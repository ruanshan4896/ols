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

// ReplaceSaltsInWPConfig thay thế hoặc chèn khối 8 khóa bảo mật vào nội dung wp-config.php một cách an toàn tuyệt đối.
// Sử dụng ReplaceAllLiteralString để đảm bảo các ký tự đặc biệt ($1, $2, $name, backslashes...) trong salt từ WordPress.org
// không bao giờ bị hiểu nhầm thành regex expansion gây lỗi cú pháp PHP Unmatched '}'.
func ReplaceSaltsInWPConfig(content, newSalts string) string {
	replacement := "// Authentication Unique Keys and Salts (Regenerated via OLS CLI)\n" + strings.TrimSpace(newSalts) + "\n"

	// Pattern tìm block 8 dòng define keys trong wp-config.php (hỗ trợ cả trường hợp đã từng regenerate hoặc có chú thích)
	reSalts := regexp.MustCompile(`(?s)(//\s*Authentication Unique Keys and Salts[^\n]*\n\s*)?(define\s*\(\s*['"]AUTH_KEY['"].*?define\s*\(\s*['"]NONCE_SALT['"][^\n]*\n?)`)

	if reSalts.MatchString(content) {
		return reSalts.ReplaceAllLiteralString(content, replacement)
	}

	// Nếu file cấu hình không có block chuẩn, chèn trước khối "if (!defined('ABSPATH'))"
	reABSPATH := regexp.MustCompile(`(?i)if\s*\(\s*!\s*defined\s*\(\s*['"]ABSPATH['"]\s*\)\s*\)`)
	if loc := reABSPATH.FindStringIndex(content); loc != nil {
		return content[:loc[0]] + replacement + "\n" + content[loc[0]:]
	}

	return strings.TrimRight(content, "\r\n") + "\n\n" + replacement
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

	updated := ReplaceSaltsInWPConfig(string(contentBytes), newSalts)

	if err := os.WriteFile(wpConfigPath, []byte(updated), 0644); err != nil {
		return false, fmt.Errorf("lỗi ghi wp-config.php: %w", err)
	}

	return true, nil
}

// GetSitePHPBinary trả về đường dẫn chính xác của binary LSPHP bên trong container của website
func (m *Manager) GetSitePHPBinary(domain string) string {
	phpShort := "82"
	siteDir := filepath.Join(m.cfg.SystemDir, "sites", domain)
	composePath := filepath.Join(siteDir, "docker-compose.yml")
	if content, err := os.ReadFile(composePath); err == nil {
		re := regexp.MustCompile(`lsphp(8[1-3])`)
		if match := re.FindStringSubmatch(string(content)); len(match) > 1 {
			phpShort = match[1]
		}
	}
	return fmt.Sprintf("/usr/local/lsws/lsphp%s/bin/php", phpShort)
}

// ResetAdminPassword đặt lại mật khẩu cho tài khoản quản trị WordPress.
// Nếu username rỗng, hàm tự động phát hiện tài khoản Administrator đầu tiên trong database.
// Hỗ trợ tìm kiếm theo cả user_login và email (nếu có ký tự @).
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

	// Escape password và username cho PHP string
	escapedPass := strings.ReplaceAll(newPassword, "\\", "\\\\")
	escapedPass = strings.ReplaceAll(escapedPass, "'", "\\'")
	escapedUser := strings.ReplaceAll(username, "\\", "\\\\")
	escapedUser = strings.ReplaceAll(escapedUser, "'", "\\'")

	phpScript := fmt.Sprintf(`
define('WP_USE_THEMES', false);
require_once '/usr/local/lsws/Example/html/wp-load.php';

$target_user = '%s';
$user = null;

if (!empty($target_user)) {
    // 1. Tìm theo username đăng nhập chính xác
    $user = get_user_by('login', $target_user);
    // 2. Nếu không thấy và target_user có dạng email, tìm theo email
    if (!$user && strpos($target_user, '@') !== false) {
        $user = get_user_by('email', $target_user);
    }
}

// Nếu người dùng không chỉ định username cụ thể, tự động lấy tài khoản Administrator đầu tiên
if (!$user && empty($target_user)) {
    $admins = get_users(array('role' => 'administrator', 'number' => 1));
    if (!empty($admins)) {
        $user = $admins[0];
    }
}

if ($user) {
    wp_set_password('%s', $user->ID);
    echo "SUCCESS:" . $user->user_login;
} else if (!empty($target_user)) {
    echo "ERROR: Không tìm thấy tài khoản quản trị viên nào với username hoặc email là '" . $target_user . "'";
} else {
    echo "ERROR: Không tìm thấy tài khoản quản trị viên nào trong cơ sở dữ liệu";
}
`, escapedUser, escapedPass)

	phpBin := m.GetSitePHPBinary(domain)
	out, err := m.dm.ExecInContainer(containerName, phpBin, "-r", phpScript)
	if err != nil {
		// Fallback sang "php" nếu phpBin không tìm thấy
		out, err = m.dm.ExecInContainer(containerName, "php", "-r", phpScript)
		if err != nil {
			return "", fmt.Errorf("lỗi thực thi đổi mật khẩu trong container: %s (%w)", out, err)
		}
	}

	trimmedOut := strings.TrimSpace(out)
	if strings.HasPrefix(trimmedOut, "SUCCESS:") {
		adminLogin := strings.TrimPrefix(trimmedOut, "SUCCESS:")
		return fmt.Sprintf("User: %s | Mật khẩu mới: %s", adminLogin, newPassword), nil
	}

	return "", fmt.Errorf("phản hồi từ WordPress: %s", trimmedOut)
}
