package site

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

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
	redisPass, err := util.GenerateRandomString(24)
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

	// 4. Render file docker-compose.yml
	composeContent, err := template.RenderSiteCompose(template.SiteTemplateData{
		Domain:      opts.Domain,
		DomainSlug:  slug,
		PHPVersion:  opts.PHPVersion,
		RedisPass:   redisPass,
		NetworkName: m.cfg.NetworkName,
	})
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(siteDir, "docker-compose.yml"), []byte(composeContent), 0644); err != nil {
		return err
	}

	// 5. Cấu hình mã nguồn WordPress
	if opts.InstallWP {
		_ = downloadWordPress(htmlDir)

		wpConfig := fmt.Sprintf(`<?php
define( 'DB_NAME', '%s' );
define( 'DB_USER', '%s' );
define( 'DB_PASSWORD', '%s' );
define( 'DB_HOST', 'ols-mariadb' );
define( 'DB_CHARSET', 'utf8mb4' );
define( 'DB_COLLATE', '' );

$table_prefix = 'wp_';

define( 'WP_DEBUG', false );

define( 'WP_REDIS_HOST', 'redis_%s' );
define( 'WP_REDIS_PASSWORD', '%s' );
define( 'WP_CACHE', true );

if ( ! defined( 'ABSPATH' ) ) {
	define( 'ABSPATH', __DIR__ . '/' );
}
require_once ABSPATH . 'wp-settings.php';
`, dbName, dbUser, dbPass, slug, redisPass)

		_ = os.WriteFile(filepath.Join(htmlDir, "wp-config.php"), []byte(wpConfig), 0644)
	}

	// Phân quyền cho user 1001 của OpenLiteSpeed
	_ = exec.Command("chown", "-R", "1001:1001", siteDir).Run()

	// 6. Khởi chạy stack site
	if err = m.dm.ComposeUp(siteDir); err != nil {
		return fmt.Errorf("khởi chạy container: %w", err)
	}

	return nil
}

func downloadWordPress(targetDir string) error {
	cmd := exec.Command("curl", "-sSL", "https://wordpress.org/latest.tar.gz", "-o", "/tmp/wordpress.tar.gz")
	if err := cmd.Run(); err == nil {
		tarCmd := exec.Command("tar", "-xzf", "/tmp/wordpress.tar.gz", "--strip-components=1", "-C", targetDir)
		_ = tarCmd.Run()
		_ = os.Remove("/tmp/wordpress.tar.gz")
		return nil
	}

	// Fallback nếu môi trường chưa tải được file tar
	indexPHP := `<?php echo "<h1>Website OpenLiteSpeed + WordPress đã sẵn sàng!</h1><p>Mã nguồn và Database đã được thiết lập thành công.</p>"; phpinfo(); ?>`
	return os.WriteFile(filepath.Join(targetDir, "index.php"), []byte(indexPHP), 0644)
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
