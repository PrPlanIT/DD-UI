# Logging System Design

- **Status:** partially-implemented. Real-time container log access exists today (SSE streaming, per-container/host/stack), and DD-UI already has the SSE plumbing this reuses. What remains planned: the **DB-backed aggregation** (centralized `log_entries` storage, cross-host query/filter), the **retention/size-pruning** job, and the **dedicated Dozzle-style Logs view**. Log storage here is an *observability* store, not reconcile state — it is optional (can be disabled entirely) and never participates in reconcile.
- **Type:** plan / being built.
- **Merged from:** the Dozzle-inspired UI notes (UI component layout + retention/pruning implementation).

## Overview

A time-sequenced logging system in the spirit of Dozzle: real-time aggregation, filtering, and viewing of logs from every managed container across hosts and stacks. Streaming is **SSE-based**, matching DD-UI's existing deployment/log stream transport.

## Data model

```go
type LogEntry struct {
    ID          int64             `json:"id"`
    Timestamp   time.Time         `json:"timestamp"`
    HostName    string            `json:"hostname"`
    StackName   string            `json:"stack_name,omitempty"`
    ServiceName string            `json:"service_name"`
    ContainerID string            `json:"container_id"`
    Level       string            `json:"level"`  // INFO, WARN, ERROR, DEBUG
    Source      string            `json:"source"` // stdout, stderr
    Message     string            `json:"message"`
    Labels      map[string]string `json:"labels,omitempty"`
}

type LogFilter struct {
    HostNames    []string  `json:"hostnames,omitempty"`
    StackNames   []string  `json:"stacks,omitempty"`
    ServiceNames []string  `json:"services,omitempty"`
    Levels       []string  `json:"levels,omitempty"`
    Since        time.Time `json:"since,omitempty"`
    Until        time.Time `json:"until,omitempty"`
    Search       string    `json:"search,omitempty"`  // full-text
    Limit        int       `json:"limit,omitempty"`
    Follow       bool      `json:"follow,omitempty"`  // real-time
}
```

### Storage schema

Optional — only when DB-backed retention is enabled (see config).

```sql
CREATE TABLE log_entries (
    id BIGSERIAL PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    stack_name VARCHAR(255),
    service_name VARCHAR(255) NOT NULL,
    container_id VARCHAR(64) NOT NULL,
    level VARCHAR(10) DEFAULT 'INFO',
    source VARCHAR(10) DEFAULT 'stdout',
    message TEXT NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_log_entries_timestamp ON log_entries (timestamp DESC);
CREATE INDEX idx_log_entries_hostname  ON log_entries (hostname);
CREATE INDEX idx_log_entries_stack     ON log_entries (stack_name);
CREATE INDEX idx_log_entries_service   ON log_entries (service_name);
CREATE INDEX idx_log_entries_level     ON log_entries (level);
CREATE INDEX idx_log_entries_composite ON log_entries (hostname, stack_name, timestamp DESC);
```

## Backend

### Collection service

```go
// src/api/logs.go
type LogCollector struct {
    db          *pgxpool.Pool
    subscribers map[string]chan LogEntry
    mu          sync.RWMutex
}

// StartLogCollection: for each host, stream Docker container logs,
// parse/structure entries, optionally persist, and fan out to SSE subscribers.
func (lc *LogCollector) StartLogCollection() { /* ... */ }

// SubscribeToLogs: SSE subscription with server-side filtering.
func (lc *LogCollector) SubscribeToLogs(filterID string, f LogFilter) <-chan LogEntry { /* ... */ }
```

### API endpoints

Following the hierarchical convention (`/api/{resource}/hosts/{hostname}/...`):

```
GET  /api/logs                                    # paginated history with filters
GET  /api/logs/hosts/{hostname}/stream            # host-wide SSE stream
GET  /api/logs/containers/{containerName}/stream  # container SSE stream
GET  /api/logs/stacks/{stackName}/stream          # stack SSE aggregation
POST /api/logs/search                             # advanced search
GET  /api/logs/export                             # export filtered logs
DELETE /api/logs/cleanup                          # manual cleanup (admin)
```

