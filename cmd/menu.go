package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/ols-cli/ols/internal/backup"
	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/site"
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

func RunInteractiveMenu(r io.Reader, w io.Writer) error {
	reader := bufio.NewReader(r)

	for {
		color.New(color.FgCyan, color.Bold).Println("\n==================================================================")
		color.New(color.FgHiGreen, color.Bold).Println("      HỆ THỐNG QUẢN TRỊ WORDPRESS & OPENLITESPEED (OLS-CLI)")
		color.New(color.FgCyan, color.Bold).Println("==================================================================")
		fmt.Println("  [1] Khởi tạo hạ tầng máy chủ VPS (Traefik, MariaDB, Redis, SSL)")
		fmt.Println("  [2] Thêm website WordPress mới (Tự động tải mã nguồn & cấu hình)")
		fmt.Println("  [3] Xem danh sách website đang chạy")
		fmt.Println("  [4] Khởi động lại website (Restart)")
		fmt.Println("  [5] Xóa website")
		fmt.Println("  [6] Sao lưu website (Backup)")
		fmt.Println("  [7] Khôi phục website từ bản sao lưu (Restore)")
		fmt.Println("  [8] Quản trị Database phpMyAdmin (Bật / Tắt)")
		fmt.Println("  [9] Kiểm tra trạng thái các container Docker")
		fmt.Println("  [0] Thoát")
		color.New(color.FgCyan, color.Bold).Println("==================================================================")

		choice := readInput(reader, "👉 Nhập lựa chọn của bạn [0-9]", "")
		if choice == "" {
			continue
		}

		if choice == "0" {
			color.Yellow("\nCảm ơn bạn đã sử dụng ols-cli. Tạm biệt!")
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
		domain := readInput(reader, "Nhập tên miền (Domain)", "")
		if domain == "" {
			color.Red("Tên miền không được để trống!")
			pauseForEnter(reader)
			return
		}
		phpVer := readInput(reader, "Phiên bản PHP (8.1, 8.2, 8.3)", "8.2")

		mgr := site.NewManager(cfg)
		color.Cyan("-> Đang tạo website %s...", domain)
		if err := mgr.CreateSite(site.CreateSiteOptions{
			Domain:     domain,
			PHPVersion: phpVer,
			WithRedis:  true,
			InstallWP:  true,
		}); err != nil {
			color.Red("Tạo website thất bại: %v", err)
		} else {
			color.Green("✓ Website https://%s đã được tạo và vận hành thành công!", domain)
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
		domain := readInput(reader, "Nhập tên miền cần sao lưu", "")
		if domain == "" {
			color.Red("Tên miền không được để trống!")
			pauseForEnter(reader)
			return
		}
		bm := backup.NewBackupManager(cfg)
		color.Cyan("-> Đang sao lưu %s...", domain)
		path, err := bm.BackupSite(domain)
		if err != nil {
			color.Red("Sao lưu thất bại: %v", err)
		} else {
			color.Green("✓ Sao lưu thành công! File lưu tại:\n  %s", path)
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

	default:
		color.Yellow("Lựa chọn không hợp lệ! Vui lòng chọn từ 0 đến 9.")
		pauseForEnter(reader)
	}
}
