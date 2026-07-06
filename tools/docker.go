package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	dockersdk "github.com/docker/go-sdk/client"
	mobyclient "github.com/moby/moby/client"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type DockerContainer struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Image  string `json:"image"`
	Status string `json:"status"`
}

type DockerImage struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	ID         string `json:"id"`
	Size       string `json:"size"`
}

type DockerInfoOutput struct {
	Containers []DockerContainer `json:"containers"`
	Images     []DockerImage     `json:"images"`
	OutputErrors
}

func withDockerClient[T any](ctx context.Context, fn func(cli dockersdk.SDKClient) (T, error)) (T, error) {
	cli, err := dockersdk.New(ctx)
	if err != nil {
		var zero T
		return zero, err
	}
	defer func() { _ = cli.Close() }()
	return fn(cli)
}

func ListDockerContainers(ctx context.Context) ([]DockerContainer, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) ([]DockerContainer, error) {
		result, err := cli.ContainerList(
			ctx,
			mobyclient.ContainerListOptions{All: true},
		)
		if err != nil {
			return nil, err
		}

		containers := make([]DockerContainer, 0, len(result.Items))
		for _, c := range result.Items {
			name := ""
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}
			id := shortID(c.ID)
			containers = append(containers, DockerContainer{
				ID: id, Name: name, Image: c.Image, Status: c.Status,
			})
		}
		return containers, nil
	})
}

func ListDockerImages(ctx context.Context) ([]DockerImage, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) ([]DockerImage, error) {
		result, err := cli.ImageList(ctx, mobyclient.ImageListOptions{})
		if err != nil {
			return nil, err
		}

		images := make([]DockerImage, 0, len(result.Items))
		for _, img := range result.Items {
			repo, tag := "<none>", "<none>"
			if len(img.RepoTags) > 0 {
				if idx := strings.LastIndex(img.RepoTags[0], ":"); idx >= 0 {
					repo, tag = img.RepoTags[0][:idx], img.RepoTags[0][idx+1:]
				} else {
					repo = img.RepoTags[0]
				}
			}
			id := shortID(img.ID)
			images = append(images, DockerImage{
				Repository: repo, Tag: tag, ID: id, Size: HumanSize(img.Size),
			})
		}
		return images, nil
	})
}

func GatherDockerInfo(ctx context.Context) (*DockerInfoOutput, error) {
	containers, err := ListDockerContainers(ctx)
	if err != nil {
		return nil, err
	}

	images, err := ListDockerImages(ctx)
	if err != nil {
		return nil, err
	}

	return &DockerInfoOutput{
		Containers: containers,
		Images:     images,
	}, nil
}

func HandleGetDockerInfo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *DockerInfoOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerInfo,
		0,
		GatherDockerInfo,
	)
}

// --- Container Detail (Inspect) ---

type GetDockerContainerDetailInput struct {
	ContainerID string `json:"container_id" jsonschema:"container name or ID"`
}

type DockerContainerMount struct {
	Type        string `json:"type"`
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Mode        string `json:"mode"`
	RW          bool   `json:"rw"`
}

type DockerContainerDetail struct {
	ID      string                 `json:"id"`
	Name    string                 `json:"name"`
	Image   string                 `json:"image"`
	Created string                 `json:"created"`
	State   map[string]any         `json:"state"`
	Status  string                 `json:"status"`
	Path    string                 `json:"path"`
	Args    []string               `json:"args"`
	Env     []string               `json:"env"`
	Mounts  []DockerContainerMount `json:"mounts"`
	Network map[string]any         `json:"network"`
	Ports   map[string]any         `json:"ports"`
}

type DockerContainerDetailOutput struct {
	Container DockerContainerDetail `json:"container"`
	OutputErrors
}

