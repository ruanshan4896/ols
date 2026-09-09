package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/site"
	"github.com/spf13/cobra"
)

var (
	secAdminUser string
	secAdminPass string
	secSaltsOnly bool
	secPassOnly  bool
	secAllSites  bool
)

var securityCmd = &cobra.Command{
	Use:   "security [domain]",
	Short: "Bảo mật website: Làm mới Salts/Keys và đổi mật khẩu quản trị wp-admin",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return fmt.Errorf("chưa khởi tạo hệ thống: %w", err)
		}

		mgr := site.NewManager(cfg)

		if secAllSites {
			sitesList, err := mgr.ListSites()
			if err != nil {
				return err
			}
			if len(sitesList) == 0 {
				color.Yellow("Hiện chưa có website nào trên hệ thống.")
				return nil
			}

			color.Cyan("-> Đang thực hiện bảo mật cho toàn bộ %d website...", len(sitesList))
			for i, s := range sitesList {
				color.Yellow("\n[%d/%d] Website: %s", i+1, len(sitesList), s.Domain)
				if !secPassOnly {
					if _, err := mgr.RegenerateSalts(s.Domain); err != nil {
						color.Red("  Làm mới Salts thất bại: %v", err)
					} else {
						color.Green("  ✓ Đã làm mới 8 Salts/Keys bảo mật từ WordPress.org API")
					}
				}
				if !secSaltsOnly {
					info, err := mgr.ResetAdminPassword(s.Domain, secAdminUser, secAdminPass)
					if err != nil {
						color.Red("  Đổi mật khẩu admin thất bại: %v", err)
					} else {
						color.Green("  ✓ %s", info)
					}
				}
			}
			color.Cyan("\n=== Hoàn tất bảo mật cho toàn bộ website ===")
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("vui lòng chỉ định tên miền cần bảo mật hoặc sử dụng cờ --all")
		}

		domain := args[0]
		color.Cyan("-> Đang thực hiện bảo mật cho website %s...", domain)

		if !secPassOnly {
			color.Cyan("-> Đang lấy Salts/Keys mới từ WordPress.org API...")
			if _, err := mgr.RegenerateSalts(domain); err != nil {
				color.Red("Làm mới Salts thất bại: %v", err)
			} else {
				color.Green("✓ Đã làm mới thành công 8 Authentication Salts & Keys vào wp-config.php!")
				color.Yellow("  (Toàn bộ cookie và phiên đăng nhập cũ trên mọi thiết bị đã bị đăng xuất)")
			}
		}

		if !secSaltsOnly {
			color.Cyan("-> Đang đặt lại mật khẩu quản trị WordPress...")
			info, err := mgr.ResetAdminPassword(domain, secAdminUser, secAdminPass)
			if err != nil {
				color.Red("Đổi mật khẩu admin thất bại: %v", err)
			} else {
				color.Green("✓ %s", info)
				color.Green("  Đăng nhập: https://%s/wp-admin", domain)
			}
		}

		return nil
	},
}

func init() {
	securityCmd.Flags().StringVar(&secAdminUser, "user", "", "Tài khoản admin cần đổi mật khẩu (để trống sẽ tự tìm Administrator đầu tiên)")
	securityCmd.Flags().StringVar(&secAdminPass, "pass", "", "Mật khẩu mới (để trống sẽ tự sinh ngẫu nhiên bảo mật)")
	securityCmd.Flags().BoolVar(&secSaltsOnly, "salts-only", false, "Chỉ làm mới Salt Keys, không đổi mật khẩu")
	securityCmd.Flags().BoolVar(&secPassOnly, "pass-only", false, "Chỉ đổi mật khẩu wp-admin, không đổi Salt Keys")
	securityCmd.Flags().BoolVar(&secAllSites, "all", false, "Áp dụng cho tất cả website trên hệ thống")

	siteCmd.AddCommand(securityCmd)
	RootCmd.AddCommand(securityCmd)
}
