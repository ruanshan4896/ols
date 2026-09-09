package cmd

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/shield"
	"github.com/ols-cli/ols/internal/site"
	"github.com/spf13/cobra"
)

var (
	shieldAllSites bool
)

var shieldCmd = &cobra.Command{
	Use:   "shield [domain]",
	Short: "Lá chắn bảo mật OLS Shield: Chống Brute-force, Khóa XML-RPC, Chặn PHP Uploads",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return fmt.Errorf("chưa khởi tạo hệ thống: %w", err)
		}

		mgr := site.NewManager(cfg)

		if len(args) == 0 {
			// Hiển thị trạng thái của toàn bộ website
			return showAllShieldStatus(mgr, cfg.SystemDir)
		}

		domain := args[0]
		shieldCfg, err := shield.GetShieldConfig(cfg.SystemDir, domain)
		if err != nil {
			return err
		}

		PrintShieldReport(domain, shieldCfg)
		return nil
	},
}

var shieldStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Xem trạng thái Lá chắn bảo mật trên toàn bộ website",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return fmt.Errorf("chưa khởi tạo hệ thống: %w", err)
		}

		mgr := site.NewManager(cfg)
		return showAllShieldStatus(mgr, cfg.SystemDir)
	},
}

var shieldEnableCmd = &cobra.Command{
	Use:   "enable [domain]",
	Short: "Bật toàn bộ lớp phòng thủ OLS Shield (Chế độ phòng thủ tối đa / Under Attack)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return fmt.Errorf("chưa khởi tạo hệ thống: %w", err)
		}

		mgr := site.NewManager(cfg)
		targetCfg := shield.DefaultShieldConfig()

		if shieldAllSites {
			color.Cyan("-> Đang kích hoạt OLS Shield toàn diện cho tất cả website...")
			succeeded, errs := mgr.ApplyShieldAll(targetCfg)
			for _, d := range succeeded {
				color.Green("  [OK] Đã bật phòng thủ toàn diện cho: %s", d)
			}
			for _, e := range errs {
				color.Red("  [LỖI] %v", e)
			}
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("vui lòng chỉ định tên miền hoặc dùng cờ --all")
		}

		domain := args[0]
		color.Cyan("-> Đang kích hoạt OLS Shield toàn diện cho website %s...", domain)
		if err := mgr.ApplyShield(domain, targetCfg); err != nil {
			return err
		}
		color.Green("  [OK] Kích hoạt thành công OLS Shield cho website %s!", domain)
		return nil
	},
}

var shieldDisableCmd = &cobra.Command{
	Use:   "disable [domain]",
	Short: "Tắt tạm thời các lớp phòng thủ OLS Shield (Chế độ debug)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return fmt.Errorf("chưa khởi tạo hệ thống: %w", err)
		}

		mgr := site.NewManager(cfg)
		targetCfg := shield.DisabledShieldConfig()

		if shieldAllSites {
			color.Cyan("-> Đang tắt OLS Shield cho tất cả website...")
			succeeded, errs := mgr.ApplyShieldAll(targetCfg)
			for _, d := range succeeded {
				color.Yellow("  [TẮT] Đã tắt phòng thủ cho: %s", d)
			}
			for _, e := range errs {
				color.Red("  [LỖI] %v", e)
			}
			return nil
		}

		if len(args) == 0 {
			return fmt.Errorf("vui lòng chỉ định tên miền hoặc dùng cờ --all")
		}

		domain := args[0]
		color.Cyan("-> Đang tắt các lớp phòng thủ cho website %s...", domain)
		if err := mgr.ApplyShield(domain, targetCfg); err != nil {
			return err
		}
		color.Yellow("  [TẮT] Đã tắt OLS Shield cho website %s!", domain)
		return nil
	},
}

