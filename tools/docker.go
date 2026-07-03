package tools

import (
	"context"
	"errors"
	"os/exec"
	"strings"

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
}

func ListDockerContainers() ([]DockerContainer, error) {
	cmd := exec.Command(
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

func ListDockerImages() ([]DockerImage, error) {
	cmd := exec.Command(
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

func GatherDockerInfo() (DockerInfoOutput, error) {
	containers, err := ListDockerContainers()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return DockerInfoOutput{}, errors.New(
				"docker is not installed",
			)
		}
		return DockerInfoOutput{}, err
	}
	images, err := ListDockerImages()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return DockerInfoOutput{}, errors.New(
				"docker is not installed",
			)
		}
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
	out, err := GatherDockerInfo()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, DockerInfoOutput{}, errors.New(
				"docker is not installed",
			)
		}
		return nil, DockerInfoOutput{}, err
	}
	return nil, out, nil
}