func GatherContainerDetail(
	ctx context.Context,
	containerID string,
) (*DockerContainerDetailOutput, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) (*DockerContainerDetailOutput, error) {
		result, err := cli.ContainerInspect(
			ctx,
			containerID,
			mobyclient.ContainerInspectOptions{},
		)
		if err != nil {
			return nil, err
		}
		c := result.Container

		name := strings.TrimPrefix(c.Name, "/")
		id := shortID(c.ID)

		state := map[string]any{}
		if c.State != nil {
			state["status"] = c.State.Status
			state["running"] = c.State.Running
			state["paused"] = c.State.Paused
			state["restarting"] = c.State.Restarting
			state["exit_code"] = c.State.ExitCode
			state["started_at"] = c.State.StartedAt
			state["finished_at"] = c.State.FinishedAt
		}

		network := map[string]any{}
		ports := map[string]any{}
		if c.NetworkSettings != nil {
			for name, ep := range c.NetworkSettings.Networks {
				network[name] = map[string]string{
					"ip_address":    ep.IPAddress.String(),
					"ip_prefix_len": fmt.Sprintf("%d", ep.IPPrefixLen),
					"gateway":       ep.Gateway.String(),
					"mac_address":   ep.MacAddress.String(),
				}
			}
			for p, bindings := range c.NetworkSettings.Ports {
				var bList []string
				for _, b := range bindings {
					bList = append(bList, b.HostIP.String()+":"+b.HostPort)
				}
				ports[p.String()] = bList
			}
		}

		mounts := make([]DockerContainerMount, 0, len(c.Mounts))
		for _, m := range c.Mounts {
			mounts = append(mounts, DockerContainerMount{
				Type:        string(m.Type),
				Source:      m.Source,
				Destination: m.Destination,
				Mode:        m.Mode,
				RW:          m.RW,
			})
		}

		var env []string
		if c.Config != nil {
			env = c.Config.Env
		}
		env = nilToEmpty(env)

		status, _ := state["status"].(string)

		args := c.Args
		args = nilToEmpty(args)

		return &DockerContainerDetailOutput{
			Container: DockerContainerDetail{
				ID:      id,
				Name:    name,
				Image:   c.Image,
				Created: c.Created,
				State:   state,
				Status:  status,
				Path:    c.Path,
				Args:    args,
				Env:     env,
				Mounts:  mounts,
				Network: network,
				Ports:   ports,
			},
		}, nil
	})
}

func HandleGetContainerDetail(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetDockerContainerDetailInput,
) (*mcp.CallToolResult, *DockerContainerDetailOutput, error) {
	if err := requireField(input.ContainerID, "container_id"); err != nil {
		return nil, nil, err
	}
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerContainerDetails,
		0,
		func(ctx context.Context) (*DockerContainerDetailOutput, error) {
			return GatherContainerDetail(ctx, input.ContainerID)
		},
	)
}

// --- Container Logs ---

type GetDockerContainerLogsInput struct {
	ContainerID string `json:"container_id"         jsonschema:"container name or ID"`
	Tail        int    `json:"tail,omitempty"       jsonschema:"number of lines to tail (default: 100, max: 10000)"`
	Timestamps  bool   `json:"timestamps,omitempty" jsonschema:"include timestamps (default: false)"`
}

type DockerContainerLogsOutput struct {
	Logs []string `json:"logs"`
	OutputErrors
}

func GatherContainerLogs(
	ctx context.Context,
	containerID string,
	tail int,
	timestamps bool,
) (*DockerContainerLogsOutput, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) (*DockerContainerLogsOutput, error) {
		if tail <= 0 {
			tail = 100
		} else if tail > 10000 {
			tail = 10000
		}

		result, err := cli.ContainerLogs(
			ctx,
			containerID,
			mobyclient.ContainerLogsOptions{
				ShowStdout: true,
				ShowStderr: true,
				Timestamps: timestamps,
				Tail:       fmt.Sprintf("%d", tail),
			},
		)
		if err != nil {
			return nil, err
		}
		defer func() { _ = result.Close() }()

		lines := make([]string, 0)
		scanner := bufio.NewScanner(result)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		return &DockerContainerLogsOutput{Logs: lines}, nil
	})
}

func HandleGetContainerLogs(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetDockerContainerLogsInput,
) (*mcp.CallToolResult, *DockerContainerLogsOutput, error) {
	if err := requireField(input.ContainerID, "container_id"); err != nil {
		return nil, nil, err
	}
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerContainerLogs,
		0,
		func(ctx context.Context) (*DockerContainerLogsOutput, error) {
			return GatherContainerLogs(
				ctx,
				input.ContainerID,
				input.Tail,
				input.Timestamps,
			)
		},
	)
}

// --- Container Stats ---

type GetDockerContainerStatsInput struct {
	ContainerIDs string `json:"container_ids" jsonschema:"container name(s) or ID(s), comma-separated, or 'all' for all running containers"`
}

type DockerContainerStats struct {
	CPUPercent    float64                      `json:"cpu_percent"`
	MemoryUsage   uint64                       `json:"memory_usage"`
	MemoryLimit   uint64                       `json:"memory_limit"`
	MemoryPercent float64                      `json:"memory_percent"`
	PIDs          uint64                       `json:"pids"`
	Network       map[string]map[string]uint64 `json:"network,omitempty"`
	BlockRead     uint64                       `json:"block_read"`
	BlockWrite    uint64                       `json:"block_write"`
}

