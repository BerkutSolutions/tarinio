package services

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DashboardContainerOverview struct {
	GeneratedAt         string                      `json:"generated_at"`
	HostUptimeSeconds   int64                       `json:"host_uptime_seconds"`
	HostUptimeHuman     string                      `json:"host_uptime_human"`
	TotalContainers     int                         `json:"total_containers"`
	RunningContainers   int                         `json:"running_containers"`
	Containers          []DashboardContainerMetrics `json:"containers"`
	TotalCPUPercent     float64                     `json:"total_cpu_percent"`
	AvgMemoryPercent    float64                     `json:"avg_memory_percent"`
	TotalNetworkInB     uint64                      `json:"total_network_in_bytes"`
	TotalNetworkOutB    uint64                      `json:"total_network_out_bytes"`
	TotalNetworkInText  string                      `json:"total_network_in_text"`
	TotalNetworkOutText string                      `json:"total_network_out_text"`
}

type DashboardContainerMetrics struct {
	ID               string  `json:"id"`
	Name             string  `json:"name"`
	Image            string  `json:"image"`
	Status           string  `json:"status"`
	State            string  `json:"state"`
	RunningFor       string  `json:"running_for"`
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryPercent    float64 `json:"memory_percent"`
	MemoryUsageText  string  `json:"memory_usage_text"`
	MemoryLimitText  string  `json:"memory_limit_text"`
	MemoryUsageBytes uint64  `json:"memory_usage_bytes"`
	MemoryLimitBytes uint64  `json:"memory_limit_bytes"`
	NetworkInText    string  `json:"network_in_text"`
	NetworkOutText   string  `json:"network_out_text"`
	NetworkInBytes   uint64  `json:"network_in_bytes"`
	NetworkOutBytes  uint64  `json:"network_out_bytes"`
	PIDs             int     `json:"pids"`
}

type DashboardContainerLogsRequest struct {
	Container string
	Since     string
	Tail      int
}

type DashboardContainerLogs struct {
	Container string                     `json:"container"`
	Since     string                     `json:"since,omitempty"`
	FetchedAt string                     `json:"fetched_at"`
	Lines     []DashboardContainerLogRow `json:"lines"`
}

type DashboardContainerLogRow struct {
	Timestamp string `json:"timestamp,omitempty"`
	Message   string `json:"message"`
}

type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type osCommandRunner struct{}

func (osCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

type ContainerRuntimeService struct {
	runner commandRunner

	mu                sync.RWMutex
	cachedOverview    DashboardContainerOverview
	hasCachedOverview bool
}

func NewContainerRuntimeService() *ContainerRuntimeService {
	return &ContainerRuntimeService{runner: osCommandRunner{}}
}

func (s *ContainerRuntimeService) Overview() (DashboardContainerOverview, error) {
	if s == nil {
		return DashboardContainerOverview{}, errors.New("container runtime service is nil")
	}
	now := time.Now().UTC()
	psOut, err := s.runDockerPS()
	if err != nil {
		if cached, ok := s.getCachedOverview(); ok {
			return cached, nil
		}
		return DashboardContainerOverview{}, err
	}
	containers := parseDockerPS(psOut)
	statsCtx, cancelStats := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancelStats()
	statsOut, statsErr := s.runner.Run(statsCtx, "docker", "stats", "--no-stream", "--format", "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.PIDs}}")
	statsByName := map[string]DashboardContainerMetrics{}
	if statsErr == nil {
		statsByName = parseDockerStats(statsOut)
	}

	uptimeSeconds := readHostUptimeSeconds()
	out := DashboardContainerOverview{
		GeneratedAt:         now.Format(time.RFC3339Nano),
		HostUptimeSeconds:   uptimeSeconds,
		HostUptimeHuman:     formatUptime(uptimeSeconds),
		TotalContainers:     len(containers),
		Containers:          make([]DashboardContainerMetrics, 0, len(containers)),
		TotalNetworkInText:  "0 B",
		TotalNetworkOutText: "0 B",
	}

	var memoryCount int
	for _, base := range containers {
		item := base
		if item.State == "running" {
			out.RunningContainers++
		}
		if stats, ok := statsByName[item.Name]; ok {
			item.CPUPercent = stats.CPUPercent
			item.MemoryPercent = stats.MemoryPercent
			item.MemoryUsageText = stats.MemoryUsageText
			item.MemoryLimitText = stats.MemoryLimitText
			item.MemoryUsageBytes = stats.MemoryUsageBytes
			item.MemoryLimitBytes = stats.MemoryLimitBytes
			item.NetworkInText = stats.NetworkInText
			item.NetworkOutText = stats.NetworkOutText
			item.NetworkInBytes = stats.NetworkInBytes
			item.NetworkOutBytes = stats.NetworkOutBytes
			item.PIDs = stats.PIDs
		}
		out.TotalCPUPercent += item.CPUPercent
		out.TotalNetworkInB += item.NetworkInBytes
		out.TotalNetworkOutB += item.NetworkOutBytes
		if item.MemoryPercent > 0 {
			out.AvgMemoryPercent += item.MemoryPercent
			memoryCount++
		}
		out.Containers = append(out.Containers, item)
	}
	if memoryCount > 0 {
		out.AvgMemoryPercent = out.AvgMemoryPercent / float64(memoryCount)
	}
	out.TotalCPUPercent = round1(out.TotalCPUPercent)
	out.AvgMemoryPercent = round1(out.AvgMemoryPercent)
	out.TotalNetworkInText = formatBytesAuto(out.TotalNetworkInB)
	out.TotalNetworkOutText = formatBytesAuto(out.TotalNetworkOutB)

	sort.Slice(out.Containers, func(i, j int) bool {
		if out.Containers[i].CPUPercent == out.Containers[j].CPUPercent {
			return out.Containers[i].Name < out.Containers[j].Name
		}
		return out.Containers[i].CPUPercent > out.Containers[j].CPUPercent
	})
	s.putCachedOverview(out)
	return out, nil
}

func (s *ContainerRuntimeService) runDockerPS() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	out, err := s.runner.Run(ctx, "docker", "ps", "-a", "--no-trunc", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.State}}\t{{.RunningFor}}")
	if err == nil {
		return out, nil
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}
	retryCtx, retryCancel := context.WithTimeout(context.Background(), 16*time.Second)
	defer retryCancel()
	return s.runner.Run(retryCtx, "docker", "ps", "-a", "--no-trunc", "--format", "{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}\t{{.State}}\t{{.RunningFor}}")
}

