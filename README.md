# <img src="ui/public/DD-UI-Logo.png" alt="DD-UI (Designated Driver UI)" width="35" height="35" /> DD-UI (Designated Driver UI)
> DD-UI is a declarative, security-first Docker orchestration engine. It compares runtime state (running containers) to declared state (your IaC repo), shows drift, and puts encryption (SOPS/AGE) and DevOps ergonomics first. **Please Docker responsibly.**

<!-- sf:project:start -->
[![badge/GitHub-source-181717?logo=github](https://img.shields.io/badge/GitHub-source-181717?logo=github)](https://github.com/PrPlanIT/DD-UI) [![badge/GitLab-source-FC6D26?logo=gitlab](https://img.shields.io/badge/GitLab-source-FC6D26?logo=gitlab)](https://gitlab.prplanit.com/PrPlanIT/dd-ui) [![Last Commit](https://img.shields.io/github/last-commit/PrPlanIT/DD-UI)](https://github.com/PrPlanIT/DD-UI/commits) [![Open Issues](https://img.shields.io/github/issues/PrPlanIT/DD-UI)](https://github.com/PrPlanIT/DD-UI/issues) [![Contributors](https://img.shields.io/github/contributors/PrPlanIT/DD-UI)](https://github.com/PrPlanIT/DD-UI/graphs/contributors)
<!-- sf:project:end -->
<!-- sf:badges:start -->
[![build](https://raw.githubusercontent.com/PrPlanIT/DD-UI/main/.stagefreight/badges/build.svg)](https://gitlab.prplanit.com/PrPlanIT/dd-ui/-/pipelines) [![license](https://raw.githubusercontent.com/PrPlanIT/DD-UI/main/.stagefreight/badges/license.svg)](https://github.com/PrPlanIT/DD-UI/blob/main/LICENSE) [![release](https://raw.githubusercontent.com/PrPlanIT/DD-UI/main/.stagefreight/badges/release.svg)](https://github.com/PrPlanIT/DD-UI/releases) ![updated](https://raw.githubusercontent.com/PrPlanIT/DD-UI/main/.stagefreight/badges/updated.svg) [![badge/donate-FF5E5B?logo=ko-fi&logoColor=white](https://img.shields.io/badge/donate-FF5E5B?logo=ko-fi&logoColor=white)](https://ko-fi.com/T6T41IT163) [![badge/sponsor-EA4AAA?logo=githubsponsors&logoColor=white](https://img.shields.io/badge/sponsor-EA4AAA?logo=githubsponsors&logoColor=white)](https://github.com/sponsors/PrPlanIT)
<!-- sf:badges:end -->
<!-- sf:image:start -->
[![badge/Docker-prplanit%2Fdd--ui-2496ED?logo=docker&logoColor=white](https://img.shields.io/badge/Docker-prplanit%2Fdd--ui-2496ED?logo=docker&logoColor=white)](https://hub.docker.com/r/prplanit/dd-ui) [![pulls](https://raw.githubusercontent.com/PrPlanIT/DD-UI/main/.stagefreight/badges/pulls.svg)](https://hub.docker.com/r/prplanit/dd-ui)
<!-- sf:image:end -->

## What is DD-UI?

DD-UI is a **Docker management engine that puts DevOps and encryption first** — in essence, a Docker-focused CI/CD pipeline with a UI.

- **Infrastructure as Code.** DD-UI manages hosts, groups, and Docker "stacks" as standardized, CI/CD-compatible IaC files. Deployment state is decoupled from the app — edit it in the editor of your choice, and DD-UI redeploys containers when the IaC changes.
- **Encryption, first-class.** Encrypt/decrypt any IaC file — compose and `.env` included — with SOPS/AGE, right from the UI. Values are censored by default, so you can stream or push configs to a repo safely, and DD-UI can still deploy them encrypted.
- **A rich experience.** Many of the features you'd expect from a Docker GUI, plus industry tools like xterm 🔥 and monaco (the editor from VS Code 🎉).
- **Free and open source** under the **AGPL-3.0-or-later** — free forever for personal, homelab, non-profit, and commercial use alike (a commercial license is available if you need proprietary terms).

<p align="center"><img src="docs/screenshots/DD-UI-Host-Stacks.png" width="760" alt="DD-UI — every stack across every host in one view" /></p>

> 📸 **[See DD-UI in action →](docs/Screenshots.md)** — SOPS-encrypted stacks decrypted on the fly, live logs, in-container terminal, drift detection, cleanup, and more.

## Status

> ⚠️ **Pre-release — ~85–95% functional.** Core works; some advanced features are partial (git-sync is currently unstable). **Before you rely on it, skim the [open issues](https://github.com/PrPlanIT/DD-UI/issues) and [Known Issues](docs/Roadmap.md) for any dealbreakers.** See **[Roadmap & Status](docs/Roadmap.md)** for scope & cadence. CHEERS!

---

## What DD-UI does today
- **Container control**: start / stop / pause / resume / kill containers.
- **Live logs**: dedicated logging view with advanced filters.
- **In-container terminal**: an xterm-powered shell for a rich experience.
- **Editor**: edit compose, `.env`, and scripts with monaco (the editor from VS Code) — no compromise vs other Docker tools.
- **Inventory**: list hosts.
- **Stacks / containers**: every running container across all your systems in one view.
- **Sync**: one click triggers an IaC scan (local repo) plus a runtime scan per host (Docker).
- **Compare**: runtime vs desired (images, services), with a per-stack drift indicator.
- **Usability**: per-host search, fixed table layout, ports rendered one mapping per line.
- **SOPS awareness**: detect encrypted files; never decrypt by default (explicit, audited reveal flow).
- **Auth**: OIDC is **mandatory** — there are no local accounts. Authenticate through your IdP (Zitadel/Keycloak/Authentik/Okta/Auth0) or you're not getting in. Session probe, login, logout (RP-logout optional).
- **API**: `/api/...` (JSON), with the static SPA served by the backend.
- **SOPS CLI integration**: the server runs `sops` for encrypt/decrypt; no plaintext secrets are stored.
- **Health pills**: health-aware state (running / healthy / exited …).
- **Stack Files page**: view/edit compose/env/scripts against runtime context, with gated SOPS decryption.
- **Docker Cleanup page**: prune or clear the build cache from the UI.

---

## 📚 Documentation

**[Full documentation →](docs/README.md)** — deploy, configure, and run DD-UI.

- **[Installation (Docker Compose)](docs/Installation.md)** — requirements, compose example, `.env`, Nginx
- **[Development](docs/Development.md)** — run the UI/API locally
- **[Environment Variables](docs/Environment_Variables.md)** — all environment variables
- **[SOPS / AGE](docs/SOPS.md)** — encryption keys, encrypt, decrypt
- **[Usage](docs/Usage.md)** — using DD-UI + IaC layout
- **[Architecture](docs/Architecture.md)** — high-level design
- **[Security](docs/Security.md)** — posture & disclosure
- **[Roadmap & Status](docs/Roadmap.md)** — scope, cadence, known issues

---

## Contributing
- File issues with steps, logs, and versions.
- Small, focused PRs are best (typos, error handling, UI polish).
- Sample IaC directories welcome!
- Security-related PRs and hardening suggestions are especially appreciated (SOPS/AGE, cookie settings, RBAC, etc.).

---

## Support / Sponsorship
If you’d like to help keep the project moving:

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/T6T41IT163)

---

## License

DD-UI is distributed under the **[AGPL-3.0-or-later](LICENSE)** license — free and open for everyone, including commercial use, under its copyleft terms. See **[LICENSING.md](docs/LICENSING.md)** for commercial/proprietary licensing options.

> _This section is a human‑readable summary and not a substitute for the license. Nothing here grants rights by itself._

---

## Disclaimer
The Software provided hereunder (“Software”) is licensed “as‑is,” without warranties of any kind—express, implied, or telepathically transmitted. The Softwarer (yes, that’s totally a word now) makes no promises about functionality, performance, compatibility, security, or availability—and absolutely no warranty of any sort. The developer shall not be held responsible, even if the software is clearly the reason your dog decided to orchestrate its own sidecar, your mom scored five tickets to Hawaii but you missed out because you were knee‑deep in a `docker compose` rabbit hole, or your stack drifted so hard it achieved sentience and renamed itself.

If using this orchestration UI leads you down a rabbit hole of obsessive network optimizations, breaks your fragile grasp of version pinning, or causes an uprising among your offline‑first containers—sorry, still not liable. Also not liable if your repo syncs so fast it rips a hole in the space‑time continuum **or** if your `.env` files multiply like Tribbles. The developer likewise claims no credit for anything that actually goes right either. Any positive experiences are owed entirely to the unstoppable force that is the Open Source community.

It’s never been a better time to be a PC user—or a homelabber. Just don’t blame me when YAML inevitably eats your weekend.
