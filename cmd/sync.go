package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/site"
	"github.com/spf13/cobra"
)

var syncAll bool

var syncCmd = &cobra.Command{
	Use:   "sync [domain]",
	Short: "Đồng bộ và nâng cấp cấu hình mới nhất cho các website đang hoạt động",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return fmt.Errorf("chưa khởi tạo hệ thống (vui lòng chạy 'ols init' trước): %w", err)
		}

		mgr := site.NewManager(cfg)

		if syncAll || len(args) == 0 {
			color.Cyan("-> Đang đồng bộ cấu hình cho toàn bộ website...")
			synced, errs := mgr.SyncAllSites()
			for _, d := range synced {
				color.Green("✓ Đã đồng bộ thành công: %s", d)
			}
			for _, e := range errs {
				color.Red("✗ Thất bại: %v", e)
			}
			if len(synced) == 0 && len(errs) == 0 {
				color.Yellow("Hiện chưa có website nào trên hệ thống.")
			}
			return nil
		}

		domain := args[0]
		color.Cyan("-> Đang đồng bộ cấu hình website %s...", domain)
		if err := mgr.SyncSite(domain); err != nil {
			return fmt.Errorf("đồng bộ website %s thất bại: %w", domain, err)
		}

		color.Green("✓ Đồng bộ cấu hình website %s thành công!", domain)
		return nil
	},
}

func init() {
	syncCmd.Flags().BoolVarP(&syncAll, "all", "a", false, "Đồng bộ tất cả các website")
	RootCmd.AddCommand(syncCmd)
}
