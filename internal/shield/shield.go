package shield

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
