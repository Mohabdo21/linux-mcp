package tools

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/Mohabdo21/linux-mcp/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetDockerInfoInput struct{}

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
	Errors     []string          `json:"errors,omitempty"`
}

func ListDockerContainers(ctx context.Context) ([]DockerContainer, error) {
	cmd := exec.CommandContext(ctx,
		"docker",
		"ps",
		"-a",
		"--format",
		"{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var containers []DockerContainer
	for line := range strings.SplitSeq(
		strings.TrimSpace(string(out)), "\n",
	) {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) != 4 {
			continue
		}
		containers = append(containers, DockerContainer{
			ID: parts[0], Name: parts[1],
			Image: parts[2], Status: parts[3],
		})
	}
	return containers, nil
}

func ListDockerImages(ctx context.Context) ([]DockerImage, error) {
	cmd := exec.CommandContext(ctx,
		"docker",
		"images",
		"--format",
		"{{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var images []DockerImage
	for line := range strings.SplitSeq(
		strings.TrimSpace(string(out)), "\n",
	) {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) != 4 {
			continue
		}
		images = append(images, DockerImage{
			Repository: parts[0], Tag: parts[1],
			ID: parts[2], Size: parts[3],
		})
	}
	return images, nil
}

func GatherDockerInfo(ctx context.Context) (DockerInfoOutput, error) {
	if _, err := exec.LookPath("docker"); err != nil {
		return DockerInfoOutput{}, errors.New("docker is not installed")
	}
	containers, err := ListDockerContainers(ctx)
	if err != nil {
		return DockerInfoOutput{}, err
	}
	images, err := ListDockerImages(ctx)
	if err != nil {
		return DockerInfoOutput{}, err
	}
	return DockerInfoOutput{
		Containers: containers, Images: images,
	}, nil
}

func HandleGetDockerInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetDockerInfoInput,
) (*mcp.CallToolResult, DockerInfoOutput, error) {
	if config.IsDisabled("get_docker_info") {
		return nil, DockerInfoOutput{},
			errors.New("tool disabled by configuration")
	}
	ctx, cancel := WithToolTimeout(
		ctx, "get_docker_info", 10*time.Second)
	defer cancel()

	start := time.Now()
	out, err := GatherDockerInfo(ctx)
	LogToolCall(ctx, "get_docker_info",
		time.Since(start), len(out.Errors))
	if err != nil {
		out.Errors = append(out.Errors, err.Error())
	}
	return nil, out, nil
}