SSE event shape:

```typescript
interface LogStreamEvent {
  type: 'log' | 'status' | 'error';
  timestamp: string;
  source: string;                              // container name or host
  level: 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';
  message: string;
  metadata?: Record<string, any>;
}
```

### Retention and pruning

Retention accepts Go duration syntax plus two sentinels: `0`/`never` keeps forever, `disabled`/`off` skips DB storage entirely. Pruning runs both size-based (keep newest under a byte cap) and time-based passes on a background ticker.

```go
func parseLogRetention() (time.Duration, error) {
    retention := env("DD_UI_LOG_RETENTION", "7d")
    switch retention {
    case "0", "never", "infinite":
        return 0, nil  // never prune
    case "disabled", "off":
        return -1, nil // don't store logs at all
    default:
        return time.ParseDuration(retention)
    }
}

func pruneOldLogs(ctx context.Context) error {
    retention, err := parseLogRetention()
    if err != nil {
        return fmt.Errorf("invalid retention config: %w", err)
    }
    if retention < 0 {
        return nil // logging disabled
    }

    // Size-based pruning first.
    if maxSize := parseByteSize(env("DD_UI_LOG_MAX_SIZE", "10GB")); maxSize > 0 {
        if err := pruneBySize(ctx, maxSize); err != nil {
            errorLog("Size-based pruning failed: %v", err)
        }
    }

    // Then time-based.
    if retention == 0 {
        debugLog("Log retention infinite - skipping time-based prune")
        return nil
    }
    cutoff := time.Now().Add(-retention)
    result, err := db.Exec(ctx, `DELETE FROM log_entries WHERE timestamp < $1`, cutoff)
    if err != nil {
        return err
    }
    debugLog("Pruned %d log entries older than %s", result.RowsAffected(), retention)
    return nil
}

// Keep newest logs up to a byte cap; delete oldest via a cumulative-size CTE.
// Deletes down to 80% of the cap to avoid pruning on every tick.
func pruneBySize(ctx context.Context, maxBytes int64) error {
    var currentSize int64
    if err := db.QueryRow(ctx, `
        SELECT COALESCE(SUM(pg_column_size(message) + pg_column_size(metadata)), 0)
        FROM log_entries`).Scan(&currentSize); err != nil {
        return err
    }
    if currentSize <= maxBytes {
        return nil
    }
    targetSize := int64(float64(maxBytes) * 0.8)
    debugLog("Log storage at %s, max %s - pruning oldest",
        humanizeBytes(currentSize), humanizeBytes(maxBytes))
    result, err := db.Exec(ctx, `
        WITH sized_logs AS (
            SELECT id,
                   SUM(pg_column_size(message) + pg_column_size(metadata))
                       OVER (ORDER BY timestamp DESC) AS cumulative_size
            FROM log_entries
        )
        DELETE FROM log_entries
        WHERE id IN (SELECT id FROM sized_logs WHERE cumulative_size > $1)`, targetSize)
    if err != nil {
        return fmt.Errorf("size-based pruning failed: %w", err)
    }
    debugLog("Size-based pruning deleted %d entries", result.RowsAffected())
    return nil
}

// Parse "10GB", "500MB", raw bytes, etc.
func parseByteSize(size string) int64 {
    size = strings.TrimSpace(strings.ToUpper(size))
    if size == "0" || size == "" {
        return 0
    }
    mult := map[string]int64{
        "B": 1, "KB": 1024, "MB": 1 << 20, "GB": 1 << 30, "TB": 1 << 40,
    }
    for suffix, m := range mult {
        if strings.HasSuffix(size, suffix) {
            if n, err := strconv.ParseFloat(strings.TrimSuffix(size, suffix), 64); err == nil {
                return int64(n * float64(m))
            }
        }
    }
    if n, err := strconv.ParseInt(size, 10, 64); err == nil {
        return n
    }
    errorLog("Invalid size format: %s, defaulting to 10GB", size)
    return 10 << 30
}

func startLogPruneJob(ctx context.Context) {
    interval := parseDurationDefault(env("DD_UI_LOG_PRUNE_INTERVAL", "1h"), time.Hour)
    ticker := time.NewTicker(interval)
    go func() {
        for {
            select {
            case <-ctx.Done():
                ticker.Stop()
                return
            case <-ticker.C:
                if err := pruneOldLogs(ctx); err != nil {
                    errorLog("Failed to prune logs: %v", err)
                }
            }
        }
    }()
}
```

