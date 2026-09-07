package cmd

import (
	"fmt"
	"os/exec"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/config"
	"github.com/spf13/cobra"
)

var pmaPort int

var pmaCmd = &cobra.Command{
	Use:   "pma",
	Short: "Quản lý container phpMyAdmin (enable/disable)",
}

var pmaEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Khởi chạy phpMyAdmin để truy cập giao diện quản trị database",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		color.Cyan("-> Đang khởi chạy phpMyAdmin trên port %d...", pmaPort)
		runCmd := exec.Command("docker", "run", "-d",
			"--name", "ols-pma",
			"--restart", "always",
			"--network", cfg.NetworkName,
			"-p", fmt.Sprintf("%d:80", pmaPort),
			"-e", "PMA_HOST=ols-mariadb",
			"phpmyadmin:latest",
		)
		if out, err := runCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("không thể khởi chạy phpmyadmin: %s (%w)", string(out), err)
		}

		color.Green("✓ phpMyAdmin đã sẵn sàng tại http://<IP_VPS>:%d", pmaPort)
		return nil
	},
}

var pmaDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Dừng và tắt phpMyAdmin để tiết kiệm RAM",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = exec.Command("docker", "rm", "-f", "ols-pma").Run()
		color.Green("✓ Đã tắt phpMyAdmin thành công!")
		return nil
	},
}

func init() {
	pmaEnableCmd.Flags().IntVar(&pmaPort, "port", 8080, "Port truy cập phpMyAdmin")
	pmaCmd.AddCommand(pmaEnableCmd)
	pmaCmd.AddCommand(pmaDisableCmd)
	RootCmd.AddCommand(pmaCmd)
}
