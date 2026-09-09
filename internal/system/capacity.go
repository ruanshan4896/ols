package system

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

type CPUInfo struct {
	Cores       int
	Load1       float64
	Load5       float64
	Load15      float64
	LoadPerCore float64
	Status      string
}

type MemInfo struct {
	TotalMB     int
	UsedMB      int
	FreeMB      int
	AvailableMB int
}

type DiskInfo struct {
	TotalGB     float64
	UsedGB      float64
	AvailableGB float64
}

type CapacityReport struct {
	// CPU
	CPUCores          int
	CPULoad1          float64
	CPULoad5          float64
	CPULoad15         float64
	CPULoadPerCore    float64
	CPUStatus         string
	// RAM & Swap
	TotalRAMMB        int
	UsedRAMMB         int
	AvailableRAMMB    int
	SwapTotalMB       int
	SwapFreeMB        int
	SwapUsedMB        int
	// Disk
	DiskTotalGB       float64
	DiskAvailableGB   float64
	// Sites & Estimate
	RunningSitesCount int
	AvgSiteRAMMB      int
	EstSafeSites      int
	EstWithSwapSites  int
	SystemStatus      string
	Recommendation    string
}

// ParseFreeOutput phân tích cú pháp kết quả từ lệnh 'free -m'
func ParseFreeOutput(output string) (MemInfo, SwapInfo) {
	var mem MemInfo
	var swap SwapInfo

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			prefix := strings.ToLower(fields[0])
			if strings.HasPrefix(prefix, "mem") {
				mem.TotalMB, _ = strconv.Atoi(fields[1])
				mem.UsedMB, _ = strconv.Atoi(fields[2])
				mem.FreeMB, _ = strconv.Atoi(fields[3])
				if len(fields) >= 7 {
					mem.AvailableMB, _ = strconv.Atoi(fields[6])
				} else {
					mem.AvailableMB = mem.FreeMB
				}
			} else if strings.HasPrefix(prefix, "swap") {
				swap.TotalMB, _ = strconv.Atoi(fields[1])
				swap.UsedMB, _ = strconv.Atoi(fields[2])
				swap.FreeMB, _ = strconv.Atoi(fields[3])
			}
		}
	}

	return mem, swap
}

// ParseDfOutput phân tích cú pháp kết quả từ lệnh 'df -m'
func ParseDfOutput(output string) DiskInfo {
	var disk DiskInfo
	lines := strings.Split(output, "\n")
	if len(lines) > 1 {
		fields := strings.Fields(lines[1])
		if len(fields) >= 4 {
			totalMB, _ := strconv.ParseFloat(fields[1], 64)
			usedMB, _ := strconv.ParseFloat(fields[2], 64)
			availMB, _ := strconv.ParseFloat(fields[3], 64)

			disk.TotalGB = totalMB / 1024.0
			disk.UsedGB = usedMB / 1024.0
			disk.AvailableGB = availMB / 1024.0
		}
	}
	return disk
}

// ParseDockerStats phân tích mức RAM của các container website (bắt đầu bằng ols_)
func ParseDockerStats(output string) (siteCount int, avgMemMB int, totalSiteMemMB int) {
	lines := strings.Split(output, "\n")
	reMem := regexp.MustCompile(`([0-9.]+)\s*(kib|mib|gib|b)?`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			parts = strings.Fields(line)
		}
		if len(parts) >= 2 {
			name := strings.TrimSpace(parts[0])
			// Chỉ tính các container website của site (ols_<slug>), bỏ qua core (ols-mariadb, ols-redis, ols-traefik, ols-pma)
			if strings.HasPrefix(name, "ols_") && !strings.HasPrefix(name, "ols-") {
				memStr := strings.ToLower(parts[1])
				// Cắt lấy phần trước dấu gạch chéo "78.4MiB / 384MiB"
				if slashIdx := strings.Index(memStr, "/"); slashIdx != -1 {
					memStr = memStr[:slashIdx]
				}

				matches := reMem.FindStringSubmatch(memStr)
				if len(matches) > 1 {
					val, _ := strconv.ParseFloat(matches[1], 64)
					unit := "mib"
					if len(matches) > 2 && matches[2] != "" {
						unit = matches[2]
					}

					var memMB float64
					switch unit {
					case "gib":
						memMB = val * 1024.0
					case "mib":
						memMB = val
					case "kib":
						memMB = val / 1024.0
					case "b":
						memMB = val / (1024.0 * 1024.0)
					default:
						memMB = val
					}

					siteCount++
					totalSiteMemMB += int(memMB)
				}
			}
		}
	}

	if siteCount > 0 {
		avgMemMB = totalSiteMemMB / siteCount
	} else {
		avgMemMB = 85 // Ước lượng mặc định 85MB nếu chưa có site nào chạy
	}

	return siteCount, avgMemMB, totalSiteMemMB
}

