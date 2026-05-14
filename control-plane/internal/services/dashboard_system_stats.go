package services

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func collectSystemStats() DashboardSystemStats {
	total, free := readMemoryFromProc()
	used := uint64(0)
	if total >= free {
		used = total - free
	}
	usedPercent := 0.0
	if total > 0 {
		usedPercent = float64(used) * 100 / float64(total)
	}

	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)
	topCPUProcesses, topMemoryProcesses := collectTopProcessStats(total)

	return DashboardSystemStats{
		CPUCores:           runtime.NumCPU(),
		CPULoadPercent:     readCPULoadPercent(),
		MemoryTotalBytes:   total,
		MemoryUsedBytes:    used,
		MemoryFreeBytes:    free,
		MemoryUsedPercent:  round1(usedPercent),
		Goroutines:         runtime.NumGoroutine(),
		ControlPlaneHeapMB: memStats.HeapAlloc / (1024 * 1024),
		TopCPUProcesses:    topCPUProcesses,
		TopMemoryProcesses: topMemoryProcesses,
	}
}

func readCPULoadPercent() float64 {
	current, ok := readCPUTimesSample()
	if !ok {
		return 0
	}

	cpuUsageState.mu.Lock()
	if cpuUsageState.hasSample {
		percent, valid := calculateCPUUsagePercent(cpuUsageState.lastSample, current)
		cpuUsageState.lastSample = current
		if valid {
			cpuUsageState.lastPercent = percent
			cpuUsageState.mu.Unlock()
			return percent
		}
		lastPercent := cpuUsageState.lastPercent
		cpuUsageState.mu.Unlock()
		return lastPercent
	}
	cpuUsageState.lastSample = current
	cpuUsageState.hasSample = true
	cpuUsageState.mu.Unlock()

	time.Sleep(120 * time.Millisecond)

	next, ok := readCPUTimesSample()
	if !ok {
		return 0
	}
	percent, valid := calculateCPUUsagePercent(current, next)
	if !valid {
		return 0
	}

	cpuUsageState.mu.Lock()
	cpuUsageState.lastSample = next
	cpuUsageState.lastPercent = percent
	cpuUsageState.hasSample = true
	cpuUsageState.mu.Unlock()
	return percent
}

func readCPUTimesSample() (cpuTimesSample, bool) {
	content, err := os.ReadFile("/proc/stat")
	if err != nil {
		return cpuTimesSample{}, false
	}
	lines := strings.Split(string(content), "\n")
	if len(lines) == 0 {
		return cpuTimesSample{}, false
	}
	fields := strings.Fields(strings.TrimSpace(lines[0]))
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuTimesSample{}, false
	}
	var total uint64
	values := make([]uint64, 0, len(fields)-1)
	for _, field := range fields[1:] {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return cpuTimesSample{}, false
		}
		values = append(values, value)
		total += value
	}
	idle := values[3]
	if len(values) > 4 {
		idle += values[4]
	}
	return cpuTimesSample{
		idle:     idle,
		total:    total,
		captured: time.Now(),
	}, true
}

func calculateCPUUsagePercent(previous, current cpuTimesSample) (float64, bool) {
	if current.total <= previous.total {
		return 0, false
	}
	totalDelta := current.total - previous.total
	if totalDelta == 0 {
		return 0, false
	}
	idleDelta := uint64(0)
	if current.idle >= previous.idle {
		idleDelta = current.idle - previous.idle
	}
	busyDelta := totalDelta - minUint64(idleDelta, totalDelta)
	percent := round1((float64(busyDelta) * 100) / float64(totalDelta))
	if percent < 0 {
		return 0, false
	}
	if percent > 100 {
		percent = 100
	}
	return percent, true
}

func minUint64(left, right uint64) uint64 {
	if left < right {
		return left
	}
	return right
}

func readMemoryFromProc() (uint64, uint64) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer file.Close()

	var totalKB uint64
	var availableKB uint64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			totalKB = parseMemInfoKB(line)
		} else if strings.HasPrefix(line, "MemAvailable:") {
			availableKB = parseMemInfoKB(line)
		}
	}
	if totalKB == 0 {
		return 0, 0
	}
	return totalKB * 1024, availableKB * 1024
}

func parseMemInfoKB(line string) uint64 {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0
	}
	value, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}
	return value
}
