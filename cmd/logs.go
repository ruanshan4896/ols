package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/site"
	"github.com/ols-cli/ols/internal/system"
	"github.com/spf13/cobra"
)

var (
	logsLines   int
	logsScan    bool
	logsClear   bool
	logsAccess  bool
	logsError   bool
	logsTopIP   bool
	logsTopURL  bool
	logsFollow  bool
)

var logsCmd = &cobra.Command{
	Use:   "logs [target]",
	Short: "Xem nhật ký và quét lỗi hệ thống (traefik, mariadb, redis, pma, hoặc domain website)",
	Example: `  ols logs traefik
  ols logs mariadb -n 100
  ols logs pma
  ols logs example.com --access
  ols logs example.com --error
  ols logs example.com --top-ip
  ols logs example.com -f
  ols logs --scan
  ols logs --clear`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if logsClear || (len(args) > 0 && (args[0] == "clear" || args[0] == "clean")) {
			return RunClearAllLogs()
		}

		if logsScan || (len(args) > 0 && args[0] == "scan") {
			return RunQuickErrorScan(logsLines)
		}

		if len(args) == 0 {
			color.Yellow("Vui lòng chỉ định dịch vụ hoặc domain cần xem log (Ví dụ: traefik, mariadb, redis, pma, <domain> hoặc --scan)")
			return nil
		}

		target := strings.ToLower(strings.TrimSpace(args[0]))
		switch target {
		case "traefik":
			return PrintContainerLog("ols-traefik", "Traefik SSL/Proxy", logsLines, logsFollow)
		case "mariadb", "mysql", "db":
			return PrintContainerLog("ols-mariadb", "MariaDB Database", logsLines, logsFollow)
		case "redis", "cache":
			return PrintContainerLog("ols-redis", "Redis Cache", logsLines, logsFollow)
		case "pma", "phpmyadmin":
			return PrintContainerLog("ols-pma", "phpMyAdmin", logsLines, logsFollow)
		default:
			// Xem log website theo domain
			if logsTopIP {
				color.Cyan("-> Đang phân tích các IP truy cập nhiều nhất cho %s...", target)
				stats, err := system.GetSiteTopIPs("/opt/ols", target, 15)
				if err != nil {
					return err
				}
				PrintTopIPReport(target, stats)
				return nil
			}

			if logsTopURL {
				color.Cyan("-> Đang phân tích các URL bị gọi nhiều nhất cho %s...", target)
				stats, err := system.GetSiteTopURLs("/opt/ols", target, 15)
				if err != nil {
					return err
				}
				PrintTopURLReport(target, stats)
				return nil
			}

			if logsFollow {
				return FollowSiteLog(target, logsAccess)
			}

			if logsAccess {
				color.Cyan("-> Đang lấy Access Log website %s (%d dòng gần nhất)...", target, logsLines)
				out, err := system.GetSiteAccessLog("/opt/ols", target, logsLines)
				if err != nil {
					return err
				}
				fmt.Println(out)
				return nil
			}

			if logsError {
				color.Cyan("-> Đang lấy Error Log website %s (%d dòng gần nhất)...", target, logsLines)
				out, err := system.GetSiteErrorLog("/opt/ols", target, logsLines)
				if err != nil {
					return err
				}
				fmt.Println(out)
				return nil
			}

			// Mặc định: xem tổng quan cả container và error log
			color.Cyan("-> Đang lấy nhật ký website %s (%d dòng gần nhất)...", target, logsLines)
			out, err := system.GetSiteLogs(target, logsLines)
			if err != nil {
				return err
			}
			fmt.Println(out)
			return nil
		}
	},
}

func PrintContainerLog(containerName, title string, lines int, follow bool) error {
	color.Cyan("-> Đang lấy nhật ký %s (%d dòng gần nhất)...", title, lines)
	if follow {
		color.Yellow("(Đang theo dõi thời gian thực. Bấm Ctrl+C để dừng...)")
		cmd := exec.Command("docker", "logs", "-f", "--tail", strconv.Itoa(lines), containerName)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	out, err := system.GetContainerLogs(containerName, lines)
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) == "" {
		color.Yellow("Không có dữ liệu log mới từ %s.", title)
		return nil
	}
	fmt.Println(out)
	return nil
}