// ParseLoadAvg phân tích cú pháp chuỗi từ /proc/loadavg
func ParseLoadAvg(content string, cores int) CPUInfo {
	if cores <= 0 {
		cores = 1
	}
	cpu := CPUInfo{Cores: cores}
	fields := strings.Fields(content)
	if len(fields) >= 3 {
		cpu.Load1, _ = strconv.ParseFloat(fields[0], 64)
		cpu.Load5, _ = strconv.ParseFloat(fields[1], 64)
		cpu.Load15, _ = strconv.ParseFloat(fields[2], 64)
	}
	calculateCPUStatus(&cpu)
	return cpu
}

// ParseUptime phân tích cú pháp dòng load average từ lệnh uptime
func ParseUptime(content string, cores int) CPUInfo {
	if cores <= 0 {
		cores = 1
	}
	cpu := CPUInfo{Cores: cores}
	lower := strings.ToLower(content)
	idx := strings.Index(lower, "load average")
	if idx != -1 {
		after := content[idx:]
		if colon := strings.Index(after, ":"); colon != -1 {
			after = after[colon+1:]
		}
		after = strings.ReplaceAll(after, ",", " ")
		fields := strings.Fields(after)
		if len(fields) >= 3 {
			cpu.Load1, _ = strconv.ParseFloat(fields[0], 64)
			cpu.Load5, _ = strconv.ParseFloat(fields[1], 64)
			cpu.Load15, _ = strconv.ParseFloat(fields[2], 64)
		}
	}
	calculateCPUStatus(&cpu)
	return cpu
}

func calculateCPUStatus(cpu *CPUInfo) {
	if cpu.Cores <= 0 {
		cpu.Cores = 1
	}
	// Lấy Load15 làm thước đo mức tải dài hạn ổn định nhất
	activeLoad := cpu.Load15
	if activeLoad == 0 {
		activeLoad = cpu.Load5
	}
	if activeLoad == 0 {
		activeLoad = cpu.Load1
	}

	cpu.LoadPerCore = activeLoad / float64(cpu.Cores)

	switch {
	case cpu.LoadPerCore < 0.5:
		cpu.Status = "Rất tốt (CPU rảnh rỗi)"
	case cpu.LoadPerCore < 0.8:
		cpu.Status = "Tốt (Mức tải tối ưu)"
	case cpu.LoadPerCore < 1.2:
		cpu.Status = "Đang tải (Nhiều tác vụ hoặc bot truy cập)"
	default:
		cpu.Status = "Quá tải (CPU nghẽn, phản hồi có thể chậm)"
	}
}

// GetCPUInfo đọc thông tin số core và load average của hệ thống
func GetCPUInfo() CPUInfo {
	cores := runtime.NumCPU()
	if cores <= 0 {
		cores = 1
	}

	data, err := os.ReadFile("/proc/loadavg")
	if err == nil {
		return ParseLoadAvg(string(data), cores)
	}

	out, err := exec.Command("uptime").CombinedOutput()
	if err == nil {
		return ParseUptime(string(out), cores)
	}

	cpu := CPUInfo{Cores: cores}
	calculateCPUStatus(&cpu)
	return cpu
}

