package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/ols-cli/ols/internal/backup"
	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/site"
	"github.com/ols-cli/ols/internal/system"
	"github.com/ols-cli/ols/internal/util"
	"github.com/spf13/cobra"
)

var menuCmd = &cobra.Command{
	Use:   "menu",
	Short: "Mở menu tương tác trực quan quản lý website và hệ thống",
	RunE: func(cmd *cobra.Command, args []string) error {
		return RunInteractiveMenu(os.Stdin, os.Stdout)
	},
}

func init() {
	RootCmd.AddCommand(menuCmd)
}

func readInput(reader *bufio.Reader, prompt string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return defaultValue
	}
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func pauseForEnter(reader *bufio.Reader) {
	fmt.Print("\n👉 Bấm phím [Enter] để quay lại menu chính...")
	_, _ = reader.ReadString('\n')
}

var (
	headerBoxStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00F0FF")).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7952DE")).
		Padding(0, 3)

	categoryHeaderStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FF79C6"))

	itemNumStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#50FA7B"))

	itemDescStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#94A3B8"))

	dividerStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#475569"))
)

// PrintMenu in giao diện menu trực quan phân nhóm với khung viền bo góc
func PrintMenu(w io.Writer) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, headerBoxStyle.Render("⚡ HỆ THỐNG QUẢN TRỊ WORDPRESS & OPENLITESPEED (OLS-CLI)"))
	fmt.Fprintln(w, "")

	fmt.Fprintln(w, " "+categoryHeaderStyle.Render("🚀 [ HẠ TẦNG CỐT LÕI ]"))
	fmt.Fprintf(w, "   %s Khởi tạo máy chủ VPS %s\n", itemNumStyle.Render("[1]"), itemDescStyle.Render("(Traefik Proxy, MariaDB 11, Redis 7, SSL)"))
	fmt.Fprintln(w, "")

	fmt.Fprintln(w, " "+categoryHeaderStyle.Render("🌐 [ QUẢN LÝ WEBSITE ]"))
	fmt.Fprintf(w, "   %s Thêm website WordPress mới %s\n", itemNumStyle.Render("[2]"), itemDescStyle.Render("(Tự động tải WP core & vhost OLS)"))
	fmt.Fprintf(w, "   %s Xem danh sách website đang chạy\n", itemNumStyle.Render("[3]"))
	fmt.Fprintf(w, "   %s Khởi động lại website %s\n", itemNumStyle.Render("[4]"), itemDescStyle.Render("(Restart container OLS)"))
	fmt.Fprintf(w, "   %s Xóa website %s\n", itemNumStyle.Render("[5]"), itemDescStyle.Render("(Xóa container, mã nguồn, database & SSL)"))
	fmt.Fprintln(w, "")

	fmt.Fprintln(w, " "+categoryHeaderStyle.Render("🛡️ [ SAO LƯU & BẢO MẬT ]"))
	fmt.Fprintf(w, "   %s Sao lưu website %s\n", itemNumStyle.Render("[6]"), itemDescStyle.Render("(Backup 1 site hoặc tất cả website)"))
	fmt.Fprintf(w, "   %s Khôi phục website từ bản sao lưu %s\n", itemNumStyle.Render("[7]"), itemDescStyle.Render("(Restore .tar.gz)"))
	fmt.Fprintf(w, "   %s Quản trị Database phpMyAdmin %s\n", itemNumStyle.Render("[8]"), itemDescStyle.Render("(Bật / Tắt qua web port 8080)"))
	fmt.Fprintf(w, "   %s Kiểm tra trạng thái các container Docker\n", itemNumStyle.Render("[9]"))
	fmt.Fprintln(w, "")

	fmt.Fprintln(w, " "+categoryHeaderStyle.Render("⚡ [ TỐI ƯU, GIÁM SÁT & DEBUG ]"))
	fmt.Fprintf(w, "  %s Đồng bộ cấu hình các website %s\n", itemNumStyle.Render("[10]"), itemDescStyle.Render("(Sync vhost, cache & Traefik)"))
	fmt.Fprintf(w, "  %s Bảo mật: Làm mới Salt Keys & Đổi pass Admin %s\n", itemNumStyle.Render("[11]"), itemDescStyle.Render("(WordPress.org API)"))
	fmt.Fprintf(w, "  %s Quản lý bộ nhớ Swap RAM %s\n", itemNumStyle.Render("[12]"), itemDescStyle.Render("(Tạo Swap 2-8GB chống sập VPS)"))
	fmt.Fprintf(w, "  %s Đánh giá tải VPS & Tính số website có thể cài thêm\n", itemNumStyle.Render("[13]"))
	fmt.Fprintf(w, "  %s Xem nhật ký lỗi & Hỗ trợ Debug %s\n", itemNumStyle.Render("[14]"), itemDescStyle.Render("(Traefik, DB, PHP Error & Quét lỗi)"))
	fmt.Fprintln(w, "")

	fmt.Fprintln(w, " "+categoryHeaderStyle.Render("🚪 [ HỆ THỐNG ]"))
	fmt.Fprintf(w, "   %s Thoát\n", itemNumStyle.Render("[0]"))
	fmt.Fprintln(w, dividerStyle.Render("───────────────────────────────────────────────────────────────────"))
}