type DockerContainerStatsOutput struct {
	Containers []DockerContainerStatEntry `json:"containers"`
	OutputErrors
}

func GatherContainerStats(
	ctx context.Context,
	containerID string,
) (*DockerContainerStatsOutput, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) (*DockerContainerStatsOutput, error) {
		result, err := cli.ContainerStats(
			ctx,
			containerID,
			mobyclient.ContainerStatsOptions{
				Stream: false,
			},
		)
		if err != nil {
			return nil, err
		}
		defer func() { _ = result.Body.Close() }()

		var stats struct {
			CPUStats struct {
				CPUUsage struct {
					TotalUsage uint64 `json:"total_usage"`
				} `json:"cpu_usage"`
				SystemUsage uint64 `json:"system_cpu_usage"`
				OnlineCPUs  uint32 `json:"online_cpus"`
			} `json:"cpu_stats"`
			PreCPUStats struct {
				CPUUsage struct {
					TotalUsage uint64 `json:"total_usage"`
				} `json:"cpu_usage"`
				SystemUsage uint64 `json:"system_cpu_usage"`
			} `json:"precpu_stats"`
			MemoryStats struct {
				Usage uint64            `json:"usage"`
				Limit uint64            `json:"limit"`
				Stats map[string]uint64 `json:"stats"`
			} `json:"memory_stats"`
			PidsStats struct {
				Current uint64 `json:"current"`
			} `json:"pids_stats"`
			BlkioStats struct {
				IoServiceBytesRecursive []struct {
					Op    string `json:"op"`
					Value uint64 `json:"value"`
				} `json:"io_service_bytes_recursive"`
			} `json:"blkio_stats"`
			Networks map[string]struct {
				RxBytes   uint64 `json:"rx_bytes"`
				TxBytes   uint64 `json:"tx_bytes"`
				RxPackets uint64 `json:"rx_packets"`
				TxPackets uint64 `json:"tx_packets"`
			} `json:"networks"`
		}

		if err := json.NewDecoder(result.Body).Decode(&stats); err != nil {
			return nil, err
		}

		cpuDelta := stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage
		sysDelta := stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage
		cpuPercent := 0.0
		if sysDelta > 0 && stats.CPUStats.OnlineCPUs > 0 {
			cpuPercent = (float64(cpuDelta) / float64(sysDelta)) * float64(
				stats.CPUStats.OnlineCPUs,
			) * 100.0
		}

		memPercent := 0.0
		if stats.MemoryStats.Limit > 0 {
			memPercent = (float64(stats.MemoryStats.Usage) / float64(stats.MemoryStats.Limit)) * 100.0
		}

		networkMap := map[string]map[string]uint64{}
		for name, net := range stats.Networks {
			networkMap[name] = map[string]uint64{
				"rx_bytes":   net.RxBytes,
				"tx_bytes":   net.TxBytes,
				"rx_packets": net.RxPackets,
				"tx_packets": net.TxPackets,
			}
		}

		var blkRead, blkWrite uint64
		for _, op := range stats.BlkioStats.IoServiceBytesRecursive {
			switch op.Op {
			case "read":
				blkRead += op.Value
			case "write":
				blkWrite += op.Value
			}
		}

		id := shortID(containerID)

		return &DockerContainerStatsOutput{
			Containers: []DockerContainerStatEntry{{
				ID:            id,
				CPUPercent:    cpuPercent,
				MemoryUsage:   stats.MemoryStats.Usage,
				MemoryLimit:   stats.MemoryStats.Limit,
				MemoryPercent: memPercent,
				PIDs:          stats.PidsStats.Current,
				Network:       networkMap,
				BlockRead:     blkRead,
				BlockWrite:    blkWrite,
			}},
		}, nil
	})
}

