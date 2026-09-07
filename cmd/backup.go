package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/backup"
	"github.com/ols-cli/ols/internal/config"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup [domain]",
	Short: "Sao lưu toàn bộ website (mã nguồn và database) thành file nén .tar.gz",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		bm := backup.NewBackupManager(cfg)
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
	RootCmd.AddCommand(backupCmd)
	RootCmd.AddCommand(restoreCmd)
}
