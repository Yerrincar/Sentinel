package docker

import (
	"context"
	"encoding/json"
	"sentinel/internal/model"
	helpers "sentinel/internal/util"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type state int

func GetMetricsFromContainer(dockerContainer string) model.ServiceRuntime {
	var r model.ServiceRuntime
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := client.New(client.FromEnv)
	if err != nil {
		r.ErrorMsg = err.Error()
		return r
	}
	resp, err := cli.ContainerStats(ctx, dockerContainer, client.ContainerStatsOptions{Stream: false})
	if err != nil {
		r.ErrorMsg = err.Error()
		return r
	}
	inspection, err := cli.ContainerInspect(ctx, dockerContainer, client.ContainerInspectOptions{})
	if err != nil {
		r.ErrorMsg = err.Error()
		return r
	}

	defer resp.Body.Close()
	var stats container.StatsResponse
	_ = json.NewDecoder(resp.Body).Decode(&stats)

	currentContainerCPU := stats.CPUStats.CPUUsage.TotalUsage
	prevContainerCPU := stats.PreCPUStats.CPUUsage.TotalUsage
	currentSystemCPUTotal := stats.CPUStats.SystemUsage
	prevSystemCPUTotal := stats.PreCPUStats.SystemUsage
	cpuDelta := currentContainerCPU - prevContainerCPU
	systemDelta := currentSystemCPUTotal - prevSystemCPUTotal

	r.Cpu = float64((cpuDelta / systemDelta) * uint64(stats.CPUStats.OnlineCPUs) * 100)

	r.Mem = "0.0"
	if (stats.MemoryStats.Usage / (1024 * 1024)) < 1024 {
		r.Mem = strconv.FormatFloat(float64(stats.MemoryStats.Usage)/(1024.0*1024.0), 'f', 1, 64) + " MiB"
	} else {
		r.Mem = strconv.FormatFloat(float64(stats.MemoryStats.Usage)/(1024.0*1024.0*1024.0), 'f', 1, 64) + " GiB"
	}

	if (stats.MemoryStats.Limit / (1024 * 1024)) < 1024 {
		r.MemLimit = strconv.FormatFloat(float64(stats.MemoryStats.Limit)/(1024.0*1024.0), 'f', 1, 64) + " MiB"
	} else {
		r.MemLimit = strconv.FormatFloat(float64(stats.MemoryStats.Limit)/(1024.0*1024.0*1024.0), 'f', 1, 64) + " GiB"
	}
	status := string(inspection.Container.State.Status)
	if status != "" {
		status = strings.ToUpper(status[:1]) + status[1:]
	}
	r.Status = status
	uptime, err := time.Parse(time.RFC3339Nano, inspection.Container.State.StartedAt)
	if err != nil {
		r.ErrorMsg = err.Error()
		return r
	}
	r.Uptime = helpers.FormatUptime(time.Since(uptime))
	return r
}