func HandleGetContainerStats(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetDockerContainerStatsInput,
) (*mcp.CallToolResult, *DockerContainerStatsOutput, error) {
	if err := requireField(input.ContainerIDs, "container_ids"); err != nil {
		return nil, nil, err
	}

	if input.ContainerIDs == "all" {
		return handleToolCall(
			ctx,
			config.ToolNameGetDockerContainerStats,
			0,
			func(ctx context.Context) (*DockerContainerStatsOutput, error) {
				all, err := GatherDockerStatsAll(ctx, nil)
				if err != nil {
					return nil, err
				}
				return (*DockerContainerStatsOutput)(all), nil
			},
		)
	}

	ids := strings.Split(input.ContainerIDs, ",")
	for i := range ids {
		ids[i] = strings.TrimSpace(ids[i])
	}

	if len(ids) == 1 {
		return handleToolCall(
			ctx,
			config.ToolNameGetDockerContainerStats,
			0,
			func(ctx context.Context) (*DockerContainerStatsOutput, error) {
				return GatherContainerStats(ctx, ids[0])
			},
		)
	}

	return handleToolCall(
		ctx,
		config.ToolNameGetDockerContainerStats,
		0,
		func(ctx context.Context) (*DockerContainerStatsOutput, error) {
			all, err := GatherDockerStatsAll(ctx, ids)
			if err != nil {
				return nil, err
			}
			return (*DockerContainerStatsOutput)(all), nil
		},
	)
}

// --- All Containers Stats (bulk) ---

type GetDockerStatsAllInput struct {
	Containers []string `json:"containers,omitempty" jsonschema:"optional list of container names or IDs to filter"`
}

type DockerContainerStatEntry struct {
	ID            string                       `json:"id"`
	Name          string                       `json:"name"`
	CPUPercent    float64                      `json:"cpu_percent"`
	MemoryUsage   uint64                       `json:"memory_usage"`
	MemoryLimit   uint64                       `json:"memory_limit"`
	MemoryPercent float64                      `json:"memory_percent"`
	PIDs          uint64                       `json:"pids"`
	Network       map[string]map[string]uint64 `json:"network,omitempty"`
	BlockRead     uint64                       `json:"block_read"`
	BlockWrite    uint64                       `json:"block_write"`
	Error         string                       `json:"error,omitempty"`
}

type DockerStatsAllOutput struct {
	Containers []DockerContainerStatEntry `json:"containers"`
	OutputErrors
}

func GatherDockerStatsAll(
	ctx context.Context,
	containers []string,
) (*DockerStatsAllOutput, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) (*DockerStatsAllOutput, error) {
		running, err := cli.ContainerList(
			ctx,
			mobyclient.ContainerListOptions{All: false},
		)
		if err != nil {
			return nil, err
		}

		filterSet := map[string]bool{}
		for _, f := range containers {
			filterSet[f] = true
		}

		type target struct{ id, name string }
		var targets []target
		for _, c := range running.Items {
			name := ""
			if len(c.Names) > 0 {
				name = strings.TrimPrefix(c.Names[0], "/")
			}
			sid := shortID(c.ID)
			if len(filterSet) > 0 && !filterSet[c.ID] && !filterSet[sid] &&
				!filterSet[name] {
				continue
			}
			targets = append(targets, target{id: c.ID, name: name})
		}

		type statResult struct {
			id    string
			name  string
			entry DockerContainerStatEntry
			err   error
		}
		ch := make(chan statResult, len(targets))
		for _, t := range targets {
			go func(containerID, containerName string) {
				out, err := GatherContainerStats(ctx, containerID)
				if err != nil {
					ch <- statResult{id: containerID, name: containerName, err: err}
				} else if len(out.Containers) > 0 {
					ch <- statResult{
						id: containerID, name: containerName, entry: out.Containers[0],
					}
				}
			}(t.id, t.name)
		}

		entries := make([]DockerContainerStatEntry, 0, len(targets))
		var errs []string
		for range targets {
			r := <-ch
			sid := shortID(r.id)
			entry := DockerContainerStatEntry{
				ID:   sid,
				Name: r.name,
			}
			if r.err != nil {
				entry.Error = r.err.Error()
				errs = append(
					errs,
					fmt.Sprintf("%s (%s): %v", r.name, sid, r.err),
				)
			} else {
				entry.CPUPercent = r.entry.CPUPercent
				entry.MemoryUsage = r.entry.MemoryUsage
				entry.MemoryLimit = r.entry.MemoryLimit
				entry.MemoryPercent = r.entry.MemoryPercent
				entry.PIDs = r.entry.PIDs
				entry.Network = r.entry.Network
				entry.BlockRead = r.entry.BlockRead
				entry.BlockWrite = r.entry.BlockWrite
			}
			entries = append(entries, entry)
		}

		out := &DockerStatsAllOutput{Containers: entries}
		out.Errors = errs
		return out, nil
	})
}

func HandleGetDockerStatsAll(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetDockerStatsAllInput,
) (*mcp.CallToolResult, *DockerStatsAllOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerStatsAll,
		0,
		func(ctx context.Context) (*DockerStatsAllOutput, error) {
			return GatherDockerStatsAll(ctx, input.Containers)
		},
	)
}

