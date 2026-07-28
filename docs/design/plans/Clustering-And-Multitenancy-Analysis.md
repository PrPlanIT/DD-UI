# Clustering and Multi-tenancy Analysis

- **Status:** superseded-parts-noted. The **value here is the inventory** — the enumeration of stateful components that block horizontal scaling, and the honest gap analysis for multi-tenancy. That inventory is still accurate against the DB-coupled form of the app. What is **superseded is the remedy**: the original recommendation was to add Redis (shared ID-token store, distributed locks, view-state) plus sticky sessions. The project has instead moved toward a **stateless, git-driven, pull-based** core, which removes most of these blockers *by design* rather than by adding a coordination store. The Redis migration phases below are retained only as a marked-superseded record.
- **Type:** plan / assessment feeding the stateless refactor.
- **Original assessment date:** September 16, 2025.

## Scores

- **Clustering readiness: 7/10** — good foundations, a handful of concrete stateful blockers.
- **Multi-tenancy readiness: 2/10** — single-tenant today; true multi-tenancy is a large, aspirational effort (see [`../Ideas.md`](../Ideas.md), "Enterprise hub / multi-tenancy").

## Clustering — strengths

1. **Session management** — secure cookie sessions (`scs`); no server-side session store; survives pod restarts/migrations.
2. **Database architecture** — external PostgreSQL, `pgxpool` pooling, configurable pool settings; persistent data in the DB.
3. **Stateless API design** — most endpoints are stateless and RESTful; no file-based storage for core operations.

## Clustering — blockers (the inventory that matters)

Each is a component holding runtime state inside a single process, so a second replica diverges.

1. **In-memory ID token store** — `auth.go`: a global `map[sessionID]idToken` guarded by a mutex. Breaks OIDC logout across replicas.
2. **View-boost performance tracker** — `main.go`: in-memory `activeViews` map + per-view timers. Boost state is lost on replica switch.
3. **Inventory cache** — `services/inventory.go`: in-memory `hosts []Host` plus a file watch on `/data/inventory`. Diverges across replicas.
4. **Background scanners and timers** — `main.go`: per-host scan timers run independently per process, with no coordination, so replicas duplicate scanning work.
5. **WebSocket / SSE connections** — container console (WebSocket), deployment and log streams (SSE). Tied to a specific process; connections drop on replica switch, with no reconnection/handoff.

### File-system dependencies

- **Temporary SOPS decryption** — uses local `/tmp` (`ddui-builds-*`). Clustering-compatible but needs pod-local temp space.
- **Inventory file watching** — watches `/data/inventory`; needs shared config or a non-file source.

### State summary

| Component | Implementation | Clustering impact | Fix complexity |
|---|---|---|---|
| Session storage | Cookie-based (scs) | None | N/A |
| ID token store | In-memory map | High | Medium |
| View-boost tracker | In-memory + timers | Medium | Low |
| Inventory cache | In-memory + file watch | High | Medium |
| WebSocket/SSE sessions | Process-local | Medium | Medium |
| Background scanners | Uncoordinated timers | Medium | High |
| DB connections | Pooled (pgxpool) | None | N/A |

## How the stateless direction resolves these

The blocker inventory above is precisely what the move to a stateless, git-driven, pull-based engine addresses without a coordination store:

- **Reconcile carries no shared runtime state.** Desired state lives in git; actual state is read live from the Docker API; the last-applied stamp is a container label. There is nothing per-process to share, so items that exist only to cache or coordinate reconcile (inventory cache, background scan timers) stop being cross-replica hazards — the source of truth is external.
- **Coordination moves to the engine's own daemon, not Redis.** A single pull-based reconcile daemon (leader, or one process at homelab scale) owns the scan loop; there is no fleet of replicas racing to scan the same host, so distributed locks aren't needed for correctness.
- **What genuinely remains process-local is the control-plane surface, not reconcile:** the OIDC ID-token store and live WebSocket/SSE streams. These are the real residual clustering work, and they are solvable with a small shared store or sticky routing *if and when* HA of the UI itself is required — but they are control-plane concerns, never reconcile.

### Superseded remedy (record only)

The original plan proposed, for clustering: add a Redis service; move the ID-token store to Redis with TTL; use `redsync` distributed locks around per-host scanners; replace the view-boost tracker with Redis+TTL; and configure `ip_hash` sticky sessions for WebSocket affinity. **This is not the chosen direction** — it treats shared runtime state as a given and adds infrastructure to coordinate it, whereas the stateless engine removes the shared state. Retained here so the trade-off isn't re-litigated without new reasons.

## Multi-tenancy — gap analysis

DD-UI is single-tenant today, with no tenant isolation. The gaps are real and large; genuine multi-tenancy is an aspirational, paid/SaaS direction (see [`../Ideas.md`](../Ideas.md)), not near-term work.

- **Auth model** — `User{Sub,Email,Name,Pic}` has no tenant/organization context.
- **Database schema** — no `tenant_id` columns, no row-level security; all authenticated users see all data.
- **API structure** — no tenant scoping (`/api/hosts`, not `/api/tenants/{id}/hosts`).
- **Resource access** — all hosts visible to all users; shared secrets/config; global inventory.
- **Data isolation** — no per-tenant query filtering, no tenant-context middleware.

| Feature | Current | Required | Effort |
|---|---|---|---|
| User model | Single-tenant | tenant_id, roles | Low |
| DB schema | No isolation | tenant_id on all tables | High |
| API routes | Global | tenant middleware | Medium |
| Query filtering | None | scope all queries | High |
| Resource access | Shared | per-tenant isolation | High |
| Secrets | Global | tenant-scoped | Medium |
| Audit logging | Basic | tenant-aware | Medium |

If multi-tenancy is ever pursued, the shape is well-understood: add a `tenants` table and `tenant_id` columns, enable Postgres row-level security keyed on `current_setting('app.tenant_id')`, carry tenant context in the user model and a request middleware, and scope every query. This is a months-long effort justified only by a concrete SaaS need.

## Recommended action

1. **Immediate** — treat the blocker inventory as the checklist the stateless refactor must satisfy; the residual process-local items (ID-token store, live streams) are the only ones needing a clustering answer.
2. **Short-term** — validate HA against the stateless engine, not against a Redis-coordinated copy of the current app.
3. **Long-term** — multi-tenancy only if a SaaS model is committed to; tracked as an idea, not a plan.
