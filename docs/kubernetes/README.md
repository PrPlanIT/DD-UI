# Deploy DD-UI on Kubernetes (FluxCD)

A GitOps deployment example under [`fluxcd/`](fluxcd/) — Kustomize `base` (DD-UI + Postgres StatefulSets and Services) plus an `overlays/production` overlay (HTTPRoute, ExternalSecret, LoadBalancer, patches).

```
fluxcd/apps/
  base/dd-ui/                 # StatefulSets, Services, kustomization
  overlays/production/dd-ui/  # httproute, externalsecret, loadbalancer, patches
```

**Provided as a starting point — adapt at your discretion.** It reflects one homelab's setup (ExternalSecrets for config, a Gateway-API HTTPRoute, a LoadBalancer IP); swap those pieces for whatever your cluster uses (plain Secrets, an Ingress, a NodePort, etc.).
