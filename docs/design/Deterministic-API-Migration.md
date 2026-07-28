# Deterministic API Migration

- **Status:** implemented. This documents the endpoint restructuring that produced DD-UI's current hierarchical API. The "after" shape below is the live convention — `/api/{resource}/hosts/{hostname}/...` — and is the structure enshrined in `CLAUDE.md` ("we do NOT change this established structure"). Kept as the record of how that structure came to be and as a complete endpoint inventory.
- **Type:** design / how it works (implemented reality).

## The convention

Every resource route is normalized to a predictable hierarchy:

- **Global** (all hosts): `/api/{resource}/global` — e.g. `POST /api/scan/global`.
- **Host-scoped:** `/api/{resource}/hosts/{hostname}/{action}` — e.g. `GET /api/containers/hosts/{hostname}`.
- **Stack-scoped:** `/api/{resource}/hosts/{hostname}/stacks/{stackname}/{action}` — with a parallel `groups/{groupname}` variant for group scope.

Name-based addressing replaced opaque `{id}` stack routes, so callers act directly (`/api/iac/hosts/server1/stacks/web-app/deploy`) instead of looking an ID up first.

## Endpoint inventory and migration map

Totals at migration time: **47 endpoints — 25 kept unchanged, 22 migrated.**

### Unchanged

Authentication & health (8): `GET /api/healthz`, `GET /healthz`, `GET /api/session`, `GET /login`, `GET /auth/login`, `GET /auth/callback`, `POST /logout`, `POST /auth/logout`.

Host management (1): `GET /api/hosts`.

GitOps configuration (10): `GET|PATCH /api/gitops/global`, `GET|PATCH /api/gitops/hosts/{name}`, `GET|PATCH /api/gitops/groups/{name}`, `GET|PATCH /api/gitops/hosts/{name}/stacks/{stackname}`, `GET|PATCH /api/gitops/groups/{name}/stacks/{stackname}`.

Global IaC operations (2): `POST /api/iac/scan` (global repository scan), `POST /api/iac/stacks` (create new stack — returns `Location` header).

Global operation (1): `POST /api/inventory/reload`.

Static assets (2): `GET /assets/*`, `GET /*`.

Health/auth totals include the paired legacy+canonical forms above.

### Migrated (before → after)

Container management (8):

```
GET  /api/hosts/{name}/containers                     → GET  /api/containers/hosts/{hostname}
GET  /api/hosts/{name}/containers/{ctr}/inspect       → GET  /api/containers/hosts/{hostname}/{ctr}/inspect
POST /api/hosts/{name}/containers/{ctr}/action        → POST /api/containers/hosts/{hostname}/{ctr}/action
GET  /api/hosts/{name}/containers/{ctr}/logs          → GET  /api/containers/hosts/{hostname}/{ctr}/logs
GET  /api/hosts/{name}/containers/{ctr}/logs/stream    → GET  /api/containers/hosts/{hostname}/{ctr}/logs/stream
GET  /ws/hosts/{name}/containers/{ctr}/exec           → GET  /ws/containers/hosts/{hostname}/{ctr}/exec
GET  /api/hosts/{name}/containers/{ctr}/stats         → GET  /api/containers/hosts/{hostname}/{ctr}/stats
POST /api/hosts/{name}/containers/{ctr}/enhanced-action → POST /api/containers/hosts/{hostname}/{ctr}/enhanced-action
```

Docker resources — images, networks, volumes (6):

```
GET  /api/hosts/{name}/images          → GET  /api/images/hosts/{hostname}
POST /api/hosts/{name}/images/delete   → POST /api/images/hosts/{hostname}/delete
GET  /api/hosts/{name}/networks        → GET  /api/networks/hosts/{hostname}
POST /api/hosts/{name}/networks/delete → POST /api/networks/hosts/{hostname}/delete
GET  /api/hosts/{name}/volumes         → GET  /api/volumes/hosts/{hostname}
POST /api/hosts/{name}/volumes/delete  → POST /api/volumes/hosts/{hostname}/delete
```

Infrastructure-as-Code — host list & per-stack CRUD/file/deploy (host-scoped, with parallel `groups/{groupname}` variants):

```
GET    /api/hosts/{name}/iac                → GET    /api/iac/hosts/{hostname}
GET    /api/hosts/{name}/enhanced-iac       → GET    /api/iac/hosts/{hostname}/enhanced
GET    /api/iac/stacks/{id}                 → GET    /api/iac/hosts/{hostname}/stacks/{stackname}
PATCH  /api/iac/stacks/{id}                 → PATCH  /api/iac/hosts/{hostname}/stacks/{stackname}
DELETE /api/iac/stacks/{id}                 → DELETE /api/iac/hosts/{hostname}/stacks/{stackname}
GET    /api/iac/stacks/{id}/files           → GET    /api/iac/hosts/{hostname}/stacks/{stackname}/files
GET    /api/iac/stacks/{id}/file            → GET    /api/iac/hosts/{hostname}/stacks/{stackname}/file
POST   /api/iac/stacks/{id}/file            → POST   /api/iac/hosts/{hostname}/stacks/{stackname}/file
DELETE /api/iac/stacks/{id}/file            → DELETE /api/iac/hosts/{hostname}/stacks/{stackname}/file
POST   /api/iac/stacks/{id}/deploy          → POST   /api/iac/hosts/{hostname}/stacks/{stackname}/deploy
GET    /api/iac/stacks/{id}/deploy-stream   → GET    /api/iac/hosts/{hostname}/stacks/{stackname}/deploy-stream
POST   /api/iac/stacks/{id}/deploy-check    → POST   /api/iac/hosts/{hostname}/stacks/{stackname}/deploy-check
GET    /api/iac/stacks/{id}/deploy-force    → GET    /api/iac/hosts/{hostname}/stacks/{stackname}/deploy-force
GET    /api/scopes/{scope}/stacks/{stackname}/deploy-stream → the host/group deploy-stream routes above
```

Scanning & operations (4):

```
POST /api/scan/host/{name} → POST /api/scan/hosts/{name}
POST /api/scan/all         → POST /api/scan/global
POST /api/hosts/{name}/ssh → POST /api/ssh/hosts/{name}
POST /api/inventory/reload → unchanged (global operation)
```

## Implementation notes

**Name-to-ID resolution.** IaC operations resolve `(scopeKind, scopeName, stackName)` to the internal stack ID:

```go
func getStackID(ctx context.Context, scopeKind, scopeName, stackName string) (int64, error) {
    var id int64
    err := db.QueryRow(ctx,
        `SELECT id FROM iac_stacks WHERE scope_kind=$1 AND scope_name=$2 AND stack_name=$3`,
        scopeKind, scopeName, stackName).Scan(&id)
    return id, err
}
```

**Dual scope.** Most IaC operations expose both `hosts/{hostname}` and `groups/{groupname}` variants over the same underlying logic, differing only in scope resolution.

**Frontend impact.** Callers no longer pre-fetch an ID:

```typescript
// Old: look the ID up first
const stacks = await fetch('/api/hosts/server1/iac');
const stack = stacks.find(s => s.name === 'web-app');
await fetch(`/api/iac/stacks/${stack.id}/deploy`);

// New: address by name directly
await fetch('/api/iac/hosts/server1/stacks/web-app/deploy');
```
