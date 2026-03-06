package systemd

import (
	"context"

	dbus "github.com/coreos/go-systemd/v22/dbus"
)

type ServiceRuntime struct {
	Cpu      float64
	Mem      string
	MemLimit string
	Status   string
	Uptime   string
	ErrorMsg string
}

func GetSystemdMetrics(unit string) ServiceRuntime {
	var r ServiceRuntime
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
	status, err := conn.ListUnitsByNamesContext(ctx, []string{unit})
	if err != nil {
		r.ErrorMsg = err.Error()
		return r
	}
	r.Status = status[0].ActiveState
	return r
}
