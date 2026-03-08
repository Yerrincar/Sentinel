package systemd

import (
	"context"
	"os"
	"sentinel/internal/model"
	helpers "sentinel/internal/util"
	"strconv"
	"strings"
	"time"

	dbus "github.com/coreos/go-systemd/v22/dbus"
)

type cpuSample struct {
	usageUsec uint64
	at        time.Time
}

type Sampler struct {
	prevByID map[string]cpuSample
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
	//status
	statusInfo, err := conn.ListUnitsByNamesContext(ctx, []string{unit})
	if err != nil {
		r.ErrorMsg = err.Error()
		return r
	}
	if len(statusInfo) <= 0 {
		r.ErrorMsg = "Status len < 0"
		return r
	}
	status := statusInfo[0].ActiveState
	if status != "" {
		status = strings.ToUpper(status[:1]) + status[1:]
	}
	r.Status = status

	//uptime
	uptimeInfo, err := conn.GetUnitPropertyContext(ctx, unit, "ActiveEnterTimestamp")
	if err != nil {
		r.ErrorMsg = err.Error()
		return r
	}
	uptime, ok := uptimeInfo.Value.Value().(uint64)
	if ok == false {
		r.ErrorMsg = "Error getting uptime"
	}
	uptimeSec := uptime / 1_000_000
	uptimeNsec := (uptime % 1_000_000) * 1_000
	startedAt := time.Unix(int64(uptimeSec), int64(uptimeNsec))

	r.Uptime = helpers.FormatUptime(time.Since(startedAt))

	//cpustats
	cpuPercentage, err := s.cpuMetrics(serviceId, unit, time.Now())
	if err != nil {
		r.ErrorMsg = "App error related to cpu calculation " + err.Error()
		return r
	}
	r.Cpu = cpuPercentage

	return r
}

func NewSampler() *Sampler {
	return &Sampler{prevByID: map[string]cpuSample{}}
}

func (s *Sampler) cpuMetrics(serviceID, unit string, now time.Time) (float64, error) {
	if s.prevByID == nil {
		s.prevByID = make(map[string]cpuSample)
	}
	currUsage, err := readUsageUsec(unit)
	if err != nil {
		return 0.0, err
	}
	prev, ok := s.prevByID[serviceID]
	s.prevByID[serviceID] = cpuSample{usageUsec: currUsage, at: now}

	if !ok {
		return 0.0, err
	}
	dt := now.Sub(prev.at).Microseconds()
	du := int64(currUsage) - int64(prev.usageUsec)
	if dt <= 0 || du < 0 {
		return 0.0, err
	}

	cpuPct := (float64(du) / float64(dt)) * 100.0
	return cpuPct, nil
}

func readUsageUsec(unit string) (uint64, error) {
	ctx := context.Background()
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	cgroupInfo, err := conn.GetUnitTypePropertyContext(ctx, unit, "Service", "ControlGroup")
	if err != nil {
		return 0, err
	}
	cg, ok := cgroupInfo.Value.Value().(string)
	if !ok {
		return 0, err
	}
	servicePath := "/sys/fs/cgroup" + cg

	file, err := os.ReadDir(servicePath)
	if err != nil {
		return 0, err
	}

	cpuUsage := uint64(0)
	for _, m := range file {
		if m.Name() == "cpu.stat" {
			cpuStats, _ := os.ReadFile(servicePath + "/cpu.stat")
			cpuStatsString := string(cpuStats)
			for _, c := range strings.Split(cpuStatsString, "\n") {
				s := strings.Fields(c)
				if len(s) > 1 && s[0] == "usage_usec" {
					cpuUsage, _ = strconv.ParseUint(s[1], 10, 64)
				}
			}
		}
	}
	return cpuUsage, nil
}