func FollowSiteLog(domain string, isAccess bool) error {
	logFileName := "error.log"
	if isAccess {
		logFileName = "access.log"
	}
	hostPath := filepath.Join("/opt/ols", "sites", domain, "logs", logFileName)
	if _, err := os.Stat(hostPath); err == nil {
		color.Yellow("(Đang theo dõi %s thời gian thực trên VPS. Bấm Ctrl+C để dừng...)", logFileName)
		cmd := exec.Command("tail", "-f", "-n", "30", hostPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	slug := site.DomainToSlug(domain)
	containerName := fmt.Sprintf("ols_%s", slug)
	containerLogPath := fmt.Sprintf("/usr/local/lsws/Example/logs/%s", logFileName)
	color.Yellow("(Đang theo dõi %s thời gian thực qua container. Bấm Ctrl+C để dừng...)", logFileName)
	cmd := exec.Command("docker", "exec", "-it", containerName, "tail", "-f", "-n", "30", containerLogPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func PrintTopIPReport(domain string, stats []system.IPStat) {
	color.New(color.FgCyan, color.Bold).Println("\n==================================================================")
	color.New(color.FgHiGreen, color.Bold).Printf("     TOP ĐỊA CHỈ IP TRUY CẬP NHIỀU NHẤT CHO: %s\n", strings.ToUpper(domain))
	color.New(color.FgCyan, color.Bold).Println("==================================================================")
	if len(stats) == 0 {
		color.Yellow("Chưa ghi nhận dữ liệu IP nào.")
		return
	}
	fmt.Printf("%-5s %-25s %-15s\n", "HẠNG", "ĐỊA CHỈ IP", "SỐ REQUEST")
	color.New(color.FgHiBlack).Println(strings.Repeat("-", 50))
	for i, s := range stats {
		fmt.Printf("#%-4d %-25s %-15d\n", i+1, s.IP, s.Count)
	}
	fmt.Println()
}

func PrintTopURLReport(domain string, stats []system.URLStat) {
	color.New(color.FgCyan, color.Bold).Println("\n==================================================================")
	color.New(color.FgHiGreen, color.Bold).Printf("     TOP ĐƯỜNG DẪN URL BỊ TRUY CẬP NHIỀU NHẤT: %s\n", strings.ToUpper(domain))
	color.New(color.FgCyan, color.Bold).Println("==================================================================")
	if len(stats) == 0 {
		color.Yellow("Chưa ghi nhận dữ liệu URL nào.")
		return
	}
	fmt.Printf("%-5s %-45s %-15s\n", "HẠNG", "ĐƯỜNG DẪN URL", "SỐ LƯỢT GỌI")
	color.New(color.FgHiBlack).Println(strings.Repeat("-", 70))
	for i, s := range stats {
		fmt.Printf("#%-4d %-45s %-15d\n", i+1, s.URL, s.Count)
	}
	fmt.Println()
}

func RunQuickErrorScan(lines int) error {
	color.Cyan("-> Đang quét các thông báo lỗi (ERROR/FATAL/PANIC/PHP) trên toàn hệ thống...")
	entries, err := system.ScanAllErrors(lines)
	if err != nil {
		return fmt.Errorf("lỗi khi quét hệ thống: %w", err)
	}

	color.New(color.FgCyan, color.Bold).Println("\n==================================================================")
	color.New(color.FgHiGreen, color.Bold).Println("           KẾT QUẢ QUÉT LỖI TOÀN DIỆN TOÀN HỆ THỐNG")
	color.New(color.FgCyan, color.Bold).Println("==================================================================")

	if len(entries) == 0 {
		color.Green("✓ Tuyệt vời! Không phát hiện lỗi nghiêm trọng nào gần đây trên toàn hệ thống.\n")
		return nil
	}

	color.Yellow("[!] Phát hiện %d dòng cảnh báo/lỗi cần lưu ý:\n", len(entries))
	for _, e := range entries {
		sourceColor := color.New(color.FgHiMagenta, color.Bold).Sprintf("[%s]", e.Source)
		fmt.Printf("  %s %s\n", sourceColor, e.Message)
	}
	fmt.Println()
	color.Yellow("Gợi ý: Nếu website gặp sự cố, bạn chỉ cần copy các dòng lỗi trên gửi cho kỹ thuật để được hỗ trợ xử lý ngay.\n")
	return nil
}

func RunClearAllLogs() error {
	color.Cyan("-> Đang tiến hành dọn sạch toàn bộ log cũ trên hệ thống...")
	count, bytesFreed, err := system.ClearAllLogs()
	if err != nil {
		return fmt.Errorf("lỗi khi xóa log: %w", err)
	}

	freedMB := float64(bytesFreed) / (1024 * 1024)
	color.Green("✓ Đã xóa sạch toàn bộ nhật ký của %d nguồn log!", count)
	if freedMB > 0 {
		color.Green("  (Giải phóng thành công %.2f MB dung lượng ổ cứng)", freedMB)
	} else {
		color.Green("  (Tất cả file log đã được reset về 0 byte)")
	}
	return nil
}

func init() {
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "Số dòng log cần xem (mặc định 50)")
	logsCmd.Flags().BoolVarP(&logsScan, "scan", "s", false, "Quét nhanh các dòng lỗi trên toàn hệ thống")
	logsCmd.Flags().BoolVarP(&logsClear, "clear", "c", false, "Xóa sạch toàn bộ log cũ của các container và website")
	logsCmd.Flags().BoolVar(&logsAccess, "access", false, "Chỉ xem nhật ký truy cập (access.log) của website")
	logsCmd.Flags().BoolVar(&logsError, "error", false, "Chỉ xem nhật ký lỗi (error.log) của website")
	logsCmd.Flags().BoolVar(&logsTopIP, "top-ip", false, "Thống kê Top 15 địa chỉ IP gửi request nhiều nhất")
	logsCmd.Flags().BoolVar(&logsTopURL, "top-url", false, "Thống kê Top 15 đường dẫn URL bị gọi nhiều nhất")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "Theo dõi nhật ký liên tục theo thời gian thực (live stream)")
	RootCmd.AddCommand(logsCmd)
}
