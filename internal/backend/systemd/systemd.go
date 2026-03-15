package systemd

import (
	"context"
	"fmt"
	"os"
	"sentinel/internal/model"
	helpers "sentinel/internal/util"
	"strconv"
	"strings"
	"time"

	dbus "github.com/coreos/go-systemd/v22/dbus"
	"github.com/coreos/go-systemd/v22/sdjournal"
)

type cpuSample struct {
	usageUsec uint64
	at        time.Time
}

type Sampler struct {
	prevByID map[string]cpuSample
}

type UnitSample struct {
	Status          string
	StartedAt       time.Time
	CPUUsageUsec    uint64
	MemCurrentBytes uint64
	MemMaxBytes     uint64
	MemUnlimited    bool
}

func (s *Sampler) GetSystemdMetrics(serviceId, unit string) model.ServiceRuntime {
	var r model.ServiceRuntime
	ctx := context.Background()
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		r.ErrorMsg = err.Error()
		return r
	}
	defer conn.Close()

	if unit == "" {
		r.ErrorMsg = "Empty unit name"
		return r
	}
	sample, err := readMetrics(ctx, conn, unit)
	if err != nil {
		r.ErrorMsg = err.Error()
		return r
	}

	r.Status = sample.Status
	r.State = mapSystemdStatus(sample.Status)
	r.Cpu = s.cpuMetrics(serviceId, sample.CPUUsageUsec, time.Now())
	if !sample.StartedAt.IsZero() {
		r.Uptime = helpers.FormatUptime(time.Since(sample.StartedAt))
	}
	r.Mem = helpers.FormatBytes(sample.MemCurrentBytes)
	if sample.MemUnlimited {
		r.MemLimit = "No limit assigned"
	} else {
		r.MemLimit = helpers.FormatBytes(sample.MemMaxBytes)
	}

	return r
}

func NewSampler() *Sampler {
	return &Sampler{prevByID: map[string]cpuSample{}}
}

func (s *Sampler) cpuMetrics(serviceID string, currUsage uint64, now time.Time) float64 {
	if s.prevByID == nil {
		s.prevByID = make(map[string]cpuSample)
	}
	prev, ok := s.prevByID[serviceID]
	s.prevByID[serviceID] = cpuSample{usageUsec: currUsage, at: now}

	if !ok {
		return 0.0
	}
	return helpers.CPUPercent(prev.usageUsec, currUsage, prev.at, now)
}

func GetUnitLogs(unitName string) (string, error) {
	r, err := sdjournal.NewJournal()
	if err != nil {
		return "", err
	}
	defer r.Close()

	err = r.AddMatch("_SYSTEMD_UNIT=" + unitName)
	if err != nil {
		return "", err
	}

	if err := r.SeekTail(); err == nil {
		_, _ = r.PreviousSkip(200)
	}

	var b strings.Builder
	for {
		n, err := r.Next()
		if err != nil {
			return "", err
		}
		if n == 0 {
			break
		}

		entry, err := r.GetEntry()
		if err != nil {
			continue
		}

		_, _ = b.WriteString(fmt.Sprintf("%s\n", entry.Fields["MESSAGE"]))
	}

	return b.String(), nil
}

func readMetrics(ctx context.Context, conn *dbus.Conn, unit string) (UnitSample, error) {
	var u UnitSample

	statusInfo, err := conn.ListUnitsByNamesContext(ctx, []string{unit})
	if err != nil {
		return u, err
	}
	if len(statusInfo) == 0 {
		return u, fmt.Errorf("unit %s not found", unit)
	}
	u.Status = statusInfo[0].ActiveState
	if u.Status != "" {
		u.Status = strings.ToUpper(u.Status[:1]) + u.Status[1:]
	}

	uptimeInfo, err := conn.GetUnitPropertyContext(ctx, unit, "ActiveEnterTimestamp")
	if err != nil {
		return u, err
	}
	if uptime, ok := uptimeInfo.Value.Value().(uint64); ok && uptime > 0 {
		uptimeSec := uptime / 1_000_000
		uptimeNsec := (uptime % 1_000_000) * 1_000
		u.StartedAt = time.Unix(int64(uptimeSec), int64(uptimeNsec))
	}

	cgroupInfo, err := conn.GetUnitTypePropertyContext(ctx, unit, "Service", "ControlGroup")
	if err != nil {
		return u, err
	}
	cg, ok := cgroupInfo.Value.Value().(string)
	if !ok {
		return u, err
	}
	servicePath := "/sys/fs/cgroup" + cg

	cpuStats, err := os.ReadFile(servicePath + "/cpu.stat")
	if err != nil {
		return u, err
	}
	for _, c := range strings.Split(string(cpuStats), "\n") {
		fields := strings.Fields(c)
		if len(fields) > 1 && fields[0] == "usage_usec" {
			u.CPUUsageUsec, err = strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				return u, err
			}
			break
		}
	}

	memCurrentRaw, err := os.ReadFile(servicePath + "/memory.current")
	if err == nil {
		u.MemCurrentBytes, _ = strconv.ParseUint(strings.TrimSpace(string(memCurrentRaw)), 10, 64)
	}

	memMaxRaw, err := os.ReadFile(servicePath + "/memory.max")
	if err == nil {
		memMax := strings.TrimSpace(string(memMaxRaw))
		if memMax == "max" {
			u.MemUnlimited = true
		} else {
			u.MemMaxBytes, _ = strconv.ParseUint(memMax, 10, 64)
		}
	}

	return u, nil
}

func SystemdStart(systemdUnit string) (int, error) {
	ctx := context.Background()
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	ch := make(chan string, 1)
	result, err := conn.StartUnitContext(ctx, systemdUnit, "replace", ch)
	return result, nil
}

func SystemdStop(systemdUnit string) (int, error) {
	ctx := context.Background()
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	ch := make(chan string, 1)
	result, err := conn.StopUnitContext(ctx, systemdUnit, "replace", ch)
	return result, nil
}

func SystemdRestart(systemdUnit string) (int, error) {
	ctx := context.Background()
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	ch := make(chan string, 1)
	result, err := conn.RestartUnitContext(ctx, systemdUnit, "replace", ch)
	return result, err
}

func mapSystemdStatus(status string) string {
	switch strings.ToLower(status) {
	case "active":
		return "running"
	case "inactive", "deactivating":
		return "inactive"
	case "failed":
		return "stopped"
	default:
		return "degraded"
	}
}