// --- Container Top (Processes) ---

type GetDockerContainerTopInput struct {
	ContainerID string   `json:"container_id"   jsonschema:"container name or ID"`
	Args        []string `json:"args,omitempty" jsonschema:"optional arguments to ps (e.g. aux)"`
}

type DockerContainerTopOutput struct {
	Titles    []string   `json:"titles"`
	Processes [][]string `json:"processes"`
	OutputErrors
}

func GatherContainerTop(
	ctx context.Context,
	containerID string,
	args []string,
) (*DockerContainerTopOutput, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) (*DockerContainerTopOutput, error) {
		result, err := cli.ContainerTop(
			ctx,
			containerID,
			mobyclient.ContainerTopOptions{
				Arguments: args,
			},
		)
		if err != nil {
			return nil, err
		}

		titles := result.Titles
		titles = nilToEmpty(titles)
		procs := result.Processes
		procs = nilToEmpty(procs)
		return &DockerContainerTopOutput{
			Titles:    titles,
			Processes: procs,
		}, nil
	})
}

func HandleGetContainerTop(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetDockerContainerTopInput,
) (*mcp.CallToolResult, *DockerContainerTopOutput, error) {
	if err := requireField(input.ContainerID, "container_id"); err != nil {
		return nil, nil, err
	}
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerContainerTop,
		0,
		func(ctx context.Context) (*DockerContainerTopOutput, error) {
			return GatherContainerTop(ctx, input.ContainerID, input.Args)
		},
	)
}

// --- Container Diff ---

type GetDockerContainerDiffInput struct {
	ContainerID string `json:"container_id" jsonschema:"container name or ID"`
}

type DockerFileChange struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type DockerContainerDiffOutput struct {
	Changes []DockerFileChange `json:"changes"`
	OutputErrors
}

func tristateKind(kind uint8) string {
	switch kind {
	case 0:
		return "modified"
	case 1:
		return "added"
	case 2:
		return "deleted"
	default:
		return fmt.Sprintf("unknown(%d)", kind)
	}
}

func GatherContainerDiff(
	ctx context.Context,
	containerID string,
) (*DockerContainerDiffOutput, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) (*DockerContainerDiffOutput, error) {
		result, err := cli.ContainerDiff(
			ctx,
			containerID,
			mobyclient.ContainerDiffOptions{},
		)
		if err != nil {
			return nil, err
		}

		changes := make([]DockerFileChange, 0, len(result.Changes))
		for _, ch := range result.Changes {
			changes = append(changes, DockerFileChange{
				Kind: tristateKind(uint8(ch.Kind)),
				Path: ch.Path,
			})
		}
		return &DockerContainerDiffOutput{Changes: changes}, nil
	})
}

func HandleGetContainerDiff(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetDockerContainerDiffInput,
) (*mcp.CallToolResult, *DockerContainerDiffOutput, error) {
	if err := requireField(input.ContainerID, "container_id"); err != nil {
		return nil, nil, err
	}
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerContainerDiff,
		0,
		func(ctx context.Context) (*DockerContainerDiffOutput, error) {
			return GatherContainerDiff(ctx, input.ContainerID)
		},
	)
}

// --- Image History ---

type GetDockerImageHistoryInput struct {
	ImageID string `json:"image_id" jsonschema:"image name or ID"`
}

type DockerImageLayer struct {
	ID        string   `json:"id"`
	Created   int64    `json:"created"`
	CreatedBy string   `json:"created_by"`
	Size      string   `json:"size"`
	Tags      []string `json:"tags,omitempty"`
	Comment   string   `json:"comment,omitempty"`
}

type DockerImageHistoryOutput struct {
	Layers []DockerImageLayer `json:"layers"`
	OutputErrors
}

func GatherImageHistory(
	ctx context.Context,
	imageID string,
) (*DockerImageHistoryOutput, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) (*DockerImageHistoryOutput, error) {
		result, err := cli.ImageHistory(ctx, imageID)
		if err != nil {
			return nil, err
		}

		layers := make([]DockerImageLayer, 0, len(result.Items))
		for _, item := range result.Items {
			id := shortID(item.ID)
			layers = append(layers, DockerImageLayer{
				ID:        id,
				Created:   item.Created,
				CreatedBy: item.CreatedBy,
				Size:      HumanSize(item.Size),
				Tags:      item.Tags,
				Comment:   item.Comment,
			})
		}
		return &DockerImageHistoryOutput{Layers: layers}, nil
	})
}

