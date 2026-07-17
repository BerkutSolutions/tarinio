package services

import (
	"os"
	"runtime"
	"testing"
)

func TestReadSingleProcessSampleDoesNotExposeCommandLineArguments(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("/proc process samples are Linux-specific")
	}

	sample, ok := readSingleProcessSample(os.Getpid())
	if !ok {
		t.Fatal("failed to read current process sample")
	}
	if sample.command != sample.name {
		t.Fatalf("dashboard sample exposed command %q instead of process name %q", sample.command, sample.name)
	}
}
