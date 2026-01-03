package collectors

import (
	"context"
	"fmt"

	//"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container" // Correct import for container options
	"github.com/docker/docker/client"
)

// DockerCollector gathers engine-level metrics and container counts.
func DockerCollector(params map[string]string) ([]map[string]interface{}, error) {
	ctx := context.Background()

	// 1. Initialize Client
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker_client_error: %w", err)
	}
	defer cli.Close()

	// 2. Fetch all containers
	// Note: container.ListOptions is strictly typed now
	containers, err := cli.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("docker_list_error: %w", err)
	}

	// 3. Process states
	var total int64 = int64(len(containers))
	var running int64 = 0
	var paused int64 = 0
	var exited int64 = 0

	for _, c := range containers {
		switch c.State {
		case "running":
			running++
		case "paused":
			paused++
		case "exited":
			exited++
		}
	}

	// 4. Fetch Engine Info
	info, err := cli.Info(ctx)
	if err != nil {
		return []map[string]interface{}{
			{
				"docker_containers_total":   total,
				"docker_containers_running": running,
				"docker_containers_paused":  paused,
				"docker_containers_exited":  exited,
			},
		}, nil
	}

	return []map[string]interface{}{
		{
			"docker_containers_total":   total,
			"docker_containers_running": running,
			"docker_containers_paused":  paused,
			"docker_containers_exited":  exited,
			"docker_images_total":       int64(info.Images),
			"docker_volumes_total":      int64(len(info.Plugins.Volume)),
			"docker_goroutines":         int64(info.NGoroutines),
		},
	}, nil
}
