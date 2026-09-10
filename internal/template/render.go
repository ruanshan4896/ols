package template

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/ols-cli/ols/templates"
)

type CoreTemplateData struct {
	NetworkName        string
	BackendNetworkName string
	DBRootPassword     string
}

type SiteTemplateData struct {
	Domain             string
	DomainSlug         string
	PHPVersion         string
	PHPVersionShort    string
	IncludeWWW         bool
	NetworkName        string
	BackendNetworkName string
	RateLimitLogin     bool
}

type SiteVhostData struct {
	Domain              string
	BlockXMLRPC         bool
	BlockSensitiveFiles bool
	BlockUploadsPHP     bool
	BlockUserScan       bool
}

func RenderCoreCompose(data CoreTemplateData) (string, error) {
	return renderFile("core/docker-compose.yml.tmpl", data)
}

func RenderTraefikConfig(email string) (string, error) {
	return renderFile("core/traefik.yml.tmpl", map[string]string{"Email": email})
}

// ShouldIncludeWWW xác định xem tên miền có nên được gắn thêm tiền tố www hay không.
// Tự động nhận diện chính xác cả tên miền gốc chuẩn quốc tế và tên miền hai đuôi quốc gia (ccTLD: .com.vn, .co.uk, v.v.)
func ShouldIncludeWWW(domain string) bool {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if strings.HasPrefix(domain, "www.") {
		return false
	}

	parts := strings.Split(domain, ".")
	if len(parts) < 2 {
		return false
	}

	// Trường hợp 1: Tên miền 2 phần (example.com, myblog.vn) -> luôn có www
	if len(parts) == 2 {
		return true
	}

	// Trường hợp 2: Tên miền 3 phần (example.com.vn, domain.co.uk)
	// Kiểm tra nếu đuôi cuối là mã quốc gia 2 ký tự (vn, uk, jp, au...) và phần giữa là SLD phổ biến
	if len(parts) == 3 {
		tld := parts[2]
		sld := parts[1]
		if len(tld) == 2 {
			commonSLDs := map[string]bool{
				"com": true, "net": true, "org": true, "edu": true,
				"gov": true, "biz": true, "info": true, "co": true,
				"me": true, "or": true, "ne": true, "ac": true,
				"pro": true, "name": true,
			}
			if commonSLDs[sld] {
				return true
			}
		}
	}

	// Các trường hợp khác (subdomain như sub.example.com, dev.domain.com.vn) -> không có www
	return false
}

func RenderSiteCompose(data SiteTemplateData) (string, error) {
	data.PHPVersionShort = strings.ReplaceAll(data.PHPVersion, ".", "")
	data.IncludeWWW = ShouldIncludeWWW(data.Domain)
	return renderFile("site/docker-compose.yml.tmpl", data)
}

func RenderSiteVhost(domain string, opts ...SiteVhostData) (string, error) {
	data := SiteVhostData{
		Domain:              domain,
		BlockXMLRPC:         true,
		BlockSensitiveFiles: true,
		BlockUploadsPHP:     true,
		BlockUserScan:       true,
	}
	if len(opts) > 0 {
		data = opts[0]
		data.Domain = domain
	}
	return renderFile("site/vhost.conf.tmpl", data)
}

func renderFile(name string, data interface{}) (string, error) {
	content, err := templates.FS.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", name, err)
	}

	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}

	return buf.String(), nil
}
