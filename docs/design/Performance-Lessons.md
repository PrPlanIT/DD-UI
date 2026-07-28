# Performance Lessons — the Sept 2025 scaling experiment (10 hosts / 300+ containers)

- **Status:** historical-lessons. Nothing here is live code — the perf subsystem it dissects was reverted and never ported. What stays accurate is the *why*: the failure modes (I/O under locks, caps set at fleet size, racy semaphores) and the forward guidance. What changed since: the engine is moving to a stateless, git-driven, pull-based core, so a future scaling attempt starts from that shape, not the DB-coupled polling loop described here.
- **See also:** the pull-based refactor plan → [`plans/2026-07-28-DD-UI-Evaluate-Pull-Based-Reconciliation-and-Other-Refactors.md`](plans/2026-07-28-DD-UI-Evaluate-Pull-Based-Reconciliation-and-Other-Refactors.md).

> Distilled from an adversarial post-mortem of the abandoned perf subsystem that lived
> in the `_host-anchorage/dd-ui` sandbox. The **code** is archived losslessly at
> `/home/kai/backups/ddui-sandbox-archive-20260704.tar.gz` (and the sandbox itself).
> **None of that code was ported** — it was slower than the flat baseline. This file is
> the *why*, so the next attempt starts ahead, not at zero.

## What was tried
A backend "scaling" subsystem: `rate_limiter.go` (GlobalRateLimiter), `ssh_pool_manager.go`
(EnhancedPool), `docker_connection_manager.go` (DockerConnManager), `scan_throttle.go`,
`container_polling.go` (diff cache + `/changes`/`/batch`/`/metrics`), plus a UI adaptive
poller. Upstream **reverted all of it** — it is not in `main`.

## Root cause: it was slower BECAUSE of the "optimizations", not despite them
The load path for a single Docker op was wrapped in **four stacked global mutexes plus a
starvation-sized pool**:

```
container_polling  (one PROCESS-WIDE write lock, held across the per-container loop incl. json.Marshal)
   → rate_limiter  (global counting semaphore; docker_scan capped at 3)
      → DockerConnManager  (RWMutex — and a 3s SSH Ping is executed UNDER the read lock)
         → EnhancedPool  (MaxConnectionsPerHost=2, MaxTotalSSHConnections=20)
            → another SSH NewSession health-check UNDER the pool's read lock
```

Consequences:
- **2 conns/host × 10 hosts = 20 = the global cap.** The pool is *exactly saturated* at the
  intended fleet size, sticky (no per-op release), so the whole system is throttled to
  **2 concurrent SSH-tunneled Docker ops per host**; the 21st request queues up to 45s.
- **Network I/O under locks.** A 3s `Ping` (DockerConnManager) and an SSH `NewSession`
  (EnhancedPool) run while holding read locks that writer-preferring RWMutexes will block
  behind → **one slow/hung host stalls Docker-client acquisition for ALL hosts.**
- The container-list "differential" endpoint still fetched **all** rows every poll (only the
  response body shrank), so DB + marshal cost stayed O(all containers) — under a single
  process-wide write lock.

**The lesson:** they didn't scale the system up; they serialized it. Concurrency limits
must leave headroom above the steady-state fleet size, and **never do remote I/O (Ping,
SSH dial/session) while holding a shared lock.**

## Real bugs found (don't reintroduce these patterns)
- `rate_limiter.go`: **slot-leak race** — if ctx-cancel/timeout fires concurrently with a
  slot hand-off, `Active` is incremented for a slot nobody holds; ratchets to the cap and
  the op type **deadlocks permanently**. (Just use `golang.org/x/sync/semaphore` — weighted,
  context-aware, correct.)
- `ssh_pool_manager.go`: cap enforcement is **racy** (queue drainer checks counters that are
  only incremented later inside spawned goroutines → caps blown past); `lastUsed` is written
  under a *read* lock → **data race** (`-race` flags it).
- `docker_connection_manager.go`: the "max 10 concurrent" **semaphore is inert** — the slot
  is released microseconds after acquisition, before the client is ever used.
- `container_polling.go`: `GetContainerMetrics` returns no metrics (stub); `scope=all` is a
  TODO returning `[]`.

## What's actually worth carrying forward (ideas, not the code)
1. **Adaptive UI polling — worth ~50 lines, behind a CORRECT hook.** Upstream's flat
   `setInterval(2s)` is fine for the single-host view, but "All hosts" fans out to ~2×N
   requests every 2s *even when the tab is hidden* — a real cost. A good version:
   tab-visibility + user-idle backoff + container-count scaling, but it MUST have a
   **reentrancy guard** (no overlapping polls when a visibility-return poll races the timer)
   and an **idle→active immediate reschedule** (don't strand a returning user on a 15s
   cadence). The sandbox's inline `HostStacksView` version dropped both; its own orphaned
   `useSmartPolling.ts` had the right *shape* but shipped stale-closure `deps` bugs. Drive
   `onPoll` through a ref so the loop never re-subscribes yet always calls the latest closure.
   Discard `containerPolling.ts` entirely (WebSocket/batch code against endpoints that never
   existed).
2. **Profiling — if wanted, do it safely.** The sandbox mounted `net/http/pprof` on the
   process-global `DefaultServeMux` under `/api/debug`, *outside* the `RequireAuth` group and
   on the public `:443` listener → unauthenticated heap/goroutine/cmdline dump + 30s-CPU DoS
   (and it 404s anyway — chi can't mount `DefaultServeMux` under a sub-path). Correct form:
   `chi/middleware.Profiler()` **inside** the auth group, or a separate `127.0.0.1`-only
   listener with its own `*http.ServeMux`.
3. **`scan_throttle.go`** was the one correct file (proper TOCTOU dedup: advisory
   `CanScanHost` + write-locked re-check in `MarkScanStarted`). It contributed nothing to the
   slowness, but if per-host scan dedup is ever needed again, lift it (delete its dead,
   write-only `scanHistory` field first).

## If you revisit scaling
Measure first. The likely wins are structural, not lock-based: reuse one `http.Transport`
per SSH host (populate the unused `SSHTransport.transport` field; switch the dial closure to
`DialContext`), rebuild dead SSH clients in the pool (not via per-request retries on a
broken `*ssh.Client`), push `since`-filtering into the DB query instead of computing diffs
in-app, and shard any cache by host rather than one global lock. Headroom over 10 hosts, not
a ceiling at it.
