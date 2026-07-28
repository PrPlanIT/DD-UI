# The Desired-State Docker Engine

- **Status:** Built and working — needs **refactoring, not rebuilding**.
- **Type:** design / how it works (concept and current shape).
- **See also:** the refactor plan → [`plans/2026-07-28-DD-UI-Evaluate-Pull-Based-Reconciliation-and-Other-Refactors.md`](plans/2026-07-28-DD-UI-Evaluate-Pull-Based-Reconciliation-and-Other-Refactors.md) · aspirational directions → [`Ideas.md`](Ideas.md).

## What it is

At its core, DD-UI is a **desired-state-native Docker orchestration engine** — Flux/Argo, but for plain Docker hosts. You declare what should run as Infrastructure-as-Code (compose + `.env`, SOPS-encrypted) in a git repo; the engine compares that declared state to the actual running containers, surfaces drift, and reconciles. **The GUI is one face on that engine; it is not the engine.**

This engine exists and works today. The refactors described elsewhere make it portable and correct — they do not build it from scratch.

## How it reconciles

Following the GitOps principle Flux and Argo proved: **git is the desired-state store, the target system is the actual-state store — reconcile needs no database.**

| Concern | Store |
|---|---|
| Desired state | Git — compose/`.env` IaC, SOPS-encrypted |
| Actual state | The Docker API, queried live |
| Last-applied stamp | Container labels (`dd.config-hash`, `dd.managed-by`) — the annotation analog Flux/Argo use |
| Fleet / targets | Git — ansible inventory + group selectors |
| Drift | derived: rendered-desired hash vs live label |
| Orphans | derived: `managed-by` containers absent from the desired set |

No database appears in the reconcile path. That is precisely what lets the engine run as a portable CLI or a bare-node agent.

## Current state — why it needs refactoring

The engine exists today in **two divergent forms**, and that duplication is the core debt:

- **DB-coupled** (`api/services/`) — full-featured, but bound to Postgres; cannot run standalone.
- **Stateless / interface-driven** (StageFreight's `src/docker/`, extracted from this codebase) — a clean rewrite: `InventorySource`, `SecretsProvider`, `HostTransport` interfaces; injectable hash stamps; no DB.

The refactor consolidates onto the **stateless** form (portable, DB-free), retires the DB-coupled copy, and relocates the engine into its own tool. Sequencing lives in the refactor plan.

## The three surfaces

One engine, three ways to reach it:

- **`dd` (CLI + `dd serve` controller)** — the engine itself: a **pull-based, always-on** reconcile daemon plus a kubectl-style remote CLI. Stateless; one multi-call binary; a single process is fine at homelab scale.
- **DD-UI** — the **stateful control plane** over the engine: auth, multitenancy, RBAC, audit, history, and the git-write-back UX. (Pull-only means the UI *commits / opens an MR* to change desired state — it does not mutate containers directly; imperative live-ops like start/stop/logs/exec stay direct but are transient.) Its database serves the control plane, **never** reconcile.
- **StageFreight** — the CI orchestrator; it *invokes* `dd` (via its toolchain) exactly as it invokes Flux for Kubernetes. **Push-based** reconcile-on-merge. It does **not** embed the engine.

Both triggers — push (StageFreight in CI) and pull (`dd serve`) — drive the same stateless engine.

## Principles

- **The engine is the product; the UI is a thin client over it.**
- **Git is truth; the target holds actual state; no third store for reconcile.**
- **Invoke, don't embed** — StageFreight orchestrates the engine as an external tool the way it orchestrates Flux, and stays slim.