func (s *ContainerRuntimeService) getCachedOverview() (DashboardContainerOverview, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.hasCachedOverview {
		return DashboardContainerOverview{}, false
	}
	return s.cachedOverview, true
}

func (s *ContainerRuntimeService) putCachedOverview(payload DashboardContainerOverview) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cachedOverview = payload
	s.hasCachedOverview = true
}

func (s *ContainerRuntimeService) Logs(req DashboardContainerLogsRequest) (DashboardContainerLogs, error) {
	if s == nil {
		return DashboardContainerLogs{}, errors.New("container runtime service is nil")
	}
	container := strings.TrimSpace(req.Container)
	if container == "" {
		return DashboardContainerLogs{}, errors.New("container is required")
	}
	tail := req.Tail
	if tail <= 0 {
		tail = 1000
	}
	if tail > 10000 {
		tail = 10000
	}

	args := []string{"logs", "--timestamps", "--details"}
	sinceValue := strings.TrimSpace(req.Since)
	if sinceValue != "" {
		sinceTime, err := time.Parse(time.RFC3339Nano, sinceValue)
		if err != nil {
			if alt, errAlt := time.Parse(time.RFC3339, sinceValue); errAlt == nil {
				sinceTime = alt
			} else {
				return DashboardContainerLogs{}, errors.New("invalid since format")
			}
		}
		args = append(args, "--since", sinceTime.UTC().Format(time.RFC3339Nano))
	} else {
		args = append(args, "--tail", strconv.Itoa(tail))
	}
	args = append(args, container)

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	output, err := s.runner.Run(ctx, "docker", args...)
	if err != nil {
		return DashboardContainerLogs{}, err
	}

	now := time.Now().UTC()
	lines := parseDockerLogLines(output)
	return DashboardContainerLogs{
		Container: container,
		Since:     sinceValue,
		FetchedAt: now.Format(time.RFC3339Nano),
		Lines:     lines,
	}, nil
}

func parseDockerPS(out []byte) []DashboardContainerMetrics {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	items := make([]DashboardContainerMetrics, 0, 8)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 6 {
			continue
		}
		items = append(items, DashboardContainerMetrics{
			ID:         strings.TrimSpace(parts[0]),
			Name:       strings.TrimSpace(parts[1]),
			Image:      strings.TrimSpace(parts[2]),
			Status:     strings.TrimSpace(parts[3]),
			State:      strings.TrimSpace(parts[4]),
			RunningFor: strings.TrimSpace(parts[5]),
		})
	}
	return items
}

