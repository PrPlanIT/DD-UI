// db_container_logs.go - persistence + query for the builtin log backend.
// The continuous collector batch-writes here via COPY; the historical/filter
// query reads here honoring the canonical LogFilter (Since/Until/host/stack/
// service/container/level/search). Retention prunes by timestamp.
package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dd-ui/common"

	"github.com/jackc/pgx/v5"
)

// InsertContainerLogs batch-inserts entries via COPY (the collector's fast path).
// No-op on an empty batch.
func InsertContainerLogs(ctx context.Context, entries []common.LogEntry) (int64, error) {
	if len(entries) == 0 {
		return 0, nil
	}
	rows := make([][]any, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, []any{
			parseLogTS(e.Timestamp), e.HostName, logNull(e.StackName), e.ServiceName,
			e.ContainerID, logNull(e.ContainerName), logDefault(e.Level, "INFO"),
			logDefault(e.Source, "stdout"), e.Message,
		})
	}
	return common.DB.CopyFrom(ctx,
		pgx.Identifier{"container_logs"},
		[]string{"timestamp", "hostname", "stack_name", "service_name",
			"container_id", "container_name", "level", "source", "message"},
		pgx.CopyFromRows(rows),
	)
}

// buildLogWhere translates a LogFilter into parameterized WHERE conditions + args:
// host/stack/service/container/level as exact `= ANY`, search as ILIKE, and a time
// window that defaults to the last hour when Since is unset. `now` is a parameter
// so the default window is deterministic in tests.
func buildLogWhere(f common.LogFilter, now time.Time) ([]string, []any) {
	var conds []string
	var args []any
	add := func(tmpl string, val any) {
		args = append(args, val)
		conds = append(conds, fmt.Sprintf(tmpl, len(args)))
	}
	if !f.Since.IsZero() {
		add("timestamp >= $%d", f.Since)
	} else {
		add("timestamp >= $%d", now.Add(-time.Hour))
	}
	if !f.Until.IsZero() {
		add("timestamp <= $%d", f.Until)
	}
	if len(f.HostNames) > 0 {
		add("hostname = ANY($%d)", f.HostNames)
	}
	if len(f.StackNames) > 0 {
		add("stack_name = ANY($%d)", f.StackNames)
	}
	if len(f.ServiceNames) > 0 {
		add("service_name = ANY($%d)", f.ServiceNames)
	}
	if len(f.Containers) > 0 {
		add("container_name = ANY($%d)", f.Containers)
	}
	if len(f.Levels) > 0 {
		add("level = ANY($%d)", f.Levels)
	}
	if f.Search != "" {
		add("message ILIKE $%d", "%"+f.Search+"%")
	}
	return conds, args
}

// QueryContainerLogs returns historical logs matching the filter, newest first,
// bounded by the time window (Since/Until; defaults to the last hour) and Limit.
func QueryContainerLogs(ctx context.Context, f common.LogFilter) ([]common.LogEntry, error) {
	conds, args := buildLogWhere(f, time.Now())

	limit := f.Limit
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}
	q := `SELECT id, timestamp, hostname, stack_name, service_name, container_id,
	             container_name, level, source, message
	      FROM container_logs`
	if len(conds) > 0 {
		q += " WHERE " + strings.Join(conds, " AND ")
	}
	q += fmt.Sprintf(" ORDER BY timestamp DESC LIMIT %d", limit)

	rows, err := common.DB.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []common.LogEntry
	for rows.Next() {
		var e common.LogEntry
		var ts time.Time
		var stack, cname *string
		if err := rows.Scan(&e.ID, &ts, &e.HostName, &stack, &e.ServiceName,
			&e.ContainerID, &cname, &e.Level, &e.Source, &e.Message); err != nil {
			return nil, err
		}
		e.Timestamp = ts.Format(time.RFC3339Nano)
		if stack != nil {
			e.StackName = *stack
		}
		if cname != nil {
			e.ContainerName = *cname
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// PruneContainerLogs deletes rows older than the cutoff; returns rows removed.
func PruneContainerLogs(ctx context.Context, before time.Time) (int64, error) {
	tag, err := common.DB.Exec(ctx, `DELETE FROM container_logs WHERE timestamp < $1`, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func parseLogTS(s string) time.Time {
	if s != "" {
		if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
			return t
		}
	}
	return time.Now()
}

func logNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func logDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
