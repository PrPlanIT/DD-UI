// src/api/utils/docker_action.go
package utils

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"
)

func PerformContainerAction(ctx context.Context, hostProvider HostProvider, dockerProvider DockerClientProvider, hostName, ctr, action string) error {
	h, err := hostProvider.GetHostByName(ctx, hostName)
	if err != nil {
		return err
	}
	cli, err := dockerProvider.DockerClientForHost(HostRow{Name: h.Name, Addr: h.Addr, Vars: h.Vars})
	if err != nil {
		return err
	}
	defer cli.Close()

	// Default graceful timeout (seconds) for stop/restart
	sec := 10
	to := &sec

	switch action {
	case "start", "play":
		_, err = cli.ContainerStart(ctx, ctr, client.ContainerStartOptions{})
	case "stop":
		_, err = cli.ContainerStop(ctx, ctr, client.ContainerStopOptions{Timeout: to})
	case "kill":
		// default to SIGKILL
		_, err = cli.ContainerKill(ctx, ctr, client.ContainerKillOptions{Signal: "KILL"})
	case "restart":
		_, err = cli.ContainerRestart(ctx, ctr, client.ContainerRestartOptions{Timeout: to})
	case "pause":
		_, err = cli.ContainerPause(ctx, ctr, client.ContainerPauseOptions{})
	case "unpause", "resume":
		_, err = cli.ContainerUnpause(ctx, ctr, client.ContainerUnpauseOptions{})
	case "remove":
		_, err = cli.ContainerRemove(ctx, ctr, client.ContainerRemoveOptions{
			Force:         true,
			RemoveVolumes: false,
		})
	default:
		return fmt.Errorf("unknown action: %s", action)
	}
	return err
}

func OneShotStats(ctx context.Context, hostProvider HostProvider, dockerProvider DockerClientProvider, hostName, ctr string) (string, error) {
	h, err := hostProvider.GetHostByName(ctx, hostName)
	if err != nil {
		return "", err
	}
	cli, err := dockerProvider.DockerClientForHost(HostRow{Name: h.Name, Addr: h.Addr, Vars: h.Vars})
	if err != nil {
		return "", err
	}
	defer cli.Close()

	// Use non-streaming stats (read once)
	resp, err := cli.ContainerStats(ctx, ctr, client.ContainerStatsOptions{})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Return raw JSON body to the caller (UI shows as text)
	buf := make([]byte, 0, 64*1024)
	tmp := make([]byte, 32*1024)
	for {
		n, er := resp.Body.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if er != nil {
			break
		}
	}
	// common.DebugLog("stats: %s len=%d", ctr, len(buf)) // Comment out - needs to be injected
	return string(buf), nil
}
