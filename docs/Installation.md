# Installation (Docker Compose)

> **Ready-made examples:** a runnable compose file at [`docker/docker-compose.yml`](docker/docker-compose.yml), config samples (inventory, `.env`, an example IaC stack) under [`config/`](config/README.md), and a Kubernetes/FluxCD deployment at [`kubernetes/fluxcd/`](kubernetes/README.md) — adapt at your discretion.

## Requirements

- Docker reachable from the DD-UI backend to each host you list (TCP or local socket).
- PostgreSQL 14+
- Node 18+ (for dev UI), Go 1.21+ (backend)
- **OIDC provider — required** (Zitadel/Keycloak/Authentik/Okta/Auth0; tested with Zitadel). DD-UI has no local accounts.
- **SOPS 3.10+** available on the backend host/container (DD-UI calls `sops` by name).
  The provided Docker image installs it to `/usr/local/bin/sops`.

## Point DD-UI at your IaC repo (local)

Mount or place your repo under a root (default `/data`) with this layout:
```bash
/data/
  docker-compose/
    <scope-name>/
      <stack-name>/
        compose.yaml|docker-compose.yaml
        .env / *.env / *_secret.env   # SOPS detection supported
        pre.sh / deploy.sh / post.sh  # optional
```
See **[config/README](config/README.md)** for a worked example of this layout (`config/docker-compose/anchorage/grafana/`) and how DD-UI deploys any valid Compose you place there.

- `<scope-name>` is either a host name or a group name.
- DD-UI auto-detects if a scope matches a host in your inventory; otherwise it’s treated as a group.

Env (if you customize):
```bash
DD_UI_IAC_ROOT="/data"
DD_UI_IAC_DIRNAME="docker-compose"
# Gated decrypt is OFF by default; see docs/SOPS.md to enable carefully.
# DD_UI_ALLOW_SOPS_DECRYPT=true
```

## Quick start (docker-compose)

A ready-to-edit compose file lives at **[`docker/docker-compose.yml`](docker/docker-compose.yml)** — DD-UI + Postgres, secrets mounted as files, OIDC, SOPS, and the scanning/logging config. Copy it, edit the values for your deployment, create the referenced secret files, then `docker compose up -d`.

See **[Environment Variables](Environment_Variables.md)** for every knob.

### `.env` file
```.env
POSTGRES_USER=prplanit
POSTGRES_DB=dd-ui
SOPS_AGE_RECIPIENTS=<placeyourkeyhere>
```

### `Nginx` Example:
```
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

server {
  listen 80;
  listen [::]:80;

  server_name dd-ui.pcfae.com;
  return 301 https://$host$request_uri;
}

server {
  listen                    443 ssl http2;
  listen                    [::]:443 ssl http2;
  server_name               dd-ui.pcfae.com;
  # return 301                $scheme://dd-ui.pcfae.com$request_uri;

  access_log                /var/log/nginx/dd-ui.pcfae.com.access.log;
  error_log                 /var/log/nginx/dd-ui.pcfae.com.error.log;

  # TLS configuration
  # sudo openssl req -x509 -newkey rsa:4096 -keyout /etc/letsencrypt/live/172.122.122.104/privkey.pem -out /etc/letsencrypt/live/172.122.122.104/fullchain.pem -sha256 -days 3650 -nodes \
  # -subj "/C=XX/ST=Washington/L=Seattle/O=PrecisionPlanIT/OU=Internal/CN=cell-membrane"
  ssl_certificate           /etc/letsencrypt/live/pcfae.com/fullchain.pem;
  ssl_certificate_key       /etc/letsencrypt/live/pcfae.com/privkey.pem;
  ssl_protocols             TLSv1.2 TLSv1.3;

  ssl_ciphers 'ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384:ECDHE-ECDSA-CHACHA20-POLY1305:ECDHE-RSA-CHACHA20-POLY1305:ECDHE-ECDSA-AES256-SHA384:ECDHE-RSA-AES256-SHA384';
  ssl_prefer_server_ciphers on;
  ssl_session_cache         shared:SSL:50m;
  ssl_session_timeout       1d;

  # OCSP Stapling ---
  # fetch OCSP records from URL in ssl_certificate and cache them
  ssl_stapling on;
  ssl_stapling_verify on;
  ssl_dhparam /etc/nginx/dhparam.pem;

  client_max_body_size 0;

  add_header 'Access-Control-Allow-Origin' 'https://apps.pcfae.com/';
  more_set_headers "Content-Security-Policy: form-action 'self' https://apps.pcfae.com/;";
  more_set_headers "Content-Security-Policy: frame-ancestors 'self' https://apps.pcfae.com/;";
  #add_header 'Content-Security-Policy' 'upgrade-insecure-requests';

  # WebSocket upgrade path (long-lived)
  location ^~ /api/ws/ {
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;

    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    proxy_read_timeout 3600s;

    # headroom for large Set-Cookie from upstream
    proxy_buffer_size   16k;
    proxy_buffers       8 32k;
    proxy_busy_buffers_size 64k;

    proxy_pass https://anchorage:3000;
  }

  location / {

    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection $connection_upgrade;

    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    proxy_read_timeout 3600s;

    # headroom for large Set-Cookie from upstream
    proxy_buffer_size   16k;
    proxy_buffers       8 32k;
    proxy_busy_buffers_size 64k;

    # proxy_redirect off;
    proxy_pass https://anchorage:3000/;
  }
}
```
