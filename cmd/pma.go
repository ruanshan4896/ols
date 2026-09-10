package cmd

import (
	"fmt"
	"os/exec"
	"strings"

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

		// Xóa container cũ nếu còn tồn tại
		_ = exec.Command("docker", "rm", "-f", "ols-pma").Run()

		// Tự động nhận diện mạng Docker mà container ols-mariadb đang tham gia
		targetNetwork := cfg.GetBackendNetwork()
		netOut, err := exec.Command("docker", "inspect", "ols-mariadb", "--format", "{{range $k, $v := .NetworkSettings.Networks}}{{$k}} {{end}}").Output()
		if err == nil {
			nets := strings.Fields(string(netOut))
			if len(nets) > 0 {
				targetNetwork = nets[0]
			}
		}

		color.Cyan("-> Đang khởi chạy phpMyAdmin trên port %d (mạng: %s)...", pmaPort, targetNetwork)
		runCmd := exec.Command("docker", "run", "-d",
			"--name", "ols-pma",
			"--restart", "always",
			"--network", targetNetwork,
			"-p", fmt.Sprintf("%d:80", pmaPort),
			"--memory", "256m",
			"--cpus", "1.0",
			"-e", "PMA_HOST=ols-mariadb",
			"phpmyadmin:latest",
		)
		if out, err := runCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("không thể khởi chạy phpmyadmin: %s (%w)", string(out), err)
		}

		// Nối thêm vào cả 2 mạng Frontend và Backend (nếu có) để đảm bảo thông suốt 100%
		_ = exec.Command("docker", "network", "connect", cfg.GetBackendNetwork(), "ols-pma").Run()
		_ = exec.Command("docker", "network", "connect", cfg.GetFrontendNetwork(), "ols-pma").Run()

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
