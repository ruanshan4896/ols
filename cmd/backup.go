package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/backup"
	"github.com/ols-cli/ols/internal/config"
	"github.com/spf13/cobra"
)

var backupAll bool

var backupCmd = &cobra.Command{
	Use:   "backup [domain]",
	Short: "Sao lưu website (mã nguồn và database) thành file nén .tar.gz",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		bm := backup.NewBackupManager(cfg)

		if backupAll || len(args) == 0 {
			if !backupAll && len(args) == 0 {
				return fmt.Errorf("vui lòng chỉ định tên miền cần sao lưu hoặc sử dụng cờ --all")
			}
			color.Cyan("-> Đang tiến hành sao lưu toàn bộ website...")
			backedUp, errs := bm.BackupAllSitesProgress(func(current, total int, domain, backupPath string, err error) {
				if err != nil {
					color.Red("  [%d/%d] Sao lưu %s: ✗ Thất bại: %v", current, total, domain, err)
				} else {
					color.Green("  [%d/%d] Sao lưu %s: ✓ Hoàn tất (%s)", current, total, domain, backupPath)
				}
			})
			if len(backedUp) == 0 && len(errs) == 0 {
				color.Yellow("Hiện chưa có website nào trên hệ thống.")
			} else {
				color.Cyan("=== Hoàn tất sao lưu: Thành công: %d | Thất bại: %d ===", len(backedUp), len(errs))
			}
			return nil
		}

		domain := args[0]
		color.Cyan("-> Đang sao lưu website %s...", domain)
		path, err := bm.BackupSite(domain)
		if err != nil {
			return fmt.Errorf("sao lưu thất bại: %w", err)
		}

		color.Green("✓ Sao lưu thành công! File lưu trữ: %s", path)
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore [domain] [backup_file.tar.gz]",
	Short: "Khôi phục website từ bản sao lưu",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		backupFile := args[1]
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		bm := backup.NewBackupManager(cfg)
		color.Cyan("-> Đang khôi phục website %s từ %s...", domain, backupFile)
		if err := bm.RestoreSite(domain, backupFile); err != nil {
			return fmt.Errorf("khôi phục thất bại: %w", err)
		}

		color.Green("✓ Khôi phục thành công website %s!", domain)
		return nil
	},
}

func init() {
	backupCmd.Flags().BoolVarP(&backupAll, "all", "a", false, "Sao lưu tất cả các website trên hệ thống")
	RootCmd.AddCommand(backupCmd)
	RootCmd.AddCommand(restoreCmd)
}
