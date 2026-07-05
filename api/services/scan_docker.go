// src/api/scan_docker.go
package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"dd-ui/common"
	"dd-ui/database"
	"dd-ui/utils"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

// ===== helpers =====

var (
	ErrSkipScan = errors.New("skip scan") // sentinel for intentional skip
	sshEnvMu    sync.Mutex
)

func IsUnixSock(url string) bool { return strings.HasPrefix(url, "unix://") }

// LocalHostAllowed checks if local Docker socket access is allowed for a host
func LocalHostAllowed(h database.HostRow) bool {
	// 1) per-host opt-in (inventory var)
	if v := strings.ToLower(strings.TrimSpace(h.Vars["docker_local"])); v == "true" || v == "1" || v == "yes" {
		return true
	}
	// 2) env mapping
	if lh := strings.TrimSpace(common.Env("DD_UI_LOCAL_HOST", "")); lh != "" && strings.EqualFold(lh, h.Name) {
		return true
	}
	// 3) obvious localhost addresses
	switch strings.ToLower(strings.TrimSpace(h.Addr)) {
	case "127.0.0.1", "::1", "localhost":
		return true
	}
	return false
}

func DockerURLFor(h database.HostRow) (string, string) {
	// per-host override
	if v := h.Vars["docker_host"]; v != "" {
		return v, h.Vars["docker_ssh_cmd"]
	}

	// Check if this host matches DD_UI_LOCAL_HOST - if so, use local socket for performance
	if lh := strings.TrimSpace(common.Env("DD_UI_LOCAL_HOST", "")); lh != "" && strings.EqualFold(lh, h.Name) {
		sock := common.Env("DOCKER_SOCK_PATH", "/var/run/docker.sock")
		return "unix://" + sock, ""
	}

	kind := common.Env("DOCKER_CONNECTION_METHOD", "ssh") // ssh|tcp|local
	switch kind {
	case "local":
		sock := common.Env("DOCKER_SOCK_PATH", "/var/run/docker.sock")
		return "unix://" + sock, ""
	case "tcp":
		host := h.Addr
		if host == "" {
			host = h.Name
		}
		port := common.Env("DOCKER_TCP_PORT", "2375")
		return fmt.Sprintf("tcp://%s:%s", host, port), ""
	default: // ssh
		user := h.Vars["ansible_user"]
		if user == "" {
			user = common.Env("SSH_USER", "root")
		}
		addr := h.Addr
		if addr == "" {
			addr = h.Name
		}

		// Build SSH command for Docker CLI SSH support
		sshCmd := "ssh"
		if keyFile := common.Env("SSH_KEY_FILE", ""); keyFile != "" {
			sshCmd += " -i " + keyFile
		}
		if common.Env("SSH_STRICT_HOST_KEY", "true") == "false" {
			sshCmd += " -o StrictHostKeyChecking=no"
		}
		if port := common.Env("SSH_PORT", ""); port != "" && port != "22" {
			sshCmd += " -p " + port
		}

		// Return SSH URL format that Docker CLI expects
		return fmt.Sprintf("ssh://%s@%s", user, addr), sshCmd
	}
}

func withSSHEnv(cmd string, fn func() error) error {
	sshEnvMu.Lock()
	defer sshEnvMu.Unlock()
	prev, had := os.LookupEnv("DOCKER_SSH_CMD")
	if cmd != "" {
		_ = os.Setenv("DOCKER_SSH_CMD", cmd)
	}
	defer func() {
		if had {
			_ = os.Setenv("DOCKER_SSH_CMD", prev)
		} else {
			_ = os.Unsetenv("DOCKER_SSH_CMD")
		}
	}()
	return fn()
}

