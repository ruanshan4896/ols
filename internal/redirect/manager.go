package redirect

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	BlockStartMarker = "# BEGIN OLS-REDIRECTS"
	BlockEndMarker   = "# END OLS-REDIRECTS"
)

type Manager struct {
	systemDir string
}

func NewManager(systemDir string) *Manager {
	if systemDir == "" {
		systemDir = "/opt/ols"
	}
	return &Manager{systemDir: systemDir}
}

// GetRedirectsFile trả về đường dẫn file redirects.json của website
func (m *Manager) GetRedirectsFile(domain string) string {
	return filepath.Join(m.systemDir, "sites", domain, "redirects.json")
}

// GetHtaccessFile trả về đường dẫn file .htaccess của website
func (m *Manager) GetHtaccessFile(domain string) string {
	return filepath.Join(m.systemDir, "sites", domain, "html", ".htaccess")
}

// ListRedirects đọc danh sách quy tắc chuyển hướng của một domain
func (m *Manager) ListRedirects(domain string) ([]RedirectRule, error) {
	filePath := m.GetRedirectsFile(domain)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []RedirectRule{}, nil
		}
		return nil, fmt.Errorf("đọc file redirects: %w", err)
	}

	var rules []RedirectRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("parse dữ liệu redirects: %w", err)
	}
	return rules, nil
}

// SaveRedirects lưu danh sách quy tắc vào redirects.json
func (m *Manager) SaveRedirects(domain string, rules []RedirectRule) error {
	filePath := m.GetRedirectsFile(domain)
	_ = os.MkdirAll(filepath.Dir(filePath), 0755)

	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return fmt.Errorf("mã hóa json: %w", err)
	}
	return os.WriteFile(filePath, data, 0664)
}

// AddRedirect thêm một quy tắc chuyển hướng mới và đồng bộ vào .htaccess
func (m *Manager) AddRedirect(domain string, rule RedirectRule) error {
	rules, err := m.ListRedirects(domain)
	if err != nil {
		return err
	}

	rules = append(rules, rule)
	if err := m.SaveRedirects(domain, rules); err != nil {
		return err
	}

	return m.ApplyToHtaccess(domain, rules)
}

// RemoveRedirect xóa một quy tắc theo ID và cập nhật lại .htaccess
func (m *Manager) RemoveRedirect(domain string, ruleID string) (bool, error) {
	rules, err := m.ListRedirects(domain)
	if err != nil {
		return false, err
	}

	var updated []RedirectRule
	found := false
	for _, r := range rules {
		if r.ID == ruleID {
			found = true
			continue
		}
		updated = append(updated, r)
	}

	if !found {
		return false, nil
	}

	if err := m.SaveRedirects(domain, updated); err != nil {
		return false, err
	}

	if err := m.ApplyToHtaccess(domain, updated); err != nil {
		return false, err
	}
	return true, nil
}

// GenerateHtaccessBlock sinh ra khối mã Apache Rewrite cho OLS từ danh sách rules
func GenerateHtaccessBlock(rules []RedirectRule) string {
	if len(rules) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(BlockStartMarker + "\n")
	sb.WriteString("<IfModule mod_rewrite.c>\n")
	sb.WriteString("RewriteEngine On\n\n")

	for _, r := range rules {
		code := r.StatusCode
		if code != 301 && code != 302 {
			code = 301
		}

		if r.Type == TypeDomain {
			// Chuyển hướng toàn bộ domain
			cleanDomain := strings.TrimSpace(r.Source)
			cleanDomain = strings.TrimPrefix(cleanDomain, "http://")
			cleanDomain = strings.TrimPrefix(cleanDomain, "https://")
			cleanDomain = strings.TrimPrefix(cleanDomain, "www.")
			cleanDomain = strings.TrimRight(cleanDomain, "/")

			escapedDomain := regexp.QuoteMeta(cleanDomain)

			targetURL := strings.TrimSpace(r.Target)
			if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
				targetURL = "https://" + targetURL
			}
			targetURL = strings.TrimRight(targetURL, "/")

			noteStr := ""
			if r.Note != "" {
				noteStr = fmt.Sprintf(" - %s", r.Note)
			}
			sb.WriteString(fmt.Sprintf("# [%s] Chuyển hướng domain: %s -> %s (%d)%s\n", r.ID, cleanDomain, targetURL, code, noteStr))
			sb.WriteString(fmt.Sprintf("RewriteCond %%{HTTP_HOST} ^(www\\.)?%s$ [NC]\n", escapedDomain))
			if r.PreservePath {
				sb.WriteString(fmt.Sprintf("RewriteRule ^(.*)$ %s/$1 [R=%d,L]\n\n", targetURL, code))
			} else {
				sb.WriteString(fmt.Sprintf("RewriteRule ^(.*)$ %s/ [R=%d,L]\n\n", targetURL, code))
			}
		} else {
			// Chuyển hướng URL / Path cụ thể
			srcPath := strings.TrimSpace(r.Source)
			srcPath = strings.TrimPrefix(srcPath, "/")
			srcPath = strings.TrimSuffix(srcPath, "/")

			target := strings.TrimSpace(r.Target)
			if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") && !strings.HasPrefix(target, "/") {
				target = "/" + target
			}

			noteStr := ""
			if r.Note != "" {
				noteStr = fmt.Sprintf(" - %s", r.Note)
			}
			sb.WriteString(fmt.Sprintf("# [%s] Chuyển hướng URL: /%s -> %s (%d)%s\n", r.ID, srcPath, target, code, noteStr))
			escapedSrc := regexp.QuoteMeta(srcPath)
			sb.WriteString(fmt.Sprintf("RewriteRule ^%s/?$ %s [R=%d,L]\n\n", escapedSrc, target, code))
		}
	}

	sb.WriteString("</IfModule>\n")
	sb.WriteString(BlockEndMarker)
	return sb.String()
}

