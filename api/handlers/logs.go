package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"dd-ui/common"
	"dd-ui/database"
	"dd-ui/services"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

// LogEntry represents a single log entry
// LogEntry / LogFilter are the canonical log types (defined in common). Aliased
// here so existing handler code keeps referring to them unqualified.
type LogEntry = common.LogEntry

type LogFilter = common.LogFilter

var (
	// Global log subscribers
	logSubscribers = make(map[string]chan LogEntry)
	subMutex       sync.RWMutex
)

// HandleLogStream handles SSE streaming of logs
func HandleLogStream(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Parse filters from query parameters
	filter := parseLogFilters(r)

	common.DebugLog("Log stream request with filter: %+v", filter)

	// Create a unique subscriber ID
	subID := fmt.Sprintf("%d", time.Now().UnixNano())
	logChan := make(chan LogEntry, 100)

	// Register subscriber
	subMutex.Lock()
	logSubscribers[subID] = logChan
	subMutex.Unlock()

	// Cleanup on disconnect
	defer func() {
		subMutex.Lock()
		delete(logSubscribers, subID)
		subMutex.Unlock()
		close(logChan)
		// Don't log disconnections as they're normal and create noise
		// common.DebugLog("Log stream subscriber %s disconnected", subID)
	}()

	// Send initial connection message
	fmt.Fprintf(w, "data: {\"type\":\"connected\",\"message\":\"Connected to log stream\"}\n\n")
	w.(http.Flusher).Flush()

	// If not following, send historical logs and return
	if !filter.Follow {
		sendHistoricalLogs(w, filter)
		return
	}

	// Live logs arrive via the background collector's broadcast — we're just a
	// subscriber (registered above). No per-request collection; history is served
	// by the LogSource path above when Follow is false.
	ctx := r.Context()

	// Stream logs to client
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Send keepalive
			fmt.Fprintf(w, ": keepalive\n\n")
			w.(http.Flusher).Flush()
		case log := <-logChan:
			// Apply filters
			if !matchesFilter(log, filter) {
				continue
			}

			// Send log entry
			data, err := json.Marshal(log)
			if err != nil {
				common.ErrorLog("Failed to marshal log entry: %v", err)
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", string(data))
			w.(http.Flusher).Flush()
		}
	}
}

// streamContainerLogs streams logs from a single container
func streamContainerLogs(ctx context.Context, cli *client.Client, hostName string, cnt container.Summary, stackName string) {
	containerName := strings.TrimPrefix(cnt.Names[0], "/")
	serviceName := cnt.Labels["com.docker.compose.service"]
	if serviceName == "" {
		serviceName = containerName
	}

	// Log at startup only, not for every message
	common.DebugLog("Starting log stream for container %s on host %s (stack: %s)", containerName, hostName, stackName)

	options := client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
		Tail:       "50", // Start with last 50 lines
	}

	reader, err := cli.ContainerLogs(ctx, cnt.ID, options)
	if err != nil {
		common.ErrorLog("Failed to get logs for container %s: %v", containerName, err)
		return
	}
	defer reader.Close()

	// Read and parse logs
	buf := make([]byte, 4096)
	for {
		select {
		case <-ctx.Done():
			// Context canceled is expected when client disconnects - don't log this as it creates noise
			// common.DebugLog("Log stream context canceled for container %s on host %s", containerName, hostName)
			return
		default:
			n, err := reader.Read(buf)
			if err != nil {
				if err == io.EOF {
					// EOF is normal when container stops or logs end - only log at trace level
					// common.DebugLog("Log stream ended for container %s on host %s", containerName, hostName)
				} else if ctx.Err() != nil {
					// Context was canceled - this is expected, don't log
					// common.DebugLog("Log stream canceled for container %s on host %s", containerName, hostName)
				} else {
					// This is an actual error
					common.ErrorLog("Error reading logs from %s on host %s: %v", containerName, hostName, err)
				}
				return
			}

			if n > 0 {
				// Parse Docker log format (first 8 bytes are header)
				if n > 8 {
					message := string(buf[8:n])

					// Parse timestamp and message
					parts := strings.SplitN(message, " ", 2)
					timestamp := time.Now().Format(time.RFC3339)
					logMessage := message

					if len(parts) >= 2 {
						// Try to parse timestamp
						if t, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
							timestamp = t.Format(time.RFC3339)
							logMessage = parts[1]
						}
					}

					// Detect log level from message
					level := detectLogLevel(logMessage)

					// Create log entry
					entry := LogEntry{
						Timestamp:     timestamp,
						HostName:      hostName,
						StackName:     stackName,
						ServiceName:   serviceName,
						ContainerID:   cnt.ID[:12],
						ContainerName: containerName,
						Level:         level,
						Source:        "stdout",
						Message:       strings.TrimSpace(logMessage),
						Labels:        cnt.Labels,
					}

					// Broadcast to all subscribers
					broadcastLog(entry)
					persistLog(entry) // builtin backend: batch-write to container_logs
				}
			}
		}
	}
}

// detectLogLevel attempts to detect the log level from the message
func detectLogLevel(message string) string {
	msgLower := strings.ToLower(message)

	if strings.Contains(msgLower, "error") || strings.Contains(msgLower, "fatal") || strings.Contains(msgLower, "panic") {
		return "ERROR"
	}
	if strings.Contains(msgLower, "warn") || strings.Contains(msgLower, "warning") {
		return "WARN"
	}
	if strings.Contains(msgLower, "debug") || strings.Contains(msgLower, "trace") {
		return "DEBUG"
	}
	return "INFO"
}

