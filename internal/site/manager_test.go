package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/shield"
)

func TestDomainToSlug(t *testing.T) {
	tests := []struct {
		domain   string
		expected string
	}{
		{"example.com", "example_com"},
		{"blog.sub.vn", "blog_sub_vn"},
		{"SHOP-ONLINE.IO", "shop_online_io"},
	}

	for _, tt := range tests {
		actual := DomainToSlug(tt.domain)
		if actual != tt.expected {
			t.Errorf("DomainToSlug(%q) = %q; want %q", tt.domain, actual, tt.expected)
		}
	}
}

func TestValidateDomain(t *testing.T) {
	valid := []string{"example.com", "sub.domain.vn", "my-site.co.uk"}
	for _, d := range valid {
		if err := ValidateDomain(d); err != nil {
			t.Errorf("expected %s to be valid, got %v", d, err)
		}
	}

	invalid := []string{"-invalid.com", "example..com", "site with spaces.com", "site;rm -rf"}
	for _, d := range invalid {
		if err := ValidateDomain(d); err == nil {
			t.Errorf("expected %s to be invalid", d)
		}
	}
}

func TestSyncSiteNonExistent(t *testing.T) {
	cfg := &config.Config{SystemDir: t.TempDir()}
	mgr := NewManager(cfg)
	err := mgr.SyncSite("notfound.com")
	if err == nil {
		t.Errorf("expected error when syncing non-existent site")
	}
}

func TestSyncCore(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		SystemDir:      tmpDir,
		ACMEEmail:      "admin@test.com",
		DBRootPassword: "pass",
	}
	mgr := NewManager(cfg)

	// Case 1: Core dir does not exist -> returns nil gracefully
	if err := mgr.SyncCore(); err != nil {
		t.Fatalf("expected nil when core dir does not exist, got: %v", err)
	}

	// Case 2: Core dir exists with traefik.yml and docker-compose.yml
	coreDir := filepath.Join(tmpDir, "core")
	traefikDir := filepath.Join(coreDir, "traefik")
	_ = os.MkdirAll(traefikDir, 0755)
	traefikFile := filepath.Join(traefikDir, "traefik.yml")
	_ = os.WriteFile(traefikFile, []byte("old traefik"), 0644)

	coreComposeFile := filepath.Join(coreDir, "docker-compose.yml")
	_ = os.WriteFile(coreComposeFile, []byte("old compose"), 0644)

	if err := mgr.SyncCore(); err != nil {
		t.Fatalf("SyncCore failed: %v", err)
	}

	newTraefik, err := os.ReadFile(traefikFile)
	if err != nil || !strings.Contains(string(newTraefik), "admin@test.com") {
		t.Errorf("expected updated traefik.yml with email, got: %s", string(newTraefik))
	}
	newCompose, err := os.ReadFile(coreComposeFile)
	if err != nil || !strings.Contains(string(newCompose), "ols-traefik") {
		t.Errorf("expected updated docker-compose.yml with ols-traefik, got: %s", string(newCompose))
	}
}

func TestSanitizeWPConfig(t *testing.T) {
	rawConfig := `<?php
define( 'DB_NAME', 'old_db' );
define( 'DB_USER', 'old_user' );
define( 'DB_PASSWORD', 'old_pass' );
define( 'DB_HOST', 'localhost' );
$table_prefix = 'wp_';

require_once ABSPATH . 'wp-settings.php';
`
	sanitized := SanitizeWPConfig(rawConfig, "test_site", 3)

	if strings.Contains(sanitized, "define( 'DB_HOST', 'localhost' )") {
		t.Errorf("expected DB_HOST localhost to be replaced, got:\n%s", sanitized)
	}
	if !strings.Contains(sanitized, "define( 'DB_HOST', 'ols-mariadb' )") {
		t.Errorf("expected DB_HOST ols-mariadb, got:\n%s", sanitized)
	}
	if !strings.Contains(sanitized, "define( 'FS_METHOD', 'direct' )") {
		t.Errorf("expected FS_METHOD direct, got:\n%s", sanitized)
	}
	if !strings.Contains(sanitized, "define( 'LITESPEED_CONF__CACHE__OBJECT_DB_ID', 3 )") {
		t.Errorf("expected Redis DB ID 3, got:\n%s", sanitized)
	}
	if !strings.Contains(sanitized, "define( 'LSOC_PREFIX', 'test_site:' )") {
		t.Errorf("expected LSOC_PREFIX test_site:, got:\n%s", sanitized)
	}
}