// ApplyToHtaccess áp dụng khối rewrite vào file .htaccess một cách an toàn
func (m *Manager) ApplyToHtaccess(domain string, rules []RedirectRule) error {
	htaccessFile := m.GetHtaccessFile(domain)
	_ = os.MkdirAll(filepath.Dir(htaccessFile), 0755)

	var currentContent string
	if data, err := os.ReadFile(htaccessFile); err == nil {
		currentContent = string(data)
	}

	cleanContent := StripHtaccessBlock(currentContent)
	block := GenerateHtaccessBlock(rules)

	var finalContent string
	if block == "" {
		finalContent = cleanContent
	} else if cleanContent == "" {
		finalContent = block + "\n"
	} else {
		finalContent = block + "\n\n" + cleanContent
	}

	return os.WriteFile(htaccessFile, []byte(strings.TrimSpace(finalContent)+"\n"), 0664)
}

// StripHtaccessBlock loại bỏ khối OLS-REDIRECTS cũ ra khỏi nội dung .htaccess
func StripHtaccessBlock(content string) string {
	startIdx := strings.Index(content, BlockStartMarker)
	if startIdx == -1 {
		return strings.TrimSpace(content)
	}

	endIdx := strings.Index(content, BlockEndMarker)
	if endIdx == -1 {
		// Nếu bị mất marker kết thúc, cắt từ start
		return strings.TrimSpace(content[:startIdx])
	}

	before := content[:startIdx]
	after := content[endIdx+len(BlockEndMarker):]
	return strings.TrimSpace(before + after)
}

// TestRedirect kiểm tra phản hồi HTTP thực tế của một URL (không tự động follow)
func TestRedirect(testURL string) (*RedirectTestResult, error) {
	if !strings.HasPrefix(testURL, "http://") && !strings.HasPrefix(testURL, "https://") {
		testURL = "https://" + testURL
	}

	start := time.Now()
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Không tự động follow redirect để bắt được mã 301/302 và header Location
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("GET", testURL, nil)
	if err != nil {
		return nil, fmt.Errorf("tạo request: %w", err)
	}
	req.Header.Set("User-Agent", "OLS-CLI-Redirect-Tester/1.0")

	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		return &RedirectTestResult{
			OriginalURL:  testURL,
			StatusCode:   0,
			IsRedirect:   false,
			ResponseTime: fmt.Sprintf("%dms", duration.Milliseconds()),
			Error:        err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	isRedir := resp.StatusCode == http.StatusMovedPermanently ||
		resp.StatusCode == http.StatusFound ||
		resp.StatusCode == http.StatusSeeOther ||
		resp.StatusCode == http.StatusTemporaryRedirect ||
		resp.StatusCode == http.StatusPermanentRedirect

	return &RedirectTestResult{
		OriginalURL:  testURL,
		StatusCode:   resp.StatusCode,
		RedirectURL:  loc,
		IsRedirect:   isRedir,
		ResponseTime: fmt.Sprintf("%dms", duration.Milliseconds()),
	}, nil
}
