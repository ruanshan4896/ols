package site

import (
	"archive/zip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	LSCachePluginURL = "https://downloads.wordpress.org/plugin/litespeed-cache.latest-stable.zip"
	LSCachePluginZip = "litespeed-cache.latest-stable.zip"
)

// GetPresetPath trả về đường dẫn file preset chung trên VPS
func (m *Manager) GetPresetPath() string {
	return filepath.Join(m.cfg.SystemDir, "config", "litespeed_preset.json")
}

// HasLSCachePreset kiểm tra xem đã có file preset khuôn mẫu được lưu chưa
func (m *Manager) HasLSCachePreset() bool {
	_, err := os.Stat(m.GetPresetPath())
	return err == nil
}

// GetDefaultLSCachePreset trả về bộ cấu hình mặc định tối ưu (Best Practices)
// Tập trung vào Tốc độ & An toàn tuyệt đối: Page Cache, Mobile Cache, Guest Mode, Instant Click, Browser Cache
func GetDefaultLSCachePreset() map[string]interface{} {
	return map[string]interface{}{
		"litespeed.cache":              1,
		"litespeed.cache-priv":         1,
		"litespeed.cache-commenter":    1,
		"litespeed.cache-rest":         1,
		"litespeed.cache-page_login":   1,
		"litespeed.cache-resources":    1,
		"litespeed.cache-mobile":       1,
		"litespeed.guest":              1,
		"litespeed.guest_optm":         1,
		"litespeed.instant_click":      1,
		"litespeed.optm-browser_cache": 1,
		"litespeed.purge-stale":        1,
		"litespeed.purge-post_all":     1,
		"litespeed.purge-post_f":       1,
		"litespeed.purge-post_h":       1,
		"litespeed.purge-post_p":       1,
	}
}

// ExportLSCachePreset trích xuất toàn bộ cấu hình LiteSpeed Cache từ một website mẫu đã cấu hình hoàn chỉnh
func (m *Manager) ExportLSCachePreset(sourceDomain string) (int, error) {
	slug := DomainToSlug(sourceDomain)
	containerName := "ols_" + slug

	if !m.dm.IsContainerRunning(containerName) {
		return 0, fmt.Errorf("container '%s' của website %s chưa được bật", containerName, sourceDomain)
	}

	phpScript := `
define('WP_USE_THEMES', false);
require_once '/usr/local/lsws/Example/html/wp-load.php';

$conf = array();

// 1. Dùng API chính thức của LiteSpeed nếu class đã nạp (dry_run = true để nhận mảng trả về)
if (class_exists('LiteSpeed\Conf')) {
    $loaded = LiteSpeed\Conf::cls()->load_options(null, true);
    if (is_array($loaded) && !empty($loaded)) {
        foreach ($loaded as $k => $v) {
            $conf['litespeed.' . $k] = $v;
        }
    }
}

// 2. Quét trực tiếp bảng wp_options (hỗ trợ mọi phiên bản v1, v2, v3, v4, v5, v6, v7)
if (empty($conf)) {
    global $wpdb;
    $results = $wpdb->get_results("SELECT option_name, option_value FROM {$wpdb->options} WHERE option_name LIKE 'litespeed.%' OR option_name = 'litespeed-cache-conf'");
    if (!empty($results)) {
        foreach ($results as $row) {
            $conf[$row->option_name] = maybe_unserialize($row->option_value);
        }
    }
}

if (empty($conf)) {
    echo "ERROR: Không tìm thấy cấu hình LiteSpeed Cache trong database của website này. Hãy đảm bảo plugin đã được cài đặt và lưu cấu hình ít nhất 1 lần trong wp-admin.";
    exit;
}

$encoded = json_encode($conf, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
echo "SUCCESS:" . base64_encode($encoded);
`

	phpBin := m.GetSitePHPBinary(sourceDomain)
	out, err := m.dm.ExecInContainer(containerName, phpBin, "-r", phpScript)
	if err != nil {
		out, err = m.dm.ExecInContainer(containerName, "php", "-r", phpScript)
		if err != nil {
			return 0, fmt.Errorf("lỗi thực thi trích xuất cấu hình trong container: %s (%w)", out, err)
		}
	}

	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(trimmed, "SUCCESS:") {
		if strings.HasPrefix(trimmed, "ERROR:") {
			return 0, fmt.Errorf("%s", strings.TrimPrefix(trimmed, "ERROR: "))
		}
		return 0, fmt.Errorf("phản hồi không hợp lệ từ WordPress: %s", trimmed)
	}

	b64Data := strings.TrimPrefix(trimmed, "SUCCESS:")
	jsonData, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return 0, fmt.Errorf("lỗi giải mã dữ liệu cấu hình: %w", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonData, &parsed); err != nil {
		return 0, fmt.Errorf("dữ liệu cấu hình không phải là JSON hợp lệ: %w", err)
	}

	configDir := filepath.Join(m.cfg.SystemDir, "config")
	_ = os.MkdirAll(configDir, 0755)

	presetPath := m.GetPresetPath()
	if err := os.WriteFile(presetPath, jsonData, 0644); err != nil {
		return 0, fmt.Errorf("không thể ghi file preset '%s': %w", presetPath, err)
	}

	return len(parsed), nil
}

