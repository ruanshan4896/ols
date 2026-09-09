package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/redirect"
	"github.com/ols-cli/ols/internal/site"
	"github.com/spf13/cobra"
)

var (
	redirType         string
	redirFrom         string
	redirTo           string
	redirCode         int
	redirPreservePath bool
	redirNote         string
)

var redirectCmd = &cobra.Command{
	Use:     "redirect",
	Aliases: []string{"redir", "r301"},
	Short:   "Quản lý chuyển hướng 301 / 302 (Redirect Manager) cho website",
}

var redirectAddCmd = &cobra.Command{
	Use:   "add [domain]",
	Short: "Thêm quy tắc chuyển hướng 301/302 mới cho website",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		systemDir := cfg.SystemDir
		if systemDir == "" {
			systemDir = "/opt/ols"
		}

		if redirFrom == "" {
			return fmt.Errorf("cần chỉ định nguồn chuyển hướng (--from hoặc -f)")
		}
		if redirTo == "" {
			return fmt.Errorf("cần chỉ định đích đến (--to)")
		}

		rType := redirect.TypeDomain
		if redirType == "path" || redirType == "url" {
			rType = redirect.TypePath
		}

		rule := redirect.NewRedirectRule(rType, redirFrom, redirTo, redirCode, redirPreservePath, redirNote)

		rm := redirect.NewManager(systemDir)
		if err := rm.AddRedirect(domain, rule); err != nil {
			return fmt.Errorf("thêm chuyển hướng: %w", err)
		}

		color.Green("✓ Đã thêm quy tắc chuyển hướng [%s] thành công!", rule.ID)
		color.Cyan("-> Đang khởi động lại website %s để OpenLiteSpeed nạp quy tắc mới...", domain)

		mgr := site.NewManager(cfg)
		if err := mgr.RestartSite(domain); err != nil {
			color.Yellow("Cảnh báo khởi động lại website: %v", err)
		} else {
			color.Green("✓ Website %s đã khởi động lại và áp dụng chuyển hướng ngay lập tức.", domain)
		}

		return nil
	},
}

var redirectListCmd = &cobra.Command{
	Use:   "list [domain]",
	Short: "Xem danh sách các quy tắc chuyển hướng đang có",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		systemDir := cfg.SystemDir
		if systemDir == "" {
			systemDir = "/opt/ols"
		}

		rm := redirect.NewManager(systemDir)
		mgr := site.NewManager(cfg)

		var targetDomains []string
		if len(args) > 0 {
			targetDomains = []string{args[0]}
		} else {
			sitesList, err := mgr.ListSites()
			if err != nil {
				return err
			}
			for _, s := range sitesList {
				targetDomains = append(targetDomains, s.Domain)
			}
		}

		if len(targetDomains) == 0 {
			color.Yellow("Không tìm thấy website nào trên hệ thống.")
			return nil
		}

		totalCount := 0
		for _, d := range targetDomains {
			rules, err := rm.ListRedirects(d)
			if err != nil {
				continue
			}
			if len(rules) == 0 && len(args) > 0 {
				color.Yellow("Website '%s' hiện chưa có quy tắc chuyển hướng nào.", d)
				return nil
			}
			if len(rules) > 0 {
				totalCount += len(rules)
				color.Cyan("\n--- Chuyển hướng của Website: %s (%d quy tắc) ---", d, len(rules))
				PrintRedirectsTable(rules)
			}
		}

		if totalCount == 0 && len(args) == 0 {
			color.Yellow("Hiện chưa có quy tắc chuyển hướng nào trên toàn hệ thống.")
		}
		return nil
	},
}

