# Environment Variables

### General

| Variable              | Default | Description                                                                                      |
| --------------------- | ------- | ------------------------------------------------------------------------------------------------ |
| `DD_UI_DEFAULT_OWNER`  | —       | Default owner/team used when creating stacks or records (namespacing/attribution in the UI).     |
| `DD_UI_BUILDS_DIR`     | —       | Directory for build outputs and artifacts (e.g., generated bundles/manifests).                   |
| `DD_UI_INVENTORY_PATH` | —       | Path to the hosts inventory file (YAML/JSON) defining remote Docker targets.                     |
| `DD_UI_LOCAL_HOST`     | `""`    | Optional override for the local host name/label; leave empty to use the tool’s implicit/default. |
| `DD_UI_BIND`           | —       | Server bind address, e.g. `:8080` or `0.0.0.0:8080`.                                             |
| `DD_UI_UI_ORIGIN`                             | empty                   | Additional allowed CORS origin for the dev UI (`http://localhost:5173` is allowed by default) |
| `DD_UI_UI_DIR`                           | `/home/dd-ui/ui/dist`    | Where built SPA is served from                                                              |


### Auth / OIDC

| Variable                                | Default                 | Description                                                                                 |
| --------------------------------------- | ----------------------- | ------------------------------------------------------------------------------------------- |
| `DD_UI_COOKIE_DOMAIN`                    | empty                   | e.g. `.example.com`                                                                         |
| `DD_UI_COOKIE_SECURE`                    | inferred                | `true/false` (if unset, inferred from redirect URL scheme)                                  |
| `OIDC_ISSUER_URL`                       | —                       | Provider discovery URL (`…/.well-known/openid-configuration`)                               |
| `OIDC_CLIENT_ID` / `OIDC_CLIENT_SECRET` | —                       | OAuth client (secret supports `@/path` indirection)                                         |
| `OIDC_CLIENT_ID_FILE` / `OIDC_CLIENT_SECRET_FILE` | —                       | Same function as above but passed in as a file for docker secrets funtionality.   |
| `OIDC_REDIRECT_URL`                     | —                       | e.g. `http://localhost:8080/auth/callback`                                                  |
| `OIDC_SCOPES`                           | `openid email profile`  | Space-separated scopes                                                                      |
| `OIDC_ALLOWED_EMAIL_DOMAIN`             | empty                   | Restrict logins to a domain                                                                 |

### Database (Postgresql)

| Variable                    | Default | Description                                                                       |
| --------------------------- | ------- | --------------------------------------------------------------------------------- |
| `DD_UI_DB_DSN`               | —       | Full connection string, e.g. `postgres://user:pass@host:5432/db?sslmode=disable`. |
| `DD_UI_DB_HOST`              | —       | Hostname/IP of the database (used when DSN is not set).                           |
| `DD_UI_DB_PORT`              | —       | Database port, e.g. `5432`.                                                       |
| `DD_UI_DB_NAME`              | —       | Database name.                                                                    |
| `DD_UI_DB_USER`              | —       | Database user.                                                                    |
| `DD_UI_DB_PASS`              | —       | Database password (prefer `DD_UI_DB_PASS_FILE` for secrets).                       |
| `DD_UI_DB_PASS_FILE`         | —       | Read password from file (Docker secrets compatible).                              |
| `DD_UI_DB_SSLMODE`           | —       | Postgres `sslmode` (`disable`, `require`, `verify-ca`, `verify-full`).            |
| `DD_UI_DB_MAX_CONNS`         | —       | Max open connections in the pool (integer).                                       |
| `DD_UI_DB_MIN_CONNS`         | —       | Minimum/idle pool size (integer).                                                 |
| `DD_UI_DB_CONN_MAX_LIFETIME` | —       | Max lifetime per connection (duration, e.g. `30m`).                               |
| `DD_UI_DB_CONN_MAX_IDLE`     | —       | Max idle time per connection (duration, e.g. `5m`).                               |
| `DD_UI_DB_HEALTH_PERIOD`     | —       | Interval between DB health checks (duration, e.g. `10s`).                         |
| `DD_UI_DB_CONNECT_TIMEOUT`   | —       | Dial/connect timeout (duration, e.g. `5s`).                                       |
| `DD_UI_DB_PING_TIMEOUT`      | —       | Timeout for readiness/`PING` checks (duration, e.g. `2s`).                        |
| `DD_UI_DB_MIGRATE`           | —       | `true/false` — run schema migrations on startup.                                  |

### Docker Connection Config

