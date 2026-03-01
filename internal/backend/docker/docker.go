package docker

import (
	"context"
	"encoding/json"
	"log"
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
	Cpu      float64
	Mem      string
	MemLimit string
	Status   state
	Uptime   int
	ErrorMsg string
}

func GetMetricsFromContainer(dockerContainer string) ServiceRuntime {
	var r ServiceRuntime
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := client.New(client.FromEnv)
	if err != nil {
		log.Fatal(err)
		r.ErrorMsg = err.Error()
		return r
	}
	resp, err := cli.ContainerStats(ctx, dockerContainer, client.ContainerStatsOptions{Stream: true})
	if err != nil {
		log.Printf("Error getting the container stats: %v", err.Error())
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
	return r
}
