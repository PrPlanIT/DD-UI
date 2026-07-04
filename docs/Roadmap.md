# Roadmap & Status

## Nearing MVP / Pre-release

> DD-UI is 85-95% functional. Core functionality is present, there are some advanced features that are partially implemented. Known issues are listed below. CHEERS!

## Project scope & cadence

- This project is built and maintained by **one person**. A large portion of the current codebase landed in **~1 week** of focused work.
- Development will continue and is driven by the maintainer’s available time.
- DD-UI is **opinionated**—it reflects how I run Docker in my homelab (declarative IaC, secrets via SOPS/AGE, minimal ceremony). If that resonates, you’ll likely feel right at home.
- Realistically, DD-UI shares my time with a handful of other projects and the orgs I support — I especially want to get **StageFreight** solid, and there's **DisplayWizard** and plenty more I'd love to push on. DD-UI isn't backed by a big community or sponsorships, so it competes with everything else on my plate, and I can only do so much at once. Expect modest but steady progress: minor fixes trickled as I can, bigger features when I get a focused stretch. It runs my own home prod, so it's not going anywhere — just don't expect a full-time pace.

## Planned / Known Issues

- **Git sync is currently unstable.** GitRepository syncing (one-tree backup/restore of stack configuration) is being reworked — pending fix.
- Bug when a file is open outside DD-UI it can create an empty temp file next to the file after saving.
- Maybe an enhanced approach for caching tags of orphaned / stranded images, the current approach for some images that are built at runtime it can be weird seeing it as just ?? in the menu. I want visibility for that.
- Groups and internal DD-UI variable are of the few things planned to test next. The GUI is ready, the inventory system can read and interpret all the files. I just need to validate that drift and prune is properly working and then its just putting this into home prod and seeing if it lets me down
- Perhaps a local admin user.
- A settings menu.
- A theme menu.
- More testing & bugfixes!
- Whatever idea I have that I suddenly think we can not live without!

Features are evolving; treat all APIs and UI as unstable for now.
Environment Variables are unlikely to change.
