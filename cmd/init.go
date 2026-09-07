package cmd

import (
	"fmt"
	"os"
	"path/filepath"

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
		if initEmail == "" {
			return fmt.Errorf("vui lòng cung cấp email Let's Encrypt qua flag --email")
		}

		color.Cyan("=== Bắt đầu khởi tạo hệ thống ols-cli ===")

		cfg := config.DefaultConfig()
		cfg.ACMEEmail = initEmail

		rootPass, err := util.GenerateRandomString(32)
		if err != nil {
			return fmt.Errorf("sinh mật khẩu root mariadb: %w", err)
		}
		cfg.DBRootPassword = rootPass

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
			NetworkName:    cfg.NetworkName,
			DBRootPassword: cfg.DBRootPassword,
		})
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(coreDir, "docker-compose.yml"), []byte(coreCompose), 0644); err != nil {
			return err
		}

		// 5. Lưu config
		configPath := filepath.Join(cfg.SystemDir, "config", "ols.yaml")
		if err := config.SaveConfig(configPath, cfg); err != nil {
			return err
		}

		// 6. Khởi động Docker Network & Stack Core
		dm := docker.NewDockerManager()
		color.Yellow("-> Tạo mạng Docker %s...", cfg.NetworkName)
		if err := dm.EnsureNetwork(cfg.NetworkName); err != nil {
			return err
		}

		color.Yellow("-> Khởi chạy container Traefik và Shared MariaDB...")
		if err := dm.ComposeUp(coreDir); err != nil {
			return err
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
