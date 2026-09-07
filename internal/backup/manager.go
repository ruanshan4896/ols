package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
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

	// 1. Xuất database qua container MariaDB
	sqlDumpPath := filepath.Join(siteDir, "database.sql")
	dumpCmd := fmt.Sprintf("mariadb-dump -uroot -p%s %s > /tmp/dump.sql && cat /tmp/dump.sql", b.cfg.DBRootPassword, dbName)
	sqlContent, err := b.dm.ExecInContainer("ols-mariadb", "sh", "-c", dumpCmd)
	if err == nil && len(sqlContent) > 0 {
		_ = os.WriteFile(sqlDumpPath, []byte(sqlContent), 0644)
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

	return nil
}
