package systemd

import (
	"context"
	"sentinel/internal/model"
	helpers "sentinel/internal/util"
	"strings"
	"time"

	dbus "github.com/coreos/go-systemd/v22/dbus"
)

func GetSystemdMetrics(unit string) model.ServiceRuntime {
	var r model.ServiceRuntime
	ctx := context.Background()
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		r.ErrorMsg = err.Error()
		return r
	}
	if unit == "" {
		r.ErrorMsg = "Empty unit name"
		return r
	}
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

	//cgroupInfo, err := conn.GetUnitPropertyContext(ctx, unit, "ControlGroup")
	//servicePath := "/sys/fs/cgroup" + cgroupInfo.Value.String()

	return r
}