## Frontend (Dozzle-inspired)

Dozzle is Vue; DD-UI adapts the same patterns onto its existing React + TypeScript stack. The layout is a set of specialized viewers plus shared controls:

```
ui/src/components/logging/
├── ContainerLogViewer.tsx    # single container
├── HostLogViewer.tsx         # host-wide
├── StackLogViewer.tsx        # stack / compose
├── MultiLogViewer.tsx        # split-screen, multiple logs
├── GroupedLogViewer.tsx      # grouped containers
├── ServiceLogViewer.tsx      # service-specific
└── common/
    ├── LogSearch.tsx         # search + filter controls
    ├── LogSideMenu.tsx       # navigation sidebar
    ├── LogTerminal.tsx       # terminal-style display
    ├── ContainerDropdown.tsx # container selection
    ├── FuzzySearchModal.tsx  # fuzzy search overlay
    └── LogControls.tsx       # play / pause / clear
```

Core component props:

```typescript
interface LogViewerProps {
  hostName: string;
  containerName?: string;
  stackName?: string;
  filters?: LogFilter;
  realTime?: boolean;
}

interface LogSearchProps {
  onSearch: (query: string) => void;
  onFilterChange: (filters: LogFilter) => void;
  searchType: 'fuzzy' | 'regex' | 'sql';
}
```

### Display and UX

- **Color-coded levels** — ERROR red, WARN yellow, INFO blue, DEBUG gray.
- **Relative timestamps** — "2 minutes ago", exact time on hover.
- **Clickable metadata** — click hostname/stack/service to filter.
- **Message formatting** — JSON pretty-print, URL highlighting, search-term highlighting.
- **Auto-scroll with pause** — follow mode with manual scroll override; connection-status indicator.
- **Virtualized scrolling** — for large volumes; debounced search; progressive loading of history.
- **Minimal chrome, monospace** — focus on log content; keyboard navigation and configurable font size for accessibility.

### Routing and navigation

```
/logs                        # landing
/logs/hosts/{hostname}       # host logs
/logs/containers/{container} # container logs
/logs/stacks/{stack}         # stack logs
/logs/search                 # advanced search
```

Contextual entry points: a "View Logs" action from the host view, stack detail, and container cards — pre-filtering to that scope.

## Configuration

```bash
# Retention (Go duration; 0/never = keep forever; disabled/off = don't store)
DD_UI_LOG_RETENTION=7d
DD_UI_LOG_MAX_SIZE=10GB
DD_UI_LOG_PRUNE_INTERVAL=1h

# Buffering / batching
DD_UI_LOG_BUFFER_SIZE=1000     # in-memory buffer per container
DD_UI_LOG_BATCH_SIZE=100       # batch insert size
DD_UI_LOG_COMPRESSION=true     # compress old logs
```

## Build order

1. **Storage + collection** — schema/migration, Docker log collection, REST query with basic filtering, a simple time-sequenced view.
2. **Streaming + filtering** — SSE streaming, filter UI (host/stack/service/level), search with highlighting, contextual navigation.
3. **Polish + performance** — retention/pruning job, virtualized scrolling for large volumes, export, display settings.

## Considerations

- **Performance** — cursor-based pagination, proper indexing, bounded in-memory buffering, efficient SSE connection management.
- **Security** — logs access gated by user permissions; sensitive-value redaction; rate-limited streaming.
- **Reliability** — graceful handling of Docker daemon disconnects; backpressure for slow consumers; preserved ordering.

Blue-sky extensions — error-rate dashboards, alert-on-pattern, and forwarding to external systems (ELK, Splunk) — are recorded in [`../Ideas.md`](../Ideas.md), not scoped here.
