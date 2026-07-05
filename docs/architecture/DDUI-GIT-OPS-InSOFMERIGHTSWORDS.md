The git sync feature in DD-UI is very particular


DD-UI initializes a clone of the remote repo when a repo is selected.
When a Git Repository is entered and sync  is enabled every 5 seconds DD-UI attempts to do either a push/pull/sync:
1. push mode every interval it pulls the  most recent state of the repo and then copies the /data/inventory, /data/docker-compose files with `rsync -avz -delete` into the remote repo and creates a new commit
2. pull mode:
    a. it clones/updates the latest version of the remote repository
    b. if the contents differ from local files it rsync -avz the local files into the remote repo, stages a commit "info: DD-UI ignored these local changes" (This preserves the ignored changes in history)
    c. it references the content of the commit prior to this new one and copies the entire content of the remote /data/inventory & /data/docker-compose to the remote repo `rsync -avz -delete` into its /data folder.
2. sync mode it looks at the modification time of the inventory and then applies which ever is the most recent change it then makes a pull or push with the same methods above rsync -avz -delete /data/docker-compose/ /data/.git-work/docker-compose/ etc.
The design allows users to fallback on git history in the incident something is changed accidentally as DD-UI never rewrites history and always checks it is up to date before edits/push.

Note for claude: There is no need for force logic. This logic above should suffice ^

● Perfect! This is a much cleaner design. Let me summarize to make sure I understand:

  DD-UI Git Sync Design

  Core Principle:

  - Always pull before push - Never lose history
  - Use rsync with --delete - Exact mirror of directories
  - Every 5 seconds - Check and sync based on mode

  Three Modes:

  1. Push Mode (Local → Remote)
  - Pull latest from remote
  - rsync -avz --delete /data/inventory /data/.git-work/inventory
  - rsync -avz --delete /data/docker-compose/ /data/.git-work/docker-compose/
  - Commit changes
  - Push to remote

  2. Pull Mode (Remote → Local)
  - Pull latest from remote
  - rsync -avz --delete /data/.git-work/inventory /data/inventory
  - rsync -avz --delete /data/.git-work/docker-compose/ /data/docker-compose/
  - Local files match remote exactly

  3. Sync Mode (Smart Bidirectional)
  - Check modification times
  - If local is newer → do Push
  - If remote is newer → do Pull
  - Whichever has the most recent change wins

  Key Benefits:

  - No force push ever needed - Always building on history
  - Perfect rollback - Every change is a git commit
  - No merge conflicts - rsync --delete ensures exact copies
  - Simple recovery - Just check git log and revert if needed

  Initial Setup:

  When repo URL is first entered:
  1. Clone remote to /data/.git-work
  2. If repo is empty → immediate push of local files
  3. If repo has files → user chooses initial direction (pull or push)

  This is beautifully simple! No complex conflict resolution, no force flags, just clean rsync copies with git history as the safety net. Should we implement this now?