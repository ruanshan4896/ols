package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/system"
	"github.com/spf13/cobra"
)

var (
	logsLines int
	logsScan  bool
)

var logsCmd = &cobra.Command{
	Use:   "logs [target]",
	Short: "Xem nhật ký và quét lỗi hệ thống (traefik, mariadb, redis, hoặc domain website)",
	Example: `  ols logs traefik
  ols logs mariadb -n 100
  ols logs example.com
  ols logs --scan`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if logsScan || (len(args) > 0 && args[0] == "scan") {
			return RunQuickErrorScan(logsLines)
		}

		if len(args) == 0 {
			color.Yellow("Vui lòng chỉ định dịch vụ hoặc domain cần xem log (Ví dụ: traefik, mariadb, redis, <domain> hoặc --scan)")
			return nil
		}

		target := strings.ToLower(strings.TrimSpace(args[0]))
		switch target {
		case "traefik":
			return PrintContainerLog("ols-traefik", "Traefik SSL/Proxy", logsLines)
		case "mariadb", "mysql", "db":
			return PrintContainerLog("ols-mariadb", "MariaDB Database", logsLines)
		case "redis", "cache":
			return PrintContainerLog("ols-redis", "Redis Cache", logsLines)
		default:
			// Xem log website theo domain
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

func PrintContainerLog(containerName, title string, lines int) error {
	color.Cyan("-> Đang lấy nhật ký %s (%d dòng gần nhất)...", title, lines)
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

func RunQuickErrorScan(lines int) error {
	color.Cyan("-> Đang quét các thông báo lỗi (ERROR/FATAL/PANIC) trên toàn hệ thống...")
	entries, err := system.ScanAllErrors(lines)
	if err != nil {
		return fmt.Errorf("lỗi khi quét hệ thống: %w", err)
	}

	color.New(color.FgCyan, color.Bold).Println("\n==================================================================")
	color.New(color.FgHiGreen, color.Bold).Println("           KẾT QUẢ QUÉT LỖI NHANH TOÀN HỆ THỐNG")
	color.New(color.FgCyan, color.Bold).Println("==================================================================")

	if len(entries) == 0 {
		color.Green("✓ Tuyệt vời! Không phát hiện lỗi nghiêm trọng nào gần đây trên toàn hệ thống.\n")
		return nil
	}

	color.Yellow("⚠️ Phát hiện %d dòng cảnh báo/lỗi cần lưu ý:\n", len(entries))
	for _, e := range entries {
		sourceColor := color.New(color.FgHiMagenta, color.Bold).Sprintf("[%s]", e.Source)
		fmt.Printf("  %s %s\n", sourceColor, e.Message)
	}
	fmt.Println()
	color.Yellow("💡 Gợi ý: Nếu website gặp sự cố, bạn chỉ cần copy các dòng lỗi trên gửi cho kỹ thuật để được hỗ trợ xử lý ngay.\n")
	return nil
}

func init() {
	logsCmd.Flags().IntVarP(&logsLines, "lines", "n", 50, "Số dòng log cần xem (mặc định 50)")
	logsCmd.Flags().BoolVarP(&logsScan, "scan", "s", false, "Quét nhanh các dòng lỗi trên toàn hệ thống")
	RootCmd.AddCommand(logsCmd)
}