// DownloadAndCacheLSCacheZip tải và lưu đệm file plugin litespeed-cache.zip từ WordPress.org
func (m *Manager) DownloadAndCacheLSCacheZip() (string, error) {
	cacheDir := filepath.Join(m.cfg.SystemDir, "cache")
	_ = os.MkdirAll(cacheDir, 0755)
	zipPath := filepath.Join(cacheDir, LSCachePluginZip)

	// Nếu đã có cache và chưa quá 14 ngày, tái sử dụng
	if stat, err := os.Stat(zipPath); err == nil && stat.Size() > 100000 {
		if time.Since(stat.ModTime()) < 14*24*time.Hour {
			return zipPath, nil
		}
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(LSCachePluginURL)
	if err != nil {
		// Nếu tải lỗi mà file cũ vẫn còn thì fallback dùng file cũ
		if _, statErr := os.Stat(zipPath); statErr == nil {
			return zipPath, nil
		}
		return "", fmt.Errorf("không thể tải LiteSpeed Cache plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tải plugin thất bại với mã lỗi HTTP %d", resp.StatusCode)
	}

	tmpFile := zipPath + ".tmp"
	out, err := os.Create(tmpFile)
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(out, resp.Body); err != nil {
		out.Close()
		_ = os.Remove(tmpFile)
		return "", err
	}
	out.Close()

	_ = os.Rename(tmpFile, zipPath)
	return zipPath, nil
}

// UnzipLSCache giải nén file zip plugin vào thư mục wp-content/plugins
func UnzipLSCache(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	_ = os.MkdirAll(destDir, 0777)

	for _, f := range r.File {
		cleanName := filepath.Clean(f.Name)
		if strings.HasPrefix(cleanName, "..") {
			continue // Chống path traversal
		}

		targetPath := filepath.Join(destDir, cleanName)

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(targetPath, 0777)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0777); err != nil {
			return err
		}

		outFile, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// EnsureLSCachePluginInstalled đảm bảo plugin LiteSpeed Cache đã có mặt trong thư mục plugins của website
func (m *Manager) EnsureLSCachePluginInstalled(domain string) error {
	pluginFile := filepath.Join(m.cfg.SystemDir, "sites", domain, "html", "wp-content", "plugins", "litespeed-cache", "litespeed-cache.php")
	if _, err := os.Stat(pluginFile); err == nil {
		return nil // Đã cài sẵn
	}

	zipPath, err := m.DownloadAndCacheLSCacheZip()
	if err != nil {
		return fmt.Errorf("lỗi chuẩn bị file plugin zip: %w", err)
	}

	pluginsDir := filepath.Join(m.cfg.SystemDir, "sites", domain, "html", "wp-content", "plugins")
	if err := UnzipLSCache(zipPath, pluginsDir); err != nil {
		return fmt.Errorf("lỗi giải nén plugin LiteSpeed Cache: %w", err)
	}

	// Đảm bảo quyền ghi cho container
	lscacheDir := filepath.Join(pluginsDir, "litespeed-cache")
	_ = exec.Command("chmod", "-R", "777", lscacheDir).Run()

	return nil
}

// ApplyLSCachePreset kích hoạt plugin và áp dụng toàn bộ cấu hình tối ưu vào website đích
func (m *Manager) ApplyLSCachePreset(targetDomain string) error {
	slug := DomainToSlug(targetDomain)
	containerName := "ols_" + slug

	if !m.dm.IsContainerRunning(containerName) {
		return fmt.Errorf("container '%s' của website chưa bật", containerName)
	}

	// 1. Đảm bảo plugin đã có mặt trong source code
	if err := m.EnsureLSCachePluginInstalled(targetDomain); err != nil {
		return fmt.Errorf("không thể cài đặt plugin cho %s: %w", targetDomain, err)
	}

	// 2. Lấy nội dung preset (ưu tiên file preset do user export, nếu chưa có thì dùng default)
	var presetJSON []byte
	if m.HasLSCachePreset() {
		data, err := os.ReadFile(m.GetPresetPath())
		if err == nil && len(data) > 0 {
			presetJSON = data
		}
	}

	if len(presetJSON) == 0 {
		defaultData, _ := json.Marshal(GetDefaultLSCachePreset())
		presetJSON = defaultData
	}

	b64Preset := base64.StdEncoding.EncodeToString(presetJSON)

	// 3. Thực thi PHP trong container để kích hoạt plugin, nạp options và purge cache
	phpScript := fmt.Sprintf(`
define('WP_USE_THEMES', false);
require_once '/usr/local/lsws/Example/html/wp-load.php';
require_once ABSPATH . 'wp-admin/includes/plugin.php';

// 1. Kích hoạt plugin nếu chưa bật
if (!is_plugin_active('litespeed-cache/litespeed-cache.php')) {
    $res = activate_plugin('litespeed-cache/litespeed-cache.php');
    if (is_wp_error($res)) {
        echo "ERROR: Không thể kích hoạt plugin: " . $res->get_error_message();
        exit;
    }
}

// 2. Cập nhật cấu hình
$raw_json = base64_decode('%s');
$new_conf = json_decode($raw_json, true);

if (!is_array($new_conf) || empty($new_conf)) {
    echo "ERROR: Dữ liệu cấu hình không hợp lệ";
    exit;
}

foreach ($new_conf as $opt_name => $opt_val) {
    if (strpos($opt_name, 'litespeed.') === 0 || $opt_name === 'litespeed-cache-conf') {
        update_option($opt_name, $opt_val);
    } else {
        update_option('litespeed.' . $opt_name, $opt_val);
    }
}

// Nạp lại cấu hình nếu class tồn tại
if (class_exists('LiteSpeed\Conf')) {
    LiteSpeed\Conf::cls()->load_options();
}

// 3. Xóa sạch cache cũ để cache mới sinh ra theo cấu hình tối ưu
if (class_exists('LiteSpeed\Purge')) {
    LiteSpeed\Purge::purge_all();
}

echo "SUCCESS";
`, b64Preset)

	phpBin := m.GetSitePHPBinary(targetDomain)
	out, err := m.dm.ExecInContainer(containerName, phpBin, "-r", phpScript)
	if err != nil {
		out, err = m.dm.ExecInContainer(containerName, "php", "-r", phpScript)
		if err != nil {
			return fmt.Errorf("lỗi thực thi nạp cấu hình trong container: %s (%w)", out, err)
		}
	}

	trimmed := strings.TrimSpace(out)
	if trimmed != "SUCCESS" {
		if strings.HasPrefix(trimmed, "ERROR:") {
			return fmt.Errorf("%s", strings.TrimPrefix(trimmed, "ERROR: "))
		}
		return fmt.Errorf("phản hồi từ WordPress: %s", trimmed)
	}

	return nil
}

// ApplyLSCachePresetToAll áp dụng cấu hình tối ưu cho tất cả các website đang có trên VPS
func (m *Manager) ApplyLSCachePresetToAll() ([]string, map[string]error) {
	sites, err := m.ListSites()
	if err != nil {
		return nil, map[string]error{"system": err}
	}

	var successes []string
	failures := make(map[string]error)

	for _, s := range sites {
		if err := m.ApplyLSCachePreset(s.Domain); err != nil {
			failures[s.Domain] = err
		} else {
			successes = append(successes, s.Domain)
		}
	}

	return successes, failures
}

// GetPresetSummary đọc thông tin tóm tắt của file khuôn mẫu hiện tại
func (m *Manager) GetPresetSummary() (string, error) {
	if !m.HasLSCachePreset() {
		return "Chưa có file khuôn mẫu riêng. Hệ thống đang dùng cấu hình mặc định (Page Cache, Guest Mode, Instant Click, Mobile Cache, Browser Cache).", nil
	}

	data, err := os.ReadFile(m.GetPresetPath())
	if err != nil {
		return "", err
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		return "", fmt.Errorf("file preset bị lỗi cấu trúc: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("✓ File khuôn mẫu: %s\n", m.GetPresetPath()))
	sb.WriteString(fmt.Sprintf("✓ Tổng số tham số cấu hình: %d tùy chọn\n", len(parsed)))

	// Điểm qua một số tính năng chủ chốt
	keyChecks := []struct {
		key  string
		name string
	}{
		{"cache", "Page Cache"},
		{"cache-mobile", "Mobile Cache"},
		{"guest", "Guest Mode"},
		{"guest_optm", "Guest Optimization"},
		{"instant_click", "Instant Click (Preload)"},
		{"optm-browser_cache", "Browser Cache"},
		{"optm-css_min", "Minify CSS"},
		{"optm-js_min", "Minify JS"},
		{"optm-html_min", "Minify HTML"},
	}

	sb.WriteString("✓ Trạng thái các tính năng nổi bật:\n")
	for _, kc := range keyChecks {
		status := "TẮT"
		val, ok := parsed["litespeed."+kc.key]
		if !ok {
			val, ok = parsed[kc.key]
		}
		if ok {
			if num, ok := val.(float64); ok && num > 0 {
				status = "BẬT"
			} else if b, ok := val.(bool); ok && b {
				status = "BẬT"
			} else if s, ok := val.(string); ok && (s == "1" || s == "true" || s == "on") {
				status = "BẬT"
			}
		}
		sb.WriteString(fmt.Sprintf("  • %-26s: %s\n", kc.name, status))
	}

	return sb.String(), nil
}
