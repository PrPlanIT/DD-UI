# DD-UI Documentation

Everything to deploy, configure, and run DD-UI. New here? Start with **[Installation](Installation.md)**.

## Guides
| Guide | |
|---|---|
| **[Installation](Installation.md)** | Deploy DD-UI with Docker Compose — requirements, `.env`, Nginx |
| **[Development](Development.md)** | Run the UI/API locally |
| **[Usage](Usage.md)** | Day-to-day use + the IaC layout |
| **[Environment Variables](Environment_Variables.md)** | Every `DD_UI_*` / OIDC / DB / SSH knob |
| **[SOPS / AGE](SOPS.md)** | Encrypt/decrypt secrets in your IaC repo |
| **[Architecture](Architecture.md)** | High-level design |
| **[Security](Security.md)** | Posture & disclosure |
| **[Roadmap & Status](Roadmap.md)** | Scope, cadence, known issues |
| **[Screenshots](Screenshots.md)** | DD-UI in action |

## Deploy DD-UI
- **[Docker Compose](docker/)** — [`docker/docker-compose.yml`](docker/docker-compose.yml), the quickest way to run DD-UI.
- **[Kubernetes / FluxCD](kubernetes/)** — a GitOps example ([`kubernetes/fluxcd/`](kubernetes/fluxcd/), base + production overlay). Provided as a starting point — **adapt at your discretion**.

## Configure DD-UI — the happy path
- **[config/](config/)** — your hosts ([`inventory.example`](config/inventory.example)), env ([`.env.example`](config/.env.example)), and the stacks DD-UI deploys ([`config/docker-compose/<host>/<stack>/docker-compose.yaml`](config/docker-compose/)). Full walkthrough in **[config/README](config/README.md)**.

## Design notes
- **[architecture/](architecture/)** — drift detection, git-sync, the groups model, and how DD-UI thinks.
