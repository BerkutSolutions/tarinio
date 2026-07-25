package services

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type dashboardContainersTestRunner struct{}

func (dashboardContainersTestRunner) Run(_ context.Context, _ string, args ...string) ([]byte, error) {
	if len(args) > 0 && args[0] == "ps" {
		return []byte(strings.Join([]string{
			"id-a\tcontainer-a\timage-a\tUp\trunning\t1 minute\tcom.docker.compose.project=unit",
			"id-b\tcontainer-b\timage-b\tUp\trunning\t1 minute\tcom.docker.compose.project=unit",
		}, "\n")), nil
	}
	if len(args) > 0 && args[0] == "stats" {
		return []byte(strings.Join([]string{
			"container-a\t120.0%\t100MiB / 1000MiB\t10.0%\t1kB / 2kB\t3",
			"container-b\t30.0%\t100MiB / 100MiB\t100.0%\t3kB / 4kB\t4",
		}, "\n")), nil
	}
	return nil, fmt.Errorf("unexpected docker invocation: %v", args)
}

func TestContainerOverviewUsesRawCPUAndWeightedMemory(t *testing.T) {
	service := NewContainerRuntimeService()
	service.runner = dashboardContainersTestRunner{}

	overview, err := service.Overview()
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if overview.TotalCPUPercent != 150 {
		t.Fatalf("total CPU must preserve docker container percentages, got %.1f", overview.TotalCPUPercent)
	}
	if overview.CPUCapacityPercent <= 0 {
		t.Fatalf("CPU capacity must be positive, got %.1f", overview.CPUCapacityPercent)
	}
	if overview.AvgMemoryPercent != 18.2 {
		t.Fatalf("memory aggregate must be usage/limit weighted, got %.1f", overview.AvgMemoryPercent)
	}
	if len(overview.Containers) != 2 || overview.Containers[0].CPUPercent != 120 || overview.Containers[1].CPUPercent != 30 {
		t.Fatalf("container CPU rows lost docker values: %#v", overview.Containers)
	}
}
