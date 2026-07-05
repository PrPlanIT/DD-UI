# Configuring DD-UI — the happy path

DD-UI is driven by three kinds of plain files — keep them in a repo, edit them in the UI, or both.

## 1. Hosts — [`inventory.example`](inventory.example)
Define the machines DD-UI manages (Ansible-format, YAML or INI):

```yaml
all:
  hosts:
    anchorage:
      ansible_host: 10.30.1.122
```

Also here: [`inventory.sample`](inventory.sample) (minimal) and [`inventory.sample.enhanced`](inventory.sample.enhanced) (every field — tags, tenant, owner, per-host env, group membership).

## 2. Environment — [`.env.example`](.env.example)
Copy it to your stack env and fill in Postgres, OIDC, SOPS recipients, etc. Full reference: **[Environment Variables](../Environment_Variables.md)**.

## 3. Stacks — `docker-compose/<host>/<stack>/docker-compose.yaml`
This is the core idea. **DD-UI deploys any _valid_ Docker Compose you place at:**

```
<docker-compose-dir>/<host-or-group>/<stack>/docker-compose.yaml
```

Nothing is bespoke — it's ordinary Compose. Alongside it, any `.env` files are interpreted **at deployment time**, and **every file is editable in the DD-UI UI** (with gated SOPS decrypt for encrypted ones). Change an IaC file and DD-UI redeploys the stack.

### Example — [`docker-compose/anchorage/grafana/`](docker-compose/anchorage/grafana/docker-compose.yaml)
```
config/docker-compose/
  anchorage/                 # a host from your inventory (or a group name)
    grafana/                 # a stack
      docker-compose.yaml    # → DD-UI deploys grafana on anchorage
```

That file is a stock `grafana/grafana` compose — and that's exactly the point: **hand DD-UI valid, arbitrary Compose and it deploys it.** Add stacks, add hosts, swap images; it's just Compose the whole way down.

> Root and dir are configurable: `DD_UI_IAC_ROOT` (default `/data`) and `DD_UI_IAC_DIRNAME` (default `docker-compose`).
