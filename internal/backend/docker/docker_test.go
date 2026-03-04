package docker

import (
	"math"
	"testing"
)

func cpuPercent(cpuDelta, systemDelta uint64, onlineCpus uint32) float64 {
	if systemDelta == 0 || cpuDelta == 0 {
		return 0
	}
	return (float64(cpuDelta) / float64(systemDelta)) * float64(onlineCpus) * 100
}

func TestCPUPercent(t *testing.T) {
	cases := []struct {
		name        string
		cpuDelta    uint64
		systemDelta uint64
		cpus        uint32
		want        float64
	}{
		{"normal", 50, 100, 2, 100.0},
		{"zero system delta", 50, 0, 2, 0.0},
		{"zero online cpus", 50, 100, 0, 0.0},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := cpuPercent(tc.cpuDelta, tc.systemDelta, tc.cpus)
			if math.Abs(got-tc.want) > 0.0001 {
				t.Fatalf("got=%f want=%f", got, tc.want)
			}
		})
	}
}
