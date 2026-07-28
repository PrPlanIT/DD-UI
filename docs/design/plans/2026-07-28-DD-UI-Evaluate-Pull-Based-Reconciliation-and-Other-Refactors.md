# DD-UI — Evaluate Pull-Based Reconciliation and Other Refactors

- **Date:** 2026-07-28
- **Status:** Proposal / evaluation — direction agreed in discussion, not yet a locked spec.
- **Scope:** The architecture DD-UI should move toward, and the DD-UI ↔ StageFreight boundary.
- **One-line:** Re-base DD-UI on a stateless, GitOps-native, pull-based Docker reconcile core (`dd`), keep the rich app features in a stateful control plane, and let StageFreight *invoke* that core the way it invokes Flux — rather than embedding a Docker engine of its own.

---

## 1. Why now

DD-UI works, but it's at the fork where a project either gets modernized or drifts to abandonware. The debt that matters:

- The **reconcile engine is DB-coupled** (`pgxpool`, `stackID`) — it can't run as a portable CLI or a bare-node agent without dragging Postgres along.
- **Git reconciliation is jank** (manual/one-shot sync), not a proper pull loop.
- The **multi-host view polls like crazy** — it refreshes details everywhere instead of only what's visible.
- The **boundary with StageFreight is blurred**: DD-UI's Docker reconcile code was extracted into StageFreight's `src/docker/`, leaving *two* engines and an unresolved "who owns this" question.

This doc resolves those by borrowing the model Flux and Argo already proved.

## 2. Guiding principle

**The engine is the product; the UI is a thin client over it.** (Same principle we're applying to DisplayWizard — it's becoming the house pattern.)

And the GitOps corollary: **Git is the desired-state store, the target system is the actual-state store. Reconcile needs no third store.** Neither Flux nor Argo runs a database; we won't either — *for reconcile*.

## 3. Target architecture

Three parts, cleanly separated:

```
        commit / MR                 invoke (toolchain)
DD-UI ───────────────▶  git repo  ◀───────────────── StageFreight (CI, slim)
(control plane)             │                          │
   │ drives                 │ pull                      │ push-based reconcile
   ▼                        ▼                           ▼
  dd  ◀───────────────  dd serve  ───────────────▶  Docker hosts
 (CLI)                (controller)                  (actual state)
```

- **`dd` — the Docker reconcile tool.** One multi-call binary (Argo model), but a *single process* is fine at homelab scale — no need for Argo's component split.
  - `dd serve` — pull-based always-on controller **+** API **+** UI, in one process.
  - `dd reconcile | get | apply | bootstrap …` — the CLI; talks to a daemon, local or remote (`--server https://…`), kubectl-style.
  - **Stateless core:** desired = git, actual = Docker API, last-applied = container labels. No DB.
