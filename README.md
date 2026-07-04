# <img src="ui/public/DD-UI-Logo.png" alt="DD-UI (Designated Driver UI)" width="35" height="35" /> DD-UI (Designated Driver UI)
> Declarative, security-first Docker orchestration. DD-UI compares runtime state (containers on your hosts) to declared state (your IaC repo), shows drift, and puts encryption (SOPS/AGE) and DevOps ergonomics first.

## What is DD-UI?
- Designated Driver UI is a Docker Managment Engine that puts DevOps and Encryption first.
- DD-UI seeks to ease the adoption of Infrastructure as Code and make it less intimidating for users to encrypt their secrets and sensitive docker values.
  - DD-UI manages all configuration of hosts/groups, and docker "stacks" as Inventory as Control (IaC) files. They are standardized format and CI/CD compatible. In essense DD-UI is a Docker focused CI/CD pipeline with a UI.
    - The state of your deployments is decoupled from the application and can be manipulated in the editor of your choice. DD-UI will redeploy containers when IaC files change.
  - DD-UI also allows you to decrypt/encrypt any IaC related files, you can deploy containers from encrypted docker-compose.yaml if you want.
    - This is good for those who like to stream while working on their servers or want to upload their compose and env to a repo as by default they are shown censored and they can be uploaded encrypted and dd-ui can actually deploy them if they are ever cloned and placed in its watch folder.
      - GitRepository syncing is currently unstable but will be fixed soon, but will allow backup/restore of stack configuration with ease.
- DD-UI seeks to bring the rewards of the DevOps mindset to those who may not have afforded them otherwise.
- DD-UI implements much of the features of other Docker GUIs and includes some industry tools like xterm 🔥 and monaco (editor used in vscode 🎉) to ensure a rich experience for the user.
- DD-UI is free forever, for non-commercial and home use. You can inquire for a commercial license. If you find us interesting feel free to give us a pull @ prplanit/dd-ui on the Docker Hub.

#### Thank you for your support!

<img src="example/screenshots/DD-UI-Decrypted-Deployed.png" width="400" /><img src="example/screenshots/DD-UI-Host-Stack_Detail-Collapsed.png" width="400" />
<img src="example/screenshots/DD-UI-Host-Stack_Detail-Expanded.png" width="400" /><img src="example/screenshots/DD-UI-Host-Stacks.png" width="400" />
<img src="example/screenshots/DD-UI-Hosts.png" width="400" /><img src="example/screenshots/DD-UI-Images.png" width="400" />
<img src="example/screenshots/DD-UI-Logs.png" width="400" /><img src="example/screenshots/DD-UI-Networks.png" width="400" />
<img src="example/screenshots/DD-UI-Stack-Compose-Editor.png" width="400" /><img src="example/screenshots/DD-UI-Stack-Encrypted.png" width="400" />
<img src="example/screenshots/DD-UI-Stats.png" width="400" /><img src="example/screenshots/DD-UI-Terminal.png" width="400" />
<img src="example/screenshots/DD-UI-Volumes.png" width="400" /><img src="example/screenshots/DD-UI-Cleanup.png" width="400" />
<img src="example/screenshots/DD-UI-Logging.png" width="400" />
---

## Status

### Nearing MVP / Pre-release
> DD-UI is 85-95% functional. Core functionality is present, there are some advanced features that are partially implemented. Known issues are listed below. CHEERS!

> See **[Roadmap & Status](docs/Roadmap.md)** for project scope, cadence, and known issues.

---

## What DD-UI does today
- Docker Management: Start/Stop/Pause/Resume/Kill containers.
- View live logs of any container (Dedicated Logging View, with advanced filters).
- Initiate a terminal session in a container. Uses xterm for a really rich experience in the shell.
- Edit docker compose, .env, and scripts. Application implements monaco editor (editor used in vscode) for a no compromise experience compared to other Docker management tools.
- **Inventory**: list hosts.
- **Stacks/Containers**: See all of your running docker containers in one view of all your combined systems.
- **Sync**: one click triggers:
  - **IaC scan** (local repo), and
  - **Runtime scan** per host (Docker).
- **Compare**: show runtime vs desired (images, services); per-stack drift indicator.
- **Usability**: per-host search, fixed table layout, ports rendered one mapping per line.
- **SOPS awareness**: detect encrypted files; don’t decrypt by default (explicit, audited reveal flow).
- **Auth**: OIDC (e.g., Zitadel/Okta/Auth0). Session probe, login, and logout (RP-logout optional).
- **API**: `/api/...` (JSON), static SPA served by backend.
- **SOPS CLI integration**: server executes `sops` for encryption/decryption; no plaintext secrets are stored.
- Health-aware state pills (running/healthy/exited etc.).
- Stack Files page: view (and optionally edit) compose/env/scripts vs runtime context; gated decryption for SOPS.
- Docker Cleanup Page: Do the equivalent of a docker prune or clear your build cache from the comfort of the UI.

---

## 📚 Documentation

- **[Installation (Docker Compose)](docs/Installation.md)** — requirements, compose example, `.env`, Nginx
- **[Development](docs/Development.md)** — run the UI/API locally
- **[Configuration](docs/Configuration.md)** — all environment variables
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

## License (Open Core for non‑commercial use)
DD-UI is offered under an **open-core, non‑commercial** model:

- For **home, personal, student, hobbyist, research, and other non‑commercial uses**, DD-UI is free to use with **all features enabled**.
- For **commercial use** (including use inside a business, paid consulting, hosted/SaaS for customers, or any revenue‑generating context), please obtain a **commercial license** from the maintainer.

The project adopts the **Prosperity Public License 3.0.0 (Noncommercial)** as the baseline, plus DD-UI‑specific **Additional Terms** to clarify that all features remain available to non‑commercial users. See `LICENSE.md` for details and contact information.

> _This section is a human‑readable summary and not a substitute for the license. Nothing here grants rights by itself._

---

## Disclaimer
The Software provided hereunder (“Software”) is licensed “as‑is,” without warranties of any kind—express, implied, or telepathically transmitted. The Softwarer (yes, that’s totally a word now) makes no promises about functionality, performance, compatibility, security, or availability—and absolutely no warranty of any sort. The developer shall not be held responsible, even if the software is clearly the reason your dog decided to orchestrate its own sidecar, your mom scored five tickets to Hawaii but you missed out because you were knee‑deep in a `docker compose` rabbit hole, or your stack drifted so hard it achieved sentience and renamed itself.

If using this orchestration UI leads you down a rabbit hole of obsessive network optimizations, breaks your fragile grasp of version pinning, or causes an uprising among your offline‑first containers—sorry, still not liable. Also not liable if your repo syncs so fast it rips a hole in the space‑time continuum **or** if your `.env` files multiply like Tribbles. The developer likewise claims no credit for anything that actually goes right either. Any positive experiences are owed entirely to the unstoppable force that is the Open Source community.

It’s never been a better time to be a PC user—or a homelabber. Just don’t blame me when YAML inevitably eats your weekend.
