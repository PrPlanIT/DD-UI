# Development (developer mode)

> Best for hacking on the UI/API locally.

See [Installation → Requirements](Installation.md#requirements) for prerequisites (Docker, PostgreSQL 14+, Node 18+, Go 1.21+, SOPS 3.10+).

1) **Postgres**
```bash
docker run -d --name dd-ui-pg -p 5432:5432 \
  -e POSTGRES_PASSWORD=devpass -e POSTGRES_USER=dd-ui -e POSTGRES_DB=dd-ui \
  postgres:15
```
Set `DATABASE_URL` for the backend:
```bash
export DATABASE_URL=postgres://dd-ui:devpass@localhost:5432/dd-ui?sslmode=disable
```

2) **OIDC (Zitadel example)**  
Create an OAuth 2.0 Web client:
- Redirect URL: `https://your-dd-ui.example.com/auth/callback` (or `http://localhost:8080/auth/callback` for dev)
- (Optional) Post-logout redirect: `http://localhost:8080/`
- Scopes: `openid email profile`

Environment (dev):
```bash
export OIDC_ISSUER_URL="https://<your-zitadel-domain>/.well-known/openid-configuration"
export OIDC_CLIENT_ID="<client-id>"
export OIDC_CLIENT_SECRET="<client-secret>"    # supports "@/path/to/secret"
export OIDC_REDIRECT_URL="http://localhost:8080/auth/callback"
# Optional hardening / ergonomics
export OIDC_SCOPES="openid email profile"
export OIDC_ALLOWED_EMAIL_DOMAIN=""            # e.g. "example.com" to restrict
export COOKIE_DOMAIN=""                         # e.g. ".example.com" in prod
# If unset, DD-UI infers COOKIE_SECURE from the redirect URL scheme
# export COOKIE_SECURE=true|false
```

3) **Run backend**
```bash
cd src/api
go run .
# or: go build -o dd-ui && ./dd-ui
```
The backend runs DB migrations automatically at startup (ensure `DATABASE_URL` is set).

4) **Run frontend**
```bash
cd ui
pnpm install
pnpm dev
```
Visit `http://localhost:5173` (or the port Vite prints).  
In production, the Go server serves the built UI; during dev it’s fine to run separately.

Build the UI once:
```bash
cd ui && pnpm install && pnpm build
```
Then hit `http://localhost:8080` (or `http://localhost:3000` if you used the mapping above).