func showAllShieldStatus(mgr *site.Manager, systemDir string) error {
	sites, err := mgr.ListSites()
	if err != nil {
		return err
	}

	if len(sites) == 0 {
		color.Yellow("Chưa có website nào trên hệ thống.")
		return nil
	}

	color.New(color.FgCyan, color.Bold).Println("\n==================================================================")
	color.New(color.FgHiGreen, color.Bold).Println("          TRẠNG THÁI LÁ CHẮN BẢO MẬT (OLS SHIELD)")
	color.New(color.FgCyan, color.Bold).Println("==================================================================")
	fmt.Printf("%-24s %-10s %-12s %-10s %-10s %-10s\n", "WEBSITE", "XML-RPC", "LOGIN RATE", "PHP UPLOAD", "USER SCAN", "FILES")
	color.New(color.FgHiBlack).Println(strings.Repeat("-", 78))

	for _, s := range sites {
		sc, _ := shield.GetShieldConfig(systemDir, s.Domain)
		fmt.Printf("%-24s %-10s %-12s %-10s %-10s %-10s\n",
			s.Domain,
			formatStatus(sc.BlockXMLRPC),
			formatStatus(sc.RateLimitLogin),
			formatStatus(sc.BlockUploadsPHP),
			formatStatus(sc.BlockUserScan),
			formatStatus(sc.BlockSensitiveFiles),
		)
	}
	fmt.Println()
	return nil
}

func formatStatus(enabled bool) string {
	if enabled {
		return color.GreenString("[BẬT]")
	}
	return color.RedString("[TẮT]")
}

func PrintShieldReport(domain string, cfg shield.SiteShieldConfig) {
	color.New(color.FgCyan, color.Bold).Println("\n==================================================================")
	color.New(color.FgHiGreen, color.Bold).Printf("          LÁ CHẮN BẢO MẬT CHO: %s\n", strings.ToUpper(domain))
	color.New(color.FgCyan, color.Bold).Println("==================================================================")

	fmt.Printf("  [1] Khóa hoàn toàn XML-RPC (xmlrpc.php)    : %s\n", formatStatus(cfg.BlockXMLRPC))
	fmt.Printf("  [2] Giới hạn lần thử đăng nhập wp-login    : %s\n", formatStatus(cfg.RateLimitLogin))
	fmt.Printf("  [3] Cấm thực thi file PHP trong /uploads/  : %s\n", formatStatus(cfg.BlockUploadsPHP))
	fmt.Printf("  [4] Chống quét dò Username quản trị viên   : %s\n", formatStatus(cfg.BlockUserScan))
	fmt.Printf("  [5] Khóa các file hệ thống nhạy cảm        : %s\n", formatStatus(cfg.BlockSensitiveFiles))
	color.New(color.FgCyan, color.Bold).Println("==================================================================")
}

