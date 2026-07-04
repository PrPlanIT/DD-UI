# DD-UI — Screenshots

A tour of DD-UI in action. ← back to the [README](../README.md).

## 🔐 Encryption on the fly (SOPS / AGE)
The headline feature: compose and `.env` values stay encrypted at rest and are revealed only on an explicit, audited request — DD-UI can still deploy them encrypted.

<img src="../example/screenshots/DD-UI-Stack-Encrypted.png" width="820" alt="A stack with SOPS-encrypted compose/env values" />

<img src="../example/screenshots/DD-UI-Decrypted-Deployed.png" width="820" alt="Decrypted and deployed" />

## 📚 Stacks & hosts
Every stack across every host in one view, with per-stack drift indicators.

<img src="../example/screenshots/DD-UI-Host-Stacks.png" width="820" alt="Host stacks overview" />

<img src="../example/screenshots/DD-UI-Hosts.png" width="820" alt="Hosts inventory with metrics" />

<img src="../example/screenshots/DD-UI-Host-Stack_Detail-Collapsed.png" width="820" alt="Stack detail — collapsed" />

<img src="../example/screenshots/DD-UI-Host-Stack_Detail-Expanded.png" width="820" alt="Stack detail — expanded" />

## ✏️ Editing
Monaco (the editor from VS Code) for compose, `.env`, and scripts.

<img src="../example/screenshots/DD-UI-Stack-Compose-Editor.png" width="820" alt="Compose editor" />

## 👁️ Observability
Live logs, a dedicated logging view, per-container stats, and an in-container terminal.

<img src="../example/screenshots/DD-UI-Logs.png" width="820" alt="Container logs" />

<img src="../example/screenshots/DD-UI-Logging.png" width="820" alt="Dedicated logging view with filters" />

<img src="../example/screenshots/DD-UI-Stats.png" width="820" alt="Container stats" />

<img src="../example/screenshots/DD-UI-Terminal.png" width="820" alt="In-container terminal (xterm)" />

## 🧱 Docker resources
Images, networks, volumes, and a cleanup/prune page.

<img src="../example/screenshots/DD-UI-Images.png" width="820" alt="Images" />

<img src="../example/screenshots/DD-UI-Networks.png" width="820" alt="Networks" />

<img src="../example/screenshots/DD-UI-Volumes.png" width="820" alt="Volumes" />

<img src="../example/screenshots/DD-UI-Cleanup.png" width="820" alt="Docker cleanup / prune" />
