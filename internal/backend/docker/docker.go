package docker

import (
	"context"
	"encoding/json"
	"log"
	"sentinel/internal/config"
	"strconv"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type state int

const (
	running state = iota
	stopped
	serviceError
	unknown
)

type ServiceRuntime struct {
	cpu      float64
	mem      string
	memLimit float64
	status   state
	uptime   int
	errorMsg string
}

func (r *ServiceRuntime) GetMetricsFromContainer(y *config.YamlConfig, dockerContainer string) string {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := client.New(client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := cli.ContainerStats(ctx, dockerContainer, client.ContainerStatsOptions{Stream: true})
	if err != nil {
		log.Printf("Error getting the container stats: %v", err.Error())
		return ""
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
	r.cpu = float64((cpuDelta / systemDelta) * uint64(stats.CPUStats.OnlineCPUs) * 100)

	if (stats.MemoryStats.Usage / (1024 * 1024)) < 1024 {
		r.mem = strconv.FormatFloat(float64(stats.MemoryStats.Usage)/(1024.0*1024.0), 'f', 1, 64) + " MiB"
	} else {
		r.mem = strconv.FormatFloat(float64(stats.MemoryStats.Usage)/(1024.0*1024.0*1024.0), 'f', 1, 64) + " GiB"
	}

	r.memLimit = float64(stats.MemoryStats.Limit) / (1024.0 * 1024.0 * 1024.0)
	return strconv.FormatFloat(r.cpu, 'f', 1, 64) + "%" + "\n" + r.mem + " / " + strconv.FormatFloat(r.memLimit, 'f', 1, 64) + " GiB"
}