// broadcastLog sends a log entry to all subscribers
func broadcastLog(entry LogEntry) {
	subMutex.RLock()
	defer subMutex.RUnlock()

	for _, ch := range logSubscribers {
		select {
		case ch <- entry:
		default:
			// Channel full, skip
		}
	}
}

// matchesFilter checks if a log entry matches the given filter
func matchesFilter(entry LogEntry, filter LogFilter) bool {
	// Special handling: when filtering for a specific container that's NOT dd-ui-app,
	// filter out DD-UI's debug logs about stream management to reduce noise
	if len(filter.Containers) > 0 && !contains(filter.Containers, "dd-ui-app") {
		// Check if this is DD-UI's stream management noise
		if entry.ContainerName == "dd-ui-app" &&
			(strings.Contains(entry.Message, "Log stream canceled") ||
				strings.Contains(entry.Message, "Log stream ended") ||
				strings.Contains(entry.Message, "Log stream context canceled") ||
				strings.Contains(entry.Message, "subscriber") ||
				strings.Contains(entry.Message, "Starting log stream for container")) {
			// Filter out these noise messages when viewing other containers
			return false
		}
	}

	// Check levels
	if len(filter.Levels) > 0 && !contains(filter.Levels, entry.Level) {
		return false
	}

	// Check search
	if filter.Search != "" && !strings.Contains(strings.ToLower(entry.Message), strings.ToLower(filter.Search)) {
		return false
	}

	// Check hosts
	if len(filter.HostNames) > 0 && !contains(filter.HostNames, entry.HostName) {
		return false
	}

	// Check stacks
	if len(filter.StackNames) > 0 && entry.StackName != "" && !contains(filter.StackNames, entry.StackName) {
		return false
	}

	// Check containers
	if len(filter.Containers) > 0 && !contains(filter.Containers, entry.ContainerName) {
		return false
	}

	return true
}

// parseLogFilters parses filter parameters from request
func parseLogFilters(r *http.Request) LogFilter {
	filter := LogFilter{
		Follow: r.URL.Query().Get("follow") == "true",
		Limit:  100,
	}

	// Parse comma-separated values
	if hosts := r.URL.Query().Get("hostnames"); hosts != "" {
		filter.HostNames = strings.Split(hosts, ",")
	}

	if stacks := r.URL.Query().Get("stacks"); stacks != "" {
		filter.StackNames = strings.Split(stacks, ",")
	}

	if containers := r.URL.Query().Get("containers"); containers != "" {
		filter.Containers = strings.Split(containers, ",")
	}

	if levels := r.URL.Query().Get("levels"); levels != "" {
		filter.Levels = strings.Split(levels, ",")
	}

	filter.Search = r.URL.Query().Get("search")

	return filter
}

// sendHistoricalLogs sends historical logs from the database
func sendHistoricalLogs(w http.ResponseWriter, filter LogFilter) {
	ctx := context.Background()
	src := LogSourceBackend()

	// The backend (builtin Postgres today, loki later) applies host/stack/service/
	// container/level/search + the Since/Until window + Limit in-store. matchesFilter
	// is a final pass for the dd-ui-app stream-noise suppression the store can't know.
	entries, err := src.Query(ctx, filter)
	if err != nil {
		common.ErrorLog("Failed to query historical logs (%s backend): %v", src.Name(), err)
		return
	}

	count := 0
	for _, entry := range entries {
		if !matchesFilter(entry, filter) {
			continue
		}
		data, _ := json.Marshal(entry)
		fmt.Fprintf(w, "data: %s\n\n", string(data))
		count++
	}

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	common.DebugLog("Sent %d historical log entries (%s)", count, src.Name())
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// HandleGetLogSources returns available log sources (hosts, stacks, containers)
func HandleGetLogSources(w http.ResponseWriter, r *http.Request) {
	sources := struct {
		Hosts      []string `json:"hosts"`
		Stacks     []string `json:"stacks"`
		Containers []struct {
			Name  string `json:"name"`
			Host  string `json:"host"`
			Stack string `json:"stack,omitempty"`
		} `json:"containers"`
	}{}

	// Get hosts
	hosts := services.GetHosts()
	for _, h := range hosts {
		sources.Hosts = append(sources.Hosts, h.Name)
	}

	// Get containers from all hosts with timeout
	ctx := context.Background()
	for _, host := range hosts {
		hostRow := database.HostRow{Name: host.Name, Addr: host.Addr, Vars: host.Vars}
		cli, err := services.DockerClientForHost(hostRow)
		if err != nil {
			common.DebugLog("Skipping unreachable host %s for log sources: %v", host.Name, err)
			continue
		}
		defer cli.Close()

		// Use timeout for container list to avoid hanging on slow hosts
		listCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		containers, err := cli.ContainerList(listCtx, client.ContainerListOptions{All: false})
		cancel()

		if err != nil {
			common.DebugLog("Failed to list containers on %s for log sources: %v", host.Name, err)
			continue
		}

		for _, cnt := range containers.Items {
			containerName := strings.TrimPrefix(cnt.Names[0], "/")
			stackName := cnt.Labels["com.docker.compose.project"]

			sources.Containers = append(sources.Containers, struct {
				Name  string `json:"name"`
				Host  string `json:"host"`
				Stack string `json:"stack,omitempty"`
			}{
				Name:  containerName,
				Host:  host.Name,
				Stack: stackName,
			})

			// Add unique stack names
			if stackName != "" && !contains(sources.Stacks, stackName) {
				sources.Stacks = append(sources.Stacks, stackName)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sources)
}