func TestSanitizeHtaccess(t *testing.T) {
	// Case 1: Empty or incomplete htaccess
	badHtaccess := "# Some other plugin rules\nRewriteEngine Off"
	sanitized := SanitizeHtaccess(badHtaccess)
	if !strings.Contains(sanitized, "RewriteRule . /index.php [L]") {
		t.Errorf("expected standard WP rewrite rule, got:\n%s", sanitized)
	}

	// Case 2: Already valid WordPress htaccess
	goodHtaccess := `# BEGIN WordPress
<IfModule mod_rewrite.c>
RewriteEngine On
RewriteRule ^index\.php$ - [L]
RewriteCond %{REQUEST_FILENAME} !-f
RewriteCond %{REQUEST_FILENAME} !-d
RewriteRule . /index.php [L]
</IfModule>
# END WordPress`
	sanitizedGood := SanitizeHtaccess(goodHtaccess)
	if sanitizedGood != goodHtaccess {
		t.Errorf("expected already valid htaccess to remain untouched, got:\n%s", sanitizedGood)
	}
}

func TestCleanConflictingCacheDropins(t *testing.T) {
	tmpDir := t.TempDir()
	wpContent := filepath.Join(tmpDir, "wp-content")
	_ = os.MkdirAll(wpContent, 0755)

	// Create third-party object-cache.php (e.g. W3 Total Cache or Memcached)
	foreignDropin := filepath.Join(wpContent, "object-cache.php")
	_ = os.WriteFile(foreignDropin, []byte("<?php /* Memcached Object Cache */"), 0644)

	// Create third-party advanced-cache.php
	foreignAdv := filepath.Join(wpContent, "advanced-cache.php")
	_ = os.WriteFile(foreignAdv, []byte("<?php /* WP Super Cache */"), 0644)

	CleanConflictingCacheDropins(wpContent)

	if _, err := os.Stat(foreignDropin); !os.IsNotExist(err) {
		t.Errorf("expected foreign object-cache.php to be removed or renamed")
	}
	if _, err := os.Stat(foreignDropin + ".bak"); os.IsNotExist(err) {
		t.Errorf("expected foreign object-cache.php.bak to exist")
	}
	if _, err := os.Stat(foreignAdv); !os.IsNotExist(err) {
		t.Errorf("expected foreign advanced-cache.php to be removed or renamed")
	}
	if _, err := os.Stat(foreignAdv + ".bak"); os.IsNotExist(err) {
		t.Errorf("expected foreign advanced-cache.php.bak to exist")
	}

	// Ensure LiteSpeed dropin is NOT renamed
	lscacheDropin := filepath.Join(wpContent, "object-cache.php")
	_ = os.WriteFile(lscacheDropin, []byte("<?php /* LiteSpeed Object Cache */"), 0644)
	CleanConflictingCacheDropins(wpContent)
	if _, err := os.Stat(lscacheDropin); os.IsNotExist(err) {
		t.Errorf("expected LiteSpeed object-cache.php to remain untouched")
	}
}

func TestParseDomainList(t *testing.T) {
	raw := "site1.com\r\n  site2.net  \r\n\r\n# This is a comment\nsite3.org\n   \nsite4.vn\r\n"
	domains := ParseDomainList(raw)
	expected := []string{"site1.com", "site2.net", "site3.org", "site4.vn"}
	if len(domains) != len(expected) {
		t.Fatalf("expected %d domains, got %d: %v", len(expected), len(domains), domains)
	}
	for i, d := range domains {
		if d != expected[i] {
			t.Errorf("at index %d: expected %q, got %q", i, expected[i], d)
		}
	}
}

