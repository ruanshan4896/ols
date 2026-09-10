package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ols-cli/ols/internal/site"
)

type ErrorLogEntry struct {
	Source  string
	Message string
}

type IPStat struct {
	IP    string
	Count int
}

type URLStat struct {
	URL   string
	Count int
}

// readLastLinesFromPath đọc n dòng cuối cùng của một tệp tin
func readLastLinesFromPath(filePath string, n int) (string, error) {
	if n <= 0 {
		n = 50
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	lines := strings.Split(strings.TrimRight(string(data), "\r\n"), "\n")
	if len(lines) <= n {
		return strings.Join(lines, "\n"), nil
	}
	return strings.Join(lines[len(lines)-n:], "\n"), nil
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

// GetSiteAccessLog đọc nhật ký truy cập (access.log) của website
func GetSiteAccessLog(systemDir, domain string, lines int) (string, error) {
	if lines <= 0 {
		lines = 50
	}
	if systemDir == "" {
		systemDir = "/opt/ols"
	}
	hostPath := filepath.Join(systemDir, "sites", domain, "logs", "access.log")
	if content, err := readLastLinesFromPath(hostPath, lines); err == nil && len(strings.TrimSpace(content)) > 0 {
		return content, nil
	}

	// Fallback sang đọc từ bên trong container nếu volume chưa mount ra host
	slug := site.DomainToSlug(domain)
	containerName := fmt.Sprintf("ols_%s", slug)
	out, err := exec.Command("docker", "exec", containerName, "tail", "-n", strconv.Itoa(lines), "/usr/local/lsws/Example/logs/access.log").CombinedOutput()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return strings.TrimSpace(string(out)), nil
	}
	return "", fmt.Errorf("chưa có nhật ký truy cập hoặc file access.log đang trống")
}

// GetSiteErrorLog đọc nhật ký lỗi (error.log) của website
func GetSiteErrorLog(systemDir, domain string, lines int) (string, error) {
	if lines <= 0 {
		lines = 50
	}
	if systemDir == "" {
		systemDir = "/opt/ols"
	}
	hostPath := filepath.Join(systemDir, "sites", domain, "logs", "error.log")
	if content, err := readLastLinesFromPath(hostPath, lines); err == nil && len(strings.TrimSpace(content)) > 0 {
		return content, nil
	}

	slug := site.DomainToSlug(domain)
	containerName := fmt.Sprintf("ols_%s", slug)
	out, err := exec.Command("docker", "exec", containerName, "tail", "-n", strconv.Itoa(lines), "/usr/local/lsws/Example/logs/error.log").CombinedOutput()
	if err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return strings.TrimSpace(string(out)), nil
	}
	return "", fmt.Errorf("chưa có nhật ký lỗi hoặc file error.log đang trống")
}

// GetSiteLogs lấy nhật ký tổng quan của một website WordPress cụ thể
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

	// Đọc thêm OpenLiteSpeed error.log
	errLog, err := GetSiteErrorLog("/opt/ols", domain, lines)
	if err == nil && len(errLog) > 0 {
		sb.WriteString("\n=== NHẬT KÝ LỖI OPENLITESPEED (error.log) ===\n")
		sb.WriteString(errLog)
		sb.WriteString("\n")
	}

	return sb.String(), nil
}

// ParseTopIPs thống kê các IP gửi nhiều request nhất từ dữ liệu access log
func ParseTopIPs(rawLog string, topN int) []IPStat {
	if topN <= 0 {
		topN = 10
	}
	ipCounts := make(map[string]int)
	lines := strings.Split(rawLog, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			ip := fields[0]
			if strings.Contains(ip, ".") || strings.Contains(ip, ":") {
				ipCounts[ip]++
			}
		}
	}

	var stats []IPStat
	for ip, count := range ipCounts {
		stats = append(stats, IPStat{IP: ip, Count: count})
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	if len(stats) > topN {
		return stats[:topN]
	}
	return stats
}

// ParseTopURLs thống kê các URL bị gọi nhiều nhất từ dữ liệu access log
func ParseTopURLs(rawLog string, topN int) []URLStat {
	if topN <= 0 {
		topN = 10
	}
	reRequest := regexp.MustCompile(`"(?:GET|POST|HEAD|PUT|DELETE|OPTIONS)\s+([^"\s?]+)`)
	urlCounts := make(map[string]int)
	lines := strings.Split(rawLog, "\n")
	for _, line := range lines {
		matches := reRequest.FindStringSubmatch(line)
		if len(matches) > 1 {
			u := matches[1]
			urlCounts[u]++
		}
	}

	var stats []URLStat
	for u, count := range urlCounts {
		stats = append(stats, URLStat{URL: u, Count: count})
	}
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Count > stats[j].Count
	})

	if len(stats) > topN {
		return stats[:topN]
	}
	return stats
}

// GetSiteTopIPs đọc log và thống kê các IP truy cập nhiều nhất
func GetSiteTopIPs(systemDir, domain string, topN int) ([]IPStat, error) {
	content, err := GetSiteAccessLog(systemDir, domain, 5000)
	if err != nil {
		return nil, err
	}
	return ParseTopIPs(content, topN), nil
}

