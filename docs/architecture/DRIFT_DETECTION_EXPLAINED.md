# DD-UI Drift Detection Explained

## What is Drift?

Drift occurs when the actual running state of your Docker containers differs from what's defined in your IaC (Infrastructure as Code) files. DD-UI uses a sophisticated two-tier hash-based system to detect drift efficiently.

## How Drift Detection Works

### Two-Tier Detection System

DD-UI uses a hierarchical approach to minimize expensive Docker API calls while maintaining accuracy:

#### Tier 1: IaC File Change Detection (Bundle Hash)
- **What it tracks**: All IaC files (compose.yml, .env, etc.) associated with a stack
- **How it works**:
  1. Computes SHA256 hash of all IaC files combined (bundle hash)
  2. Compares with previously stored bundle hash
  3. If different, drift is detected immediately
- **Trigger**: Any change to IaC files (content, addition, deletion)

#### Tier 2: Container Configuration Detection (Docker Config Hash)
- **What it tracks**: Docker's internal configuration hash for each container
- **How it works**:
  1. Only checked if IaC files haven't changed (Tier 1 passes)
  2. Reads `com.docker.compose.config-hash` label from containers
  3. Compares with cached hashes from last successful deployment
- **Trigger**: Container configuration changes outside of DD-UI

## What Triggers Drift?

### Common Drift Scenarios

1. **IaC File Changes** (Tier 1)
   - Editing compose.yml, docker-compose.yml files
   - Modifying .env files
   - Adding/removing service definitions
   - Changing image versions in compose files
   - Modifying volume/network configurations

2. **Container Changes** (Tier 2)
   - Manual container updates via `docker` CLI
   - Container recreated with different configuration
   - Service scaled manually outside DD-UI
   - Container removed but still defined in IaC
   - Container exists but not defined in IaC (orphaned)

3. **Specific Drift Reasons You'll See**:
   - `"IaC files changed since last deployment"` - Your stack files were modified
   - `"Container configurations changed"` - Containers were modified outside DD-UI
   - `"No drift detected"` - Everything matches expected state
   - `"Unable to verify container state"` - Docker API error (treated as no drift)

## Understanding the Status Cards

On the Stacks page, you'll see metrics like:
- **Stacks**: Total number of stacks on the selected host
- **Containers**: Total running containers
- **Drift** (amber number): Count of stacks with detected drift
- **Errors** (red number): Containers in error states (dead, exited abnormally)

## How to Resolve Drift

When a stack shows drift:

1. **Review the Changes**
   - Click on the stack to see the Stack Detail view
   - Compare the "Runtime" vs "IaC" states
   - Look for services marked as "missing" or image mismatches

2. **Deploy to Sync**
   - If IaC is correct: Deploy the stack to update containers
   - If containers are correct: Update IaC files to match reality

3. **Auto DevOps**
   - Enable Auto DevOps for automatic drift correction
   - DD-UI will automatically deploy when drift is detected

## Performance Optimizations

- **Caching**: Docker config hashes are cached to avoid repeated API calls
- **Bundle Hash**: Quick file comparison before expensive container checks
- **Lazy Evaluation**: Drift only checked when viewing stacks or during scans
- **Database Storage**: Uses `stack_drift_cache` table for persistence

## Database Schema

The drift detection system uses:
```sql
stack_drift_cache (
  stack_id: int64          -- References iac_stacks
  bundle_hash: string      -- SHA256 of all IaC files
  docker_config_cache: jsonb -- Map of service->config_hash
  last_updated: timestamp
)
```

## When Drift Detection Runs

1. **On-Demand**: When viewing the Stacks page
2. **After Deployment**: Cache updated on successful deploy
3. **During Scan**: When clicking "Sync" button
4. **API Calls**: When fetching enhanced IaC data

## Testing URL

For future testing of DD-UI:
- **URL**: https://dd-ui.pcfae.com
- **Note**: You'll need proper authentication credentials to access

## Troubleshooting

### Stack shows drift but shouldn't?
- Check if someone manually updated containers
- Verify SOPS decryption is working (encrypted values cause drift)
- Look for orphaned containers with old project labels

### Drift not detecting changes?
- Ensure you're viewing the latest data (click Sync)
- Check that IaC files are properly saved to database
- Verify Docker API connectivity to the host

### Performance issues?
- Large numbers of containers may slow detection
- Consider increasing Docker API timeout
- Check database performance for cache queries

## Technical Implementation

The core logic lives in:
- `/srv/DDUI/api/utils/hash.go` - Hash computation and caching
- `/srv/DDUI/api/utils/hash_drift.go` - Wrapper functions
- `/srv/DDUI/api/services/db_iac.go` - Integration with enhanced IaC

The detection follows this flow:
1. Compute current bundle hash from IaC files
2. Compare with stored hash - if different, report drift
3. If same, check Docker config hashes
4. Compare with cached hashes - if different, report drift
5. Update caches as needed
