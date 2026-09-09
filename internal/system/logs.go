package system

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/ols-cli/ols/internal/site"
)

type ErrorLogEntry struct {
	Source  string
	Message string
}

// GetContainerLogs lấy n dòng log gần nhất của một container Docker
func GetContainerLogs(containerName string, lines int) (string, error) {
	if lines <= 0 {
		lines = 50
	}
	out, err := exec.Command("docker", "logs", "--tail", strconv.Itoa(lines), containerName).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("không thể đọc log container %s: %w (%s)", containerName, err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// GetSiteLogs lấy nhật ký lỗi của một website WordPress cụ thể
func GetSiteLogs(domain string, lines int) (string, error) {
	if lines <= 0 {
		lines = 50
	}
	slug := site.DomainToSlug(domain)
	containerName := fmt.Sprintf("ols_%s", slug)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== NHẬT KÝ CONTAINER DOCKER (%s) ===\n", containerName))

	cLogs, err := GetContainerLogs(containerName, lines)
	if err != nil {
		sb.WriteString(fmt.Sprintf("(Lỗi đọc container log: %v)\n", err))
	} else if strings.TrimSpace(cLogs) == "" {
		sb.WriteString("(Không có log stdout/stderr mới)\n")
	} else {
		sb.WriteString(cLogs)
		sb.WriteString("\n")
	}

	// Cố gắng đọc thêm OpenLiteSpeed error.log từ bên trong container nếu có
	vhOut, err := exec.Command("docker", "exec", containerName, "tail", "-n", strconv.Itoa(lines), "/usr/local/lsws/Example/logs/error.log").CombinedOutput()
	if err == nil && len(strings.TrimSpace(string(vhOut))) > 0 {
		sb.WriteString("\n=== NHẬT KÝ LỖI OPENLITESPEED (error.log) ===\n")
		sb.WriteString(strings.TrimSpace(string(vhOut)))
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// FilterErrorLines lọc các dòng chứa từ khóa lỗi từ một khối log text
func FilterErrorLines(source string, rawLog string) []ErrorLogEntry {
	var entries []ErrorLogEntry
	if strings.TrimSpace(rawLog) == "" {
		return entries
	}

	errRegex := regexp.MustCompile(`(?i)\b(error|fatal|exception|critical|panic|emerg|alert|connection refused|oom|access denied)\b`)

	lines := strings.Split(rawLog, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if errRegex.MatchString(line) {
			entries = append(entries, ErrorLogEntry{
				Source:  source,
				Message: line,
			})
		}
	}
	return entries
}

// ScanAllErrors quét nhanh các lỗi gần đây trên toàn bộ hạ tầng và các website
func ScanAllErrors(linesPerTarget int) ([]ErrorLogEntry, error) {
	if linesPerTarget <= 0 {
		linesPerTarget = 50
	}

	var allErrors []ErrorLogEntry

	// 1. Quét Core containers
	coreContainers := []struct {
		name string
		desc string
	}{
		{"ols-traefik", "Traefik SSL/Proxy"},
		{"ols-mariadb", "MariaDB Database"},
		{"ols-redis", "Redis Cache"},
	}

	for _, c := range coreContainers {
		out, err := GetContainerLogs(c.name, linesPerTarget)
		if err == nil {
			errs := FilterErrorLines(c.desc, out)
			allErrors = append(allErrors, errs...)
		}
	}

	// 2. Tìm các container website đang chạy
	psOut, err := exec.Command("docker", "ps", "--format", "{{.Names}}").CombinedOutput()
	if err == nil {
		names := strings.Split(strings.TrimSpace(string(psOut)), "\n")
		for _, name := range names {
			name = strings.TrimSpace(name)
			if strings.HasPrefix(name, "ols_") && !strings.HasPrefix(name, "ols-") {
				out, err := GetContainerLogs(name, linesPerTarget)
				if err == nil {
					errs := FilterErrorLines(name, out)
					allErrors = append(allErrors, errs...)
				}
			}
		}
	}

	return allErrors, nil
}
