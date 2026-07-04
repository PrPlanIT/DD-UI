# Using DD-UI

1. **Log in (OIDC)**. You’ll be redirected to `/auth/login` if no session.
2. **Add hosts to inventory.** Currently hosts are stored in the DB; the API supports reload from a path if you want to seed via file:
   ```bash
   # POST /api/inventory/reload with an optional { "path": "/data/inventory.yaml" }
   curl -sS -X POST -H "Content-Type: application/json" \
     -d '{"path":"/data/inventory.yaml"}' \
     http://localhost:8080/api/inventory/reload
   ```
   `inventory.yaml` (example) (Ansible formatted inventory, yaml/ini supported)
   ```yaml
   all:
     hosts:
   # GPU Accelerated:
       anchorage:
         ansible_host: 10.30.1.122
       leaf-cutter:
         ansible_host: 10.13.37.141
   ```
3. **Click Sync** on the Hosts page (or “Scan” per host). This will:
   - Scan IaC (`/data/docker-compose/...`), persist stacks/services/files.
   - Scan runtime per host (containers, images, ports, health).
4. **Drill into a host** to see:
   - Stacks merged from runtime and IaC.
   - For each row: name, state, image (runtime → desired), created, IP, ports (one per line), owner.
   - Per-host search box (filters rows).
5. **Metrics**: Hosts, Stacks, Containers, Drift, Errors aggregate across filtered hosts.
6. **SOPS**: encrypted `.env` files are detected (marked). Use the gated reveal if enabled.

---

## IaC layout details
- DD-UI walks `<root>/<dirname>/<scope>/<stack>` (defaults `/data/docker-compose/*/*`).
- It records:
  - compose file (if present),
  - env files (SOPS detection via markers / file suffixes),
  - scripts `pre.sh`, `deploy.sh`, `post.sh`,
  - parsed services (image, labels, ports, volumes, env keys).
- **Scopes**
  - If `<scope>` equals a known host, it’s a host scope.
  - Otherwise it’s a group scope (applies to any host in that group).
- **Drift**
  - Different image than desired, a missing desired container/service, or IaC with no runtime ⇒ **drift**.