// RunInteractiveShieldUI quản lý tương tác menu cho OLS Shield
func RunInteractiveShieldUI(reader *bufio.Reader, mgr *site.Manager, systemDir string) {
	sites, err := mgr.ListSites()
	if err != nil || len(sites) == 0 {
		color.Yellow("Hiện chưa có website nào trên hệ thống để bảo vệ.")
		return
	}

	color.Cyan("\n--- [15] Quản lý lá chắn bảo vệ OLS Shield ---")
	color.New(color.FgWhite, color.Bold).Println("Danh sách website:")
	for i, s := range sites {
		sc, _ := shield.GetShieldConfig(systemDir, s.Domain)
		summary := "[Bảo vệ toàn diện]"
		if !sc.BlockXMLRPC || !sc.RateLimitLogin || !sc.BlockUploadsPHP || !sc.BlockUserScan || !sc.BlockSensitiveFiles {
			summary = "[Tùy chỉnh riêng]"
		}
		fmt.Printf("  [%d] %-25s %s\n", i+1, s.Domain, summary)
	}
	fmt.Printf("  [A] Áp dụng BẬT phòng thủ toàn diện cho TẤT CẢ website\n")
	fmt.Printf("  [D] Áp dụng TẮT phòng thủ cho TẤT CẢ website\n")
	fmt.Printf("  [0] Quay lại menu chính\n")
	fmt.Println()

	choice := readInput(reader, "Nhập số thứ tự website cần quản lý [hoặc A/D/0]", "1")
	if choice == "0" || choice == "" {
		return
	}

	if strings.ToUpper(choice) == "A" {
		confirm := readInput(reader, "Bạn có chắc muốn BẬT toàn bộ lớp bảo vệ cho TẤT CẢ website? (y/N)", "y")
		if strings.ToLower(confirm) == "y" {
			color.Cyan("-> Đang kích hoạt cho toàn bộ website...")
			succeeded, errs := mgr.ApplyShieldAll(shield.DefaultShieldConfig())
			for _, d := range succeeded {
				color.Green("  [OK] Đã bảo vệ: %s", d)
			}
			for _, e := range errs {
				color.Red("  [LỖI] %v", e)
			}
		}
		return
	}

	if strings.ToUpper(choice) == "D" {
		confirm := readInput(reader, "Bạn có chắc muốn TẮT toàn bộ lớp bảo vệ cho TẤT CẢ website? (y/N)", "N")
		if strings.ToLower(confirm) == "y" {
			color.Cyan("-> Đang tắt phòng thủ cho toàn bộ website...")
			succeeded, errs := mgr.ApplyShieldAll(shield.DisabledShieldConfig())
			for _, d := range succeeded {
				color.Yellow("  [TẮT] Đã tắt: %s", d)
			}
			for _, e := range errs {
				color.Red("  [LỖI] %v", e)
			}
		}
		return
	}

	// Chọn website cụ thể
	var selectedDomain string
	var idx int
	if _, err := fmt.Sscanf(choice, "%d", &idx); err == nil && idx >= 1 && idx <= len(sites) {
		selectedDomain = sites[idx-1].Domain
	} else {
		color.Red("Lựa chọn không hợp lệ!")
		return
	}

	// Menu cấu hình từng lớp bảo vệ cho website đã chọn
	for {
		curCfg, _ := shield.GetShieldConfig(systemDir, selectedDomain)
		PrintShieldReport(selectedDomain, curCfg)

		fmt.Println("Tùy chọn thao tác:")
		fmt.Println("  [1] Bật/Tắt: Khóa XML-RPC (xmlrpc.php)")
		fmt.Println("  [2] Bật/Tắt: Giới hạn 5 lần thử đăng nhập/phút (wp-login.php)")
		fmt.Println("  [3] Bật/Tắt: Cấm thực thi file PHP trong /uploads/")
		fmt.Println("  [4] Bật/Tắt: Chống quét Username qua ?author= và REST API")
		fmt.Println("  [5] Bật/Tắt: Khóa các file hệ thống nhạy cảm")
		fmt.Println("  [6] BẬT TOÀN BỘ (Chế độ phòng thủ tối đa / Under Attack)")
		fmt.Println("  [7] TẮT TOÀN BỘ (Chế độ gỡ lỗi / Debug)")
		fmt.Println("  [0] Xong / Quay lại")
		fmt.Println()

		sub := readInput(reader, "Nhập số [0-7] để bật/tắt hoặc hoàn tất", "0")
		if sub == "0" || sub == "" {
			break
		}

		switch sub {
		case "1":
			curCfg.BlockXMLRPC = !curCfg.BlockXMLRPC
		case "2":
			curCfg.RateLimitLogin = !curCfg.RateLimitLogin
		case "3":
			curCfg.BlockUploadsPHP = !curCfg.BlockUploadsPHP
		case "4":
			curCfg.BlockUserScan = !curCfg.BlockUserScan
		case "5":
			curCfg.BlockSensitiveFiles = !curCfg.BlockSensitiveFiles
		case "6":
			curCfg = shield.DefaultShieldConfig()
		case "7":
			curCfg = shield.DisabledShieldConfig()
		default:
			color.Yellow("Lựa chọn không hợp lệ!")
			continue
		}

		color.Cyan("-> Đang áp dụng thay đổi cấu hình cho %s...", selectedDomain)
		if err := mgr.ApplyShield(selectedDomain, curCfg); err != nil {
			color.Red("Lỗi áp dụng: %v", err)
		} else {
			color.Green("  [OK] Cập nhật lá chắn bảo mật thành công!")
		}
	}
}

func init() {
	shieldEnableCmd.Flags().BoolVar(&shieldAllSites, "all", false, "Áp dụng cho tất cả website trên VPS")
	shieldDisableCmd.Flags().BoolVar(&shieldAllSites, "all", false, "Áp dụng cho tất cả website trên VPS")

	shieldCmd.AddCommand(shieldStatusCmd)
	shieldCmd.AddCommand(shieldEnableCmd)
	shieldCmd.AddCommand(shieldDisableCmd)

	RootCmd.AddCommand(shieldCmd)
}