- **DD-UI — the stateful control plane** over `dd`: auth/local-login, multitenancy, RBAC, audit, log history, cached views, and the git-write-back UX. Owns a DB — but only for *control-plane* concerns, never reconcile.
- **StageFreight — the slim orchestrator.** Its `docker` lifecycle backend *invokes* `dd` (added to SF's toolchain), exactly as its `gitops` backend invokes Flux. SF does **not** embed a Docker engine.

**The symmetry that makes this correct:** `SF → flux` (K8s CD) and `SF → dd` (Docker CD) are the same shape. SF stays true to its invoke-don't-absorb identity.

## 4. State model (the "no DB for reconcile" detail)

| Concern | Store | Notes |
|---|---|---|
| Desired state | **Git** (compose/`.env` IaC, SOPS-encrypted) | The source of truth |
| Actual state | **Docker API** | Queried live per host |
| Last-applied stamp | **Container labels** (`dd.config-hash`, `dd.managed-by`) | The annotation analog Flux/Argo use |
| Fleet / targets | **Git** (ansible inventory + group selectors) | Already how SF's engine resolves hosts |
| Drift | derived | rendered-desired hash vs live label |
| Orphans | derived | `managed-by=dd` containers not in desired set |

No database appears in the reconcile path. This is what lets `dd` run on a bare node.

## 5. Engine consolidation (decision + prerequisite)

Two Docker engines exist today and must collapse to one:

- **DD-UI `api/services/`** — full-featured, but **DB-coupled** (needs Postgres).
- **StageFreight `src/docker/`** — a **stateless, interface-driven rewrite** (`InventorySource`, `SecretsProvider`, `HostTransport`; injectable `HashStamps`; no DB).

**Decision:** SF's `src/docker/` is the better base for `dd` — it's already stateless and portable. It **relocates out of StageFreight into `dd`**, DD-UI's DB-coupled engine **retires**, and SF's `docker` backend shrinks to a thin shell that invokes `dd`.

**Prerequisite (Phase 0):** a feature-completeness diff of SF `src/docker/` vs DD-UI `services/` — to know exactly what moves and what (if anything) from DD-UI's engine still needs to fold in. Do not delete either engine before this diff.

## 6. Pull-only ⇒ git write-back

Because the controller owns "what runs," changing desired state means **changing git, not mutating containers directly.** DD-UI splits its actions:

- **Declarative changes** (what's deployed, config, image tags, scale) → DD-UI **commits to the reconciled branch, or opens an MR** → controller reconciles the merge. DD-UI gains git-*write* (commit/push + forge MR API), not just today's read-side sync.
- **Imperative live-ops** (start/stop/restart/logs/exec) → still direct Docker, but **transient**: a manual change to a managed stack drifts and gets reconciled back. The UI must say so ("live override — commit it to make it stick").

**MR-based write-back + RBAC = the multitenancy story:** a tenant proposes (MR), an authorized role approves, the controller reconciles the merge — with an audit trail. Propose → approve → reconcile.

## 7. Control-plane features (all preserved)

Everything on the wishlist is a *control-plane* feature and lives in DD-UI's DB, unaffected by the stateless engine:

- **Local login** — feasible. **Note:** reverses DD-UI's current "OIDC-mandatory, no local accounts" posture; a deliberate product choice, not a freebie.
- **Multitenancy** — control-plane scoping over (git repos × inventory groups).
- **RBAC** — gates who may see / trigger / commit / approve.
- **In-flight features** (log persistence, etc.) — control-plane/UI; untouched by the split.

Argo is the proof: stateless reconcile, full RBAC + projects (its multitenancy) + SSO as control-plane config.

## 8. `dd bootstrap` (like `flux bootstrap`)

```
dd bootstrap --repo <git> --path infra/dd --hosts <group>
```

1. Connect (or create) the git repo via the forge API.
2. Commit `dd`'s **own** compose stack + reconcile config into it.
3. `compose up` the `dd serve` controller on the target host(s).
4. Point it at that repo — including reconciling **itself**.

Payoff: **self-management** — thereafter you upgrade `dd` by committing to git (bump its image tag → it reconciles itself). Deploying `dd` as a compose stack is exactly the intended shape.

## 9. Known problems to fix

- **Multi-host polling storm** *(concrete)* — the multi-hosts view refreshes container details everywhere instead of only visible items. Fix: scope refresh to visible/lazy rows; move to event-driven or on-demand-per-visible; back off blanket background polling. (Cross-reference [`../Performance-Lessons.md`](../Performance-Lessons.md).)
- **Jank git-sync** — replaced by the proper pull-based reconcile loop in the core (Section 3).

## 10. StageFreight-side changes

- Relocate `src/docker/` engine → `dd`; slim the `docker` backend to shell `dd`; **add `dd` to SF's toolchain** (provisioned on demand → Docker reconcile is only present for docker-mode pipelines, keeping SF slim).
- **Multi-mode dispatch** — allow one config to run more than one reconcile backend, so a single lab IaC repo can reconcile K8s (flux) **and** Docker (`dd`) together. `lifecycle.mode` is currently a single scalar; this is the one real extension the one-repo-whole-lab reality needs.
- **Multi-cluster gitops** — `Cluster` → `Clusters` (Flux monorepo pattern: many clusters, shared bases/overlays, one repo).

## 11. Future / deferred axes

Explicitly **not now**; some may never be needed. Recorded so the design doesn't foreclose them.

- **Docker Swarm support** — a stated ultimate goal. Mechanism TBD; would slot as another engine target behind the same interface. (How Swarm's service model maps to our stack-based reconcile is an open research item — not scoped here.)
- **Podman** — likely cheap (Docker-compatible socket); another backend target.
- **Multi-component / microservice split** (Argo-style, for scale-out) — considered, **not committed**, and may never be warranted at homelab scale. The single-binary/single-process design does not preclude splitting later.
- **Cross-node application scaling / coordinating services across unlinked nodes** — an intriguing capability the current cross-node coordination hints at. **Blue-sky, not a requirement**; needs a concrete use-case before any design.
- **True peer mesh** — considered and set aside in favor of **hub-and-spoke** (which is what Flux/Argo themselves do).

## 12. Proposed sequencing

Ordered by dependency; reorder by appetite.

- **Phase 0 (now):** this doc + the engine completeness diff (SF `src/docker/` vs DD-UI `services/`).
- **Phase 1:** `dd` core — relocate SF's stateless engine; label-based stamps; DB-free; retire DD-UI's engine.
- **Phase 2:** `dd` one binary — `serve` (controller+API+UI), CLI, remote `--server`; the pull-based reconcile loop (replaces the jank).
- **Phase 3:** DD-UI git write-back (commit/MR); fix the multi-host polling storm.
- **Phase 4:** SF thin `docker` backend → shell `dd` + toolchain entry; multi-mode + multi-cluster.
- **Phase 5:** control-plane features as needed — local login, RBAC, multitenancy, MR-approval flow.
- **Phase 6:** `dd bootstrap`.
- **Deferred:** Swarm, Podman, scaling axes (Section 11).

## 13. Open questions / decisions needed

1. **Engine completeness** — is SF's `src/docker/` feature-complete vs DD-UI's, or a clean-but-partial refactor? (Phase 0 answers.)
2. **Local login** — commit to reversing the OIDC-only posture?
3. **Label schema** — exact stamp keys and what's hashed (compose bundle vs rendered config).
4. **DD-UI ↔ `dd` integration** — subprocess (recommended; matches how DD-UI already shells `sops` and how SF shells `flux`) vs a shared Go SDK. Start subprocess; promote only if the CLI boundary chafes.
5. **Single process vs component split** — start single; revisit only under real load.
6. **Bootstrap UX** — repo creation, forge auth, what lands in git.

---

*Relationship recap:* **StageFreight** = build + slim orchestrator (invokes `flux` for K8s, `dd` for Docker). **`dd`** = the stateless Docker reconcile tool (CLI + controller + UI, one binary). **DD-UI** = the stateful control plane over `dd` + the git-write-back UX. The DB stops being an engine dependency and becomes the correct home for auth, tenancy, RBAC, audit, and history.
