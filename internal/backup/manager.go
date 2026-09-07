package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/docker"
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

	// 1. Nếu có database.sql, nạp vào MariaDB bằng streaming stdin (hỗ trợ DB dung lượng lớn)
	sqlDumpPath := filepath.Join(siteDir, "database.sql")
	if _, err := os.Stat(sqlDumpPath); err == nil {
		dbName := "wp_" + slug
		sqlFile, errOpen := os.Open(sqlDumpPath)
		if errOpen == nil {
			importCmd := exec.Command("docker", "exec", "-i", "ols-mariadb", "mariadb", "-uroot", "-p"+b.cfg.DBRootPassword, dbName)
			importCmd.Stdin = sqlFile
			_ = importCmd.Run()
			sqlFile.Close()
		}
		_ = os.Remove(sqlDumpPath)
	}

	htmlDir := filepath.Join(siteDir, "html")

	// 2. Đảm bảo file .htaccess tồn tại để rewrite permalinks không bị 404
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

	// 3. Phân quyền chuẩn xác cho user nobody (UID 65534) của OpenLiteSpeed
	_ = exec.Command("chown", "-R", "65534:65534", htmlDir).Run()
	_ = exec.Command("chmod", "-R", "775", htmlDir).Run()
	wpContentDir := filepath.Join(htmlDir, "wp-content")
	_ = os.MkdirAll(filepath.Join(wpContentDir, "upgrade"), 0777)
	_ = os.MkdirAll(filepath.Join(wpContentDir, "uploads"), 0777)
	_ = os.MkdirAll(filepath.Join(wpContentDir, "plugins"), 0777)
	_ = exec.Command("chmod", "-R", "777", wpContentDir).Run()

	// 4. Khởi động lại container OpenLiteSpeed để nạp cấu hình mới & rewrite
	_ = b.dm.ComposeRestart(siteDir)

	return nil
}