var redirectRemoveCmd = &cobra.Command{
	Use:     "remove [domain] [rule-id]",
	Aliases: []string{"rm", "del", "delete"},
	Short:   "Xóa một quy tắc chuyển hướng",
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		ruleID := args[1]

		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		systemDir := cfg.SystemDir
		if systemDir == "" {
			systemDir = "/opt/ols"
		}

		rm := redirect.NewManager(systemDir)
		deleted, err := rm.RemoveRedirect(domain, ruleID)
		if err != nil {
			return fmt.Errorf("xóa quy tắc: %w", err)
		}
		if !deleted {
			color.Yellow("Không tìm thấy quy tắc có ID '%s' trên website '%s'.", ruleID, domain)
			return nil
		}

		color.Green("✓ Đã xóa quy tắc [%s] khỏi website %s!", ruleID, domain)
		color.Cyan("-> Đang khởi động lại website %s để gỡ bỏ quy tắc khỏi bộ nhớ OpenLiteSpeed...", domain)

		mgr := site.NewManager(cfg)
		if err := mgr.RestartSite(domain); err != nil {
			color.Yellow("Cảnh báo khởi động lại website: %v", err)
		} else {
			color.Green("✓ Đã cập nhật OpenLiteSpeed thành công.")
		}

		return nil
	},
}

var redirectTestCmd = &cobra.Command{
	Use:   "test [url]",
	Short: "Kiểm tra phản hồi HTTP chuyển hướng thực tế của một URL",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		testURL := args[0]
		color.Cyan("-> Đang gửi yêu cầu kiểm tra tới %s...", testURL)

		res, err := redirect.TestRedirect(testURL)
		if err != nil {
			return err
		}

		PrintTestResult(res)
		return nil
	},
}

func init() {
	redirectAddCmd.Flags().StringVarP(&redirType, "type", "t", "domain", "Loại chuyển hướng: 'domain' (toàn bộ website) hoặc 'path' (đường dẫn con)")
	redirectAddCmd.Flags().StringVarP(&redirFrom, "from", "f", "", "Nguồn chuyển hướng (Tên miền hoặc URL con)")
	redirectAddCmd.Flags().StringVar(&redirTo, "to", "", "Đích đến (URL mới)")
	redirectAddCmd.Flags().IntVarP(&redirCode, "code", "c", 301, "Mã phản hồi HTTP (301: Vĩnh viễn, 302: Tạm thời)")
	redirectAddCmd.Flags().BoolVarP(&redirPreservePath, "preserve-path", "p", true, "Giữ nguyên đường dẫn con khi chuyển domain ($1)")
	redirectAddCmd.Flags().StringVarP(&redirNote, "note", "n", "", "Ghi chú mục đích chuyển hướng")

	redirectCmd.AddCommand(redirectAddCmd)
	redirectCmd.AddCommand(redirectListCmd)
	redirectCmd.AddCommand(redirectRemoveCmd)
	redirectCmd.AddCommand(redirectTestCmd)

	RootCmd.AddCommand(redirectCmd)
}

// PrintRedirectsTable in danh sách rules dưới dạng bảng
func PrintRedirectsTable(rules []redirect.RedirectRule) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"STT", "Mã ID", "Loại", "Nguồn (Source)", "Đích đến (Target)", "HTTP Code", "Giữ link con", "Ghi chú"})
	table.SetHeaderColor(
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiCyanColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiYellowColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiMagentaColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiGreenColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlueColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiWhiteColor},
		tablewriter.Colors{tablewriter.Bold, tablewriter.FgHiBlackColor},
	)
	table.SetBorder(true)
	table.SetAutoWrapText(false)

	for i, r := range rules {
		preserveStr := "Không"
		if r.PreservePath {
			preserveStr = color.GreenString("Có ($1)")
		}

		typeStr := "Đường dẫn"
		if r.Type == redirect.TypeDomain {
			typeStr = color.CyanString("Tên miền")
		}

		codeStr := fmt.Sprintf("%d", r.StatusCode)
		if r.StatusCode == 301 {
			codeStr = color.GreenString("301")
		} else {
			codeStr = color.YellowString("302")
		}

		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			r.ID,
			typeStr,
			r.Source,
			r.Target,
			codeStr,
			preserveStr,
			r.Note,
		})
	}
	table.Render()
}

