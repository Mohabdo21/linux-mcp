package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetGPUInfoInput struct{}

type GPUDevice struct {
	Index         int     `json:"index"`
	Name          string  `json:"name"`
	UsagePercent  float64 `json:"usage_percent"`
	MemoryUsedMB  int64   `json:"memory_used_mb"`
	MemoryTotalMB int64   `json:"memory_total_mb"`
	TemperatureC  int64   `json:"temperature_c"`
	PowerDrawW    float64 `json:"power_draw_w"`
}

type GPUInfoOutput struct {
	Vendor string      `json:"vendor"`
	GPUs   []GPUDevice `json:"gpus"`
}

func GatherNvidiaGPU() (GPUInfoOutput, error) {
	cmd := exec.Command(
		"nvidia-smi",
		"--query-gpu=index,name,utilization.gpu,memory.used,memory.total,temperature.gpu,power.draw",
		"--format=csv,noheader,nounits",
	)
	out, err := cmd.Output()
	if err != nil {
		return GPUInfoOutput{}, err
	}
	var gpus []GPUDevice
	for line := range strings.SplitSeq(
		strings.TrimSpace(string(out)), "\n",
	) {
		if line == "" {
			continue
		}
		fields := strings.Split(line, ", ")
		if len(fields) < 7 {
			continue
		}
		idx, _ := strconv.Atoi(fields[0])
		usage, _ := strconv.ParseFloat(fields[2], 64)
		memUsed, _ := strconv.ParseInt(
			strings.TrimSpace(fields[3]), 10, 64)
		memTotal, _ := strconv.ParseInt(
			strings.TrimSpace(fields[4]), 10, 64)
		temp, _ := strconv.ParseInt(
			strings.TrimSpace(fields[5]), 10, 64)
		power, _ := strconv.ParseFloat(
			strings.TrimSpace(fields[6]), 64)
		gpus = append(gpus, GPUDevice{
			Index:         idx,
			Name:          fields[1],
			UsagePercent:  usage,
			MemoryUsedMB:  memUsed,
			MemoryTotalMB: memTotal,
			TemperatureC:  temp,
			PowerDrawW:    power,
		})
	}
	if len(gpus) == 0 {
		return GPUInfoOutput{}, errors.New("no NVIDIA GPUs found")
	}
	return GPUInfoOutput{Vendor: "nvidia", GPUs: gpus}, nil
}

func GatherAMDGPU() (GPUInfoOutput, error) {
	cmd := exec.Command("rocm-smi", "--json")
	out, err := cmd.Output()
	if err != nil {
		return GPUInfoOutput{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(out, &raw); err != nil {
		return GPUInfoOutput{}, err
	}
	var gpus []GPUDevice
	for key, val := range raw {
		if !strings.HasPrefix(key, "card") {
			continue
		}
		dev, ok := val.(map[string]any)
		if !ok {
			continue
		}
		idxStr := strings.TrimPrefix(key, "card")
		idx, _ := strconv.Atoi(idxStr)
		name, _ := dev["Name"].(string)
		gpu := GPUDevice{Index: idx, Name: name}
		if usage, ok := dev["GPU use"].(string); ok {
			usage = strings.TrimSuffix(usage, "%")
			gpu.UsagePercent, _ = strconv.ParseFloat(usage, 64)
		}
		if memInfo, ok := dev["VRAM Total Memory (MB)"].(string); ok {
			memTotal, _ := strconv.ParseInt(
				strings.TrimSpace(memInfo), 10, 64)
			gpu.MemoryTotalMB = memTotal
		}
		if memInfo, ok := dev["VRAM Used Memory (MB)"].(string); ok {
			memUsed, _ := strconv.ParseInt(
				strings.TrimSpace(memInfo), 10, 64)
			gpu.MemoryUsedMB = memUsed
		}
		if temp, ok := dev["Temperature (Sensor) (C)"].(string); ok {
			tempInt, _ := strconv.ParseInt(
				strings.TrimSpace(temp), 10, 64)
			gpu.TemperatureC = tempInt
		}
		if power, ok := dev["Average Graphics Package Power (W)"].(string); ok {
			powerF, _ := strconv.ParseFloat(
				strings.TrimSpace(power), 64)
			gpu.PowerDrawW = powerF
		}
		gpus = append(gpus, gpu)
	}
	if len(gpus) == 0 {
		return GPUInfoOutput{}, errors.New("no AMD GPUs found")
	}
	return GPUInfoOutput{Vendor: "amd", GPUs: gpus}, nil
}

func GatherGPUInfo() (GPUInfoOutput, error) {
	if _, err := exec.LookPath("nvidia-smi"); err == nil {
		if out, err := GatherNvidiaGPU(); err == nil {
			return out, nil
		}
	}
	if _, err := exec.LookPath("rocm-smi"); err == nil {
		if out, err := GatherAMDGPU(); err == nil {
			return out, nil
		}
	}
	if _, err := exec.LookPath("intel_gpu_top"); err == nil {
		return GPUInfoOutput{
			Vendor: "intel",
			GPUs:   []GPUDevice{{Name: "Intel GPU (detected)"}},
		}, nil
	}
	return GPUInfoOutput{}, errors.New(
		"no GPU tools found (tried nvidia-smi, rocm-smi, intel_gpu_top)",
	)
}

func HandleGetGPUInfo(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ GetGPUInfoInput,
) (*mcp.CallToolResult, GPUInfoOutput, error) {
	out, err := GatherGPUInfo()
	if err != nil {
		return nil, GPUInfoOutput{}, err
	}
	return nil, out, nil
}
