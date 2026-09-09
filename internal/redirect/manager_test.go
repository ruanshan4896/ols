package redirect

import (
	"os"
	"strings"
	"testing"
)

func TestRedirectRulesAndHtaccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ols-redirect-test")
	if err != nil {
		t.Fatalf("tạo thư mục tạm: %v", err)
	}
	defer os.RemoveAll(tempDir)

	domain := "mytest.local"
	mgr := NewManager(tempDir)

	// Ban đầu chưa có rule nào
	rules, err := mgr.ListRedirects(domain)
	if err != nil {
		t.Fatalf("ListRedirects ban đầu lỗi: %v", err)
	}
	if len(rules) != 0 {
		t.Fatalf("kỳ vọng 0 rules, nhận: %d", len(rules))
	}

	// 1. Thêm Domain Redirect (301, preserve path)
	rule1 := NewRedirectRule(TypeDomain, "mytest.local", "https://newtarget.com", 301, true, "SEO Migration")
	if err := mgr.AddRedirect(domain, rule1); err != nil {
		t.Fatalf("AddRedirect rule 1 thất bại: %v", err)
	}

	// 2. Thêm Path Redirect (302)
	rule2 := NewRedirectRule(TypePath, "/old-contact", "/contact-us", 302, false, "Contact page change")
	if err := mgr.AddRedirect(domain, rule2); err != nil {
		t.Fatalf("AddRedirect rule 2 thất bại: %v", err)
	}

	// Kiểm tra danh sách rules đã lưu
	rules, err = mgr.ListRedirects(domain)
	if err != nil {
		t.Fatalf("ListRedirects sau khi thêm lỗi: %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("kỳ vọng 2 rules, nhận: %d", len(rules))
	}

	// Kiểm tra nội dung file .htaccess
	htaccessPath := mgr.GetHtaccessFile(domain)
	data, err := os.ReadFile(htaccessPath)
	if err != nil {
		t.Fatalf("đọc file .htaccess: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, BlockStartMarker) || !strings.Contains(content, BlockEndMarker) {
		t.Errorf("kỳ vọng .htaccess chứa marker bắt đầu và kết thúc")
	}
	if !strings.Contains(content, "RewriteCond %{HTTP_HOST} ^(www\\.)?mytest\\.local$ [NC]") {
		t.Errorf("kỳ vọng RewriteCond cho domain")
	}
	if !strings.Contains(content, "RewriteRule ^(.*)$ https://newtarget.com/$1 [R=301,L]") {
		t.Errorf("kỳ vọng RewriteRule 301 preserve path")
	}
	if !strings.Contains(content, "RewriteRule ^old-contact/?$ /contact-us [R=302,L]") {
		t.Errorf("kỳ vọng RewriteRule 302 cho path")
	}

	// 3. Xóa rule 1
	deleted, err := mgr.RemoveRedirect(domain, rule1.ID)
	if err != nil {
		t.Fatalf("RemoveRedirect thất bại: %v", err)
	}
	if !deleted {
		t.Fatalf("kỳ vọng xóa được rule 1")
	}

	rules, _ = mgr.ListRedirects(domain)
	if len(rules) != 1 {
		t.Fatalf("kỳ vọng còn 1 rule, nhận: %d", len(rules))
	}
	if rules[0].ID != rule2.ID {
		t.Errorf("kỳ vọng rule còn lại là rule 2")
	}

	// Kiểm tra .htaccess sau khi xóa rule 1
	data, _ = os.ReadFile(htaccessPath)
	content = string(data)
	if strings.Contains(content, rule1.ID) {
		t.Errorf("rule 1 ID vẫn còn trong .htaccess sau khi xóa")
	}
	if !strings.Contains(content, rule2.ID) {
		t.Errorf("rule 2 ID phải còn trong .htaccess")
	}

	// 4. Xóa nốt rule 2
	deleted, err = mgr.RemoveRedirect(domain, rule2.ID)
	if err != nil || !deleted {
		t.Fatalf("xóa rule 2 thất bại")
	}

	rules, _ = mgr.ListRedirects(domain)
	if len(rules) != 0 {
		t.Fatalf("kỳ vọng 0 rules, nhận: %d", len(rules))
	}

	// Kiểm tra .htaccess sau khi xóa hết rules
	data, _ = os.ReadFile(htaccessPath)
	content = string(data)
	if strings.Contains(content, BlockStartMarker) {
		t.Errorf("khối OLS-REDIRECTS phải được dọn dẹp sạch khi không còn rule")
	}
}

func TestStripHtaccessBlockWithWordPressContent(t *testing.T) {
	wpBlock := `# BEGIN WordPress
<IfModule mod_rewrite.c>
RewriteEngine On
RewriteBase /
RewriteRule ^index\.php$ - [L]
RewriteCond %{REQUEST_FILENAME} !-f
RewriteCond %{REQUEST_FILENAME} !-d
RewriteRule . /index.php [L]
</IfModule>
# END WordPress`

	rules := []RedirectRule{
		NewRedirectRule(TypePath, "/foo", "/bar", 301, false, "test"),
	}
	redirectBlock := GenerateHtaccessBlock(rules)

	combined := redirectBlock + "\n\n" + wpBlock

	stripped := StripHtaccessBlock(combined)
	if strings.Contains(stripped, BlockStartMarker) {
		t.Errorf("StripHtaccessBlock không loại bỏ được BlockStartMarker")
	}
	if !strings.Contains(stripped, "# BEGIN WordPress") {
		t.Errorf("StripHtaccessBlock làm mất nội dung WordPress")
	}
	if !strings.Contains(stripped, "# END WordPress") {
		t.Errorf("StripHtaccessBlock làm mất nội dung kết thúc WordPress")
	}
}