func parseDockerStats(out []byte) map[string]DashboardContainerMetrics {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	items := map[string]DashboardContainerMetrics{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 6 {
			continue
		}
		name := strings.TrimSpace(parts[0])
		if name == "" {
			continue
		}
		memUsedText, memLimitText, memUsed, memLimit := parseMemoryUsageField(parts[2])
		netInText, netOutText, netIn, netOut := parseNetworkField(parts[4])
		items[name] = DashboardContainerMetrics{
			Name:             name,
			CPUPercent:       parsePercent(parts[1]),
			MemoryPercent:    parsePercent(parts[3]),
			MemoryUsageText:  memUsedText,
			MemoryLimitText:  memLimitText,
			MemoryUsageBytes: memUsed,
			MemoryLimitBytes: memLimit,
			NetworkInText:    netInText,
			NetworkOutText:   netOutText,
			NetworkInBytes:   netIn,
			NetworkOutBytes:  netOut,
			PIDs:             parseInt(parts[5]),
		}
	}
	return items
}

func parseDockerLogLines(out []byte) []DashboardContainerLogRow {
	scanner := bufio.NewScanner(bytes.NewReader(out))
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	lines := make([]DashboardContainerLogRow, 0, 256)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		firstSpace := strings.IndexByte(line, ' ')
		if firstSpace > 0 {
			ts := strings.TrimSpace(line[:firstSpace])
			msg := strings.TrimSpace(line[firstSpace+1:])
			if _, err := time.Parse(time.RFC3339Nano, ts); err == nil {
				lines = append(lines, DashboardContainerLogRow{Timestamp: ts, Message: msg})
				continue
			}
		}
		lines = append(lines, DashboardContainerLogRow{Message: strings.TrimSpace(line)})
	}
	return lines
}

func parsePercent(value string) float64 {
	raw := strings.TrimSpace(strings.TrimSuffix(value, "%"))
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return round1(f)
}

func parseInt(value string) int {
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return n
}

func parseMemoryUsageField(value string) (string, string, uint64, uint64) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		token := strings.TrimSpace(value)
		bytesValue := parseSizeToBytes(token)
		return token, "", bytesValue, 0
	}
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	return left, right, parseSizeToBytes(left), parseSizeToBytes(right)
}

func parseNetworkField(value string) (string, string, uint64, uint64) {
	parts := strings.Split(value, "/")
	if len(parts) != 2 {
		token := strings.TrimSpace(value)
		return token, "", parseSizeToBytes(token), 0
	}
	left := strings.TrimSpace(parts[0])
	right := strings.TrimSpace(parts[1])
	return left, right, parseSizeToBytes(left), parseSizeToBytes(right)
}

var sizePattern = regexp.MustCompile(`(?i)^\s*([0-9]+(?:\.[0-9]+)?)\s*([kmgtpe]?i?b)?\s*$`)

func parseSizeToBytes(value string) uint64 {
	match := sizePattern.FindStringSubmatch(strings.TrimSpace(value))
	if len(match) < 3 {
		return 0
	}
	number, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0
	}
	unit := strings.ToUpper(strings.TrimSpace(match[2]))
	switch unit {
	case "", "B":
		return uint64(number)
	case "KB":
		return uint64(number * 1000)
	case "MB":
		return uint64(number * 1000 * 1000)
	case "GB":
		return uint64(number * 1000 * 1000 * 1000)
	case "TB":
		return uint64(number * 1000 * 1000 * 1000 * 1000)
	case "KIB":
		return uint64(number * 1024)
	case "MIB":
		return uint64(number * 1024 * 1024)
	case "GIB":
		return uint64(number * 1024 * 1024 * 1024)
	case "TIB":
		return uint64(number * 1024 * 1024 * 1024 * 1024)
	default:
		return 0
	}
}

func formatBytesAuto(value uint64) string {
	if value == 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	v := float64(value)
	unit := 0
	for v >= 1024 && unit < len(units)-1 {
		v /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%.0f %s", v, units[unit])
	}
	return fmt.Sprintf("%.1f %s", v, units[unit])
}

func readHostUptimeSeconds() int64 {
	content, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(content))
	if len(fields) < 1 {
		return 0
	}
	value, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0
	}
	if value < 0 {
		return 0
	}
	return int64(value)
}

func formatUptime(seconds int64) string {
	if seconds <= 0 {
		return "n/a"
	}
	days := seconds / 86400
	hours := (seconds % 86400) / 3600
	minutes := (seconds % 3600) / 60
	parts := make([]string, 0, 3)
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || days > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	parts = append(parts, fmt.Sprintf("%dm", minutes))
	return strings.Join(parts, " ")
}