func RunInteractiveMenu(r io.Reader, w io.Writer) error {
	reader := bufio.NewReader(r)

	for {
		PrintMenu(w)

		choice := readInput(reader, "👉 Nhập lựa chọn của bạn [0-14]", "")
		if choice == "" {
			continue
		}

		if choice == "0" {
			color.Yellow("\nCảm ơn bạn đã sử dụng ols-cli. Tạm biệt!\n")
			break
		}

		handleMenuChoice(choice, reader)
	}

	return nil
}

func handleMenuChoice(choice string, reader *bufio.Reader) {
	configPath := "/opt/ols/config/ols.yaml"
	cfg, errCfg := config.LoadConfig(configPath)

	switch choice {
	case "1":
		color.Cyan("\n--- [1] Khởi tạo hạ tầng VPS ---")
		email := readInput(reader, "Nhập Email đăng ký chứng chỉ SSL Let's Encrypt", "admin@example.com")
		initEmail = email
		if err := initCmd.RunE(initCmd, []string{}); err != nil {
			color.Red("Lỗi khởi tạo: %v", err)
		}
		pauseForEnter(reader)

	case "2":
		if errCfg != nil {
			color.Red("\nHệ thống chưa được khởi tạo! Vui lòng chọn [1] trước.")
			pauseForEnter(reader)
			return
		}
		color.Cyan("\n--- [2] Thêm website WordPress mới ---")
		mode := readInput(reader, "Chọn: [1] Thêm 1 website đơn lẻ | [2] Thêm nhiều website từ file TXT", "1")
		mgr := site.NewManager(cfg)

		if mode == "2" {
			filePath := readInput(reader, "Nhập đường dẫn file TXT chứa danh sách domain", "domains.txt")
			data, err := os.ReadFile(filePath)
			if err != nil {
				color.Red("Không thể đọc file '%s': %v", filePath, err)
				pauseForEnter(reader)
				return
			}
			domains := site.ParseDomainList(string(data))
			if len(domains) == 0 {
				color.Yellow("File '%s' không chứa tên miền nào hợp lệ!", filePath)
				pauseForEnter(reader)
				return
			}

			color.Cyan("Tìm thấy %d tên miền trong file.", len(domains))
			phpVer := readInput(reader, "Phiên bản PHP chung (8.1, 8.2, 8.3)", "8.2")
			adminUser := readInput(reader, "Tên tài khoản quản trị wp-admin", "admin")
			adminPass := readInput(reader, "Mật khẩu quản trị wp-admin", "")
			if adminPass == "" {
				color.Red("Mật khẩu quản trị không được để trống!")
				pauseForEnter(reader)
				return
			}
			adminEmail := readInput(reader, "Email quản trị viên", cfg.ACMEEmail)

			color.Cyan("\n-> Bắt đầu khởi tạo và cài đặt tự động %d website...", len(domains))
			successCount := 0
			failCount := 0
			for i, d := range domains {
				color.Yellow("\n[%d/%d] Đang xử lý website: %s...", i+1, len(domains), d)
				err := mgr.CreateSite(site.CreateSiteOptions{
					Domain:        d,
					PHPVersion:    phpVer,
					WithRedis:     true,
					InstallWP:     true,
					AdminUser:     adminUser,
					AdminPassword: adminPass,
					AdminEmail:    adminEmail,
				})
				if err != nil {
					color.Red("  ✗ Thất bại: %v", err)
					failCount++
				} else {
					color.Green("  ✓ Hoàn tất! Đăng nhập: https://%s/wp-admin", d)
					successCount++
				}
			}
			color.Cyan("\n=== Tổng kết: Thành công: %d | Thất bại: %d ===", successCount, failCount)
		} else {
			domain := readInput(reader, "Nhập tên miền (Domain)", "")
			if domain == "" {
				color.Red("Tên miền không được để trống!")
				pauseForEnter(reader)
				return
			}
			phpVer := readInput(reader, "Phiên bản PHP (8.1, 8.2, 8.3)", "8.2")
			adminUser := readInput(reader, "Tên tài khoản quản trị wp-admin", "admin")
			adminPass := readInput(reader, "Mật khẩu quản trị wp-admin [Enter để tự sinh ngẫu nhiên]", "")
			if adminPass == "" {
				generatedPass, errGen := util.GenerateRandomString(16)
				if errGen == nil {
					adminPass = generatedPass
				} else {
					adminPass = "Admin@" + domain
				}
				color.Yellow("-> Đã tự động sinh mật khẩu quản trị an toàn: %s", adminPass)
			}
			adminEmail := readInput(reader, "Email quản trị viên", cfg.ACMEEmail)

			color.Cyan("-> Đang tạo và tự động cài đặt hoàn chỉnh WordPress cho %s...", domain)
			if err := mgr.CreateSite(site.CreateSiteOptions{
				Domain:        domain,
				PHPVersion:    phpVer,
				WithRedis:     true,
				InstallWP:     true,
				AdminUser:     adminUser,
				AdminPassword: adminPass,
				AdminEmail:    adminEmail,
			}); err != nil {
				color.Red("Tạo website thất bại: %v", err)
			} else {
				color.Green("✓ Website https://%s đã được tạo và cài đặt hoàn chỉnh thành công!", domain)
				color.Green("  Đăng nhập: https://%s/wp-admin", domain)
				color.Green("  Tài khoản: %s", adminUser)
				color.Green("  Mật khẩu:  %s", adminPass)
			}
		}
		pauseForEnter(reader)

	case "3":
		if errCfg != nil {
			color.Red("\nHệ thống chưa được khởi tạo!")
			pauseForEnter(reader)
			return
		}
		color.Cyan("\n--- [3] Danh sách website ---")
		mgr := site.NewManager(cfg)
		sites, err := mgr.ListSites()
		if err != nil {
			color.Red("Lỗi lấy danh sách: %v", err)
		} else if len(sites) == 0 {
			color.Yellow("Hiện chưa có website nào được tạo.")
		} else {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"STT", "Tên miền (Domain)", "Trạng thái", "URL"})
			for i, s := range sites {
				table.Append([]string{fmt.Sprintf("%d", i+1), s.Domain, s.Status, "https://" + s.Domain})
			}
			table.Render()
		}
		pauseForEnter(reader)

	case "4":
		if errCfg != nil {
			color.Red("\nHệ thống chưa được khởi tạo!")
			pauseForEnter(reader)
			return
		}
		color.Cyan("\n--- [4] Khởi động lại website ---")
		domain := readInput(reader, "Nhập tên miền cần khởi động lại", "")
		if domain == "" {
			color.Red("Tên miền không được để trống!")
			pauseForEnter(reader)
			return
		}
		mgr := site.NewManager(cfg)
		if err := mgr.RestartSite(domain); err != nil {
			color.Red("Khởi động lại thất bại: %v", err)
		} else {
			color.Green("✓ Website %s đã được khởi động lại thành công!", domain)
		}
		pauseForEnter(reader)

	case "5":
		if errCfg != nil {
			color.Red("\nHệ thống chưa được khởi tạo!")
			pauseForEnter(reader)
			return
		}
		color.Cyan("\n--- [5] Xóa website ---")
		domain := readInput(reader, "Nhập tên miền cần xóa", "")
		if domain == "" {
			color.Red("Tên miền không được để trống!")
			pauseForEnter(reader)
			return
		}
		confirm := readInput(reader, fmt.Sprintf("CẢNH BÁO: Dữ liệu của %s sẽ bị xóa vĩnh viễn. Xác nhận xóa? (y/N)", domain), "n")
		if strings.ToLower(confirm) == "y" || strings.ToLower(confirm) == "yes" {
			mgr := site.NewManager(cfg)
			if err := mgr.DeleteSite(domain, true); err != nil {
				color.Red("Xóa website thất bại: %v", err)
			} else {
				color.Green("✓ Website %s đã được xóa hoàn toàn!", domain)
			}
		} else {
			color.Yellow("Đã hủy thao tác xóa.")
		}
		pauseForEnter(reader)

	case "6":
		if errCfg != nil {
			color.Red("\nHệ thống chưa được khởi tạo!")
			pauseForEnter(reader)
			return
		}
		color.Cyan("\n--- [6] Sao lưu website (Backup) ---")
		sub := readInput(reader, "Chọn: [1] Sao lưu 1 website cụ thể | [2] Sao lưu TẤT CẢ website", "1")
		bm := backup.NewBackupManager(cfg)

		if sub == "1" {
			domain := readInput(reader, "Nhập tên miền cần sao lưu", "")
			if domain == "" {
				color.Red("Tên miền không được để trống!")
				pauseForEnter(reader)
				return
			}
			color.Cyan("-> Đang sao lưu %s...", domain)
			path, err := bm.BackupSite(domain)
			if err != nil {
				color.Red("Sao lưu thất bại: %v", err)
			} else {
				color.Green("✓ Sao lưu thành công! File lưu tại:\n  %s", path)
			}
		} else {
			color.Cyan("-> Đang tiến hành sao lưu toàn bộ website...")
			backedUp, errs := bm.BackupAllSitesProgress(func(current, total int, domain, backupPath string, err error) {
				if err != nil {
					color.Red("  [%d/%d] Sao lưu %s: ✗ Thất bại: %v", current, total, domain, err)
				} else {
					color.Green("  [%d/%d] Sao lưu %s: ✓ Hoàn tất (%s)", current, total, domain, backupPath)
				}
			})
			if len(backedUp) == 0 && len(errs) == 0 {
				color.Yellow("Hiện chưa có website nào trên hệ thống.")
			} else {
				color.Cyan("=== Hoàn tất sao lưu: Thành công: %d | Thất bại: %d ===", len(backedUp), len(errs))
			}
		}
		pauseForEnter(reader)

	case "7":
		if errCfg != nil {
			color.Red("\nHệ thống chưa được khởi tạo!")
			pauseForEnter(reader)
			return
		}
		color.Cyan("\n--- [7] Khôi phục website (Restore) ---")
		domain := readInput(reader, "Nhập tên miền cần khôi phục", "")
		if domain == "" {
			color.Red("Tên miền không được để trống!")
			pauseForEnter(reader)
			return
		}

		// Gợi ý file backup nếu có
		backupDir := filepath.Join(cfg.SystemDir, "backups", domain)
		entries, _ := os.ReadDir(backupDir)
		defaultFile := ""
		if len(entries) > 0 {
			defaultFile = filepath.Join(backupDir, entries[len(entries)-1].Name())
		}

		backupFile := readInput(reader, "Đường dẫn file backup (.tar.gz)", defaultFile)
		if backupFile == "" {
			color.Red("Đường dẫn file backup không được để trống!")
			pauseForEnter(reader)
			return
		}

		bm := backup.NewBackupManager(cfg)
		color.Cyan("-> Đang khôi phục %s từ %s...", domain, backupFile)
		if err := bm.RestoreSite(domain, backupFile); err != nil {
			color.Red("Khôi phục thất bại: %v", err)
		} else {
			color.Green("✓ Khôi phục thành công website %s!", domain)
		}
		pauseForEnter(reader)

	case "8":
		if errCfg != nil {
			color.Red("\nHệ thống chưa được khởi tạo!")
			pauseForEnter(reader)
			return
		}
		color.Cyan("\n--- [8] Quản trị phpMyAdmin ---")
		subChoice := readInput(reader, "Chọn: [1] Bật phpMyAdmin (Port 8080) | [2] Tắt phpMyAdmin", "1")
		if subChoice == "1" {
			pmaPort = 8080
			_ = pmaEnableCmd.RunE(pmaEnableCmd, []string{})
		} else {
			_ = pmaDisableCmd.RunE(pmaDisableCmd, []string{})
		}
		pauseForEnter(reader)

	case "9":
		color.Cyan("\n--- [9] Trạng thái các container Docker ---")
		cmd := exec.Command("docker", "ps", "--format", "table {{.Names}}\t{{.Status}}\t{{.Ports}}")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
		pauseForEnter(reader)

	case "10":
		if errCfg != nil {
			color.Red("\nHệ thống chưa được khởi tạo!")
			pauseForEnter(reader)
			return
		}
		color.Cyan("\n--- [10] Đồng bộ cấu hình các website ---")
		sub := readInput(reader, "Chọn: [1] Đồng bộ 1 website cụ thể | [2] Đồng bộ TẤT CẢ website", "2")
		mgr := site.NewManager(cfg)
		if sub == "1" {
			domain := readInput(reader, "Nhập tên miền cần đồng bộ", "")
			if domain == "" {
				color.Red("Tên miền không được để trống!")
				pauseForEnter(reader)
				return
			}
			color.Cyan("-> Đang đồng bộ cấu hình website %s...", domain)
			if err := mgr.SyncSite(domain); err != nil {
				color.Red("Đồng bộ thất bại: %v", err)
			} else {
				color.Green("✓ Đồng bộ cấu hình website %s thành công!", domain)
			}
		} else {
			color.Cyan("-> Đang đồng bộ cấu hình hạ tầng Core (Traefik, Services)...")
			if err := mgr.SyncCore(); err != nil {
				color.Yellow("  Cảnh báo đồng bộ hạ tầng Core: %v", err)
			} else {
				color.Green("✓ Đồng bộ cấu hình hạ tầng Core hoàn tất")
			}

			color.Cyan("-> Đang đồng bộ cấu hình toàn bộ website...")
			synced, errs := mgr.SyncAllSitesProgress(func(current, total int, domain string, err error) {
				if err != nil {
					color.Red("  [%d/%d] Đồng bộ %s: ✗ Thất bại: %v", current, total, domain, err)
				} else {
					color.Green("  [%d/%d] Đồng bộ %s: ✓ Hoàn tất", current, total, domain)
				}
			})
			if len(synced) == 0 && len(errs) == 0 {
				color.Yellow("Hiện chưa có website nào trên hệ thống.")
			} else {
				color.Cyan("=== Hoàn tất đồng bộ: Thành công: %d | Thất bại: %d ===", len(synced), len(errs))
			}
		}
		pauseForEnter(reader)

	case "11":
		if errCfg != nil {
			color.Red("\nHệ thống chưa được khởi tạo!")
			pauseForEnter(reader)
			return
		}
		color.Cyan("\n--- [11] Bảo mật: Làm mới Salt Keys & Đổi mật khẩu Admin ---")
		sub := readInput(reader, "Chọn: [1] Bảo mật 1 website cụ thể | [2] Bảo mật TẤT CẢ website", "1")
		mgr := site.NewManager(cfg)

		if sub == "1" {
			domain := readInput(reader, "Nhập tên miền website cần bảo mật", "")
			if domain == "" {
				color.Red("Tên miền không được để trống!")
				pauseForEnter(reader)
				return
			}
			optType := readInput(reader, "Chọn tác vụ: [1] Cả hai (Làm mới Salts & Đổi Pass) | [2] Chỉ làm mới Salts | [3] Chỉ đổi Pass", "1")

			if optType == "1" || optType == "2" {
				color.Cyan("-> Đang lấy 8 Authentication Salts/Keys mới từ WordPress.org API...")
				if _, err := mgr.RegenerateSalts(domain); err != nil {
					color.Red("Làm mới Salts thất bại: %v", err)
				} else {
					color.Green("✓ Đã làm mới 8 Salts/Keys bảo mật vào wp-config.php thành công!")
					color.Yellow("  (Toàn bộ cookie và phiên đăng nhập cũ trên mọi thiết bị đã bị đăng xuất)")
				}
			}

			if optType == "1" || optType == "3" {
				adminUser := readInput(reader, "Tài khoản admin cần đổi mật khẩu [Enter để tự phát hiện]", "")
				adminPass := readInput(reader, "Mật khẩu mới [Enter để tự sinh ngẫu nhiên bảo mật]", "")
				color.Cyan("-> Đang đặt lại mật khẩu quản trị WordPress...")
				info, err := mgr.ResetAdminPassword(domain, adminUser, adminPass)
				if err != nil {
					color.Red("Đổi mật khẩu thất bại: %v", err)
				} else {
					color.Green("✓ %s", info)
					color.Green("  Đăng nhập quản trị: https://%s/wp-admin", domain)
				}
			}
		} else {
			sitesList, err := mgr.ListSites()
			if err != nil {
				color.Red("Lỗi lấy danh sách website: %v", err)
				pauseForEnter(reader)
				return
			}
			if len(sitesList) == 0 {
				color.Yellow("Hiện chưa có website nào trên hệ thống.")
				pauseForEnter(reader)
				return
			}

			optType := readInput(reader, "Chọn tác vụ: [1] Cả hai (Làm mới Salts & Đổi Pass) | [2] Chỉ làm mới Salts | [3] Chỉ đổi Pass", "1")
			globalPass := ""
			if optType == "1" || optType == "3" {
				globalPass = readInput(reader, "Mật khẩu quản trị mới dùng chung [Enter để tự sinh mật khẩu riêng cho từng site]", "")
			}

			color.Cyan("\n-> Bắt đầu xử lý bảo mật cho %d website...", len(sitesList))
			for i, s := range sitesList {
				color.Yellow("\n[%d/%d] Website: %s", i+1, len(sitesList), s.Domain)
				if optType == "1" || optType == "2" {
					if _, err := mgr.RegenerateSalts(s.Domain); err != nil {
						color.Red("  Làm mới Salts thất bại: %v", err)
					} else {
						color.Green("  ✓ Đã làm mới 8 Salts/Keys bảo mật từ WordPress.org API")
					}
				}
				if optType == "1" || optType == "3" {
					info, err := mgr.ResetAdminPassword(s.Domain, "", globalPass)
					if err != nil {
						color.Red("  Đổi mật khẩu thất bại: %v", err)
					} else {
						color.Green("  ✓ %s", info)
					}
				}
			}
			color.Cyan("\n=== Hoàn tất xử lý bảo mật cho toàn bộ website ===")
		}
		pauseForEnter(reader)

	case "12":
		color.Cyan("\n--- [12] Quản lý bộ nhớ Swap RAM chống sập VPS ---")
		info, err := system.GetSwapInfo()
		if err == nil {
			color.New(color.FgWhite, color.Bold).Println("Trạng thái Swap hiện tại:")
			fmt.Printf("  - Tổng dung lượng : %d MB (%.1f GB)\n", info.TotalMB, float64(info.TotalMB)/1024.0)
			fmt.Printf("  - Đang sử dụng    : %d MB\n", info.UsedMB)
			fmt.Printf("  - Còn trống       : %d MB\n", info.FreeMB)
			if info.Path != "" {
				fmt.Printf("  - Tệp tin Swap    : %s\n", info.Path)
			}
			fmt.Println()
		}

		sub := readInput(reader, "Chọn: [1] Tạo mới / Thay đổi dung lượng Swap | [2] Tắt và xóa Swap", "1")
		if sub == "1" {
			sizeStr := readInput(reader, "Nhập dung lượng Swap mong muốn (GB) [Ví dụ: 2, 4, 8]", "2")
			sizeGB, _ := strconv.Atoi(sizeStr)
			if sizeGB <= 0 {
				sizeGB = 2
			}

			color.Cyan("-> Đang tiến hành tạo bộ nhớ Swap RAM %d GB (tối ưu swappiness=10)...", sizeGB)
			if err := system.CreateSwap(sizeGB); err != nil {
				color.Red("Tạo Swap thất bại: %v", err)
			} else {
				color.Green("✓ Đã tạo thành công %d GB Swap RAM cho VPS!", sizeGB)
				color.Yellow("  (Tự động kích hoạt lại khi khởi động lại VPS qua /etc/fstab & tối ưu vm.swappiness=10)")
			}
		} else {
			confirm := readInput(reader, "Bạn có chắc chắn muốn tắt và xóa Swap? (y/N)", "N")
			if strings.ToLower(confirm) == "y" {
				color.Yellow("-> Đang tắt và xóa Swap...")
				if err := system.DisableSwap(); err != nil {
					color.Red("Lỗi tắt Swap: %v", err)
				} else {
					color.Green("✓ Đã tắt và xóa hoàn toàn Swap trên VPS!")
				}
			} else {
				color.Cyan("Đã hủy thao tác.")
			}
		}
		pauseForEnter(reader)

	case "13":
		color.Cyan("\n--- [13] Đánh giá năng lực & Khả năng chịu tải VPS ---")
		systemDir := "/opt/ols"
		if errCfg == nil && cfg.SystemDir != "" {
			systemDir = cfg.SystemDir
		}
		report, err := system.GetVPSCapacity(systemDir)
		if err != nil {
			color.Red("Lỗi đo lường tài nguyên: %v", err)
		} else {
			PrintCapacityReport(report)
		}
		pauseForEnter(reader)

	case "14":
		color.Cyan("\n--- [14] Xem nhật ký lỗi & Hỗ trợ Debug ---")
		color.New(color.FgWhite, color.Bold).Println("Chọn dịch vụ hoặc thành phần cần kiểm tra lỗi:")
		fmt.Println("  [1] Traefik Reverse Proxy & SSL (Lỗi chứng chỉ SSL, 502 Bad Gateway)")
		fmt.Println("  [2] MariaDB Database (Lỗi kết nối cơ sở dữ liệu, crash)")
		fmt.Println("  [3] Redis Cache (Lỗi bộ nhớ đệm, connection refused)")
		fmt.Println("  [4] Xem lỗi của một Website cụ thể (PHP Fatal, 500, lỗi plugin)")
		fmt.Println("  [5] ⚡ QUÉT NHANH TOÀN HỆ THỐNG (Tự động lọc các lỗi gần nhất)")
		fmt.Println("  [6] 🗑️  XÓA TOÀN BỘ NHẬT KÝ CŨ (Reset log Docker & Web về 0 byte)")
		fmt.Println("  [0] Quay lại")
		fmt.Println()

		subChoice := readInput(reader, "👉 Nhập lựa chọn của bạn [0-6]", "5")
		switch subChoice {
		case "1":
			_ = PrintContainerLog("ols-traefik", "Traefik SSL/Proxy", 60)
		case "2":
			_ = PrintContainerLog("ols-mariadb", "MariaDB Database", 60)
		case "3":
			_ = PrintContainerLog("ols-redis", "Redis Cache", 60)
		case "4":
			domain := readInput(reader, "Nhập tên miền website cần xem nhật ký", "")
			if domain != "" {
				color.Cyan("-> Đang lấy nhật ký website %s...", domain)
				out, err := system.GetSiteLogs(domain, 60)
				if err != nil {
					color.Red("Lỗi: %v", err)
				} else {
					fmt.Println(out)
				}
			}
		case "5":
			_ = RunQuickErrorScan(50)
		case "6":
			confirm := readInput(reader, "Bạn có chắc chắn muốn xóa sạch toàn bộ log của tất cả container? (y/N)", "N")
			if strings.ToLower(confirm) == "y" {
				_ = RunClearAllLogs()
			} else {
				color.Cyan("Đã hủy thao tác.")
			}
		default:
			color.Cyan("Đã quay lại menu chính.")
		}
		pauseForEnter(reader)

	default:
		color.Yellow("Lựa chọn không hợp lệ! Vui lòng chọn từ 0 đến 14.")
		pauseForEnter(reader)
	}
}