func TestListSitesFilterValidOnly(t *testing.T) {
	tmpDir := t.TempDir()
	sitesDir := filepath.Join(tmpDir, "sites")
	_ = os.MkdirAll(sitesDir, 0755)

	// Valid site: has docker-compose.yml
	site1 := filepath.Join(sitesDir, "valid-site1.com")
	_ = os.MkdirAll(site1, 0755)
	_ = os.WriteFile(filepath.Join(site1, "docker-compose.yml"), []byte("services:"), 0644)

	// Valid site: has html/ dir
	site2 := filepath.Join(sitesDir, "valid-site2.com")
	_ = os.MkdirAll(filepath.Join(site2, "html"), 0755)

	// Invalid site: empty folder or trash folder
	trash := filepath.Join(sitesDir, "some-trash-folder")
	_ = os.MkdirAll(trash, 0755)

	// File, not a dir
	_ = os.WriteFile(filepath.Join(sitesDir, "random.txt"), []byte("hello"), 0644)

	cfg := &config.Config{SystemDir: tmpDir}
	mgr := NewManager(cfg)
	sites, err := mgr.ListSites()
	if err != nil {
		t.Fatalf("ListSites failed: %v", err)
	}

	if len(sites) != 2 {
		t.Fatalf("expected 2 valid sites, got %d: %v", len(sites), sites)
	}
}

func TestDeployMUPlugins(t *testing.T) {
	tmpDir := t.TempDir()
	err := DeployMUPlugins(tmpDir)
	if err != nil {
		t.Fatalf("DeployMUPlugins failed: %v", err)
	}

	muFile := filepath.Join(tmpDir, "wp-content", "mu-plugins", "ols-cleanup.php")
	content, err := os.ReadFile(muFile)
	if err != nil {
		t.Fatalf("expected ols-cleanup.php to exist, got: %v", err)
	}

	if !strings.Contains(string(content), "rest_output_link_header") || !strings.Contains(string(content), "wp_shortlink_header") {
		t.Errorf("expected ols-cleanup.php to remove link headers, got: %s", string(content))
	}
}

func TestApplyShield(t *testing.T) {
	tmpDir := t.TempDir()
	domain := "shieldsite.local"
	siteDir := filepath.Join(tmpDir, "sites", domain)
	_ = os.MkdirAll(filepath.Join(siteDir, "ols", "conf"), 0755)
	_ = os.WriteFile(filepath.Join(siteDir, "docker-compose.yml"), []byte("services:\n  ols:\n    image: litespeedtech/openlitespeed:1.8.2-lsphp82\n"), 0644)

	cfg := &config.Config{SystemDir: tmpDir}
	mgr := NewManager(cfg)

	shieldCfg := shield.SiteShieldConfig{
		BlockXMLRPC:         false,
		RateLimitLogin:      true,
		BlockUploadsPHP:     true,
		BlockUserScan:       false,
		BlockSensitiveFiles: true,
	}

	// Apply shield
	if err := mgr.ApplyShield(domain, shieldCfg); err != nil {
		t.Fatalf("ApplyShield failed: %v", err)
	}

	// Verify shield.json saved
	savedCfg, err := shield.GetShieldConfig(tmpDir, domain)
	if err != nil {
		t.Fatalf("GetShieldConfig failed: %v", err)
	}
	if savedCfg.BlockXMLRPC != false || savedCfg.BlockUploadsPHP != true {
		t.Errorf("saved shield config mismatch: %+v", savedCfg)
	}

	// Verify vhost.conf rendered correctly
	vhostData, err := os.ReadFile(filepath.Join(siteDir, "ols", "conf", "vhost.conf"))
	if err != nil {
		t.Fatalf("read vhost.conf failed: %v", err)
	}
	if strings.Contains(string(vhostData), "xmlrpc.php") {
		t.Errorf("expected xmlrpc to NOT be in vhost when BlockXMLRPC=false")
	}
	if !strings.Contains(string(vhostData), "uploads") {
		t.Errorf("expected uploads rule to be in vhost when BlockUploadsPHP=true")
	}
}

