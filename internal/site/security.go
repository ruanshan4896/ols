package site

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ols-cli/ols/internal/util"
)

// FetchWordPressSalts lấy 8 khóa bảo mật từ WordPress.org API chính chủ.
// Nếu thất bại (mất mạng, timeout), tự động fallback sang sinh chuỗi ngẫu nhiên cryptographic.
func FetchWordPressSalts() (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("https://api.wordpress.org/secret-key/1.1/salt/")
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		body, errRead := io.ReadAll(resp.Body)
		if errRead == nil && strings.Contains(string(body), "define('AUTH_KEY'") {
			return strings.TrimSpace(string(body)), nil
		}
	}

	// Fallback nếu API WordPress không phản hồi
	keys := []string{
		"AUTH_KEY", "SECURE_AUTH_KEY", "LOGGED_IN_KEY", "NONCE_KEY",
		"AUTH_SALT", "SECURE_AUTH_SALT", "LOGGED_IN_SALT", "NONCE_SALT",
	}

	var sb strings.Builder
	for _, k := range keys {
		val, errGen := util.GenerateRandomString(64)
		if errGen != nil {
			return "", errGen
		}
		sb.WriteString(fmt.Sprintf("define( '%s', '%s' );\n", k, val))
	}

	return strings.TrimSpace(sb.String()), nil
}

// RegenerateSalts làm mới 8 khóa bảo mật trong file wp-config.php của website
func (m *Manager) RegenerateSalts(domain string) (bool, error) {
	siteDir := filepath.Join(m.cfg.SystemDir, "sites", domain)
	wpConfigPath := filepath.Join(siteDir, "html", "wp-config.php")

	contentBytes, err := os.ReadFile(wpConfigPath)
	if err != nil {
		return false, fmt.Errorf("không tìm thấy wp-config.php cho website %s: %w", domain, err)
	}

	newSalts, err := FetchWordPressSalts()
	if err != nil {
		return false, fmt.Errorf("không thể tạo salts mới: %w", err)
	}

	content := string(contentBytes)

	// Pattern tìm block 8 dòng define keys trong wp-config.php
	// Hỗ trợ cả định dạng: define( 'AUTH_KEY' ... hoặc define('AUTH_KEY' ...
	reSalts := regexp.MustCompile(`(?s)(//\s*Authentication Unique Keys and Salts\s*)?(define\s*\(\s*['"]AUTH_KEY['"].*?define\s*\(\s*['"]NONCE_SALT['"][^\n]*\n)`)

	replacement := "// Authentication Unique Keys and Salts (Regenerated via OLS CLI)\n" + newSalts + "\n"

	if reSalts.MatchString(content) {
		content = reSalts.ReplaceAllString(content, replacement)
	} else {
		// Nếu file cấu hình không có block chuẩn, chèn trước "if ( ! defined( 'ABSPATH' ) )" hoặc cuối file
		marker := "if ( ! defined( 'ABSPATH' ) )"
		if strings.Contains(content, marker) {
			content = strings.Replace(content, marker, replacement+"\n"+marker, 1)
		} else {
			content = content + "\n" + replacement
		}
	}

	if err := os.WriteFile(wpConfigPath, []byte(content), 0644); err != nil {
		return false, fmt.Errorf("lỗi ghi wp-config.php: %w", err)
	}

	return true, nil
}

// ResetAdminPassword đặt lại mật khẩu cho tài khoản quản trị WordPress.
// Nếu username rỗng, hàm tự động phát hiện tài khoản Administrator đầu tiên trong database.
func (m *Manager) ResetAdminPassword(domain string, username string, newPassword string) (string, error) {
	slug := DomainToSlug(domain)
	containerName := "ols_" + slug

	if !m.dm.IsContainerRunning(containerName) {
		return "", fmt.Errorf("container '%s' của website chưa bật, vui lòng khởi động container trước", containerName)
	}

	if newPassword == "" {
		generated, err := util.GenerateRandomString(16)
		if err != nil {
			return "", err
		}
		newPassword = generated
	}

	// Escape password cho PHP string
	escapedPass := strings.ReplaceAll(newPassword, "\\", "\\\\")
	escapedPass = strings.ReplaceAll(escapedPass, "'", "\\'")
	escapedUser := strings.ReplaceAll(username, "'", "\\'")

	phpScript := fmt.Sprintf(`
define('WP_USE_THEMES', false);
require_once '/usr/local/lsws/Example/html/wp-load.php';

$target_user = '%s';
$user = null;

if (!empty($target_user)) {
    $user = get_user_by('login', $target_user);
}

if (!$user) {
    $admins = get_users(array('role' => 'administrator', 'number' => 1));
    if (!empty($admins)) {
        $user = $admins[0];
    }
}

if ($user) {
    wp_set_password('%s', $user->ID);
    echo "SUCCESS:" . $user->user_login;
} else {
    echo "ERROR: Không tìm thấy tài khoản quản trị viên nào trong cơ sở dữ liệu";
}
`, escapedUser, escapedPass)

	out, err := m.dm.ExecInContainer(containerName, "php", "-r", phpScript)
	if err != nil {
		return "", fmt.Errorf("lỗi thực thi đổi mật khẩu trong container: %s (%w)", out, err)
	}

	trimmedOut := strings.TrimSpace(out)
	if strings.HasPrefix(trimmedOut, "SUCCESS:") {
		adminLogin := strings.TrimPrefix(trimmedOut, "SUCCESS:")
		return fmt.Sprintf("User: %s | Mật khẩu mới: %s", adminLogin, newPassword), nil
	}

	return "", fmt.Errorf("phản hồi từ WordPress: %s", trimmedOut)
}
