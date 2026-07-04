# Installation (Docker Compose)

## Requirements

- Docker reachable from the DD-UI backend to each host you list (TCP or local socket).
- PostgreSQL 14+
- Node 18+ (for dev UI), Go 1.21+ (backend)
- OIDC provider (tested with Zitadel) or run in “local only” with `/api/session` returning no user (login page will redirect).
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
This is a working docker-compose example.
Please edit the values to be specific to your deployment.
Don't forget to create the secret files and add the correct values.

```yaml
version: "3.8"
services:
  
  dd-ui-postgres:
    container_name: dd-ui-postgres
    image: postgres:16-alpine
    environment:
      - POSTGRES_DB=dd-ui
      - POSTGRES_USER=prplanit
      - POSTGRES_PASSWORD_FILE=/run/secrets/postgres_pass
    ports:
      - 5432:5432
    volumes:
      - /opt/docker/dd-ui/postgres:/var/lib/postgresql/data
    secrets:
      - postgres_pass
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $POSTGRES_USER -d $POSTGRES_DB"]
      interval: 5s
      timeout: 3s
      retries: 20
      
  dd-ui-app:
    container_name: dd-ui-app
    depends_on:
      dd-ui-postgres:
        condition: service_healthy
    image: prplanit/dd-ui:v0.4.7
    ports:
      - "3000:443"
    env_file: stack.env
    environment:
      # General Config
      #- DD_UI_BIND=0.0.0.0:443
      # - DD_UI_DEFAULT_OWNER= # (email)
      - DD_UI_INVENTORY_PATH=/data/inventory
      - DD_UI_LOCAL_HOST=anchorage
      - DD_UI_UI_ORIGIN=https://dd-ui.pcfae.com
      
      # Authentication / OIDC
      - DD_UI_COOKIE_SECURE=true
      - DD_UI_COOKIE_DOMAIN=dd-ui.pcfae.com
      - OIDC_CLIENT_ID_FILE=/run/secrets/oidc_client_id
      - OIDC_CLIENT_SECRET_FILE=/run/secrets/oidc_client_secret
      - OIDC_ISSUER_URL=https://sso.prplanit.com
      - OIDC_REDIRECT_URL=https://dd-ui.pcfae.com/auth/callback
      - OIDC_POST_LOGOUT_REDIRECT_URL=https://dd-ui.pcfae.com/login
      - OIDC_SCOPES=openid email profile
        # - OIDC_ALLOWED_EMAIL_DOMAIN # (optional; blocks others)
      
      # Database (Postgres) Configuration:
      - DD_UI_DB_HOST=dd-ui-postgres
      - DD_UI_DB_PORT=5432
      - DD_UI_DB_NAME=dd-ui
      - DD_UI_DB_USER=prplanit
      - DD_UI_DB_PASS_FILE=/run/secrets/postgres_pass
      - DD_UI_DB_SSLMODE=disable
      - DD_UI_DB_MIGRATE=true
        # or provide a single DSN:
        # - DD_UI_DB_DSN=postgres://dd-ui:...@db:5432/dd-ui?sslmode=disable

      # Docker Connection Config
      - DOCKER_CONNECTION_METHOD=local
      
      # Encryption / SOPS Config
      - DD_UI_ALLOW_SOPS_DECRYPT=true
      - SOPS_AGE_KEY_FILE=/run/secrets/sops_age_key
      - DD_UI_SESSION_SECRET_FILE=/run/secrets/session_secret
      
      # SSH Config
      - SSH_USER=kai           # or a limited user in docker group
      - SSH_PORT=22
      - SSH_KEY_FILE=/run/secrets/ssh_key
      - SSH_USE_SUDO=false      # true if your user needs sudo
      - SSH_STRICT_HOST_KEY=false
      
      # Auto DevOps Config
      - DD_UI_DEVOPS_APPLY=false
      
      # Scanning Config - Docker Host(s) States
      - DD_UI_SCAN_DOCKER_AUTO=true
      - DD_UI_SCAN_DOCKER_INTERVAL=1m
      - DD_UI_SCAN_DOCKER_HOST_TIMEOUT=45s
      - DD_UI_SCAN_DOCKER_CONCURRENCY=3
      - DD_UI_SCAN_DOCKER_ON_START=true
      - DD_UI_SCAN_DOCKER_DEBUG=true
      
      # Scannning Config - IAC
      - DD_UI_IAC_ROOT=/data
      - DD_UI_IAC_DIRNAME=docker-compose
      - DD_UI_SCAN_IAC_AUTO=true
      - DD_UI_SCAN_IAC_INTERVAL=90s

    secrets:
      - oidc_client_id
      - oidc_client_secret
      - postgres_pass
      - session_secret
      - sops_age_key
      - ssh_key
    volumes:
      - /opt/docker/dd-ui/data:/data
      - /var/run/docker.sock:/var/run/docker.sock

secrets:
  oidc_client_id:
    file: /opt/docker/dd-ui/secrets/oidc_client_id
  oidc_client_secret:
    file: /opt/docker/dd-ui/secrets/oidc_client_secret
  postgres_pass:
    file: /opt/docker/dd-ui/secrets/postgres_password
  session_secret:
    file: /opt/docker/dd-ui/secrets/session_secret
  sops_age_key:
    file: /opt/docker/dd-ui/secrets/sops_age_key
  ssh_key:
    file: /opt/docker/dd-ui/secrets/id_ed25519   # your private key
```

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