// PrintTestResult in kết quả kiểm tra HTTP redirect
func PrintTestResult(res *redirect.RedirectTestResult) {
	fmt.Println()
	color.New(color.FgWhite, color.Bold).Println("=== KẾT QUẢ KIỂM TRA CHUYỂN HƯỚNG HTTP ===")
	fmt.Printf("  - URL kiểm tra    : %s\n", res.OriginalURL)
	if res.Error != "" {
		color.Red("  - Lỗi kết nối     : %s\n", res.Error)
		return
	}

	fmt.Printf("  - Thời gian phản hồi: %s\n", res.ResponseTime)
	if res.IsRedirect {
		color.Green("  - Mã HTTP phản hồi: %d (Đang CHUYỂN HƯỚNG thành công)\n", res.StatusCode)
		color.Cyan("  - Đích chuyển đến  : %s\n", res.RedirectURL)
	} else {
		color.Yellow("  - Mã HTTP phản hồi: %d (Không phải chuyển hướng)\n", res.StatusCode)
		if res.RedirectURL != "" {
			fmt.Printf("  - Location header : %s\n", res.RedirectURL)
		}
	}
	fmt.Println()
}

// RunInteractiveRedirectUI giao diện tương tác từng bước trong menu OLS-CLI
func RunInteractiveRedirectUI(reader *bufio.Reader, mgr *site.Manager, systemDir string) {
	sites, err := mgr.ListSites()
	if err != nil || len(sites) == 0 {
		color.Yellow("Hiện chưa có website nào trên hệ thống.")
		return
	}

	color.Cyan("\n--- Quản lý chuyển hướng 301 / 302 (Redirects) ---")
	printOption("[1]", "Thêm chuyển hướng mới (Toàn bộ website hoặc URL cụ thể)")
	printOption("[2]", "Xem danh sách các chuyển hướng đang hoạt động")
	printOption("[3]", "Xóa một quy tắc chuyển hướng")
	printOption("[4]", "Kiểm tra chuyển hướng trực tiếp (Test HTTP 301/302)")
	printOption("[0]", "Quay lại menu chính")
	fmt.Println()

	sub := readInput(reader, "Nhập lựa chọn của bạn [0-4]", "2")
	rm := redirect.NewManager(systemDir)

	switch sub {
	case "1":
		// Thêm chuyển hướng
		color.Cyan("\n--- [Thêm chuyển hướng] Chọn website cần cấu hình ---")
		for i, s := range sites {
			fmt.Printf("  [%d] %s\n", i+1, s.Domain)
		}
		domainChoice := readInput(reader, "Nhập STT website", "1")
		var idx int
		if _, err := fmt.Sscanf(domainChoice, "%d", &idx); err != nil || idx < 1 || idx > len(sites) {
			color.Red("Lựa chọn website không hợp lệ!")
			return
		}
		domain := sites[idx-1].Domain

		color.Cyan("\nChọn hình thức chuyển hướng cho %s:", domain)
		printOption("[1]", "Chuyển hướng TOÀN BỘ WEBSITE sang tên miền mới (Domain 301)")
		printOption("[2]", "Chuyển hướng một ĐƯỜNG DẪN URL cụ thể (Path Redirect)")
		fmt.Println()

		typeChoice := readInput(reader, "Nhập lựa chọn [1-2]", "1")
		var rType redirect.RedirectType
		var source string
		var target string
		var preservePath bool

		if typeChoice == "1" {
			rType = redirect.TypeDomain
			source = domain
			target = readInput(reader, "Nhập tên miền đích mới (VD: https://newbrand.com)", "")
			if target == "" {
				color.Red("Tên miền đích không được để trống!")
				return
			}
			pChoice := readInput(reader, "Giữ nguyên đường dẫn con ($1)? (Y/n)", "Y")
			preservePath = strings.ToLower(pChoice) != "n"
		} else {
			rType = redirect.TypePath
			source = readInput(reader, "Nhập đường dẫn URL nguồn cần chuyển (VD: /bai-viet-cu)", "")
			if source == "" {
				color.Red("Đường dẫn nguồn không được để trống!")
				return
			}
			target = readInput(reader, "Nhập URL đích đến (VD: /bai-viet-moi hoặc https://link-ngoai.com)", "")
			if target == "" {
				color.Red("Đích đến không được để trống!")
				return
			}
			preservePath = false
		}

		codeStr := readInput(reader, "Mã chuyển hướng HTTP (301: Vĩnh viễn chuẩn SEO, 302: Tạm thời)", "301")
		code, _ := strconv.Atoi(codeStr)
		if code != 301 && code != 302 {
			code = 301
		}

		note := readInput(reader, "Ghi chú mục đích (Enter để bỏ qua)", "")

		rule := redirect.NewRedirectRule(rType, source, target, code, preservePath, note)
		color.Cyan("-> Đang ghi cấu hình vào .htaccess của %s...", domain)
		if err := rm.AddRedirect(domain, rule); err != nil {
			color.Red("Thêm chuyển hướng thất bại: %v", err)
			return
		}

		color.Green("✓ Đã thêm quy tắc chuyển hướng [%s] thành công!", rule.ID)
		color.Cyan("-> Đang khởi động lại OpenLiteSpeed container của %s...", domain)
		if err := mgr.RestartSite(domain); err != nil {
			color.Yellow("Cảnh báo khởi động lại website: %v", err)
		} else {
			color.Green("✓ Website %s đã khởi động lại và nạp chuyển hướng ngay lập tức!", domain)
		}

	case "2":
		// Xem danh sách chuyển hướng
		color.Cyan("\n--- Danh sách chuyển hướng đang hoạt động ---")
		totalFound := 0
		for _, s := range sites {
			rules, err := rm.ListRedirects(s.Domain)
			if err == nil && len(rules) > 0 {
				totalFound += len(rules)
				color.Cyan("\nWebsite: %s (%d quy tắc)", s.Domain, len(rules))
				PrintRedirectsTable(rules)
			}
		}
		if totalFound == 0 {
			color.Yellow("Hiện tại chưa có website nào thiết lập quy tắc chuyển hướng.")
		}

	case "3":
		// Xóa chuyển hướng
		color.Cyan("\n--- [Xóa chuyển hướng] Chọn website ---")
		for i, s := range sites {
			fmt.Printf("  [%d] %s\n", i+1, s.Domain)
		}
		domainChoice := readInput(reader, "Nhập STT website", "1")
		var idx int
		if _, err := fmt.Sscanf(domainChoice, "%d", &idx); err != nil || idx < 1 || idx > len(sites) {
			color.Red("Lựa chọn không hợp lệ!")
			return
		}
		domain := sites[idx-1].Domain

		rules, err := rm.ListRedirects(domain)
		if err != nil || len(rules) == 0 {
			color.Yellow("Website %s không có quy tắc chuyển hướng nào để xóa.", domain)
			return
		}

		color.Cyan("\nDanh sách quy tắc của %s:", domain)
		PrintRedirectsTable(rules)

		ruleInput := readInput(reader, "Nhập STT (1, 2, ...) hoặc Mã ID quy tắc cần xóa", "")
		if ruleInput == "" {
			color.Yellow("Đã hủy thao tác.")
			return
		}

		targetID := ruleInput
		var ruleIdx int
		if _, err := fmt.Sscanf(ruleInput, "%d", &ruleIdx); err == nil && ruleIdx >= 1 && ruleIdx <= len(rules) {
			targetID = rules[ruleIdx-1].ID
		}

		color.Yellow("-> Đang gỡ bỏ quy tắc [%s]...", targetID)
		deleted, err := rm.RemoveRedirect(domain, targetID)
		if err != nil {
			color.Red("Lỗi xóa quy tắc: %v", err)
			return
		}
		if !deleted {
			color.Red("Không tìm thấy quy tắc [%s] trên website %s!", targetID, domain)
			return
		}

		color.Green("✓ Đã xóa quy tắc [%s] và dọn dẹp file .htaccess!", targetID)
		color.Cyan("-> Đang khởi động lại website %s...", domain)
		_ = mgr.RestartSite(domain)
		color.Green("✓ Hoàn tất!")

	case "4":
		// Kiểm tra chuyển hướng thực tế
		testURL := readInput(reader, "Nhập URL cần kiểm tra (VD: https://mywebsite.com/old-link)", "")
		if testURL == "" {
			color.Yellow("Chưa nhập URL.")
			return
		}
		color.Cyan("-> Đang gửi yêu cầu kiểm tra phản hồi HTTP tới %s...", testURL)
		res, err := redirect.TestRedirect(testURL)
		if err != nil {
			color.Red("Lỗi kiểm tra: %v", err)
			return
		}
		PrintTestResult(res)

	default:
		color.Cyan("Đã quay lại menu chính.")
	}
}