| Variable                   | Default                | Description                                                                                                                                                     |
| -------------------------- | ---------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `DOCKER_CONNECTION_METHOD` | `ssh`                  | How to connect to Docker: `ssh`, `tcp`, or `local` (Unix socket).                                                                                               |
| `DOCKER_SOCK_PATH`         | `/var/run/docker.sock` | Path to local Docker socket (used when method=`local`).                                                                                                         |
| `DOCKER_TCP_PORT`          | `2375`                 | Docker TCP port (used when method=`tcp`).                                                                                                                       |
| `SSH_USER`                 | `root`                 | Remote user for SSH Docker connections (see **SSH (Remote)** for keys/port options).                                                                            |
| `DOCKER_SSH_CMD`           | —                      | Advanced override: full SSH command (binary + flags). If set, it supersedes `SSH_*` vars. E.g. `ssh -i /run/secrets/ssh_key -p 22 -o StrictHostKeyChecking=no`. |


### Encryption & SOPS

| Variable                                | Default                 | Description                                                                                 |
| --------------------------------------- | ----------------------- | ------------------------------------------------------------------------------------------- |
| `DD_UI_ALLOW_SOPS_DECRYPT`               | unset                   | Enable gated decrypt API (`true/1/yes/on`), requires `X-Confirm-Reveal: yes` header         |
| `SOPS_AGE_KEY_FILE` / `SOPS_AGE_KEY`    | unset                   | AGE private key (file path or raw), enables server-side **decrypt**                         |
| `SOPS_AGE_RECIPIENTS`                   | unset                   | Space-separated AGE recipients, enables **encrypt** even without `.sops.yaml`               |
| `DD_UI_SESSION_SECRET`                   | —                       | Session/cookie HMAC secret. Generate via `DD_UI_SESSION_SECRET="$(openssl rand -hex 64)"`    |
| `DD_UI_SESSION_SECRET_FILE`              | —                       | Same function as above but passed in as a file for docker secrets funtionality.             |

### SSH Config

| Variable              | Default | Description                                                                   |
| --------------------- | ------- | ----------------------------------------------------------------------------- |
| `SSH_USER`            | —       | Remote username.                                                              |
| `SSH_PORT`            | —       | SSH port (e.g. `22`).                                                         |
| `SSH_KEY`             | —       | Inline private key (OpenSSH/PEM). Preserve newlines; prefer file for secrets. |
| `SSH_KEY_FILE`        | —       | Read private key from file (Docker secrets compatible).                       |
| `SSH_USE_SUDO`        | —       | `true/false` — run remote commands via `sudo`.                                |
| `SSH_STRICT_HOST_KEY` | —       | `true/false` — verify host key (disable to skip checks; not recommended).     |


### Auto DevOps

| Variable                                | Default                 | Description                                                            |
| --------------------------------------- | ----------------------- | ---------------------------------------------------------------------- |
| `DD_UI_DEVOPS_APPLY`                     | `true`                  | Enables Automated Deployments via IaC / DevOps                         |

### Scanning Docker

| Variable                        | Default | Description                                                   |
| ------------------------------- | ------- | ------------------------------------------------------------- |
| `DD_UI_SCAN_DOCKER_AUTO`         | `true`  | `true/false` — enable the periodic Docker scan scheduler.     |
| `DD_UI_SCAN_DOCKER_INTERVAL`     | `1m`    | How often to run scans (Go duration, e.g. `30s`, `5m`, `1h`). |
| `DD_UI_SCAN_DOCKER_HOST_TIMEOUT` | `45s`   | Per-host scan timeout (Go duration).                          |
| `DD_UI_SCAN_DOCKER_CONCURRENCY`  | `3`     | Max number of hosts scanned in parallel (integer).            |
| `DD_UI_SCAN_DOCKER_ON_START`     | `true`  | `true/false` — run an initial scan at startup.                |
| `DD_UI_SCAN_DOCKER_DEBUG`        | `false` | `true/false` — verbose logging for the Docker scanner.        |


### Scanning IaC

| Variable                 | Default | Description                                                                             |
| ------------------------ | ------- | --------------------------------------------------------------------------------------- |
| `DD_UI_SCAN_IAC_AUTO`     | `true`  | `true/false` — enable the periodic IaC (compose) scan scheduler.                        |
| `DD_UI_SCAN_IAC_INTERVAL` | `90s`   | How often to run IaC scans (Go duration, e.g. `30s`, `5m`, `1h`).                       |
| `DD_UI_IAC_ROOT`          | —       | Root path to scan for IaC (Docker Compose) files; recommended `/data`.   |
| `DD_UI_IAC_DIRNAME`       | `empty` | Optional subfolder under the root to scope scans; leave empty to use the root directly; recommended `docker-compose`. |

