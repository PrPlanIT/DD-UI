# Deploy DD-UI with Docker Compose

These compose files run **DD-UI itself** — not the stacks it manages (those live in [`../config/docker-compose/`](../config/docker-compose/)).

- **[`docker-compose.yml`](docker-compose.yml)** — production-style: DD-UI + Postgres, secrets mounted as files, OIDC, SOPS. Edit the values for your deployment and create the referenced secret files.
- **[`docker-compose.test.yml`](docker-compose.test.yml)** — a local/test variant.

Full walkthrough — requirements, `.env`, and an Nginx reverse-proxy example: **[Installation](../Installation.md)**.
