package shield

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SiteShieldConfig chứa trạng thái bật/tắt của các lớp bảo vệ OLS Shield
type SiteShieldConfig struct {
	BlockXMLRPC         bool `json:"block_xmlrpc"`          // Khóa xmlrpc.php (403)
	RateLimitLogin      bool `json:"rate_limit_login"`      // Giới hạn 5 req/phút wp-login.php (429)
	BlockUploadsPHP     bool `json:"block_uploads_php"`     // Cấm chạy file PHP trong /wp-content/uploads/ (403)
	BlockUserScan       bool `json:"block_user_scan"`       // Chống quét username qua ?author= và REST API users (403)
	BlockSensitiveFiles bool `json:"block_sensitive_files"` // Khóa wp-config.php, .env, .git, readme.html... (403)
}

// DefaultShieldConfig trả về cấu hình an toàn mặc định (Bật toàn bộ)
func DefaultShieldConfig() SiteShieldConfig {
	return SiteShieldConfig{
		BlockXMLRPC:         true,
		RateLimitLogin:      true,
		BlockUploadsPHP:     true,
		BlockUserScan:       true,
		BlockSensitiveFiles: true,
	}
}

// DisabledShieldConfig trả về cấu hình tắt toàn bộ (chế độ gỡ lỗi / mở cửa hoàn toàn)
func DisabledShieldConfig() SiteShieldConfig {
	return SiteShieldConfig{
		BlockXMLRPC:         false,
		RateLimitLogin:      false,
		BlockUploadsPHP:     false,
		BlockUserScan:       false,
		BlockSensitiveFiles: false,
	}
}

// GetShieldConfig đọc cấu hình bảo mật của một website. Nếu chưa có file shield.json sẽ trả về mặc định.
func GetShieldConfig(systemDir, domain string) (SiteShieldConfig, error) {
	configPath := filepath.Join(systemDir, "sites", domain, "shield.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultShieldConfig(), nil
		}
		return DefaultShieldConfig(), fmt.Errorf("đọc shield.json: %w", err)
	}

	cfg := DefaultShieldConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultShieldConfig(), fmt.Errorf("giải mã shield.json: %w", err)
	}

	return cfg, nil
}

// SaveShieldConfig lưu cấu hình bảo mật cho website
func SaveShieldConfig(systemDir, domain string, cfg SiteShieldConfig) error {
	siteDir := filepath.Join(systemDir, "sites", domain)
	if err := os.MkdirAll(siteDir, 0755); err != nil {
		return fmt.Errorf("tạo thư mục website: %w", err)
	}

	configPath := filepath.Join(siteDir, "shield.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("mã hóa cấu hình shield: %w", err)
	}

	return os.WriteFile(configPath, data, 0644)
}

const (
	BlockShieldStartMarker = "# BEGIN OLS-SHIELD"
	BlockShieldEndMarker   = "# END OLS-SHIELD"
)

// GenerateShieldHtaccess sinh khối mã rewrite bảo vệ cho file .htaccess
func GenerateShieldHtaccess(cfg SiteShieldConfig) string {
	var rules []string
	if cfg.BlockXMLRPC {
		rules = append(rules, "  # Khoa brute-force bot XML-RPC\n  RewriteRule ^/?xmlrpc\\.php$ - [F,L,NC]")
	}
	if cfg.BlockSensitiveFiles {
		rules = append(rules, "  # Khoa cac file he thong nhay cam\n  RewriteRule ^/?(wp-config\\.php|readme\\.html|license\\.txt|\\.env|\\.git|debug\\.log) - [F,L,NC]")
	}
	if cfg.BlockUploadsPHP {
		rules = append(rules, "  # Cam chay ma nguon PHP trong thu muc uploads\n  RewriteRule ^/?wp-content/uploads/.*\\.php$ - [F,L,NC]")
	}
	if cfg.BlockUserScan {
		rules = append(rules, "  # Chong do quet Username qua author query va REST API users\n  RewriteCond %{QUERY_STRING} ^author=([0-9]+) [NC]\n  RewriteRule ^ - [F,L]\n  RewriteRule ^/?wp-json/wp/v2/users - [F,L,NC]")
	}

	if len(rules) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(BlockShieldStartMarker + "\n")
	sb.WriteString("<IfModule mod_rewrite.c>\n")
	sb.WriteString("RewriteEngine On\n")
	for _, r := range rules {
		sb.WriteString(r + "\n")
	}
	sb.WriteString("</IfModule>\n")
	sb.WriteString(BlockShieldEndMarker)
	return sb.String()
}

// ApplyShieldToHtaccess chèn hoặc gỡ bỏ khối OLS-SHIELD trong nội dung .htaccess
func ApplyShieldToHtaccess(currentContent string, cfg SiteShieldConfig) string {
	content := currentContent
	startIdx := strings.Index(content, BlockShieldStartMarker)
	endIdx := strings.Index(content, BlockShieldEndMarker)
	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		endIdx += len(BlockShieldEndMarker)
		content = strings.TrimSpace(content[:startIdx] + content[endIdx:])
	}

	newBlock := GenerateShieldHtaccess(cfg)
	if newBlock == "" {
		return content
	}

	if strings.TrimSpace(content) == "" {
		return newBlock
	}

	// Đặt lên đầu file để OpenLiteSpeed / Apache ưu tiên chặn trước
	return newBlock + "\n\n" + strings.TrimLeft(content, "\r\n")
}

