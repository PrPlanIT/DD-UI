// common/logs.go - canonical container-log types shared across the collector
// (write), the query backends (read), and the SSE handler. Kept in the leaf
// package so database/ and handlers/ both depend on them without a cycle.
package common

import "time"

// LogEntry is one container log line.
type LogEntry struct {
	ID            int64             `json:"id,omitempty"`
	Timestamp     string            `json:"timestamp"`
	HostName      string            `json:"hostname"`
	StackName     string            `json:"stack_name,omitempty"`
	ServiceName   string            `json:"service_name"`
	ContainerID   string            `json:"container_id"`
	ContainerName string            `json:"container_name,omitempty"`
	Level         string            `json:"level"`
	Source        string            `json:"source"`
	Message       string            `json:"message"`
	Labels        map[string]string `json:"labels,omitempty"`
}

// LogFilter is the canonical query model every log backend satisfies
// (builtin/loki/live): the same GUI filters map onto it regardless of source.
type LogFilter struct {
	HostNames    []string  `json:"hostnames,omitempty"`
	StackNames   []string  `json:"stacks,omitempty"`
	ServiceNames []string  `json:"services,omitempty"`
	Containers   []string  `json:"containers,omitempty"`
	Levels       []string  `json:"levels,omitempty"`
	Since        time.Time `json:"since,omitempty"`
	Until        time.Time `json:"until,omitempty"`
	Search       string    `json:"search,omitempty"`
	Limit        int       `json:"limit,omitempty"`
	Follow       bool      `json:"follow,omitempty"`
}
