package site

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/docker"
	"github.com/ols-cli/ols/internal/mariadb"
	"github.com/ols-cli/ols/internal/template"
	"github.com/ols-cli/ols/internal/util"
)

type Manager struct {
	cfg *config.Config
	dm  *docker.DockerManager
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		cfg: cfg,
		dm:  docker.NewDockerManager(),
	}
}

func DomainToSlug(domain string) string {
	cleaned := strings.ToLower(domain)
	re := regexp.MustCompile(`[^a-z0-9_]`)
	return re.ReplaceAllString(cleaned, "_")
}

func ValidateDomain(domain string) error {
	re := regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$`)
	if !re.MatchString(domain) {
		return fmt.Errorf("tên miền '%s' không hợp lệ", domain)
	}
	return nil
}

type CreateSiteOptions struct {
	Domain     string
	PHPVersion string
	WithRedis  bool
	InstallWP  bool
}

type SiteInfo struct {
	Domain     string
	Status     string
	PHPVersion string
	WithRedis  bool
}

func (m *Manager) CreateSite(opts CreateSiteOptions) (err error) {
	if err := ValidateDomain(opts.Domain); err != nil {
		return err
	}

	siteDir := filepath.Join(m.cfg.SystemDir, "sites", opts.Domain)
	if _, err := os.Stat(siteDir); !os.IsNotExist(err) {
		return fmt.Errorf("website '%s' đã tồn tại", opts.Domain)
	}

	slug := DomainToSlug(opts.Domain)
	dbName := "wp_" + slug
	dbUser := "usr_" + slug
	dbPass, err := util.GenerateRandomString(24)
	if err != nil {
		return err
	}

	// 1. Tạo database và user trực tiếp qua container MariaDB
	dbClient := mariadb.NewContainerClient("ols-mariadb", m.cfg.DBRootPassword)
	if err := dbClient.CreateDatabaseAndUser(dbName, dbUser, dbPass); err != nil {
		return fmt.Errorf("tạo database: %w", err)
	}

	// Rollback handler nếu có lỗi ở các bước sau
	defer func() {
		if err != nil {
			_ = m.dm.ComposeDown(siteDir, true)
			_ = dbClient.DropDatabaseAndUser(dbName, dbUser)
			_ = os.RemoveAll(siteDir)
		}
	}()

	// 2. Tạo thư mục site
	htmlDir := filepath.Join(siteDir, "html")
	olsConfDir := filepath.Join(siteDir, "ols", "conf")
	logsDir := filepath.Join(siteDir, "logs")
	for _, d := range []string{htmlDir, olsConfDir, logsDir} {
		if err = os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("tạo thư mục: %w", err)
		}
	}

	// 3. Render file vhost.conf
	vhostContent, err := template.RenderSiteVhost(opts.Domain)
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(olsConfDir, "vhost.conf"), []byte(vhostContent), 0644); err != nil {
		return err
	}

	// 4. Render file docker-compose.yml (chỉ gồm OpenLiteSpeed, dùng Shared Redis ở Core)
	composeContent, err := template.RenderSiteCompose(template.SiteTemplateData{
		Domain:      opts.Domain,
		DomainSlug:  slug,
		PHPVersion:  opts.PHPVersion,
		NetworkName: m.cfg.NetworkName,
	})
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(siteDir, "docker-compose.yml"), []byte(composeContent), 0644); err != nil {
		return err
	}

	// 5. Tải WordPress core và cấu hình wp-config.php đồng bộ
	if opts.InstallWP {
		// Tải và giải nén WordPress core trực tiếp
		if err = downloadWordPress(htmlDir); err != nil {
			return fmt.Errorf("tải bộ cài WordPress thất bại: %w", err)
		}

		// Sinh các chuỗi Salt bảo mật
		salts := make([]string, 8)
		for i := 0; i < 8; i++ {
			s, _ := util.GenerateRandomString(48)
			salts[i] = s
		}

		// Cấp phát tự động Redis Database ID độc quyền cho site (0 - 15)
		redisDB := m.GetSiteRedisDB(opts.Domain)

		// Tạo wp-config.php với FS_METHOD direct và cấu hình cách ly Redis độc quyền
		wpConfig := fmt.Sprintf(`<?php
define( 'DB_NAME', '%s' );
define( 'DB_USER', '%s' );
define( 'DB_PASSWORD', '%s' );
define( 'DB_HOST', 'ols-mariadb' );
define( 'DB_CHARSET', 'utf8mb4' );
define( 'DB_COLLATE', '' );

$table_prefix = 'wp_';

define( 'WP_DEBUG', false );

// Khắc phục triệt để lỗi hỏi FTP khi cài/cập nhật plugin & theme
define( 'FS_METHOD', 'direct' );

// Tự động cấu hình & cách ly tuyệt đối Redis Object Cache cho LiteSpeed Cache
define( 'LITESPEED_CONF', true );
define( 'LITESPEED_CONF__CACHE__OBJECT', true );
define( 'LITESPEED_CONF__CACHE__OBJECT_KIND', 2 ); // 2 = Redis
define( 'LITESPEED_CONF__CACHE__OBJECT_HOST', 'ols-redis' );
define( 'LITESPEED_CONF__CACHE__OBJECT_PORT', 6379 );
define( 'LITESPEED_CONF__CACHE__OBJECT_DB_ID', %d );
define( 'LSOC_PREFIX', '%s:' );

// Cấu hình cách ly cho plugin Redis Object Cache & WordPress Core
define( 'WP_REDIS_HOST', 'ols-redis' );
define( 'WP_REDIS_PORT', 6379 );
define( 'WP_REDIS_DATABASE', %d );
define( 'WP_REDIS_PREFIX', '%s:' );
define( 'WP_CACHE_KEY_SALT', '%s:' );
define( 'WP_CACHE', true );

// Authentication Unique Keys and Salts
define( 'AUTH_KEY',         '%s' );
define( 'SECURE_AUTH_KEY',  '%s' );
define( 'LOGGED_IN_KEY',    '%s' );
define( 'NONCE_KEY',        '%s' );
define( 'AUTH_SALT',        '%s' );
define( 'SECURE_AUTH_SALT', '%s' );
define( 'LOGGED_IN_SALT',   '%s' );
define( 'NONCE_SALT',       '%s' );

if ( ! defined( 'ABSPATH' ) ) {
	define( 'ABSPATH', __DIR__ . '/' );
}
require_once ABSPATH . 'wp-settings.php';
`, dbName, dbUser, dbPass, redisDB, slug, redisDB, slug, slug, salts[0], salts[1], salts[2], salts[3], salts[4], salts[5], salts[6], salts[7])

		if err = os.WriteFile(filepath.Join(htmlDir, "wp-config.php"), []byte(wpConfig), 0644); err != nil {
			return fmt.Errorf("tạo wp-config.php: %w", err)
		}
	}

	// Đảm bảo file .htaccess tồn tại cho các đường dẫn tĩnh (permalinks) WordPress
	htaccessPath := filepath.Join(htmlDir, ".htaccess")
	if _, err := os.Stat(htaccessPath); os.IsNotExist(err) {
		defaultHtaccess := `# BEGIN WordPress
<IfModule mod_rewrite.c>
RewriteEngine On
RewriteRule .* - [E=HTTP_AUTHORIZATION:%{HTTP:Authorization}]
RewriteBase /
RewriteRule ^index\.php$ - [L]
RewriteCond %{REQUEST_FILENAME} !-f
RewriteCond %{REQUEST_FILENAME} !-d
RewriteRule . /index.php [L]
</IfModule>
# END WordPress
`
		_ = os.WriteFile(htaccessPath, []byte(defaultHtaccess), 0664)
	}

	// Phân quyền cho user nobody (UID 65534) của OpenLiteSpeed và cấp toàn quyền ghi cho WordPress
	_ = exec.Command("chown", "-R", "65534:65534", htmlDir).Run()
	_ = exec.Command("chmod", "-R", "775", htmlDir).Run()
	wpContentDir := filepath.Join(htmlDir, "wp-content")
	_ = os.MkdirAll(filepath.Join(wpContentDir, "upgrade"), 0777)
	_ = os.MkdirAll(filepath.Join(wpContentDir, "uploads"), 0777)
	_ = os.MkdirAll(filepath.Join(wpContentDir, "plugins"), 0777)
	_ = exec.Command("chmod", "-R", "777", wpContentDir).Run()

	// 6. Khởi chạy stack site
	if err = m.dm.ComposeUp(siteDir); err != nil {
		return fmt.Errorf("khởi chạy container: %w", err)
	}

	// 7. Cài đặt các extension tối ưu hóa WordPress (Redis, ImageMagick, Msgpack, Igbinary, Intl) chạy ngầm
	phpShort := strings.ReplaceAll(opts.PHPVersion, ".", "")
	go func() {
		time.Sleep(3 * time.Second)
		pkgList := fmt.Sprintf("lsphp%s-redis lsphp%s-imagick lsphp%s-msgpack lsphp%s-igbinary lsphp%s-intl", phpShort, phpShort, phpShort, phpShort, phpShort)
		installCmd := fmt.Sprintf("dpkg -l | grep -q 'lsphp.*-redis' || (apt-get update -qq && apt-get install -y -qq %s && touch /tmp/lshttpd/restart.txt)", pkgList)
		_, _ = m.dm.ExecInContainer("ols_"+slug, "sh", "-c", installCmd)
	}()

	return nil
}

func downloadWordPress(targetDir string) error {
	cmd := exec.Command("curl", "-sSL", "https://wordpress.org/latest.tar.gz", "-o", "/tmp/wordpress.tar.gz")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("curl wordpress.tar.gz: %w", err)
	}
	defer os.Remove("/tmp/wordpress.tar.gz")

	tarCmd := exec.Command("tar", "-xzf", "/tmp/wordpress.tar.gz", "--strip-components=1", "-C", targetDir)
	if err := tarCmd.Run(); err != nil {
		return fmt.Errorf("tar extract wordpress: %w", err)
	}

	return nil
}

func (m *Manager) DeleteSite(domain string, force bool) error {
	siteDir := filepath.Join(m.cfg.SystemDir, "sites", domain)
	if _, err := os.Stat(siteDir); os.IsNotExist(err) {
		return fmt.Errorf("website '%s' không tồn tại", domain)
	}

	_ = m.dm.ComposeDown(siteDir, true)

	slug := DomainToSlug(domain)
	dbClient := mariadb.NewContainerClient("ols-mariadb", m.cfg.DBRootPassword)
	_ = dbClient.DropDatabaseAndUser("wp_"+slug, "usr_"+slug)

	return os.RemoveAll(siteDir)
}

func (m *Manager) ListSites() ([]SiteInfo, error) {
	sitesDir := filepath.Join(m.cfg.SystemDir, "sites")
	entries, err := os.ReadDir(sitesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []SiteInfo
	for _, entry := range entries {
		if entry.IsDir() {
			domain := entry.Name()
			slug := DomainToSlug(domain)
			status := "Stopped"
			if m.dm.IsContainerRunning("ols_" + slug) {
				status = "Running"
			}
			results = append(results, SiteInfo{
				Domain: domain,
				Status: status,
			})
		}
	}
	return results, nil
}

func (m *Manager) RestartSite(domain string) error {
	siteDir := filepath.Join(m.cfg.SystemDir, "sites", domain)
	return m.dm.ComposeRestart(siteDir)
}

func (m *Manager) SyncSite(domain string) error {
	siteDir := filepath.Join(m.cfg.SystemDir, "sites", domain)
	if _, err := os.Stat(siteDir); os.IsNotExist(err) {
		return fmt.Errorf("website '%s' không tồn tại", domain)
	}

	slug := DomainToSlug(domain)

	// 1. Cập nhật file vhost.conf mới nhất
	olsConfDir := filepath.Join(siteDir, "ols", "conf")
	_ = os.MkdirAll(olsConfDir, 0755)
	vhostContent, err := template.RenderSiteVhost(domain)
	if err != nil {
		return fmt.Errorf("render vhost: %w", err)
	}
	if err = os.WriteFile(filepath.Join(olsConfDir, "vhost.conf"), []byte(vhostContent), 0644); err != nil {
		return fmt.Errorf("ghi vhost.conf: %w", err)
	}

	// 2. Nhận diện phiên bản PHP từ docker-compose hiện tại
	phpVer := "8.2"
	composePath := filepath.Join(siteDir, "docker-compose.yml")
	if content, err := os.ReadFile(composePath); err == nil {
		re := regexp.MustCompile(`lsphp(8[1-3])`)
		if match := re.FindStringSubmatch(string(content)); len(match) > 1 {
			switch match[1] {
			case "81":
				phpVer = "8.1"
			case "82":
				phpVer = "8.2"
			case "83":
				phpVer = "8.3"
			}
		}
	}

	// 3. Cập nhật file docker-compose.yml
	composeContent, err := template.RenderSiteCompose(template.SiteTemplateData{
		Domain:      domain,
		DomainSlug:  slug,
		PHPVersion:  phpVer,
		NetworkName: m.cfg.NetworkName,
	})
	if err == nil {
		_ = os.WriteFile(composePath, []byte(composeContent), 0644)
	}

	// 4. Đảm bảo file .htaccess tồn tại
	htmlDir := filepath.Join(siteDir, "html")
	htaccessPath := filepath.Join(htmlDir, ".htaccess")
	if _, err := os.Stat(htaccessPath); os.IsNotExist(err) {
		defaultHtaccess := `# BEGIN WordPress
<IfModule mod_rewrite.c>
RewriteEngine On
RewriteRule .* - [E=HTTP_AUTHORIZATION:%{HTTP:Authorization}]
RewriteBase /
RewriteRule ^index\.php$ - [L]
RewriteCond %{REQUEST_FILENAME} !-f
RewriteCond %{REQUEST_FILENAME} !-d
RewriteRule . /index.php [L]
</IfModule>
# END WordPress
`
		_ = os.WriteFile(htaccessPath, []byte(defaultHtaccess), 0664)
	}

	// 5. Đồng bộ phân quyền user nobody (UID 65534)
	_ = exec.Command("chown", "-R", "65534:65534", htmlDir).Run()
	_ = exec.Command("chmod", "-R", "775", htmlDir).Run()
	wpContentDir := filepath.Join(htmlDir, "wp-content")
	_ = os.MkdirAll(filepath.Join(wpContentDir, "upgrade"), 0777)
	_ = os.MkdirAll(filepath.Join(wpContentDir, "uploads"), 0777)
	_ = os.MkdirAll(filepath.Join(wpContentDir, "plugins"), 0777)
	_ = exec.Command("chmod", "-R", "777", wpContentDir).Run()

	// 6. Tự động đồng bộ và cấu hình cách ly tuyệt đối Redis vào wp-config.php
	redisDB := m.GetSiteRedisDB(domain)
	wpConfigPath := filepath.Join(htmlDir, "wp-config.php")
	if configBytes, err := os.ReadFile(wpConfigPath); err == nil {
		configStr := string(configBytes)
		if !strings.Contains(configStr, "LITESPEED_CONF__CACHE__OBJECT_DB_ID") {
			redisDirectives := fmt.Sprintf(`
// Tự động cấu hình & cách ly tuyệt đối Redis Object Cache (ols-cli)
define( 'LITESPEED_CONF', true );
define( 'LITESPEED_CONF__CACHE__OBJECT', true );
define( 'LITESPEED_CONF__CACHE__OBJECT_KIND', 2 );
define( 'LITESPEED_CONF__CACHE__OBJECT_HOST', 'ols-redis' );
define( 'LITESPEED_CONF__CACHE__OBJECT_PORT', 6379 );
define( 'LITESPEED_CONF__CACHE__OBJECT_DB_ID', %d );
define( 'LSOC_PREFIX', '%s:' );

define( 'WP_REDIS_HOST', 'ols-redis' );
define( 'WP_REDIS_PORT', 6379 );
define( 'WP_REDIS_DATABASE', %d );
define( 'WP_REDIS_PREFIX', '%s:' );
define( 'WP_CACHE_KEY_SALT', '%s:' );
define( 'WP_CACHE', true );
`, redisDB, slug, redisDB, slug, slug)

			if idx := strings.Index(configStr, "require_once ABSPATH"); idx != -1 {
				newConfig := configStr[:idx] + redisDirectives + "\n" + configStr[idx:]
				_ = os.WriteFile(wpConfigPath, []byte(newConfig), 0644)
			}
		}
	}

	// 7. Tự động xóa sạch toàn bộ cache cũ đang bị lẫn lộn trên Redis
	_, _ = m.dm.ExecInContainer("ols-redis", "redis-cli", "FLUSHALL")

	// 8. Tái khởi động lại container để nạp cấu hình mới
	_ = m.dm.ComposeUp(siteDir)

	// 7. Cài đặt các extension tối ưu hóa chạy ngầm nếu chưa có
	phpShort := strings.ReplaceAll(phpVer, ".", "")
	go func() {
		time.Sleep(2 * time.Second)
		pkgList := fmt.Sprintf("lsphp%s-redis lsphp%s-imagick lsphp%s-msgpack lsphp%s-igbinary lsphp%s-intl", phpShort, phpShort, phpShort, phpShort, phpShort)
		installCmd := fmt.Sprintf("dpkg -l | grep -q 'lsphp.*-redis' || (apt-get update -qq && apt-get install -y -qq %s && touch /tmp/lshttpd/restart.txt)", pkgList)
		_, _ = m.dm.ExecInContainer("ols_"+slug, "sh", "-c", installCmd)
	}()

	return nil
}

func (m *Manager) SyncAllSites() ([]string, []error) {
	sites, err := m.ListSites()
	if err != nil {
		return nil, []error{err}
	}

	var synced []string
	var errs []error
	for _, s := range sites {
		if err := m.SyncSite(s.Domain); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", s.Domain, err))
		} else {
			synced = append(synced, s.Domain)
		}
	}
	return synced, errs
}

func (m *Manager) GetSiteRedisDB(domain string) int {
	sites, err := m.ListSites()
	if err != nil || len(sites) == 0 {
		return 0
	}
	for i, s := range sites {
		if s.Domain == domain {
			return i % 256
		}
	}
	return len(sites) % 256
}
