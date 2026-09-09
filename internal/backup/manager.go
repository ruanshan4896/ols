package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/docker"
	"github.com/ols-cli/ols/internal/mariadb"
	"github.com/ols-cli/ols/internal/site"
)

type BackupManager struct {
	cfg *config.Config
	dm  *docker.DockerManager
}

func NewBackupManager(cfg *config.Config) *BackupManager {
	return &BackupManager{
		cfg: cfg,
		dm:  docker.NewDockerManager(),
	}
}

func FormatBackupFilename(domain string, t time.Time) string {
	return fmt.Sprintf("%s_%s.tar.gz", t.Format("20060102_150405"), domain)
}

func (b *BackupManager) BackupSite(domain string) (string, error) {
	siteDir := filepath.Join(b.cfg.SystemDir, "sites", domain)
	if _, err := os.Stat(siteDir); os.IsNotExist(err) {
		return "", fmt.Errorf("website '%s' không tồn tại", domain)
	}

	slug := site.DomainToSlug(domain)
	dbName := "wp_" + slug

	// 1. Xuất database qua container MariaDB bằng streaming trực tiếp ra file
	sqlDumpPath := filepath.Join(siteDir, "database.sql")
	dumpFile, errCreate := os.Create(sqlDumpPath)
	if errCreate == nil {
		dumpCmd := exec.Command("docker", "exec", "ols-mariadb", "mariadb-dump", "-uroot", "-p"+b.cfg.DBRootPassword, dbName)
		dumpCmd.Stdout = dumpFile
		_ = dumpCmd.Run()
		dumpFile.Close()
	}

	// 2. Tạo file tar.gz
	backupDir := filepath.Join(b.cfg.SystemDir, "backups", domain)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", err
	}

	tarPath := filepath.Join(backupDir, FormatBackupFilename(domain, time.Now()))
	tarFile, err := os.Create(tarPath)
	if err != nil {
		return "", err
	}
	defer tarFile.Close()

	gw := gzip.NewWriter(tarFile)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	_ = filepath.Walk(siteDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(siteDir, path)
		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(relPath)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			defer f.Close()
			_, _ = io.Copy(tw, f)
		}
		return nil
	})

	_ = os.Remove(sqlDumpPath)
	return tarPath, nil
}

// BackupAllSitesProgress sao lưu toàn bộ website trên hệ thống với tiến trình thời gian thực
func (b *BackupManager) BackupAllSitesProgress(progressFn func(current, total int, domain, backupPath string, err error)) ([]string, []error) {
	siteMgr := site.NewManager(b.cfg)
	sites, err := siteMgr.ListSites()
	if err != nil {
		return nil, []error{err}
	}

	var backedUp []string
	var errs []error
	total := len(sites)

	for i, s := range sites {
		path, backupErr := b.BackupSite(s.Domain)
		if backupErr != nil {
			errs = append(errs, fmt.Errorf("%s: %w", s.Domain, backupErr))
		} else {
			backedUp = append(backedUp, path)
		}
		if progressFn != nil {
			progressFn(i+1, total, s.Domain, path, backupErr)
		}
	}

	return backedUp, errs
}

// BackupAllSites sao lưu toàn bộ website
func (b *BackupManager) BackupAllSites() ([]string, []error) {
	return b.BackupAllSitesProgress(nil)
}

func (b *BackupManager) RestoreSite(domain string, backupFile string) error {
	siteDir := filepath.Join(b.cfg.SystemDir, "sites", domain)
	f, err := os.Open(backupFile)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(siteDir, filepath.FromSlash(header.Name))
		switch header.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(target, 0755)
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			_, _ = io.Copy(outFile, tr)
			outFile.Close()
		}
	}

	slug := site.DomainToSlug(domain)
	htmlDir := filepath.Join(siteDir, "html")

	// 1. Phục hồi / Khởi tạo Database & User trong MariaDB
	dbName := "wp_" + slug
	dbUser := "usr_" + slug
	dbPass := ""

	// Trích xuất DB_PASSWORD từ wp-config.php nếu có
	wpConfigPath := filepath.Join(htmlDir, "wp-config.php")
	if wpConfigBytes, err := os.ReadFile(wpConfigPath); err == nil {
		wpConfigStr := string(wpConfigBytes)
		rePass := regexp.MustCompile(`define\(\s*['"]DB_PASSWORD['"]\s*,\s*['"]([^'"]+)['"]\s*\)`)
		if matches := rePass.FindStringSubmatch(wpConfigStr); len(matches) > 1 {
			dbPass = matches[1]
		}
		reUser := regexp.MustCompile(`define\(\s*['"]DB_USER['"]\s*,\s*['"]([^'"]+)['"]\s*\)`)
		if matches := reUser.FindStringSubmatch(wpConfigStr); len(matches) > 1 {
			dbUser = matches[1]
		}
		reDB := regexp.MustCompile(`define\(\s*['"]DB_NAME['"]\s*,\s*['"]([^'"]+)['"]\s*\)`)
		if matches := reDB.FindStringSubmatch(wpConfigStr); len(matches) > 1 {
			dbName = matches[1]
		}
	}

	// Đảm bảo Database và User luôn tồn tại và có quyền truy cập
	if dbPass != "" {
		dbClient := mariadb.NewContainerClient("ols-mariadb", b.cfg.DBRootPassword)
		_ = dbClient.CreateDatabaseAndUser(dbName, dbUser, dbPass)
	}

	// 2. Nếu có database.sql, nạp vào MariaDB bằng streaming stdin (hỗ trợ DB dung lượng lớn)
	sqlDumpPath := filepath.Join(siteDir, "database.sql")
	if _, err := os.Stat(sqlDumpPath); err == nil {
		sqlFile, errOpen := os.Open(sqlDumpPath)
		if errOpen == nil {
			importCmd := exec.Command("docker", "exec", "-i", "ols-mariadb", "mariadb", "-uroot", "-p"+b.cfg.DBRootPassword, dbName)
			importCmd.Stdin = sqlFile
			_ = importCmd.Run()
			sqlFile.Close()
		}
		_ = os.Remove(sqlDumpPath)
	}

	// 3. Đảm bảo file .htaccess tồn tại để rewrite permalinks không bị 404
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

	// 4. Phân quyền chuẩn xác cho user nobody (UID 65534) của OpenLiteSpeed
	_ = exec.Command("chown", "-R", "65534:65534", htmlDir).Run()
	_ = exec.Command("chmod", "-R", "775", htmlDir).Run()
	wpContentDir := filepath.Join(htmlDir, "wp-content")
	_ = os.MkdirAll(filepath.Join(wpContentDir, "upgrade"), 0777)
	_ = os.MkdirAll(filepath.Join(wpContentDir, "uploads"), 0777)
	_ = os.MkdirAll(filepath.Join(wpContentDir, "plugins"), 0777)
	_ = exec.Command("chmod", "-R", "777", wpContentDir).Run()

	// 5. Khởi chạy stack container OpenLiteSpeed (up -d để đảm bảo tự tạo container mới nếu site từng bị xóa)
	_ = b.dm.ComposeUp(siteDir)
	_ = b.dm.ComposeRestart(siteDir)

	return nil
}
