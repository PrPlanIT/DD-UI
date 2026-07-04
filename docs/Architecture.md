# Architecture (high level)

- **Backend (Go)**:
  - OIDC auth, sessions (cookie).
  - Scans: Docker hosts (runtime) + IaC repo (local).
  - Postgres for persistence (migrations in `src/api/migrations`).
  - Serves the SPA.
  - Calls out to the **`sops`** executable on the server for encrypt/decrypt (expects `sops` on `PATH`).
- **Frontend (Vite/React + Tailwind/shadcn)**:
  - Hosts page (metrics + search + Sync).
  - Host detail (stacks, drift, per-host search).
  - “Reveal SOPS” UX sends an explicit confirmation header to the backend.

