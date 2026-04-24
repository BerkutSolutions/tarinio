package source

import (
	"bufio"
	"encoding/json"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var accessLogPattern = regexp.MustCompile(`^(\S+) \S+ \S+ \[([^\]]+)\] "([A-Z]+) ([^"]*?) HTTP/[^"]*" (\d{3}) (\S+) "([^"]*)" "([^"]*)" "([^"]*)"$`)

type FileBackend struct {
	path string
}

type jsonAccess struct {
	Timestamp string `json:"timestamp"`
	ClientIP  string `json:"client_ip"`
	Site      string `json:"site"`
	Status    int    `json:"status"`
	Method    string `json:"method"`
	URI       string `json:"uri"`
	UserAgent string `json:"user_agent"`
}

func NewFileBackend(path string) *FileBackend {
	return &FileBackend{path: strings.TrimSpace(path)}
}

func (b *FileBackend) Read(offset int64) ([]Event, int64, error) {
	file, err := os.Open(b.path)
	if err != nil {
		return nil, offset, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, offset, err
	}
	if offset > stat.Size() {
		offset = 0
	}
	if _, err := file.Seek(offset, 0); err != nil {
		return nil, offset, err
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	out := make([]Event, 0, 32)
	for scanner.Scan() {
		item, ok := ParseAccessLine(strings.TrimSpace(scanner.Text()))
		if ok {
			out = append(out, item)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, offset, err
	}
	pos, err := file.Seek(0, 1)
	if err != nil {
		return nil, offset, err
	}
	return out, pos, nil
}

func ParseAccessLine(line string) (Event, bool) {
	if strings.HasPrefix(line, "{") {
		var item jsonAccess
		if err := json.Unmarshal([]byte(line), &item); err == nil {
			ip := strings.TrimSpace(item.ClientIP)
			if ip != "" && item.Status > 0 {
				when, err := time.Parse(time.RFC3339, strings.TrimSpace(item.Timestamp))
				if err == nil {
					return Event{
						IP:        ip,
						Site:      strings.TrimSpace(item.Site),
						Status:    item.Status,
						Method:    strings.TrimSpace(item.Method),
						Path:      strings.TrimSpace(item.URI),
						UserAgent: strings.TrimSpace(item.UserAgent),
						When:      when.UTC(),
					}, true
				}
			}
		}
	}

	matches := accessLogPattern.FindStringSubmatch(line)
	if len(matches) != 10 {
		return Event{}, false
	}
	ip := strings.TrimSpace(matches[1])
	if ip == "" {
		return Event{}, false
	}
	status, err := strconv.Atoi(matches[5])
	if err != nil {
		return Event{}, false
	}
	when, err := time.Parse("02/Jan/2006:15:04:05 -0700", matches[2])
	if err != nil {
		return Event{}, false
	}
	return Event{
		IP:        ip,
		Site:      strings.TrimSpace(matches[9]),
		Status:    status,
		Method:    strings.TrimSpace(matches[3]),
		Path:      strings.TrimSpace(matches[4]),
		UserAgent: strings.TrimSpace(matches[8]),
		When:      when.UTC(),
	}, true
}
