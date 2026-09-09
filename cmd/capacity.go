package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/system"
	"github.com/spf13/cobra"
)

var capacityCmd = &cobra.Command{
	Use:   "capacity",
	Short: "Đánh giá tài nguyên máy chủ và ước tính số website có thể cài thêm",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		systemDir := "/opt/ols"
		if err == nil && cfg.SystemDir != "" {
			systemDir = cfg.SystemDir
		}

		color.Cyan("-> Đang kiểm tra và đo lường tài nguyên thực tế của VPS...")
		report, err := system.GetVPSCapacity(systemDir)
		if err != nil {
			return fmt.Errorf("không thể đo lường tài nguyên VPS: %w", err)
		}

		PrintCapacityReport(report)
		return nil
	},
}

func PrintCapacityReport(report *system.CapacityReport) {
	color.New(color.FgCyan, color.Bold).Println("\n==================================================================")
	color.New(color.FgHiGreen, color.Bold).Println("           ĐÁNH GIÁ NĂNG LỰC & KHẢ NĂNG CHỊU TẢI VPS")
	color.New(color.FgCyan, color.Bold).Println("==================================================================")

	color.New(color.FgWhite, color.Bold).Println("  📊 TÀI NGUYÊN HỆ THỐNG:")
	fmt.Printf("  - Vi xử lý (CPU)      : %d vCore (Load Avg 1m/5m/15m: %.2f, %.2f, %.2f | %s)\n", report.CPUCores, report.CPULoad1, report.CPULoad5, report.CPULoad15, report.CPUStatus)
	fmt.Printf("  - Tổng RAM vật lý     : %d MB (Đang dùng: %d MB | Khả dụng: %d MB)\n", report.TotalRAMMB, report.UsedRAMMB, report.AvailableRAMMB)
	if report.SwapTotalMB > 0 {
		fmt.Printf("  - Bộ nhớ Swap RAM     : %d MB (Đang dùng: %d MB | Còn trống: %d MB)\n", report.SwapTotalMB, report.SwapUsedMB, report.SwapFreeMB)
	} else {
		fmt.Printf("  - Bộ nhớ Swap RAM     : Chưa kích hoạt (Khuyên dùng: tạo 2GB - 4GB Swap qua Menu [12])\n")
	}
	fmt.Printf("  - Ổ cứng khả dụng     : %.1f GB trống (Tổng: %.1f GB)\n", report.DiskAvailableGB, report.DiskTotalGB)

	fmt.Println()
	color.New(color.FgWhite, color.Bold).Println("  🌐 HIỆN TRẠNG WEBSITE TRÊN VPS:")
	fmt.Printf("  - Số website đang chạy : %d website\n", report.RunningSitesCount)
	fmt.Printf("  - Mức RAM trung bình   : ~%d MB / website\n", report.AvgSiteRAMMB)
	fmt.Printf("  - Trạng thái tải       : %s\n", report.SystemStatus)

	fmt.Println()
	color.New(color.FgHiYellow, color.Bold).Println("  🎯 DỰ BÁO KHẢ NĂNG CÀI ĐẶT THÊM:")
	color.Green("  👉 Có thể cài thêm an toàn   : ~%d website\n", report.EstSafeSites)
	if report.SwapTotalMB > 0 {
		color.Green("  👉 Mức cao nhất khi kèm Swap : ~%d website\n", report.EstWithSwapSites)
	}

	color.New(color.FgCyan, color.Bold).Println("==================================================================")
	color.Yellow("💡 Khuyến nghị: %s\n", report.Recommendation)
}

func init() {
	RootCmd.AddCommand(capacityCmd)
}
