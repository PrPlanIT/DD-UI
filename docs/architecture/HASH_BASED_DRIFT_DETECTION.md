# DD-UI Hash-Based Drift Detection Strategy

## Overview
This document defines the comprehensive strategy for efficient drift detection using hash-based comparison. The system uses bundle hashes for IaC file changes and Docker config hashes as the source of truth for container state.

## Core Philosophy: Hash-Based Efficiency

### What This Replaces (Complete Replacement)
- **Container name matching logic** - No more fuzzy string matching
- **Live container inspection** - No more expensive Docker API calls for drift detection
- **Service-to-container mapping** - Docker's config hashes are the source of truth
- **Complex matching algorithms** - Simple hash comparison only

### New Hash-Based Approach
- **Bundle hash comparison** - Detect IaC file changes via hash comparison
- **Docker config hashes** - Use Docker Compose's own config hashes as source of truth
- **Two-tier detection** - Separate file changes from container changes
- **Cache invalidation** - Smart cache clearing when bundle changes

## Variable Interpolation & SOPS (Deterministic Rendering)

### CRITICAL: All Files May Be SOPS-Encrypted
**IMPORTANT**: The current implementation supports SOPS encryption for **ALL** files in the stack:
- `docker-compose.yml` / `docker-compose.yaml` - **CAN BE ENCRYPTED**
- `.env` files (project-level and service-specific) - **CAN BE ENCRYPTED**
- Any `env_file` references - **CAN BE ENCRYPTED**

### SOPS Integration for Hash Calculation
**Bundle hashes MUST use decrypted content:**

```go
// Bundle hash calculation using decrypted content
func computeCurrentBundleHash(ctx context.Context, stackID int64) (string, error) {
    // 1. Stage all files with SOPS decryption
    stageDir, _, cleanup, err := stageStackForCompose(ctx, stackID)
    if cleanup != nil {
        defer cleanup()
    }

    // 2. Hash the decrypted content in staging directory
    return hashDirectoryContents(stageDir) // Uses decrypted files
}
```

**Key Points:**
- Bundle hash reflects **actual deployment state** (post-decryption, post-interpolation)
- Hash changes when encrypted values change, even if raw files look the same
- Staging directory contains decrypted content for accurate hashing

## Hash-Based Drift Detection Algorithm

### Two-Tier Detection System

```go
func detectDriftViaHashes(stackID int64) (bool, string, error) {
    // TIER 1: IaC File Change Detection
    currentBundleHash, err := computeCurrentBundleHash(ctx, stackID)
    if err != nil {
        return false, "", err
    }

    storedBundleHash, err := getStoredBundleHash(stackID)
    if err != nil {
        return false, "", err
    }

    // IaC files changed?
    if currentBundleHash != storedBundleHash {
        // Clear cached Docker hashes - forces container recheck
        if err := clearCachedDockerConfigHashes(stackID); err != nil {
            return false, "", err
        }

        // Update stored bundle hash
        if err := updateStoredBundleHash(stackID, currentBundleHash); err != nil {
            return false, "", err
        }

        return true, "IaC files changed since last deployment", nil
    }

    // TIER 2: Container Configuration Change Detection
    cachedDockerHashes, err := getCachedDockerConfigHashes(stackID)
    if err != nil {
        return false, "", err
    }

    actualDockerHashes, err := getActualDockerConfigHashes(stackID)
    if err != nil {
        return false, "", err
    }

    // Container configs changed?
    if !hashMapsEqual(cachedDockerHashes, actualDockerHashes) {
        // Update cache with current reality
        if err := storeCachedDockerConfigHashes(stackID, actualDockerHashes); err != nil {
            return false, "", err
        }

        return true, "Container configurations changed", nil
    }

    return false, "No drift detected", nil
}
```

### Cache Invalidation Strategy

When bundle hash changes, clear Docker config cache:

```go
func clearCachedDockerConfigHashes(stackID int64) error {
    // Clear the cache - next check will fetch fresh Docker hashes
    return db.Exec(ctx, `
        UPDATE stack_drift_cache
        SET docker_config_cache = '{}', last_updated = NOW()
        WHERE stack_id = $1
    `, stackID)
}
```

## Docker Config Hashes as Source of Truth

### Docker Compose Config Hash Labels

Docker Compose automatically sets config hash labels on containers:

```bash
# Example container labels
com.docker.compose.config-hash=sha256:a1b2c3d4e5f6...
com.docker.compose.service=web
com.docker.compose.project=myproject
```

### Lightweight Hash Collection

