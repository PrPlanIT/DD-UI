# Ideas — Aspirational / Blue-Sky

> **Status: NOT committed. Possibly never.** This is the junk drawer for directions we've *considered* and want to remember — not a plan, not a promise. Anything that becomes real graduates to [`plans/`](plans/); anything that describes how something actually works graduates to a design doc. If it's still speculative, it lives here.

## Container runtimes beyond Docker

- **Docker Swarm** — a stated long-term goal. Mechanism is TBD: Swarm's *service* model doesn't map 1:1 onto our stack-based reconcile, so this needs real design before it's more than an intention.
- **Podman** — likely cheap: it exposes a Docker-compatible socket, so it may slot in as another runtime target behind the same interface with little work.

## Scale-out (only if a real need appears)

- **Cross-node application scaling / coordinating a service across unlinked nodes** — the engine already reaches many hosts; the blue-sky version coordinates one service *across* them. Intriguing, but the need is unproven. No design until a concrete use-case exists.
- **Multi-component / microservice split** (Argo-style: separate controller / API / UI processes) — for horizontal scale. The single-binary design doesn't preclude splitting later; it may never be warranted at homelab scale.

## Considered and set aside

- **Peer mesh of agents** — set aside in favor of **hub-and-spoke** (which is what Flux and Argo themselves do). Recorded so we don't re-litigate it without new reasons.
- **Redis-shared-state clustering** — an early clustering plan proposed a shared Redis (ID-token store, distributed scan locks, view-state) plus sticky WebSocket sessions to run multiple replicas. Rejected in favor of the stateless, git-driven, pull-based direction: reconcile carries no shared runtime state to coordinate, so the blockers are removed by design rather than by adding a coordination store. See [`plans/Clustering-And-Multitenancy-Analysis.md`](plans/Clustering-And-Multitenancy-Analysis.md).

## From legacy planning docs

Directions surfaced while curating the old planning docs. Recorded here because they're aspirational, not committed.

### Pluggable secret backends

SOPS stays the default open-source secret provider. Beyond it, a pluggable secret-backend interface could let deployments resolve secrets from external managers — Infisical, Bitwarden, 1Password Connect, HashiCorp Vault, AWS Secrets Manager, Azure Key Vault, Google Secret Manager — with a small SDK for organization-specific providers and per-environment backend selection (different provider for dev/staging/prod). The interesting part is the seam: one `SecretsProvider` interface with SOPS as the reference implementation. Nothing here is built.

### Enterprise hub / multi-tenancy

A paid "hub" direction, from the remote-operations notes and the clustering analysis: each DD-UI install acts as a management hub that connects to and manages remote/foreign DD-UI servers from one UI — federated container/stack operations, aggregate monitoring across instances, per-customer environment isolation, remote authentication, and usage/billing. This is the SaaS end state the clustering analysis scores 2/10 against today; it needs `tenant_id` isolation, row-level security, and tenant-scoped queries before it's more than an intention. Hub-and-spoke (not a peer mesh) is the assumed shape — consistent with the note above.

### Cleanup and logging blue-sky

- **Intelligent cleanup** — usage-pattern analysis and predictive/space-management recommendations, scheduled per-host cleanup policies, and cross-platform cleanup (Kubernetes clusters, container registries, cloud storage). The concrete prune operations are a real plan ([`plans/Docker-Cleanup-System-Plan.md`](plans/Docker-Cleanup-System-Plan.md)); these are the speculative tail.
- **Log analytics** — error-rate dashboards, alert-on-pattern, and forwarding logs to external systems (ELK, Splunk). The core viewer and retention are planned ([`plans/Logging-System-Design.md`](plans/Logging-System-Design.md)); these forwarders and analytics are aspirational.
