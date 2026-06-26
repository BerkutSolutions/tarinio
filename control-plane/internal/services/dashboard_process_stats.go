package services

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
)

// collectTopProcessStats reads the current process snapshot from /proc and
// computes CPU percentages relative to the previous call. On the very first
// call there is no previous snapshot, so all cpu_percent values are 0; from
// the second call onward (typically 15 s apart) the values are meaningful.
func collectTopProcessStats(totalMemoryBytes uint64) ([]DashboardProcessStats, []DashboardProcessStats) {
	if runtime.GOOS != "linux" {
		return nil, nil
	}

	current, currentTotal, ok := readProcessSamples()
	if !ok {
		return nil, nil
	}

	processState.mu.Lock()
	previous := processState.lastSamples
	previousTotal := processState.lastTotal
	hadSample := processState.hasSample
	processState.lastSamples = current
	processState.lastTotal = currentTotal
	processState.hasSample = true
	processState.mu.Unlock()

	totalDelta := uint64(0)
	if hadSample && currentTotal > previousTotal {
		totalDelta = currentTotal - previousTotal
	}

	items := make([]DashboardProcessStats, 0, len(current))
	for pid, cur := range current {
		if cur.name == "" {
			continue
		}
		cpuPercent := 0.0
		if hadSample && totalDelta > 0 {
			if prev, ok := previous[pid]; ok && cur.cpuJiffies >= prev.cpuJiffies {
				cpuPercent = round1(float64(cur.cpuJiffies-prev.cpuJiffies) * 100 / float64(totalDelta))
			}
		}

		memoryPercent := 0.0
		if totalMemoryBytes > 0 {
			memoryPercent = round1(float64(cur.rssBytes) * 100 / float64(totalMemoryBytes))
		}

		items = append(items, DashboardProcessStats{
			PID:            pid,
			Name:           cur.name,
			Command:        cur.command,
			State:          cur.state,
			Threads:        cur.threads,
			CPUPercent:     cpuPercent,
			MemoryRSSBytes: cur.rssBytes,
			MemoryPercent:  memoryPercent,
		})
	}

	topCPU := append([]DashboardProcessStats(nil), items...)
	sort.Slice(topCPU, func(i, j int) bool {
		if topCPU[i].CPUPercent == topCPU[j].CPUPercent {
			if topCPU[i].MemoryRSSBytes == topCPU[j].MemoryRSSBytes {
				return topCPU[i].PID < topCPU[j].PID
			}
			return topCPU[i].MemoryRSSBytes > topCPU[j].MemoryRSSBytes
		}
		return topCPU[i].CPUPercent > topCPU[j].CPUPercent
	})
	if len(topCPU) > 10 {
		topCPU = topCPU[:10]
	}

	topMemory := append([]DashboardProcessStats(nil), items...)
	sort.Slice(topMemory, func(i, j int) bool {
		if topMemory[i].MemoryRSSBytes == topMemory[j].MemoryRSSBytes {
			if topMemory[i].CPUPercent == topMemory[j].CPUPercent {
				return topMemory[i].PID < topMemory[j].PID
			}
			return topMemory[i].CPUPercent > topMemory[j].CPUPercent
		}
		return topMemory[i].MemoryRSSBytes > topMemory[j].MemoryRSSBytes
	})
	if len(topMemory) > 10 {
		topMemory = topMemory[:10]
	}

	return topCPU, topMemory
}

func readProcessSamples() (map[int]processSample, uint64, bool) {
	totalSample, ok := readCPUTimesSample()
	if !ok {
		return nil, 0, false
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, 0, false
	}

	samples := make(map[int]processSample, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		sample, ok := readSingleProcessSample(pid)
		if !ok {
			continue
		}
		samples[pid] = sample
	}

	return samples, totalSample.total, true
}

func readSingleProcessSample(pid int) (processSample, bool) {
	procRoot := filepath.Join("/proc", strconv.Itoa(pid))
	statContent, err := os.ReadFile(filepath.Join(procRoot, "stat"))
	if err != nil {
		return processSample{}, false
	}
	sample, ok := parseProcessStat(pid, string(statContent))
	if !ok {
		return processSample{}, false
	}

	if cmdlineContent, err := os.ReadFile(filepath.Join(procRoot, "cmdline")); err == nil {
		command := strings.TrimSpace(strings.ReplaceAll(string(cmdlineContent), "\x00", " "))
		if command != "" {
			sample.command = command
		}
	}
	if sample.command == "" {
		sample.command = sample.name
	}
	return sample, true
}

func parseProcessStat(pid int, raw string) (processSample, bool) {
	raw = strings.TrimSpace(raw)
	closing := strings.LastIndex(raw, ")")
	opening := strings.Index(raw, "(")
	if opening < 0 || closing <= opening {
		return processSample{}, false
	}

	name := strings.TrimSpace(raw[opening+1 : closing])
	rest := strings.Fields(strings.TrimSpace(raw[closing+1:]))
	if len(rest) < 22 {
		return processSample{}, false
	}

	utime, err := strconv.ParseUint(rest[11], 10, 64)
	if err != nil {
		return processSample{}, false
	}
	stime, err := strconv.ParseUint(rest[12], 10, 64)
	if err != nil {
		return processSample{}, false
	}
	threads, err := strconv.Atoi(rest[17])
	if err != nil {
		threads = 0
	}
	rssPages, err := strconv.ParseInt(rest[21], 10, 64)
	if err != nil {
		rssPages = 0
	}

	return processSample{
		pid:        pid,
		name:       name,
		state:      strings.TrimSpace(rest[0]),
		threads:    threads,
		rssBytes:   uint64(maxInt64(rssPages, 0)) * uint64(os.Getpagesize()),
		cpuJiffies: utime + stime,
	}, true
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
