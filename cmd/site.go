package cmd

import (
	"fmt"
	"os"
	"strings"

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
	siteShowPass   bool
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
	Short: "Liệt kê tất cả các website và thông tin Database",
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

		if len(sites) == 0 {
			color.Yellow("Chưa có website nào trên hệ thống.")
			return nil
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"STT", "Domain", "Status", "PHP", "Database", "DB User", "DB Password"})
		for i, s := range sites {
			statusStr := s.Status
			if s.Status == "Running" {
				statusStr = color.GreenString("Running")
			} else {
				statusStr = color.RedString("Stopped")
			}
			table.Append([]string{
				fmt.Sprintf("%d", i+1),
				s.Domain,
				statusStr,
				s.PHPVersion,
				s.DBName,
				s.DBUser,
				s.DBPassword,
			})
		}
		table.Render()
		return nil
	},
}

var siteInfoCmd = &cobra.Command{
	Use:   "info [domain]",
	Short: "Xem toàn bộ thông tin chi tiết cấu hình và Database của website",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		mgr := site.NewManager(cfg)
		info, err := mgr.GetSiteInfo(domain)
		if err != nil {
			return err
		}

		PrintSiteDetailedInfo(cfg.SystemDir, info)
		return nil
	},
}

func PrintSiteDetailedInfo(systemDir string, s *site.SiteInfo) {
	color.New(color.FgCyan, color.Bold).Println("\n==================================================================")
	color.New(color.FgHiGreen, color.Bold).Printf("          THÔNG TIN CHI TIẾT WEBSITE: %s\n", strings.ToUpper(s.Domain))
	color.New(color.FgCyan, color.Bold).Println("==================================================================")
	fmt.Printf("  - Tên miền         : %s (https://%s)\n", s.Domain, s.Domain)
	statusText := color.GreenString("Đang chạy (Running)")
	if s.Status != "Running" {
		statusText = color.RedString("Đã dừng (Stopped)")
	}
	fmt.Printf("  - Trạng thái       : %s\n", statusText)
	fmt.Printf("  - Phiên bản PHP    : PHP %s (OpenLiteSpeed)\n", s.PHPVersion)
	fmt.Printf("  - Thư mục mã nguồn : %s/sites/%s/html\n", systemDir, s.Domain)
	fmt.Printf("  - Thư mục cấu hình : %s/sites/%s/ols/conf\n", systemDir, s.Domain)
	fmt.Println("  --------------------------------------------------------------")
	color.New(color.FgHiYellow, color.Bold).Println("  THÔNG TIN CƠ SỞ DỮ LIỆU (MARIADB):")
	fmt.Printf("  - Tên Database     : %s\n", s.DBName)
	fmt.Printf("  - Tài khoản DB     : %s\n", s.DBUser)
	fmt.Printf("  - Mật khẩu DB      : %s\n", s.DBPassword)
	fmt.Printf("  - Máy chủ DB (Host): %s (Port nội bộ 3306)\n", s.DBHost)
	fmt.Println("  - Quản trị web     : http://<IP_VPS>:8080 (Bật bằng Menu [8])")
	fmt.Println("  --------------------------------------------------------------")
	color.New(color.FgHiMagenta, color.Bold).Println("  THÔNG TIN QUẢN TRỊ WORDPRESS:")
	fmt.Printf("  - Link đăng nhập   : https://%s/wp-login.php\n", s.Domain)
	fmt.Println("  - Đổi pass admin   : Sử dụng Menu [11] để đổi mật khẩu admin bất cứ lúc nào")
	color.New(color.FgCyan, color.Bold).Println("==================================================================")
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
	siteCmd.AddCommand(siteInfoCmd)
	siteCmd.AddCommand(siteDeleteCmd)
	siteCmd.AddCommand(siteRestartCmd)
	RootCmd.AddCommand(siteCmd)
}
