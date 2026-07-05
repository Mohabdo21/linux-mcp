package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type GetPowerAnalyticsInput struct{}

type PowerAnalyticsOutput struct {
	ACOnline            bool    `json:"ac_online"`
	BatteryPercent      float64 `json:"battery_percent"`
	DischargeRateWatts  float64 `json:"discharge_rate_watts"`
	CapacityDegradation float64 `json:"capacity_degradation_percent"`
	OutputErrors
}

type batteryInfo struct {
	status           string
	capacity         float64
	powerNow         float64
	energyFull       float64
	energyFullDesign float64
}

func readPowerSysfs(dir, key string) (string, error) {
	data, err := os.ReadFile(filepath.Join(dir, key))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func parseBatteryDir(dir string) (*batteryInfo, error) {
	status, err := readPowerSysfs(dir, "status")
	if err != nil {
		return nil, fmt.Errorf("reading status: %w", err)
	}

	capacity, err := readPowerSysfs(dir, "capacity")
	if err != nil {
		return nil, fmt.Errorf("reading capacity: %w", err)
	}
	capVal, err := strconv.ParseFloat(capacity, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing capacity: %w", err)
	}

	info := &batteryInfo{
		status:   status,
		capacity: capVal,
	}

	pn, err := readPowerSysfs(dir, "power_now")
	if err == nil {
		pnVal, parseErr := strconv.ParseFloat(pn, 64)
		if parseErr == nil {
			info.powerNow = pnVal
		}
	}

	ef, err := readPowerSysfs(dir, "energy_full")
	if err == nil {
		efVal, parseErr := strconv.ParseFloat(ef, 64)
		if parseErr == nil {
			info.energyFull = efVal
		}
	}

	efd, err := readPowerSysfs(dir, "energy_full_design")
	if err == nil {
		efdVal, parseErr := strconv.ParseFloat(efd, 64)
		if parseErr == nil {
			info.energyFullDesign = efdVal
		}
	}

	return info, nil
}

func GatherPowerAnalytics(ctx context.Context) (*PowerAnalyticsOutput, error) {
	var out PowerAnalyticsOutput
	var errs ErrList

	batts, err := os.ReadDir("/sys/class/power_supply")
	if err != nil {
		out.AppendError(fmt.Sprintf("reading power supply class: %v", err))
		return &out, nil
	}

	for _, entry := range batts {
		dir := filepath.Join("/sys/class/power_supply", entry.Name())

		uevent, err := readPowerSysfs(dir, "uevent")
		if err != nil {
			continue
		}

		isBattery := strings.Contains(uevent, "POWER_SUPPLY_TYPE=Battery")
		isMains := strings.Contains(uevent, "POWER_SUPPLY_TYPE=Mains")

		if isMains {
			online, err := readPowerSysfs(dir, "online")
			if err == nil {
				out.ACOnline = strings.TrimSpace(online) == "1"
			}
			continue
		}

		if isBattery {
			batInfo, err := parseBatteryDir(dir)
			if err != nil {
				errs.Add(entry.Name(), err)
				continue
			}

			out.BatteryPercent = batInfo.capacity

			if batInfo.powerNow > 0 {
				out.DischargeRateWatts = batInfo.powerNow / 1_000_000
			}

			if batInfo.energyFullDesign > 0 {
				full := batInfo.energyFull
				if full == 0 {
					full = batInfo.energyFullDesign
				}
				degradation := (1 - full/batInfo.energyFullDesign) * 100
				out.CapacityDegradation = degradation
			}
		}
	}

	out.Errors = errs
	return &out, nil
}

func HandleGetPowerAnalytics(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ GetPowerAnalyticsInput,
) (*mcp.CallToolResult, *PowerAnalyticsOutput, error) {
	return handleToolCall(
		ctx,
		"get_power_analytics",
		0,
		GatherPowerAnalytics,
	)
}