func HandleGetImageHistory(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetDockerImageHistoryInput,
) (*mcp.CallToolResult, *DockerImageHistoryOutput, error) {
	if err := requireField(input.ImageID, "image_id"); err != nil {
		return nil, nil, err
	}
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerImageHistory,
		0,
		func(ctx context.Context) (*DockerImageHistoryOutput, error) {
			return GatherImageHistory(ctx, input.ImageID)
		},
	)
}

// --- Image Detail (Inspect) ---

type GetDockerImageDetailInput struct {
	ImageID string `json:"image_id" jsonschema:"image name or ID"`
}

type DockerImageDetail struct {
	ID           string            `json:"id"`
	RepoTags     []string          `json:"repo_tags"`
	RepoDigests  []string          `json:"repo_digests"`
	Created      string            `json:"created"`
	Author       string            `json:"author"`
	Architecture string            `json:"architecture"`
	OS           string            `json:"os"`
	Size         string            `json:"size"`
	Entrypoint   []string          `json:"entrypoint,omitempty"`
	Cmd          []string          `json:"cmd,omitempty"`
	Env          []string          `json:"env,omitempty"`
	WorkingDir   string            `json:"working_dir,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Layers       []string          `json:"layers,omitempty"`
}

type DockerImageDetailOutput struct {
	Image DockerImageDetail `json:"image"`
	OutputErrors
}

func GatherImageDetail(
	ctx context.Context,
	imageID string,
) (*DockerImageDetailOutput, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) (*DockerImageDetailOutput, error) {
		result, err := cli.ImageInspect(ctx, imageID)
		if err != nil {
			return nil, err
		}

		id := shortID(result.ID)

		var entrypoint, cmd, env []string
		var workingDir string
		var labels map[string]string
		if result.Config != nil {
			entrypoint = result.Config.Entrypoint
			cmd = result.Config.Cmd
			env = result.Config.Env
			workingDir = result.Config.WorkingDir
			labels = result.Config.Labels
		}

		repoTags := result.RepoTags
		repoTags = nilToEmpty(repoTags)
		repoDigests := result.RepoDigests
		repoDigests = nilToEmpty(repoDigests)

		return &DockerImageDetailOutput{
			Image: DockerImageDetail{
				ID:           id,
				RepoTags:     repoTags,
				RepoDigests:  repoDigests,
				Created:      result.Created,
				Author:       result.Author,
				Architecture: result.Architecture,
				OS:           result.Os,
				Size:         HumanSize(result.Size),
				Entrypoint:   entrypoint,
				Cmd:          cmd,
				Env:          env,
				WorkingDir:   workingDir,
				Labels:       labels,
				Layers:       result.RootFS.Layers,
			},
		}, nil
	})
}

func HandleGetImageDetail(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	input GetDockerImageDetailInput,
) (*mcp.CallToolResult, *DockerImageDetailOutput, error) {
	if err := requireField(input.ImageID, "image_id"); err != nil {
		return nil, nil, err
	}
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerImageDetails,
		0,
		func(ctx context.Context) (*DockerImageDetailOutput, error) {
			return GatherImageDetail(ctx, input.ImageID)
		},
	)
}

// --- Networks ---

type DockerNetworkSummary struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Scope      string            `json:"scope"`
	Attachable bool              `json:"attachable"`
	Internal   bool              `json:"internal"`
	Ingress    bool              `json:"ingress"`
	IPv6       bool              `json:"ipv6"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type DockerNetworksOutput struct {
	Networks []DockerNetworkSummary `json:"networks"`
	OutputErrors
}

func GatherDockerNetworks(ctx context.Context) (*DockerNetworksOutput, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) (*DockerNetworksOutput, error) {
		result, err := cli.NetworkList(ctx, mobyclient.NetworkListOptions{})
		if err != nil {
			return nil, err
		}

		networks := make([]DockerNetworkSummary, 0, len(result.Items))
		for _, n := range result.Items {
			id := shortID(n.ID)
			networks = append(networks, DockerNetworkSummary{
				ID:         id,
				Name:       n.Name,
				Driver:     n.Driver,
				Scope:      n.Scope,
				Attachable: n.Attachable,
				Internal:   n.Internal,
				Ingress:    n.Ingress,
				IPv6:       n.EnableIPv6,
				Labels:     n.Labels,
			})
		}
		return &DockerNetworksOutput{Networks: networks}, nil
	})
}

func HandleGetDockerNetworks(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *DockerNetworksOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerNetworks,
		0,
		GatherDockerNetworks,
	)
}

// --- Volumes ---

