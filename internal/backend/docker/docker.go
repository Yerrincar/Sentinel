package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"sentinel/internal/model"
	helpers "sentinel/internal/util"
	"strconv"
	"strings"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type state int

func GetMetricsFromContainer(dockerContainer string) model.ServiceRuntime {
	var r model.ServiceRuntime
	statusOut, err := exec.Command("systemctl", "is-active", "docker.service").Output()
	if err != nil || strings.TrimSpace(string(statusOut)) != "active" {
		r.Status = "Inactive"
		r.State = "inactive"
		r.ErrorMsg = "docker daemon inactive"
		return r
	}

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
	if cpuDelta != 0 && systemDelta != 0 {
		r.Cpu = float64((cpuDelta / systemDelta) * uint64(stats.CPUStats.OnlineCPUs) * 100)
	}

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
	rawStatus := strings.ToLower(string(inspection.Container.State.Status))
	r.Status, r.State = mapDockerStatus(rawStatus)
	uptime, err := time.Parse(time.RFC3339Nano, inspection.Container.State.StartedAt)
	if err != nil {
		r.ErrorMsg = err.Error()
		return r
	}
	if inspection.Container.State.Running {
		r.Uptime = helpers.FormatUptime(time.Since(uptime))
		return r
	}

	finishedAt, err := time.Parse(time.RFC3339Nano, inspection.Container.State.FinishedAt)
	if err != nil {
		r.Uptime = helpers.FormatUptime(time.Since(uptime))
		return r
	}
	r.Uptime = helpers.FormatUptime(finishedAt.Sub(uptime))
	return r
}

func GetLogsFromContainer(dockerContainerName string) (string, error) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := client.New(client.FromEnv)
	if err != nil {
		return "", err
	}

	logs, err := cli.ContainerLogs(ctx, dockerContainerName, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       "200",
	})
	if err != nil {
		return err.Error(), err
	}
	defer logs.Close()

	var outBuf, errBuf bytes.Buffer
	_, err = stdcopy.StdCopy(&outBuf, &errBuf, logs)
	if err != nil {
		return err.Error(), err
	}
	return outBuf.String(), nil
}
func DockerStart(dockerContainerName string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}
	_, err = cli.ContainerStart(ctx, dockerContainerName, client.ContainerStartOptions{})
	if err != nil {
		return err
	}
	return nil
}

func DockerStop(dockerContainerName string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}
	_, err = cli.ContainerStop(ctx, dockerContainerName, client.ContainerStopOptions{})
	if err != nil {
		return err
	}
	return nil
}

func DockerRestart(dockerContainerName string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := client.New(client.FromEnv)
	if err != nil {
		return err
	}
	_, err = cli.ContainerRestart(ctx, dockerContainerName, client.ContainerRestartOptions{})
	if err != nil {
		return err
	}
	return nil
}

func mapDockerStatus(raw string) (string, string) {
	switch raw {
	case "running":
		return "Running", "running"
	case "restarting", "paused":
		return strings.ToUpper(raw[:1]) + raw[1:], "degraded"
	case "created", "exited":
		return "Inactive", "inactive"
	case "dead", "removing":
		return strings.ToUpper(raw[:1]) + raw[1:], "stopped"
	default:
		if raw == "" {
			return "Unknown", "degraded"
		}
		return strings.ToUpper(raw[:1]) + raw[1:], "degraded"
	}
}