// CalculateCapacity tổng hợp các chỉ số và tính toán năng lực chịu tải
func CalculateCapacity(mem MemInfo, swap SwapInfo, disk DiskInfo, cpu CPUInfo, siteCount int, avgSiteMemMB int) CapacityReport {
	if avgSiteMemMB <= 0 {
		avgSiteMemMB = 85
	}

	// Đệm an toàn giữ lại cho hệ thống OS và MariaDB tránh crash (15% RAM, tối thiểu 150MB)
	safeBufferMB := int(float64(mem.TotalMB) * 0.15)
	if safeBufferMB < 150 {
		safeBufferMB = 150
	}
	if safeBufferMB > 400 {
		safeBufferMB = 400
	}

	usableRAM := mem.AvailableMB - safeBufferMB
	if usableRAM < 0 {
		usableRAM = 0
	}

	estSafe := usableRAM / avgSiteMemMB

	// Tính thêm khả năng gánh tải khi có Swap
	usableSwap := int(float64(swap.FreeMB) * 0.6) // Chỉ dùng an toàn 60% Swap free
	estWithSwap := estSafe + (usableSwap / avgSiteMemMB)

	// Kiểm tra giới hạn bởi ổ cứng (mỗi site ước tính 400MB bao gồm backup)
	estByDisk := int((disk.AvailableGB * 1024.0) / 400.0)
	if estByDisk < estSafe {
		estSafe = estByDisk
	}
	if estByDisk < estWithSwap {
		estWithSwap = estByDisk
	}

	if estSafe < 0 {
		estSafe = 0
	}
	if estWithSwap < estSafe {
		estWithSwap = estSafe
	}

	// Đánh giá tình trạng hệ thống
	status := "Rất tốt (Hệ thống dư dả tài nguyên)"
	if mem.TotalMB > 0 {
		ratio := float64(mem.AvailableMB) / float64(mem.TotalMB)
		if ratio < 0.15 {
			status = "Cảnh báo (RAM sắp đầy, không nên thêm site)"
		} else if ratio < 0.30 {
			status = "Bình thường (Đang tải vừa phải)"
		}
	}

	// Cảnh báo nếu CPU quá tải
	if cpu.LoadPerCore >= 1.5 {
		status = "Cảnh báo (CPU đang quá tải, phản hồi có thể chậm)"
	}

	recommendation := fmt.Sprintf("VPS của bạn đang vận hành ổn định. Có thể cài thêm an toàn khoảng %d - %d website.", estSafe, estWithSwap)
	if estSafe == 0 {
		if swap.TotalMB == 0 {
			recommendation = "RAM khả dụng còn ít. Hãy tạo thêm 2GB - 4GB Swap RAM (Menu [12]) để có thể cài thêm website an toàn."
		} else {
			recommendation = "Tài nguyên máy chủ đang ở ngưỡng giới hạn. Nên cân nhắc nâng cấp gói VPS hoặc tách bớt site sang VPS mới."
		}
	}

	if cpu.LoadPerCore >= 1.5 {
		recommendation += fmt.Sprintf(" ⚠️ Lưu ý: CPU đang chịu tải cao (Load: %.2f / %d Core). Dù RAM còn chỗ cho thêm website, hãy kiểm tra traffic đột biến hoặc cấu hình cache trước khi cài thêm.", cpu.Load15, cpu.Cores)
	} else if cpu.LoadPerCore >= 1.0 {
		recommendation += fmt.Sprintf(" (Lưu ý: CPU đang xử lý khối lượng tác vụ tương đối cao: %.2f / %d Core).", cpu.Load15, cpu.Cores)
	}

	return CapacityReport{
		CPUCores:          cpu.Cores,
		CPULoad1:          cpu.Load1,
		CPULoad5:          cpu.Load5,
		CPULoad15:         cpu.Load15,
		CPULoadPerCore:    cpu.LoadPerCore,
		CPUStatus:         cpu.Status,
		TotalRAMMB:        mem.TotalMB,
		UsedRAMMB:         mem.UsedMB,
		AvailableRAMMB:    mem.AvailableMB,
		SwapTotalMB:       swap.TotalMB,
		SwapFreeMB:        swap.FreeMB,
		SwapUsedMB:        swap.UsedMB,
		DiskTotalGB:       disk.TotalGB,
		DiskAvailableGB:   disk.AvailableGB,
		RunningSitesCount: siteCount,
		AvgSiteRAMMB:      avgSiteMemMB,
		EstSafeSites:      estSafe,
		EstWithSwapSites:  estWithSwap,
		SystemStatus:      status,
		Recommendation:    recommendation,
	}
}

// GetVPSCapacity đo lường tài nguyên thực tế của VPS và trả về báo cáo
func GetVPSCapacity(systemDir string) (*CapacityReport, error) {
	// 1. Đọc thông tin CPU
	cpu := GetCPUInfo()

	// 2. Đọc bộ nhớ qua 'free -m'
	freeOut, err := exec.Command("free", "-m").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("đọc thông tin bộ nhớ: %w", err)
	}
	mem, swap := ParseFreeOutput(string(freeOut))

	// 3. Đọc dung lượng ổ cứng qua 'df -m'
	targetPath := systemDir
	if targetPath == "" {
		targetPath = "/"
	}
	dfOut, _ := exec.Command("df", "-m", targetPath).CombinedOutput()
	disk := ParseDfOutput(string(dfOut))

	// 4. Đọc số liệu RAM các container website qua 'docker stats'
	statsOut, _ := exec.Command("docker", "stats", "--no-stream", "--format", "{{.Name}}\t{{.MemUsage}}").CombinedOutput()
	siteCount, avgSiteMemMB, _ := ParseDockerStats(string(statsOut))

	report := CalculateCapacity(mem, swap, disk, cpu, siteCount, avgSiteMemMB)
	return &report, nil
}
