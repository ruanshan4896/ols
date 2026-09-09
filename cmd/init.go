package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/docker"
	"github.com/ols-cli/ols/internal/template"
	"github.com/ols-cli/ols/internal/util"
	"github.com/spf13/cobra"
)

var initEmail string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Khởi tạo hệ thống máy chủ: Traefik, MariaDB và Docker network",
	RunE: func(cmd *cobra.Command, args []string) error {
		cleanEmail, err := util.ValidateAndSanitizeEmail(initEmail)
		if err != nil {
			return fmt.Errorf("email Let's Encrypt không hợp lệ: %w", err)
		}
		if cleanEmail != initEmail {
			color.Yellow("-> Đã tự động làm sạch ký tự tiếng Việt / non-ASCII trong email: %s -> %s", initEmail, cleanEmail)
		}

		color.Cyan("=== Bắt đầu khởi tạo hệ thống ols-cli ===")

		cfg := config.DefaultConfig()
		cfg.ACMEEmail = cleanEmail

		configPath := filepath.Join(cfg.SystemDir, "config", "ols.yaml")
		existingCfg, err := config.LoadConfig(configPath)
		if err == nil && existingCfg.DBRootPassword != "" {
			cfg.DBRootPassword = existingCfg.DBRootPassword
		} else {
			rootPass, err := util.GenerateRandomString(32)
			if err != nil {
				return fmt.Errorf("sinh mật khẩu root mariadb: %w", err)
			}
			cfg.DBRootPassword = rootPass
		}

		// 1. Tạo các thư mục
		coreDir := filepath.Join(cfg.SystemDir, "core")
		dirs := []string{
			filepath.Join(cfg.SystemDir, "bin"),
			filepath.Join(cfg.SystemDir, "config"),
			filepath.Join(cfg.SystemDir, "sites"),
			filepath.Join(cfg.SystemDir, "backups"),
			coreDir,
			filepath.Join(coreDir, "traefik"),
			filepath.Join(coreDir, "mariadb", "conf.d"),
			filepath.Join(coreDir, "mariadb", "data"),
		}
		for _, d := range dirs {
			if err := os.MkdirAll(d, 0755); err != nil {
				return fmt.Errorf("tạo thư mục %s: %w", d, err)
			}
		}

		// 2. Tạo file acme.json với quyền 600
		acmePath := filepath.Join(coreDir, "traefik", "acme.json")
		if _, err := os.Stat(acmePath); os.IsNotExist(err) {
			if err := os.WriteFile(acmePath, []byte("{}"), 0600); err != nil {
				return fmt.Errorf("tạo acme.json: %w", err)
			}
		}

		// 3. Render traefik.yml
		traefikYaml, err := template.RenderTraefikConfig(cfg.ACMEEmail)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(coreDir, "traefik", "traefik.yml"), []byte(traefikYaml), 0644); err != nil {
			return err
		}

		// 4. Render core docker-compose.yml
		coreCompose, err := template.RenderCoreCompose(template.CoreTemplateData{
			NetworkName:        cfg.GetFrontendNetwork(),
			BackendNetworkName: cfg.GetBackendNetwork(),
			DBRootPassword:     cfg.DBRootPassword,
		})
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(coreDir, "docker-compose.yml"), []byte(coreCompose), 0644); err != nil {
			return err
		}

		// 5. Lưu config
		if err := config.SaveConfig(configPath, cfg); err != nil {
			return err
		}

		// 6. Khởi động Docker Network (Dual-Network: Frontend & Backend) & Stack Core
		dm := docker.NewDockerManager()
		color.Yellow("-> Tạo mạng Docker Frontend %s...", cfg.GetFrontendNetwork())
		if err := dm.EnsureNetwork(cfg.GetFrontendNetwork()); err != nil {
			return err
		}
		color.Yellow("-> Tạo mạng Docker Backend bảo mật %s...", cfg.GetBackendNetwork())
		if err := dm.EnsureNetwork(cfg.GetBackendNetwork()); err != nil {
			return err
		}

		color.Yellow("-> Khởi chạy container Traefik và Shared MariaDB...")
		if err := dm.ComposeUp(coreDir); err != nil {
			return err
		}

		// 7. Chờ MariaDB khởi động và sẵn sàng nhận kết nối
		color.Yellow("-> Đang chờ MariaDB khởi động và sẵn sàng nhận kết nối...")
		for i := 0; i < 30; i++ {
			pingCmd := exec.Command("docker", "exec", "ols-mariadb", "mariadb-admin", "ping", "-uroot", "-p"+cfg.DBRootPassword, "--silent")
			if err := pingCmd.Run(); err == nil {
				break
			}
			time.Sleep(1 * time.Second)
		}

		color.Green("✓ Hệ thống đã khởi tạo thành công!")
		color.Green("  - Thư mục quản trị: %s", cfg.SystemDir)
		color.Green("  - Traefik Reverse Proxy đang lắng nghe Port 80, 443")
		color.Green("  - MariaDB Root Password đã được lưu an toàn tại: %s", configPath)

		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initEmail, "email", "", "Email quản trị viên dùng đăng ký SSL Let's Encrypt")
	RootCmd.AddCommand(initCmd)
}
