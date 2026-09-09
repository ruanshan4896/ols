package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/site"
	"github.com/spf13/cobra"
)

var (
	sitePHP        string
	siteRedis      bool
	siteWP         bool
	siteForce      bool
	siteAdminUser  string
	siteAdminPass  string
	siteAdminEmail string
)

var siteCmd = &cobra.Command{
	Use:   "site",
	Short: "Quản lý vòng đời website (create, delete, list, restart)",
}

var siteCreateCmd = &cobra.Command{
	Use:   "create [domain]",
	Short: "Tạo một website WordPress mới với OpenLiteSpeed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return fmt.Errorf("chưa khởi tạo hệ thống (vui lòng chạy 'ols-cli init' trước): %w", err)
		}

		color.Cyan("-> Đang tạo website %s...", domain)
		mgr := site.NewManager(cfg)
		if err := mgr.CreateSite(site.CreateSiteOptions{
			Domain:        domain,
			PHPVersion:    sitePHP,
			WithRedis:     siteRedis,
			InstallWP:     siteWP,
			AdminUser:     siteAdminUser,
			AdminPassword: siteAdminPass,
			AdminEmail:    siteAdminEmail,
		}); err != nil {
			return err
		}

		color.Green("✓ Website %s đã được tạo và khởi chạy thành công!", domain)
		if siteAdminPass != "" {
			color.Green("  Đăng nhập quản trị: https://%s/wp-admin", domain)
			color.Green("  Tài khoản: %s | Mật khẩu: %s", siteAdminUser, siteAdminPass)
		}
		return nil
	},
}

var siteListCmd = &cobra.Command{
	Use:   "list",
	Short: "Liệt kê tất cả các website đang quản lý",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return fmt.Errorf("chưa khởi tạo hệ thống: %w", err)
		}

		mgr := site.NewManager(cfg)
		sites, err := mgr.ListSites()
		if err != nil {
			return err
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Domain", "Status"})
		for _, s := range sites {
			statusStr := s.Status
			if s.Status == "Running" {
				statusStr = color.GreenString("Running")
			} else {
				statusStr = color.RedString("Stopped")
			}
			table.Append([]string{s.Domain, statusStr})
		}
		table.Render()
		return nil
	},
}

var siteDeleteCmd = &cobra.Command{
	Use:   "delete [domain]",
	Short: "Xóa website, container và database tương ứng",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		mgr := site.NewManager(cfg)
		if err := mgr.DeleteSite(domain, siteForce); err != nil {
			return err
		}

		color.Green("✓ Đã xóa hoàn toàn website %s!", domain)
		return nil
	},
}

var siteRestartCmd = &cobra.Command{
	Use:   "restart [domain]",
	Short: "Khởi động lại container của website",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		mgr := site.NewManager(cfg)
		if err := mgr.RestartSite(domain); err != nil {
			return err
		}

		color.Green("✓ Đã khởi động lại website %s!", domain)
		return nil
	},
}

func init() {
	siteCreateCmd.Flags().StringVar(&sitePHP, "php", "8.2", "Phiên bản PHP (8.1, 8.2, 8.3)")
	siteCreateCmd.Flags().BoolVar(&siteRedis, "redis", true, "Kích hoạt Redis Object Cache riêng")
	siteCreateCmd.Flags().BoolVar(&siteWP, "wp", true, "Tự động tải và cấu hình WordPress")
	siteCreateCmd.Flags().StringVar(&siteAdminUser, "admin-user", "admin", "Tên tài khoản quản trị wp-admin")
	siteCreateCmd.Flags().StringVar(&siteAdminPass, "admin-pass", "", "Mật khẩu quản trị wp-admin (tự động cài đặt hoàn chỉnh)")
	siteCreateCmd.Flags().StringVar(&siteAdminEmail, "admin-email", "", "Email quản trị viên")
	siteDeleteCmd.Flags().BoolVar(&siteForce, "force", false, "Xóa không cần hỏi lại")

	siteCmd.AddCommand(siteCreateCmd)
	siteCmd.AddCommand(siteListCmd)
	siteCmd.AddCommand(siteDeleteCmd)
	siteCmd.AddCommand(siteRestartCmd)
	RootCmd.AddCommand(siteCmd)
}
