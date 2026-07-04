# Roadmap & Status

## Nearing MVP / Pre-release

> DD-UI is 85-95% functional. Core functionality is present, there are some advanced features that are partially implemented. Known issues are listed below. CHEERS!

## Project scope & cadence

- This project is built and maintained by **one person**. A large portion of the current codebase landed in **~1 week** of focused work.
- Development will continue and is driven by the maintainer’s available time.
- DD-UI is **opinionated**—it reflects how I run Docker in my homelab (declarative IaC, secrets via SOPS/AGE, minimal ceremony). If that resonates, you’ll likely feel right at home.
- I am working on two other projects currently, the most important one being one I am being paid to work on. And the other one is a *suprise* for some of you who endorsed my other projects. This will be running home prod for me so it does not make it any less of a priority, but now I have achieved functionality preview I am feeling I might slow the base and trickle minor bugfixes daily. It's honestly not that much left to do and I wanna make sure I am not neglecting the project I am getting paid for.

## Planned / Known Issues

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