```go
func getActualDockerConfigHashes(stackID int64) (map[string]string, error) {
    // Get project label for filtering
    projectLabel := composeProjectLabelFromStack(stackName)

    // Filter containers by project
    filters := dockerFilters.NewArgs()
    filters.Add("label", "com.docker.compose.project="+projectLabel)

    containers, err := dockerClient.ContainerList(ctx, container.ListOptions{
        Filters: filters,
    })
    if err != nil {
        return nil, err
    }

    // Extract config hashes (lightweight operation)
    hashes := make(map[string]string)
    for _, cont := range containers {
        serviceName := cont.Labels["com.docker.compose.service"]
        configHash := cont.Labels["com.docker.compose.config-hash"]

        if serviceName != "" && configHash != "" {
            hashes[serviceName] = configHash
        }
    }

    return hashes, nil
}
```

## Database Schema

### Stack Drift Cache Table
```sql
CREATE TABLE stack_drift_cache (
    stack_id            BIGINT PRIMARY KEY REFERENCES iac_stacks(id),
    bundle_hash         TEXT NOT NULL,
    docker_config_cache JSONB NOT NULL DEFAULT '{}', -- service_name -> config_hash mapping
    last_updated        TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    INDEX idx_stack_drift_cache_updated (last_updated),
    INDEX idx_stack_drift_cache_bundle (bundle_hash)
);
```

### Cache Data Structure
```json
{
    "stack_id": 7,
    "bundle_hash": "sha256:abc123def456...",
    "docker_config_cache": {
        "web": "sha256:111aaa222bbb...",
        "api": "sha256:333ccc444ddd...",
        "database": "sha256:555eee666fff..."
    },
    "last_updated": "2024-12-12T10:30:00Z"
}
```

## Performance Characteristics

### Hash-Based Performance
| Operation | Performance |
|-----------|-------------|
| **Bundle unchanged + Docker cache hit** | ~1ms (hash string comparison only) |
| **Bundle unchanged + Docker cache miss** | ~50ms (Docker label query + cache update) |
| **Bundle changed** | ~200ms (SOPS decrypt + hash + cache clear) |

### Scalability
- **O(1) hash comparison** vs O(n×m) container×service matching
- **Minimal Docker API usage** - only labels, no full container inspection
- **Low memory footprint** - hash strings instead of container objects

## Integration Points

### Enhanced IAC API Integration
```go
func listEnhancedIacStacksForHost(ctx context.Context, hostName string) ([]EnhancedIacStackOut, error) {
    // ... existing logic ...

    for _, stack := range baseStacks {
        enhanced := EnhancedIacStackOut{IacStackOut: stack}

        // Use hash-based drift detection
        hasDrift, driftReason, err := detectDriftViaHashes(stack.ID)
        if err != nil {
            debugLog("Hash-based drift detection failed for stack %s: %v", stack.Name, err)
            return nil, err
        }

        enhanced.DriftDetected = hasDrift
        enhanced.DriftReason = driftReason
        debugLog("Stack %s: drift=%v, reason=%s", stack.Name, hasDrift, driftReason)

        // ... rest of existing logic ...
    }
}
```

### Deployment Integration
```go
func onSuccessfulDeployment(stackID int64) error {
    // Calculate and store bundle hash after successful deployment
    bundleHash, err := computeCurrentBundleHash(ctx, stackID)
    if err != nil {
        return err
    }

    // Get Docker config hashes from newly deployed containers
    dockerHashes, err := getActualDockerConfigHashes(stackID)
    if err != nil {
        return err
    }

    // Store both in cache
    return updateStackDriftCache(stackID, bundleHash, dockerHashes)
}
```

## Debug Logging Strategy

### Hash-Based Operations
```go
debugLog("Stack %s: bundle hash current=%s, stored=%s", stackName, currentHash, storedHash)
debugLog("Stack %s: bundle hash changed, clearing Docker config cache", stackName)
debugLog("Stack %s: Docker config hashes cached=%v, actual=%v", stackName, cachedHashes, actualHashes)
debugLog("Stack %s: drift detection via hashes: drift=%v, reason=%s", stackName, hasDrift, reason)
```

### SOPS Integration Logging
```go
debugLog("SOPS: staging %d files for bundle hash calculation", fileCount)
debugLog("SOPS: decrypted %d files, %d were encrypted", totalFiles, encryptedFiles)
debugLog("Bundle hash: calculated from %d decrypted files in %dms", fileCount, duration)
```

## Error Handling

### Cache Corruption Recovery
```go
if err := validateStackDriftCache(stackID); err != nil {
    debugLog("Stack %s: cache corrupted, rebuilding: %v", stackName, err)
    return rebuildStackDriftCache(stackID)
}
```

### Docker API Failures
```go
dockerHashes, err := getActualDockerConfigHashes(stackID)
if err != nil {
    debugLog("Stack %s: Docker API failed, using cached hashes: %v", stackName, err)
    return false, "Unable to verify container state", nil
}
```

This hash-based approach completely replaces the container matching system with a performant, deterministic solution that properly handles SOPS encryption throughout the pipeline.
