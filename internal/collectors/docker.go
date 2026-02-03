package collectors

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

// Global variables to support connection reuse
var (
	dockerCli *client.Client
	dockerMu  sync.Mutex
)

// getDockerClient ensures we reuse the same TCP connection 
func getDockerClient() (*client.Client, error) {
	dockerMu.Lock()
	defer dockerMu.Unlock()

	// Return existing client if healthy
	if dockerCli != nil {
		return dockerCli, nil
	}

	// Initialize new client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	
	// Lightweight ping to verify connection
	_, err = cli.Ping(context.Background())
	if err != nil {
		return nil, err
	}

	dockerCli = cli
	return dockerCli, nil
}

// DockerCollector gathers detailed engine metrics efficiently.
func DockerCollector(params map[string]string) ([]map[string]interface{}, error) {
	ctx := context.Background()

	// 1. Get Client
	cli, err := getDockerClient()
	if err != nil {
		return nil, fmt.Errorf("docker_client_init_error: %w", err)
	}

	// 2. Fetch Containers
	// We disable size calculation to prevent timeouts on 5s intervals.
	containers, err := cli.ContainerList(ctx, container.ListOptions{
		All:  true,
		Size: false, 
	})
	if err != nil {
		// If the client is actually dead, force a reset for next time
		dockerMu.Lock()
		dockerCli = nil
		dockerMu.Unlock()
		return nil, fmt.Errorf("docker_list_error: %w", err)
	}

	// 3. Calculate States
	var (
		running, paused, exited, created int64
		restarting, removing, dead       int64
	)

	for _, c := range containers {
		switch c.State {
		case "running":
			running++
		case "paused":
			paused++
		case "exited":
			exited++
		case "created":
			created++
		case "restarting":
			restarting++
		case "removing":
			removing++
		case "dead":
			dead++
		}
	}

	// 4. Fetch System Info 
	info, err := cli.Info(ctx)
	if err != nil {
		return []map[string]interface{}{
			{
				"docker_containers_total":   int64(len(containers)),
				"docker_containers_running": running,
				"docker_containers_paused":  paused,
				"docker_containers_exited":  exited,
				"docker_error":              1,
			},
		}, nil
	}

	// Helpers
	boolToInt := func(b bool) int64 {
		if b { return 1 }
		return 0
	}

	swarmStateToInt := func(s string) int64 {
		switch strings.ToLower(s) {
		case "inactive": return 0
		case "pending": return 1
		case "active": return 2
		case "error": return 3
		case "locked": return 4
		default: return -1
		}
	}
	
	
	volumeCount := int64(len(info.Plugins.Volume))

	return []map[string]interface{}{
		{
			// --- OLD COMPATIBILITY KEYS ---
			"docker_containers_total":      int64(len(containers)),
			"docker_containers_running":    running,
			"docker_containers_paused":     paused,
			"docker_containers_exited":     exited,
			"docker_images_total":          int64(info.Images),
			"docker_volumes_total":         volumeCount,
			"docker_goroutines":            int64(info.NGoroutines),

			// Granular States
			"docker_containers_created":    created,
			"docker_containers_restarting": restarting,
			"docker_containers_removing":   removing,
			"docker_containers_dead":       dead,

			// System Resources
			"docker_ncpu":             int64(info.NCPU),
			"docker_mem_total_bytes":  int64(info.MemTotal),
			"docker_file_descriptors": int64(info.NFd),
			"docker_events_listeners": int64(info.NEventsListener),
			
			// Swarm Info
			"docker_swarm_state":          swarmStateToInt(string(info.Swarm.LocalNodeState)),
			"docker_swarm_managers":       int64(info.Swarm.Managers),
			"docker_swarm_nodes":          int64(info.Swarm.Nodes),

			// Capabilities
			"docker_cap_swap_limit":       boolToInt(info.SwapLimit),
			"docker_cap_memory_limit":     boolToInt(info.MemoryLimit),
			"docker_cap_oom_kill_disable": boolToInt(info.OomKillDisable),
			"docker_cap_ipv4_forwarding":  boolToInt(info.IPv4Forwarding),
		},
	}, nil
}
