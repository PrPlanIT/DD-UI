package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"dd-ui/common"
	"dd-ui/database"
	"dd-ui/middleware"
	"dd-ui/services"
)

// GitSyncHandlers handles Git synchronization API endpoints
type GitSyncHandlers struct{}

// NewGitSyncHandlers creates a new Git sync handler
func NewGitSyncHandlers() *GitSyncHandlers {
	return &GitSyncHandlers{}
}

// GetConfig returns the current Git sync configuration
func (h *GitSyncHandlers) GetConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	config, err := database.GetGitSyncConfig(ctx)
	if err != nil {
		common.ErrorLog("Failed to get git config: %v", err)
		http.Error(w, "Failed to get configuration", http.StatusInternalServerError)
		return
	}

	// If config is nil (shouldn't happen with our fixes), return empty config
	if config == nil {
		config = &database.GitSyncConfig{
			RepoURL:           "",
			Branch:            "main",
			CommitAuthorName:  "DD-UI Bot",
			CommitAuthorEmail: "ddui@localhost",
			PullIntervalMins:  5,
		}
	}

	// Don't expose sensitive fields in full
	response := map[string]interface{}{
		"repo_url":            config.RepoURL,
		"branch":              config.Branch,
		"has_token":           config.AuthToken != "",
		"has_ssh_key":         config.SSHKey != "",
		"commit_author_name":  config.CommitAuthorName,
		"commit_author_email": config.CommitAuthorEmail,
		"sync_enabled":        config.SyncEnabled,
		"sync_mode":           config.SyncMode,
		"force_on_conflict":   config.ForceOnConflict,
		"auto_push":           config.AutoPush,
		"auto_pull":           config.AutoPull,
		"pull_interval_mins":  config.PullIntervalMins,
		"push_on_change":      config.PushOnChange,
		"sync_path":           config.SyncPath,
	}

	common.DebugLog("Returning config: repo=%s, branch=%s, author=%s <%s>",
		response["repo_url"], response["branch"],
		response["commit_author_name"], response["commit_author_email"])

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateConfig updates the Git sync configuration
func (h *GitSyncHandlers) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUserEmail(ctx)

	var config database.GitSyncConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		common.ErrorLog("Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Force push_on_change to always be true
	config.PushOnChange = true
	// Ignore sync_path from frontend - derive from DD_UI_IAC_ROOT
	config.SyncPath = strings.TrimSpace(common.Env("DD_UI_IAC_ROOT", "/data"))

	common.DebugLog("Received git config update: repo=%s, branch=%s, has_token=%v, has_key=%v",
		config.RepoURL, config.Branch, config.AuthToken != "", config.SSHKey != "")
	common.DebugLog("Full config received: author_name=%s, author_email=%s, sync_enabled=%v, sync_mode=%s, force=%v, path=%s",
		config.CommitAuthorName, config.CommitAuthorEmail, config.SyncEnabled,
		config.SyncMode, config.ForceOnConflict, config.SyncPath)

	// Get existing config to preserve tokens if not updated
	existing, existErr := database.GetGitSyncConfig(ctx)
	if existErr != nil {
		common.DebugLog("No existing config or error: %v", existErr)
	}

	if existing != nil {
		// Only preserve credentials if they weren't provided
		// The frontend should send actual values for all other fields
		if config.AuthToken == "" || config.AuthToken == "***UNCHANGED***" {
			config.AuthToken = existing.AuthToken
		}
		if config.SSHKey == "" || config.SSHKey == "***UNCHANGED***" {
			config.SSHKey = existing.SSHKey
		}
	}

	// Update configuration - this should always save, regardless of connection status
	gitSync := services.GetGitSync()
	if err := gitSync.UpdateConfig(ctx, &config); err != nil {
		common.ErrorLog("Failed to update git config: %v", err)
		http.Error(w, fmt.Sprintf("Failed to update configuration: %v", err), http.StatusInternalServerError)
		return
	}

	common.InfoLog("Git sync config updated by %s", user)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Configuration saved successfully",
	})
}