// GetSiteTopURLs đọc log và thống kê các URL bị gọi nhiều nhất
func GetSiteTopURLs(systemDir, domain string, topN int) ([]URLStat, error) {
	content, err := GetSiteAccessLog(systemDir, domain, 5000)
	if err != nil {
		return nil, err
	}
	return ParseTopURLs(content, topN), nil
}

// FilterErrorLines lọc các dòng chứa từ khóa lỗi từ một khối log text
func FilterErrorLines(source string, rawLog string) []ErrorLogEntry {
	var entries []ErrorLogEntry
	if strings.TrimSpace(rawLog) == "" {
		return entries
	}

	errRegex := regexp.MustCompile(`(?i)\b(error|fatal|exception|critical|panic|emerg|alert|connection refused|oom|access denied|allowed memory size)\b`)

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

// ScanAllErrors quét nhanh các lỗi gần đây trên toàn bộ hạ tầng Core và các website
func ScanAllErrors(linesPerTarget int) ([]ErrorLogEntry, error) {
	if linesPerTarget <= 0 {
		linesPerTarget = 50
	}
	systemDir := "/opt/ols"

	var allErrors []ErrorLogEntry

	// 1. Quét Core containers: Traefik, MariaDB, Redis, PMA
	coreContainers := []struct {
		name string
		desc string
	}{
		{"ols-traefik", "Traefik SSL/Proxy"},
		{"ols-mariadb", "MariaDB Database"},
		{"ols-redis", "Redis Cache"},
		{"ols-pma", "phpMyAdmin"},
	}

	for _, c := range coreContainers {
		out, err := GetContainerLogs(c.name, linesPerTarget)
		if err == nil {
			errs := FilterErrorLines(c.desc, out)
			allErrors = append(allErrors, errs...)
		}
	}

	// 2. Quét các container website và ĐỌC TRỰC TIẾP error.log của từng website
	sitesDir := filepath.Join(systemDir, "sites")
	if entries, err := os.ReadDir(sitesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				domain := entry.Name()
				// Đọc error.log của website
				errLog, err := GetSiteErrorLog(systemDir, domain, linesPerTarget)
				if err == nil && len(errLog) > 0 {
					errs := FilterErrorLines(domain+" (error.log)", errLog)
					allErrors = append(allErrors, errs...)
				}
				// Đọc thêm container log stdout
				slug := site.DomainToSlug(domain)
				cName := "ols_" + slug
				cLog, err := GetContainerLogs(cName, linesPerTarget)
				if err == nil && len(cLog) > 0 {
					errs := FilterErrorLines(cName+" (stdout)", cLog)
					allErrors = append(allErrors, errs...)
				}
			}
		}
	}

	return allErrors, nil
}

// ClearAllLogs làm sạch an toàn toàn bộ log của tất cả container Docker và OpenLiteSpeed
func ClearAllLogs() (int, int64, error) {
	systemDir := "/opt/ols"
	out, err := exec.Command("docker", "ps", "-a", "--format", "{{.ID}}").CombinedOutput()
	if err != nil {
		return 0, 0, fmt.Errorf("không thể lấy danh sách container: %w", err)
	}

	containerIDs := strings.Fields(string(out))
	clearedCount := 0
	var totalBytesFreed int64

	// 1. Truncate log file Docker container
	for _, cid := range containerIDs {
		pathOut, err := exec.Command("docker", "inspect", "--format={{.LogPath}}", cid).CombinedOutput()
		if err == nil {
			logPath := strings.TrimSpace(string(pathOut))
			if logPath != "" && logPath != "<no value>" {
				if fi, err := os.Stat(logPath); err == nil {
					size := fi.Size()
					if size > 0 {
						if truncErr := os.Truncate(logPath, 0); truncErr == nil {
							clearedCount++
							totalBytesFreed += size
						} else {
							_ = exec.Command("truncate", "-s", "0", logPath).Run()
							clearedCount++
							totalBytesFreed += size
						}
					}
				}
			}
		}
	}

	// 2. Dọn sạch các file log trên đĩa cứng VPS trong thư mục sites/*/logs/*.log
	sitesDir := filepath.Join(systemDir, "sites")
	if entries, err := os.ReadDir(sitesDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				siteLogsDir := filepath.Join(sitesDir, entry.Name(), "logs")
				if logFiles, err := os.ReadDir(siteLogsDir); err == nil {
					for _, lf := range logFiles {
						if !lf.IsDir() && strings.HasSuffix(lf.Name(), ".log") {
							fPath := filepath.Join(siteLogsDir, lf.Name())
							if fi, err := os.Stat(fPath); err == nil {
								size := fi.Size()
								if size > 0 {
									_ = os.Truncate(fPath, 0)
									totalBytesFreed += size
									clearedCount++
								}
							}
						}
					}
				}
			}
		}
	}

	// 3. Dọn sạch trong container OpenLiteSpeed nếu có file chưa đồng bộ
	psRunning, _ := exec.Command("docker", "ps", "--format", "{{.Names}}").CombinedOutput()
	names := strings.Split(strings.TrimSpace(string(psRunning)), "\n")
	for _, name := range names {
		name = strings.TrimSpace(name)
		if strings.HasPrefix(name, "ols_") && !strings.HasPrefix(name, "ols-") {
			_ = exec.Command("docker", "exec", name, "sh", "-c", "truncate -s 0 /usr/local/lsws/Example/logs/*.log 2>/dev/null || true").Run()
		}
	}

	return clearedCount, totalBytesFreed, nil
}
