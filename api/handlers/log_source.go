// log_source.go - the pluggable historical-log backend. The live SSE stream is
// served by the collector's broadcast regardless of backend; this seam only
// governs where history/search comes from, so builtin (Postgres) today and loki
// (query logging.pcfae.com) later are swappable behind one interface + config
// (DD_UI_LOG_BACKEND). The GUI + LogFilter model stay identical across backends.
package handlers

import (
	"context"
	"sync"

	"dd-ui/common"
	"dd-ui/database"
)

// LogSource answers historical/filter queries against whatever store backs logs.
type LogSource interface {
	Query(ctx context.Context, f common.LogFilter) ([]common.LogEntry, error)
	Name() string
}

var (
	logSourceOnce sync.Once
	logSource     LogSource
)

// LogSourceBackend resolves the configured historical backend (default builtin).
func LogSourceBackend() LogSource {
	logSourceOnce.Do(func() {
		switch logBackend() {
		case "loki":
			// TODO: loki adapter over logging.pcfae.com. Until wired, no history.
			logSource = liveSource{}
		case "live":
			logSource = liveSource{}
		default:
			logSource = builtinSource{}
		}
	})
	return logSource
}

// builtinSource reads dd-ui's own persisted logs from Postgres.
type builtinSource struct{}

func (builtinSource) Name() string { return "builtin" }
func (builtinSource) Query(ctx context.Context, f common.LogFilter) ([]common.LogEntry, error) {
	return database.QueryContainerLogs(ctx, f)
}

// liveSource keeps no history — queries return empty; the UI still has the live
// stream. Placeholder for `live` and (until wired) `loki`.
type liveSource struct{}

func (liveSource) Name() string { return "live" }
func (liveSource) Query(context.Context, common.LogFilter) ([]common.LogEntry, error) {
	return nil, nil
}
