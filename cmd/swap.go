package cmd

import (
	"fmt"
	"strconv"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/system"
	"github.com/spf13/cobra"
)

var (
	swapSizeGB int
	swapStatus bool
	swapOff    bool
)

var swapCmd = &cobra.Command{
	Use:   "swap",
	Short: "Quản lý bộ nhớ Swap RAM chống sập máy chủ VPS",
	RunE: func(cmd *cobra.Command, args []string) error {
		if swapStatus {
			info, err := system.GetSwapInfo()
			if err != nil {
				return err
			}
			color.Cyan("=== THÔNG TIN BỘ NHỚ SWAP TRÊN VPS ===")
			fmt.Printf("  - Tổng dung lượng Swap : %d MB (%.1f GB)\n", info.TotalMB, float64(info.TotalMB)/1024.0)
			fmt.Printf("  - Đang sử dụng         : %d MB\n", info.UsedMB)
			fmt.Printf("  - Còn trống            : %d MB\n", info.FreeMB)
			if info.Path != "" {
				fmt.Printf("  - Tệp tin Swap         : %s\n", info.Path)
			}
			return nil
		}

		if swapOff {
			color.Yellow("-> Đang tắt và xóa file Swap...")
			if err := system.DisableSwap(); err != nil {
				return err
			}
			color.Green("✓ Đã tắt và xóa hoàn toàn Swap trên VPS!")
			return nil
		}

		if len(args) > 0 {
			if s, err := strconv.Atoi(args[0]); err == nil && s > 0 {
				swapSizeGB = s
			}
		}

		if swapSizeGB <= 0 {
			swapSizeGB = 2 // Mặc định 2GB
		}

		color.Cyan("-> Đang tiến hành tạo bộ nhớ Swap RAM %d GB (tối ưu swappiness=10)...", swapSizeGB)
		if err := system.CreateSwap(swapSizeGB); err != nil {
			return fmt.Errorf("tạo swap thất bại: %w", err)
		}

		color.Green("✓ Đã khởi tạo thành công %d GB Swap RAM chống quá tải cho VPS!", swapSizeGB)
		color.Yellow("  (Tự động kích hoạt lại khi khởi động VPS qua /etc/fstab & tối ưu vm.swappiness=10)")
		return nil
	},
}

func init() {
	swapCmd.Flags().IntVarP(&swapSizeGB, "size", "s", 2, "Dung lượng Swap tính bằng GB (ví dụ: 2, 4, 8)")
	swapCmd.Flags().BoolVar(&swapStatus, "status", false, "Kiểm tra thông tin bộ nhớ Swap hiện tại")
	swapCmd.Flags().BoolVar(&swapOff, "off", false, "Tắt và xóa bộ nhớ Swap")
	RootCmd.AddCommand(swapCmd)
}