func DockerClientForURL(ctx context.Context, url, sshCmd string) (*client.Client, func(), error) {
	var cli *client.Client

	// Handle SSH connections with proper SSH transport
	if strings.HasPrefix(url, "ssh://") {
		common.DebugLog("SSH connection detected, using SSH transport")

		// Parse SSH URL to extract user and host
		user, host, err := utils.ParseSSHURL(url)
		if err != nil {
			return nil, nil, fmt.Errorf("invalid SSH URL: %v", err)
		}

		// Get SSH key file from environment
		keyFile := common.Env("SSH_KEY_FILE", "")
		if keyFile == "" {
			return nil, nil, fmt.Errorf("SSH_KEY_FILE not configured")
		}

		// Create Docker client with SSH transport
		cli, cleanup, err := utils.CreateSSHDockerClient(user, host, keyFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create SSH Docker client: %v", err)
		}

		// Test connection
		pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		_, err = cli.Ping(pctx, client.PingOptions{})
		if err != nil {
			cleanup()
			return nil, nil, fmt.Errorf("SSH Docker connection test failed: %v", err)
		}

		common.DebugLog("SSH Docker client created successfully for %s@%s", user, host)
		return cli, cleanup, nil
	}

	err := withSSHEnv(sshCmd, func() error {
		var err error

		// Set DOCKER_HOST environment variable
		prevHost := os.Getenv("DOCKER_HOST")
		if url != "" {
			os.Setenv("DOCKER_HOST", url)
		}
		defer func() {
			if prevHost != "" {
				os.Setenv("DOCKER_HOST", prevHost)
			} else if url != "" {
				os.Unsetenv("DOCKER_HOST")
			}
		}()

		cli, err = client.NewClientWithOpts(
			client.FromEnv,
			client.WithAPIVersionNegotiation(),
		)
		if err != nil {
			return err
		}
		pctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		_, err = cli.Ping(pctx, client.PingOptions{})
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	return cli, func() { _ = cli.Close() }, nil
}

// ===== small utils =====

func flattenPorts(pm network.PortMap) []map[string]any {
	out := make([]map[string]any, 0, len(pm))
	for port, binds := range pm {
		privateStr := port.Port()
		private, _ := strconv.Atoi(privateStr)
		typ := string(port.Proto())
		if len(binds) == 0 {
			out = append(out, map[string]any{
				"IP": "", "PublicPort": 0, "PrivatePort": private, "Type": typ,
			})
			continue
		}
		for _, b := range binds {
			pub, _ := strconv.Atoi(b.HostPort)
			hostIP := ""
			if b.HostIP.IsValid() {
				hostIP = b.HostIP.String()
			}
			out = append(out, map[string]any{
				"IP": hostIP, "PublicPort": pub, "PrivatePort": private, "Type": typ,
			})
		}
	}
	return out
}

// ===== main scan =====

func ScanHostContainers(ctx context.Context, hostName string) (int, error) {
	h, err := database.GetHostByName(ctx, hostName)
	if err != nil {
		return 0, err
	}
	url, sshCmd := DockerURLFor(h)

	// refuse scanning many hosts through a single local sock
	if IsUnixSock(url) && !LocalHostAllowed(h) {
		database.ScanLog(ctx, h.ID, "info", "skip local sock for non-local host",
			map[string]any{"url": url, "host": h.Name})
		return 0, ErrSkipScan
	}
	if strings.EqualFold(common.Env("DD_UI_SCAN_DOCKER_DEBUG", "false"), "true") {
		common.InfoLog("scan: host=%s docker_url=%s", h.Name, url)
	}

	cli, done, err := DockerClientForURL(ctx, url, sshCmd)
	if err != nil {
		database.ScanLog(ctx, h.ID, "error", "docker connect failed", map[string]any{"error": err.Error(), "url": url})
		return 0, err
	}
	defer done()

	listRes, err := cli.ContainerList(ctx, client.ContainerListOptions{All: true, Filters: client.Filters{}})
	if err != nil {
		database.ScanLog(ctx, h.ID, "error", "container list failed", map[string]any{"error": err.Error()})
		return 0, err
	}
	list := listRes.Items

	seen := make([]string, 0, len(list))
	saved := 0

	for _, c := range list {
		seen = append(seen, c.ID)

		ciRes, err := cli.ContainerInspect(ctx, c.ID, client.ContainerInspectOptions{})
		if err != nil {
			database.ScanLog(ctx, h.ID, "warn", "inspect failed", map[string]any{"id": c.ID, "error": err.Error()})
			continue
		}
		ci := ciRes.Container

		labels := map[string]string{}
		if ci.Config != nil && ci.Config.Labels != nil {
			labels = ci.Config.Labels
		}

		project := labels["com.docker.compose.project"]
		if project == "" {
			project = labels["com.docker.stack.namespace"]
		}
		var stackIDPtr *int64
		if project != "" {
			if sid, err := database.EnsureStack(ctx, h.ID, project, h.Owner); err == nil {
				stackID := sid
				stackIDPtr = &stackID
			} else {
				database.ScanLog(ctx, h.ID, "warn", "ensure stack failed", map[string]any{"project": project, "error": err.Error()})
			}
		}

		// ports/IP/env/networks/mounts/created
		var portsOut []map[string]any
		ip := ""
		if ci.NetworkSettings != nil {
			if ci.NetworkSettings.Ports != nil {
				portsOut = flattenPorts(ci.NetworkSettings.Ports)
			}
			// v29: NetworkSettings has no top-level IPAddress; the container IP
			// now lives on each network endpoint. Prefer the default bridge (matches
			// v28's top-level IPAddress); otherwise pick deterministically by network
			// name, since map iteration order is random.
			if ci.NetworkSettings.Networks != nil {
				if ep := ci.NetworkSettings.Networks["bridge"]; ep != nil && ep.IPAddress.IsValid() {
					ip = ep.IPAddress.String()
				} else {
					names := make([]string, 0, len(ci.NetworkSettings.Networks))
					for name := range ci.NetworkSettings.Networks {
						names = append(names, name)
					}
					sort.Strings(names)
					for _, name := range names {
						if ep := ci.NetworkSettings.Networks[name]; ep != nil && ep.IPAddress.IsValid() {
							ip = ep.IPAddress.String()
							break
						}
					}
				}
			}
		}
		var envOut []string
		if ci.Config != nil && ci.Config.Env != nil {
			envOut = ci.Config.Env
		}
		var networksOut any = map[string]any{}
		if ci.NetworkSettings != nil && ci.NetworkSettings.Networks != nil {
			networksOut = ci.NetworkSettings.Networks
		}
		mountsOut := any(ci.Mounts)

		var createdPtr *time.Time
		if ci.Created != "" {
			if t, err := time.Parse(time.RFC3339Nano, ci.Created); err == nil {
				createdPtr = &t
			}
		}
		if createdPtr == nil && c.Created > 0 {
			t := time.Unix(c.Created, 0).UTC()
			createdPtr = &t
		}

		name := ""
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		if err := database.UpsertContainer(
			ctx, h.ID, stackIDPtr, c.ID, name, c.Image, string(c.State), c.Status, h.Owner,
			createdPtr, ip, portsOut, labels, envOut, networksOut, mountsOut,
		); err != nil {
			database.ScanLog(ctx, h.ID, "error", "upsert container failed", map[string]any{"name": name, "id": c.ID, "error": err.Error()})
			continue
		}

		saved++
		database.ScanLog(ctx, h.ID, "info", "container discovered",
			map[string]any{"name": name, "image": c.Image, "state": c.State, "status": c.Status, "project": project})
	}

	// prune gone containers
	if pruned, err := database.PruneMissingContainers(ctx, h.ID, seen); err == nil && pruned > 0 {
		database.ScanLog(ctx, h.ID, "info", "pruned missing containers", map[string]any{"count": pruned})
	}

	database.ScanLog(ctx, h.ID, "info", "scan complete", map[string]any{"containers": saved})
	return saved, nil
}
