# Git Sync — Atomic State Design

- **Status:** superseded-parts-noted. The **durable principles** below (atomic whole-state replacement, never rewrite history, preserve ignored local changes) still hold. The **mechanism** it originally specified — a bidirectional `push`/`pull`/`sync` mode selector with `rsync --delete` mirroring and conflict "force" flags, driven on a 5-second timer against a working clone — is **superseded** by the stateless, git-driven, pull-based direction: git is the desired-state store, the engine reconciles from it, and DD-UI changes desired state by committing / opening an MR rather than by two-way syncing a local `/data` tree. See [`../Desired-State-Docker-Engine.md`](../Desired-State-Docker-Engine.md) and the pull-based plan [`2026-07-28-DD-UI-Evaluate-Pull-Based-Reconciliation-and-Other-Refactors.md`](2026-07-28-DD-UI-Evaluate-Pull-Based-Reconciliation-and-Other-Refactors.md).
- **Type:** plan / being reworked.

## Durable principles (carry forward)

These survive the shift to pull-based and should shape whatever git write-back looks like.

1. **Atomic whole-state replacement.** Treat `docker-compose/` and `inventory` each as one unit. When you write, you write the *whole* state, never a partial set of files. Mixing "stack A from one side, stack B from the other" produces a broken, un-reproducible config. All-or-nothing writes keep the tree internally consistent.

2. **Never rewrite history; always be up to date before writing.** Fetch/verify the remote tip before any write. Build *on top of* history — no force-push, no history rewrite. Every change is an ordinary commit, so any accidental change is recoverable by reading `git log` and reverting. Git is the audit-and-rollback trail.

3. **Preserve ignored local changes in history.** *(from the DDUI-GIT-OPS notes.)* Before a pull overwrites local files from the remote, first capture whatever diverged locally as its own commit — e.g. `info: DD-UI ignored these local changes` — *then* apply the remote state. Accidental or out-of-band local edits are never silently discarded; they remain in history and can be recovered, even when the resolution is "remote wins."

Because these three hold, an explicit "force on conflict" flag becomes unnecessary: you always pull before you write, and any overwritten local delta is preserved as a commit rather than lost.

## Superseded: the mode-based bidirectional mechanism

Recorded for context — this is **not** the current direction.

The original design polled every 5 seconds and, per a configured mode, mirrored the local `/data` tree and the remote clone with `rsync -avz --delete`:

- **Push** (local → remote): pull latest, mirror local `inventory` + `docker-compose/` into the clone, commit, push.
- **Pull** (remote → local): pull latest; if local diverged, first stage the "ignored local changes" commit (principle 3), then mirror the remote tree down onto local.
- **Sync** (bidirectional): compare modification times and run whichever of push/pull the most-recent change implied; stop on a two-sided conflict rather than force.

It also carried DB-backed config (`sync_mode`, `force_on_conflict`, `last_sync_hash`) and pre-write backups under `/data/.backup-{timestamp}/`.

**Why it's abandoned.** Two-way sync of a mutable local working tree makes the local filesystem a second source of truth competing with git, which is exactly the DB-coupled/bidirectional coupling the stateless engine removes. In the pull-based model git is the single desired-state store; the engine reconciles the target from git, and the UI mutates desired state through commits/MRs. The atomic-write and never-rewrite-history guarantees move into that commit path; the rsync mirror, mode selector, and force logic do not survive.
