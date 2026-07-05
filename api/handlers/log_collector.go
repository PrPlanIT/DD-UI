// log_collector.go - the continuous background log collector for the builtin
// backend. Unlike the old per-viewer collection, this runs from boot: it keeps a
// log stream open for every container on every inventory host, broadcasts to live
// SSE subscribers, and (for the builtin backend) batch-persists to container_logs
// so historical search actually has data. A dead host is skipped fast via a
// per-host Ping timeout (mirroring the scan path) and retried on the next tick.
package handlers

import (
	"context"
	"os"
	"strings"
	"sync"
	"time"

	"dd-ui/common"
	"dd-ui/database"
	"dd-ui/services"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

var (
	logPersistQueue  chan common.LogEntry
	logCollectorOnce sync.Once
)

// StartLogCollector launches the collector once. Idempotent; safe to call at boot.
func StartLogCollector(ctx context.Context) {
	logCollectorOnce.Do(func() {
		if !logCollectionEnabled() {
			common.InfoLog("Log collector disabled (DD_UI_LOG_COLLECTION=false)")
			return
		}
		persist := logBackend() == "builtin"
		if persist {
			logPersistQueue = make(chan common.LogEntry, 10000)
			go runLogWriter(ctx)
			go runLogRetention(ctx)
		}
		go runHostCollectionManager(ctx)
		common.InfoLog("Log collector started (backend=%s persist=%v retention=%s)",
			logBackend(), persist, logRetention())
	})
}

// persistLog enqueues an entry for the batched writer. Non-blocking: if the queue
// is saturated we drop rather than stall the stream or balloon memory.
func persistLog(entry common.LogEntry) {
	q := logPersistQueue
	if q == nil {
		return
	}
	select {
	case q <- entry:
	default:
	}
}

// runHostCollectionManager (re)establishes per-host collection so new/removed
// inventory hosts are picked up. One goroutine per host, cancelled when a host
// leaves the inventory.
func runHostCollectionManager(ctx context.Context) {
	active := map[string]context.CancelFunc{}
	tick := time.NewTicker(60 * time.Second)
	defer tick.Stop()
	reconcile := func() {
		hosts := services.GetHosts()
		seen := map[string]bool{}
		for _, h := range hosts {
			seen[h.Name] = true
			if _, ok := active[h.Name]; ok {
				continue
			}
			hctx, cancel := context.WithCancel(ctx)
			active[h.Name] = cancel
			go collectHostContinuously(hctx, h)
		}
		for name, cancel := range active {
			if !seen[name] {
				cancel()
				delete(active, name)
			}
		}
	}
	reconcile()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			reconcile()
		}
	}
}

// collectHostContinuously keeps one Docker client per host, re-lists containers
// periodically, and streams each. On a ping failure the host is treated as down:
// the client is dropped and all its streams cancelled, then retried next tick.
func collectHostContinuously(ctx context.Context, host common.Host) {
	hostRow := database.HostRow{Name: host.Name, Addr: host.Addr, Vars: host.Vars}
	// streaming is owned SOLELY by this goroutine; stream goroutines report
	// completion on `ended` (a channel), never touching the map themselves.
	streaming := map[string]context.CancelFunc{}
	ended := make(chan string, 64)
	var cli *client.Client
	drop := func() {
		for id, c := range streaming {
			c()
			delete(streaming, id)
		}
		if cli != nil {
			cli.Close()
			cli = nil
		}
	}
	defer drop()

	tick := time.NewTicker(30 * time.Second)
	defer tick.Stop()
	for {
		if cli == nil {
			if c, err := services.DockerClientForHost(hostRow); err == nil {
				cli = c
			}
		}
		if cli != nil {
			if !hostAlive(ctx, cli) {
				drop() // host down — reconnect next tick
			} else {
				refreshHostStreams(ctx, cli, host, streaming, ended)
			}
		}
		select {
		case <-ctx.Done():
			return
		case id := <-ended:
			delete(streaming, id) // single-owner delete; next refresh re-picks it up if still running
		case <-tick.C:
		}
	}
}

// hostAlive pings under a per-host timeout so an unresponsive host is detected in
// seconds instead of hanging collection (the gap the log path never had).
func hostAlive(ctx context.Context, cli *client.Client) bool {
	pctx, cancel := context.WithTimeout(ctx, logHostTimeout())
	defer cancel()
	_, err := cli.Ping(pctx, client.PingOptions{})
	return err == nil
}

// refreshHostStreams starts a stream for each not-yet-streamed container and
// cancels streams for containers that have gone away.
func refreshHostStreams(ctx context.Context, cli *client.Client, host common.Host, streaming map[string]context.CancelFunc, ended chan<- string) {
	lctx, cancel := context.WithTimeout(ctx, logHostTimeout())
	list, err := cli.ContainerList(lctx, client.ContainerListOptions{All: false})
	cancel()
	if err != nil {
		return
	}
	seen := map[string]bool{}
	for _, cnt := range list.Items {
		seen[cnt.ID] = true
		if _, ok := streaming[cnt.ID]; ok {
			continue
		}
		stackName := cnt.Labels["com.docker.compose.project"]
		cctx, ccancel := context.WithCancel(ctx)
		streaming[cnt.ID] = ccancel
		go func(c container.Summary, sn, id string) {
			streamContainerLogs(cctx, cli, host.Name, c, sn)
			// stream ended (container stopped / EOF) — report so the owner can
			// re-pick it up next refresh if it's still running.
			select {
			case ended <- id:
			case <-cctx.Done():
			}
		}(cnt, stackName, cnt.ID)
	}
	for id, ccancel := range streaming {
		if !seen[id] {
			ccancel()
			delete(streaming, id)
		}
	}
}

// runLogWriter batches queued entries and flushes via COPY every second or every
// 500 rows, whichever comes first.
func runLogWriter(ctx context.Context) {
	batch := make([]common.LogEntry, 0, 500)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		wctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		if _, err := database.InsertContainerLogs(wctx, batch); err != nil {
			common.ErrorLog("log persist: %v", err)
		}
		cancel()
		batch = batch[:0]
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			flush()
			return
		case e := <-logPersistQueue:
			batch = append(batch, e)
			if len(batch) >= 500 {
				flush()
			}
		case <-tick.C:
			flush()
		}
	}
}

// runLogRetention prunes logs older than the retention window every 10 minutes.
func runLogRetention(ctx context.Context) {
	tick := time.NewTicker(10 * time.Minute)
	defer tick.Stop()
	prune := func() {
		wctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		n, err := database.PruneContainerLogs(wctx, time.Now().Add(-logRetention()))
		cancel()
		if err != nil {
			common.ErrorLog("log retention prune: %v", err)
		} else if n > 0 {
			common.DebugLog("log retention pruned %d rows", n)
		}
	}
	prune()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			prune()
		}
	}
}

// ── config ──────────────────────────────────────────────────────────────────

func logBackend() string {
	if b := strings.ToLower(strings.TrimSpace(os.Getenv("DD_UI_LOG_BACKEND"))); b != "" {
		return b
	}
	return "builtin"
}

func logCollectionEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("DD_UI_LOG_COLLECTION")))
	return v != "false" && v != "0" && v != "off"
}

func logRetention() time.Duration {
	if v := os.Getenv("DD_UI_LOG_RETENTION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 48 * time.Hour
}

func logHostTimeout() time.Duration {
	if v := os.Getenv("DD_UI_SCAN_DOCKER_HOST_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return 10 * time.Second
}