type DockerVolumeSummary struct {
	Name       string            `json:"name"`
	Driver     string            `json:"driver"`
	Mountpoint string            `json:"mountpoint"`
	Scope      string            `json:"scope"`
	CreatedAt  string            `json:"created_at"`
	Size       string            `json:"size,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

type DockerVolumesOutput struct {
	Volumes []DockerVolumeSummary `json:"volumes"`
	OutputErrors
}

func GatherDockerVolumes(ctx context.Context) (*DockerVolumesOutput, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) (*DockerVolumesOutput, error) {
		result, err := cli.VolumeList(ctx, mobyclient.VolumeListOptions{})
		if err != nil {
			return nil, err
		}

		volumes := make([]DockerVolumeSummary, 0, len(result.Items))
		for _, v := range result.Items {
			size := ""
			if v.UsageData != nil {
				size = HumanSize(v.UsageData.Size)
			}
			volumes = append(volumes, DockerVolumeSummary{
				Name:       v.Name,
				Driver:     v.Driver,
				Mountpoint: v.Mountpoint,
				Scope:      v.Scope,
				CreatedAt:  v.CreatedAt,
				Size:       size,
				Labels:     v.Labels,
			})
		}
		return &DockerVolumesOutput{Volumes: volumes}, nil
	})
}

func HandleGetDockerVolumes(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *DockerVolumesOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerVolumes,
		0,
		GatherDockerVolumes,
	)
}

// --- System Info ---

type DockerSystemInfoSummary struct {
	ID                string         `json:"id"`
	ServerVersion     string         `json:"server_version"`
	Architecture      string         `json:"architecture"`
	OSType            string         `json:"os_type"`
	OperatingSystem   string         `json:"operating_system"`
	KernelVersion     string         `json:"kernel_version"`
	NCPU              int            `json:"ncpu"`
	MemTotal          string         `json:"mem_total"`
	Driver            string         `json:"driver"`
	LoggingDriver     string         `json:"logging_driver"`
	CgroupDriver      string         `json:"cgroup_driver"`
	CgroupVersion     string         `json:"cgroup_version"`
	DefaultRuntime    string         `json:"default_runtime"`
	Runtimes          []string       `json:"runtimes,omitempty"`
	ContainersTotal   int            `json:"containers_total"`
	ContainersRunning int            `json:"containers_running"`
	ContainersPaused  int            `json:"containers_paused"`
	ContainersStopped int            `json:"containers_stopped"`
	ImagesTotal       int            `json:"images_total"`
	DockerRootDir     string         `json:"docker_root_dir"`
	SecurityOptions   []string       `json:"security_options,omitempty"`
	Swarm             map[string]any `json:"swarm,omitempty"`
}

type DockerSystemInfoOutput struct {
	Info DockerSystemInfoSummary `json:"info"`
	OutputErrors
}

func GatherDockerSystemInfo(
	ctx context.Context,
) (*DockerSystemInfoOutput, error) {
	return withDockerClient(ctx, func(cli dockersdk.SDKClient) (*DockerSystemInfoOutput, error) {
		result, err := cli.Info(ctx, mobyclient.InfoOptions{})
	if err != nil {
		return nil, err
	}

	sysInfo := result.Info
	id := shortID(sysInfo.ID)

	runtimes := make([]string, 0, len(sysInfo.Runtimes))
	for name := range sysInfo.Runtimes {
		runtimes = append(runtimes, name)
	}

	swarm := map[string]any{}
	swarm["node_id"] = sysInfo.Swarm.NodeID
	swarm["control_available"] = sysInfo.Swarm.ControlAvailable
	swarm["local_node_state"] = sysInfo.Swarm.LocalNodeState

	return &DockerSystemInfoOutput{
		Info: DockerSystemInfoSummary{
			ID:                id,
			ServerVersion:     sysInfo.ServerVersion,
			Architecture:      sysInfo.Architecture,
			OSType:            sysInfo.OSType,
			OperatingSystem:   sysInfo.OperatingSystem,
			KernelVersion:     sysInfo.KernelVersion,
			NCPU:              sysInfo.NCPU,
			MemTotal:          HumanSize(int64(sysInfo.MemTotal)),
			Driver:            sysInfo.Driver,
			LoggingDriver:     sysInfo.LoggingDriver,
			CgroupDriver:      sysInfo.CgroupDriver,
			CgroupVersion:     sysInfo.CgroupVersion,
			DefaultRuntime:    sysInfo.DefaultRuntime,
			Runtimes:          runtimes,
			ContainersTotal:   sysInfo.Containers,
			ContainersRunning: sysInfo.ContainersRunning,
			ContainersPaused:  sysInfo.ContainersPaused,
			ContainersStopped: sysInfo.ContainersStopped,
			ImagesTotal:       sysInfo.Images,
			DockerRootDir:     sysInfo.DockerRootDir,
			SecurityOptions:   sysInfo.SecurityOptions,
			Swarm:             swarm,
		},
	}, nil
}

func HandleGetDockerSystemInfo(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *DockerSystemInfoOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerSystemInfo,
		0,
		GatherDockerSystemInfo,
	)
}

// --- Disk Usage ---

type DockerDiskUsageCategory struct {
	ActiveCount int64  `json:"active_count"`
	TotalCount  int64  `json:"total_count"`
	Reclaimable string `json:"reclaimable"`
	TotalSize   string `json:"total_size"`
}

type DockerDiskUsageOutput struct {
	Containers DockerDiskUsageCategory `json:"containers"`
	Images     DockerDiskUsageCategory `json:"images"`
	Volumes    DockerDiskUsageCategory `json:"volumes"`
	BuildCache DockerDiskUsageCategory `json:"build_cache"`
	OutputErrors
}

func GatherDockerDiskUsage(
	ctx context.Context,
) (*DockerDiskUsageOutput, error) {
	cli, err := newDockerClient(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cli.Close() }()

	result, err := cli.DiskUsage(ctx, mobyclient.DiskUsageOptions{
		Containers: true,
		Images:     true,
		BuildCache: true,
		Volumes:    true,
	})
	if err != nil {
		return nil, err
	}

	return &DockerDiskUsageOutput{
		Containers: DockerDiskUsageCategory{
			ActiveCount: result.Containers.ActiveCount,
			TotalCount:  result.Containers.TotalCount,
			Reclaimable: HumanSize(result.Containers.Reclaimable),
			TotalSize:   HumanSize(result.Containers.TotalSize),
		},
		Images: DockerDiskUsageCategory{
			ActiveCount: result.Images.ActiveCount,
			TotalCount:  result.Images.TotalCount,
			Reclaimable: HumanSize(result.Images.Reclaimable),
			TotalSize:   HumanSize(result.Images.TotalSize),
		},
		Volumes: DockerDiskUsageCategory{
			ActiveCount: result.Volumes.ActiveCount,
			TotalCount:  result.Volumes.TotalCount,
			Reclaimable: HumanSize(result.Volumes.Reclaimable),
			TotalSize:   HumanSize(result.Volumes.TotalSize),
		},
		BuildCache: DockerDiskUsageCategory{
			ActiveCount: result.BuildCache.ActiveCount,
			TotalCount:  result.BuildCache.TotalCount,
			Reclaimable: HumanSize(result.BuildCache.Reclaimable),
			TotalSize:   HumanSize(result.BuildCache.TotalSize),
		},
	}, nil
}

func HandleGetDockerDiskUsage(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *DockerDiskUsageOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerDiskUsage,
		0,
		GatherDockerDiskUsage,
	)
}

// --- Docker System Snapshot ---

type DockerSystemSnapshotOutput struct {
	Info      DockerInfoOutput      `json:"info"`
	Stats     DockerStatsAllOutput  `json:"stats"`
	DiskUsage DockerDiskUsageOutput `json:"disk_usage"`
	OutputErrors
}

func HandleGetDockerSystemSnapshot(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ NoArgs,
) (*mcp.CallToolResult, *DockerSystemSnapshotOutput, error) {
	return handleToolCall(
		ctx,
		config.ToolNameGetDockerSystemSnapshot,
		0,
		func(ctx context.Context) (*DockerSystemSnapshotOutput, error) {
			var snapshot DockerSystemSnapshotOutput
			var errs []string

			snapshot.Info = collectOrFallback(ctx,
				"info", GatherDockerInfo,
				DockerInfoOutput{Containers: []DockerContainer{},
					Images: []DockerImage{}}, &errs)
			snapshot.Stats = collectOrFallback(
				ctx,
				"stats",
				func(ctx context.Context) (*DockerStatsAllOutput, error) {
					return GatherDockerStatsAll(ctx, nil)
				},
				DockerStatsAllOutput{Containers: []DockerContainerStatEntry{}},
				&errs,
			)
			snapshot.DiskUsage = collectOrFallback(ctx,
				"disk_usage", GatherDockerDiskUsage,
				DockerDiskUsageOutput{}, &errs)

			snapshot.Errors = errs
			return &snapshot, nil
		},
	)
}
