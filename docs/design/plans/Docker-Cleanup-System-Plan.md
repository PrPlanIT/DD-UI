# Docker Cleanup System Plan

- **Status:** partially-implemented. Cleanup endpoints exist and follow the hierarchical API convention (`/api/cleanup/hosts/{hostname}/...`, `/api/cleanup/global/...` — see `CLAUDE.md`). What remains planned: the **dedicated cleanup UI**, **job tracking/history**, and the later phases (scheduling, custom policies). Job tracking is an *observability* concern (a job queue), not reconcile state.
- **Type:** plan / partially built.
- **Merged from:** the cleanup UI requirements (dedicated page, build-cache prominence, host dropdown).

## Overview

Give administrators safe, targeted control over Docker disk reclamation — `system prune`, image/container/volume/network prune, and build-cache prune — per host or across the fleet, with dry-run previews and confirmation.

## Operations

- **System prune** — `docker system prune -af` (all unused containers, networks, images, build cache).
- **Image prune** — `docker image prune -af`.
- **Container prune** — `docker container prune -f` (stopped containers).
- **Volume prune** — `docker volume prune -f`.
- **Network prune** — `docker network prune -f`.
- **Build-cache prune** — `docker buildx prune -af`.
- **Complete cleanup** — all of the above in sequence.

Scope: a single host, all hosts in parallel, or (future) host groups.

## Safety model

- **Dry run** — preview what would be removed, with space-reclamation estimates; no Docker mutation performed.
- **Confirmation required** — destructive operations require an explicit confirmation token; `force` mode adds warnings.
- **Exclude filters** — protect specific images/containers/volumes/networks from cleanup.
- **Default excludes** — DD-UI's own containers, base/runtime images, volumes named `*data*`/`*storage*`, networks named `*prod*`.
- **Per-host error isolation** — partial success is a first-class outcome; one host failing doesn't abort the others.
- **Audit + rate limiting** — cleanup actions are logged and rate-limited.

## API

Hierarchical routes (host-scoped and global), plus job status/streaming:

```
POST /api/cleanup/hosts/{hostname}/{operation}   # system|images|containers|volumes|networks|build-cache|complete
POST /api/cleanup/global/{operation}             # same operations, all hosts in parallel
GET  /api/cleanup/hosts/{hostname}/preview       # preview reclaimable data (single host)
GET  /api/cleanup/global/preview                 # preview across all hosts
GET  /api/cleanup/jobs/{jobId}                   # job status
GET  /api/cleanup/jobs/{jobId}/stream            # SSE progress stream
```

Request:

```json
{
  "dry_run": false,
  "force": false,
  "exclude_filters": {
    "images": ["mysql:*", "postgres:*"],
    "containers": ["important-*"],
    "volumes": ["data-*"],
    "networks": ["production-*"]
  },
  "confirmation_token": "user-provided-random-string"
}
```

Response carries `job_id`, `operation`, `scope`, `target`, `status`, timestamps, a `progress` object (total/completed hosts, current host + operation), and per-host `results` (`space_reclaimed`, `items_removed`, `errors`).

## Backend

- **`src/api/cleanup.go`** — core prune operations via the Docker API; per-host functions; parallel all-hosts execution; progress reporting.
- **`src/api/cleanup_jobs.go`** — job queue for long-running operations; status tracking; SSE progress streaming.

```go
func performSystemPrune(ctx context.Context, host string, opts CleanupOptions) (*CleanupResult, error)
func performImagePrune(ctx context.Context, host string, opts CleanupOptions) (*CleanupResult, error)
// ...container / volume / network variants...
func performCleanupAllHosts(ctx context.Context, operation string, opts CleanupOptions) (jobID string, err error)

func createCleanupJob(ctx context.Context, operation, scope, target, owner string, opts CleanupOptions) (*CleanupJob, error)
func updateJobProgress(ctx context.Context, jobID string, progress map[string]any) error
func getCleanupJob(ctx context.Context, jobID string) (*CleanupJob, error)
```

Job tracking (optional, observability only):

```sql
CREATE TABLE cleanup_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operation VARCHAR(50) NOT NULL,
    scope VARCHAR(20) NOT NULL,      -- 'single_host' | 'all_hosts'
    target VARCHAR(100) NOT NULL,    -- hostname | 'all'
    status VARCHAR(20) DEFAULT 'queued',
    dry_run BOOLEAN DEFAULT false,
    force BOOLEAN DEFAULT false,
    exclude_filters JSONB,
    created_at TIMESTAMPTZ DEFAULT now(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    progress JSONB DEFAULT '{}',
    results JSONB DEFAULT '{}',
    owner VARCHAR(200) NOT NULL
);
CREATE INDEX idx_cleanup_jobs_status  ON cleanup_jobs(status);
CREATE INDEX idx_cleanup_jobs_created ON cleanup_jobs(created_at DESC);
CREATE INDEX idx_cleanup_jobs_owner   ON cleanup_jobs(owner);
```

## Frontend

The cleanup UI is a **dedicated page for pruneable data**, deliberately separate from the existing volumes/images resource-management pages — its job is disk-space reclamation, not resource browsing.

- **Dedicated route** — its own page and clear nav entry; not folded into resource pages.
- **Build cache is the headline** — build cache and other temporary artifacts are the primary, most-prominent target; system prune is secondary; the rest are additional options.
- **Per-host granularity** — a host-selection dropdown following DD-UI's existing pattern, plus an "All Hosts" bulk option and a clear indication of the selected target.
- **Minimal, focused interface** — uncluttered, optimized for quick, safe cleanup; prominent disk-space info; clear separation between cleanup types.

Components:

- **`CleanupPanel.tsx`** — operation selection, host dropdown, safety options (dry run / force / exclude filters), confirmation dialog with warnings.
- **`CleanupProgress.tsx`** — real-time SSE progress, per-host results, space-reclaimed stats, error reporting.
- **`CleanupHistory.tsx`** — past jobs, filterable by date/operation/host, with detail view.

UX goals: quick access, clear targets, safe operations (explicit warnings/confirmation for destructive actions), live progress, and a results summary of space reclaimed.

## Configuration

```bash
DD_UI_CLEANUP_TIMEOUT=300s
DD_UI_CLEANUP_PARALLEL_HOSTS=5
DD_UI_CLEANUP_REQUIRE_CONFIRMATION=true
DD_UI_CLEANUP_DEFAULT_DRY_RUN=true
DD_UI_CLEANUP_MAX_CONCURRENT_JOBS=3
DD_UI_CLEANUP_HISTORY_DAYS=30
```

## Build order

1. **Core backend** — cleanup functions in `cleanup.go`, single-host endpoints, job queue.
2. **Multi-host** — parallel all-hosts execution with per-host error handling, SSE progress, safety features.
3. **Frontend** — the dedicated cleanup page (build-cache-forward), live progress, history viewer.

## Testing

- **Unit** — core prune functions against mock Docker clients; job-queue management; exclude-filter validation; error paths.
- **Integration** — full workflows, multi-host runs, SSE streaming, job tracking.
- **Safety** — dry-run accuracy, exclude-filter effectiveness, confirmation enforcement.

Speculative directions — usage-pattern/ML cleanup recommendations, predictive space management, and cross-platform cleanup (Kubernetes, registries, cloud storage) — are recorded in [`../Ideas.md`](../Ideas.md), not scoped here.
