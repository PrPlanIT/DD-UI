# Git Sync Feature Documentation

## Overview
The Git Sync feature enables DD-UI to synchronize its entire configuration (inventory files and Docker Compose stacks) with a Git repository. This provides version control, backup, and multi-instance synchronization capabilities.

## Quick Start

### 1. Configure Repository
1. Navigate to **Settings → Git Sync** in the UI
2. Enter your repository URL (HTTPS or SSH)
3. Add authentication:
   - **For GitHub**: Use a Personal Access Token
   - **For SSH**: Paste your private key
4. Set author information for commits
5. Click **Save Configuration**

### 2. Test Connection
Click **Test Connection** to verify repository access before enabling sync.

### 3. Enable Sync Options
- **Enable sync**: Master toggle for Git synchronization
- **Auto-pull changes**: Periodically fetch remote changes
- **Auto-push changes**: Automatically push local changes
- **Push on configuration change**: Immediate push when configs change

### 4. Manual Operations
- **Pull**: Fetch and merge remote changes
- **Push**: Commit and push local changes
- **Sync**: Pull then push (full synchronization)

## Features

### Authentication Support
- **HTTPS with Token**: GitHub/GitLab personal access tokens
- **SSH with Private Key**: Standard SSH key authentication
- Credentials encrypted with SOPS when available

### Synchronization Modes
1. **Manual Only**: User controls all sync operations
2. **Auto-Pull**: Periodic pulls from remote (configurable interval)
3. **Auto-Push**: Automatic push on local changes
4. **Bidirectional**: Both auto-pull and auto-push enabled

### Conflict Detection
- Automatic conflict detection during pull
- Conflicts logged and displayed in UI
- Manual resolution required for conflicts

### What Gets Synced
```
/data/
├── inventory.yml           # Host inventory
├── hosts/                  # Per-host configurations
│   ├── server1/
│   │   ├── nginx-stack.yml # Docker Compose files
│   │   └── app-stack.yml
│   └── server2/
│       └── db-stack.yml
└── .git/                  # Git repository
```

## UI Components

### Settings Page
Full configuration interface at **Settings → Git Sync**:
- Repository configuration
- Authentication setup
- Sync options
- Operation logs
- Conflict management

### Header Toggle
Quick sync toggle in Hosts and Stacks pages:
- Green: Sync enabled and active
- Gray: Sync disabled
- Click to enable/disable quickly

## Security

### Credential Protection
- Tokens/keys encrypted in database
- Never exposed in logs
- Sanitized in API responses

### Best Practices
- Use dedicated service accounts
- Limit repository permissions
- Regular credential rotation
- Monitor sync logs for anomalies

## Troubleshooting

### Connection Issues
If connection test fails:
1. Verify repository URL is correct
2. Check token/SSH key permissions
3. Ensure repository exists and is accessible
4. For private repos, authentication is required

### Save Configuration Failing
If configuration won't save:
1. Check all required fields are filled
2. Verify repository URL format
3. Review browser console for errors
4. Check API logs with `DD_UI_LOG_LEVEL=debug`

### Sync Conflicts
When conflicts occur:
1. Check the conflicts section in settings
2. Manually resolve in repository
3. Pull changes again
4. Mark conflicts as resolved in UI

### Common Error Messages

| Error | Cause | Solution |
|-------|-------|----------|
| "Authentication failed" | Invalid token/key | Verify credentials have repository access |
| "Branch not found" | Branch doesn't exist | Create branch or use existing one |
| "Permission denied" | Insufficient access | Check repository permissions |
| "Merge conflict" | Conflicting changes | Manually resolve conflicts |

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/git/config` | GET | Get current configuration |
| `/api/git/config` | POST | Update configuration |
| `/api/git/status` | GET | Get sync status |
| `/api/git/pull` | POST | Trigger manual pull |
| `/api/git/push` | POST | Trigger manual push |
| `/api/git/sync` | POST | Full sync operation |
| `/api/git/logs` | GET | Get operation history |
| `/api/git/test` | POST | Test repository connection |

## Advanced Configuration

### Environment Variables
- `DD_UI_DATA_PATH`: Base sync directory (default: `/data`)
- `DD_UI_GIT_PATH`: Git directory (default: `/data/.git`)
- `DD_UI_LOG_LEVEL=debug`: Enable debug logging

### Commit Messages
Automatic commits use descriptive messages:
- "Update 2 inventory, 3 stacks, 1 hosts"
- Manual pushes can specify custom messages

### Auto-Pull Interval
Default: 5 minutes
Configurable: 1-60 minutes in settings

## Integration with DD-UI

### Inventory Reload
When inventory files change:
1. Changes detected during pull
2. Inventory automatically reloaded
3. New hosts appear in UI

### Stack Management
- Compose files synced between instances
- Deployment state remains local
- Re-deploy required after pulling stack changes

## Best Practices

### Repository Setup
1. Create dedicated repository for DD-UI configs
2. Initialize with README explaining structure
3. Use branch protection for production
4. Enable audit logging

### Team Collaboration
1. Use meaningful commit messages
2. Pull before making changes
3. Test configurations locally
4. Document custom stack configurations

### Backup Strategy
1. Enable auto-pull for backup instance
2. Use separate branch for testing
3. Regular repository backups
4. Monitor sync logs for failures

## Limitations

### Current Limitations
- No selective file sync (all or nothing)
- Conflicts require manual resolution
- No built-in diff viewer
- Single branch support only

### Not Synced
- Container runtime state
- Deployment timestamps
- Local logs and metrics
- UI preferences

## Example Workflows

### Initial Setup
```bash
# 1. Create repository
git init dd-ui-config
cd dd-ui-config

# 2. Add initial structure
mkdir -p hosts
echo "# DD-UI Configuration" > README.md

# 3. Push to remote
git remote add origin https://github.com/org/dd-ui-config
git push -u origin main

# 4. Configure in DD-UI Settings
```

### Disaster Recovery
```bash
# 1. Deploy fresh DD-UI instance
# 2. Configure Git sync with backup repo
# 3. Pull configuration
# 4. All hosts and stacks restored
```

### Multi-Instance Sync
```yaml
# Instance A: Production
sync_enabled: true
auto_push: true
auto_pull: false

# Instance B: Backup
sync_enabled: true  
auto_push: false
auto_pull: true
```

## Support

For issues or questions:
1. Check operation logs in settings
2. Enable debug logging for details
3. Review this documentation
4. Check GitHub issues for known problems

## Summary
Git Sync provides powerful version control integration for DD-UI configurations, enabling backup, collaboration, and multi-instance synchronization with automatic conflict detection and secure credential management.