package site

import (
	"fmt"
	"os"
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

	// Kết nối MariaDB
	dbClient, err := mariadb.NewClient("127.0.0.1", 3306, "root", m.cfg.DBRootPassword)
	if err == nil {
		defer dbClient.Close()
		if err := dbClient.CreateDatabaseAndUser(dbName, dbUser, dbPass); err != nil {
			return fmt.Errorf("tạo database: %w", err)
		}
	}

	// Rollback handler nếu lỗi
	defer func() {
		if err != nil {
			_ = m.dm.ComposeDown(siteDir, true)
			if dbClient != nil {
				_ = dbClient.DropDatabaseAndUser(dbName, dbUser)
			}
			_ = os.RemoveAll(siteDir)
		}
	}()

	// Tạo thư mục site
	htmlDir := filepath.Join(siteDir, "html")
	olsConfDir := filepath.Join(siteDir, "ols", "conf")
	logsDir := filepath.Join(siteDir, "logs")
	for _, d := range []string{htmlDir, olsConfDir, logsDir} {
		if err = os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("tạo thư mục: %w", err)
		}
	}

	// Render file vhost.conf
	vhostContent, err := template.RenderSiteVhost(opts.Domain)
	if err != nil {
		return err
	}
	if err = os.WriteFile(filepath.Join(olsConfDir, "vhost.conf"), []byte(vhostContent), 0644); err != nil {
		return err
	}

	// Render file docker-compose.yml
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

	// Khởi chạy stack site
	if err = m.dm.ComposeUp(siteDir); err != nil {
		return fmt.Errorf("khởi chạy container: %w", err)
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
	dbClient, err := mariadb.NewClient("127.0.0.1", 3306, "root", m.cfg.DBRootPassword)
	if err == nil {
		defer dbClient.Close()
		_ = dbClient.DropDatabaseAndUser("wp_"+slug, "usr_"+slug)
	}

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