// GetStatus returns the current Git sync status
func (h *GitSyncHandlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get status from database
	dbStatus, err := database.GetGitSyncStatus(ctx)
	if err != nil {
		common.ErrorLog("Failed to get git status: %v", err)
		http.Error(w, "Failed to get status", http.StatusInternalServerError)
		return
	}

	// Get runtime status from service
	gitSync := services.GetGitSync()
	runtimeStatus := gitSync.GetStatus()

	// Merge statuses
	for k, v := range runtimeStatus {
		dbStatus[k] = v
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dbStatus)
}

// Pull triggers a manual pull from the remote repository
func (h *GitSyncHandlers) Pull(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUserEmail(ctx)

	gitSync := services.GetGitSync()
	if err := gitSync.Pull(ctx, user); err != nil {
		common.ErrorLog("Git pull failed: %v", err)

		response := map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		}

		// Check for conflicts
		if conflicts, _ := database.GetUnresolvedConflicts(ctx); len(conflicts) > 0 {
			response["conflicts"] = conflicts
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	common.InfoLog("Git pull completed by %s", user)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Repository pulled successfully",
	})
}

// Push triggers a manual push to the remote repository
func (h *GitSyncHandlers) Push(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUserEmail(ctx)

	// Get commit message from request
	var req struct {
		Message string `json:"message"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	gitSync := services.GetGitSync()
	if err := gitSync.Push(ctx, req.Message, user); err != nil {
		common.ErrorLog("Git push failed: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	common.InfoLog("Git push completed by %s", user)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Changes pushed successfully",
	})
}

// Sync performs a full sync (pull then push)
func (h *GitSyncHandlers) Sync(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUserEmail(ctx)

	gitSync := services.GetGitSync()
	if err := gitSync.Sync(ctx, user); err != nil {
		common.ErrorLog("Git sync failed: %v", err)

		response := map[string]interface{}{
			"status":  "error",
			"message": err.Error(),
		}

		// Check for conflicts
		if conflicts, _ := database.GetUnresolvedConflicts(ctx); len(conflicts) > 0 {
			response["conflicts"] = conflicts
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	}

	common.InfoLog("Git sync completed by %s", user)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Repository synchronized successfully",
	})
}

// GetLogs returns recent Git sync operation logs
func (h *GitSyncHandlers) GetLogs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get limit from query params
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	logs, err := database.GetGitSyncLogs(ctx, limit)
	if err != nil {
		common.ErrorLog("Failed to get git logs: %v", err)
		http.Error(w, "Failed to get logs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(logs)
}

// GetConflicts returns unresolved Git conflicts
func (h *GitSyncHandlers) GetConflicts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	conflicts, err := database.GetUnresolvedConflicts(ctx)
	if err != nil {
		common.ErrorLog("Failed to get conflicts: %v", err)
		http.Error(w, "Failed to get conflicts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(conflicts)
}

// ResolveConflict marks a conflict as resolved
func (h *GitSyncHandlers) ResolveConflict(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUserEmail(ctx)

	var req struct {
		ConflictID     int    `json:"conflict_id"`
		ResolutionType string `json:"resolution_type"` // "local", "remote", "manual"
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := database.ResolveGitConflict(ctx, req.ConflictID, req.ResolutionType, user); err != nil {
		common.ErrorLog("Failed to resolve conflict: %v", err)
		http.Error(w, "Failed to resolve conflict", http.StatusInternalServerError)
		return
	}

	common.InfoLog("Conflict %d resolved by %s using %s", req.ConflictID, user, req.ResolutionType)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Conflict resolved",
	})
}

// CheckInitialSetupConflict checks if both local and remote have files during initial setup
func (h *GitSyncHandlers) CheckInitialSetupConflict(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get current config
	config, err := database.GetGitSyncConfig(ctx)
	if err != nil || config == nil || config.RepoURL == "" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"has_conflict": false,
			"message":      "No repository configured",
		})
		return
	}

	// Check for initial setup conflict
	hasConflict := false
	message := ""

	// Check if local has docker-compose or inventory files
	hasLocalFiles := false
	syncPath := "/data"

	// Check for local docker-compose directory
	if info, err := os.Stat(syncPath + "/docker-compose"); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(syncPath + "/docker-compose")
		if len(entries) > 0 {
			hasLocalFiles = true
		}
	}

	// Check for local inventory file
	if _, err := os.Stat(syncPath + "/inventory"); err == nil {
		hasLocalFiles = true
	}

	// Only check remote if we have local files
	if hasLocalFiles {
		// Test connection to see if the remote exists and already carries the branch.
		// An unreachable or empty remote is not a conflict, so any error here simply
		// leaves hasConflict false — same as the previous behaviour, which ignored a
		// failed ls-remote.
		if heads, err := listRemoteHeads(config.RepoURL, config.AuthToken, config.SSHKey); err == nil && branchInHeads(heads, config.Branch) {
			// Remote exists and has content - this indicates potential conflict
			hasConflict = true
			message = "Both local (/data) and remote repository contain files. Choose Push to overwrite remote with local, Pull to overwrite local with remote, or manually resolve before enabling sync."
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"has_conflict":    hasConflict,
		"message":         message,
		"has_local_files": hasLocalFiles,
	})
}

// TestConnection tests the Git repository connection
func (h *GitSyncHandlers) TestConnection(w http.ResponseWriter, r *http.Request) {

	common.DebugLog("TestConnection: Starting connection test")

	var req struct {
		RepoURL   string `json:"repo_url"`
		Branch    string `json:"branch"`
		AuthToken string `json:"auth_token"`
		SSHKey    string `json:"ssh_key"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		common.ErrorLog("TestConnection: Failed to decode request: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Invalid request format",
		})
		return
	}

	// Default branch if not specified
	if req.Branch == "" {
		req.Branch = "main"
	}

	common.DebugLog("TestConnection: Testing repo=%s, branch=%s, has_token=%v, has_key=%v",
		req.RepoURL, req.Branch, req.AuthToken != "", req.SSHKey != "")

	// Validate inputs
	if req.RepoURL == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "Repository URL is required",
		})
		return
	}

	// Validate URL and authentication combination
	isHTTPS := strings.HasPrefix(req.RepoURL, "https://") || strings.HasPrefix(req.RepoURL, "http://")
	isSSH := strings.HasPrefix(req.RepoURL, "git@") || strings.Contains(req.RepoURL, "ssh://")

	if isHTTPS && req.SSHKey != "" && req.AuthToken == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "HTTPS URL requires a token, not SSH key. Either use SSH URL format (git@gitlab.prplanit.com:user/repo.git) or provide a Personal Access Token instead of SSH key.",
		})
		return
	}

	if isSSH && req.AuthToken != "" && req.SSHKey == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "SSH URL requires an SSH key, not a token. Either use HTTPS URL format or provide an SSH key instead of token.",
		})
		return
	}

	// If no credentials provided in request, try to use stored credentials
	if req.AuthToken == "" && req.SSHKey == "" {
		// Try to get stored config
		if storedConfig, err := database.GetGitSyncConfig(r.Context()); err == nil && storedConfig != nil {
			common.DebugLog("TestConnection: No credentials in request, using stored credentials")
			// Use stored credentials if available
			if storedConfig.AuthToken != "" {
				req.AuthToken = storedConfig.AuthToken
			}
			if storedConfig.SSHKey != "" {
				req.SSHKey = storedConfig.SSHKey
			}
		}
	}

	// Test the connection by listing the remote's branches. No git binary is involved:
	// the runtime image has none, so the previous `git ls-remote` failed there for every
	// repository and credential alike.
	common.DebugLog("TestConnection: Listing remote heads for %s", req.RepoURL)
	heads, listErr := listRemoteHeads(req.RepoURL, req.AuthToken, req.SSHKey)
	if listErr != nil {
		if errorMsg := remoteErrorMessage(listErr); errorMsg != "" {
			common.ErrorLog("TestConnection: %v", listErr)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "error",
				"message": errorMsg,
			})
			return
		}
		// An empty remote is reachable and authorized — the connection test passed,
		// there is simply no branch on it yet.
		heads = nil
	}

	// Check if branch exists
	branchExists := branchInHeads(heads, req.Branch)

	response := map[string]interface{}{
		"status":        "success",
		"message":       "Connection successful",
		"branch_exists": branchExists,
	}

	if !branchExists {
		response["message"] = fmt.Sprintf("Connection successful but branch '%s' not found", req.Branch)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
