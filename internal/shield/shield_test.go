package shield

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultShieldConfig(t *testing.T) {
	cfg := DefaultShieldConfig()
	if !cfg.BlockXMLRPC || !cfg.RateLimitLogin || !cfg.BlockUploadsPHP || !cfg.BlockUserScan || !cfg.BlockSensitiveFiles {
		t.Errorf("DefaultShieldConfig must have all protections enabled, got %+v", cfg)
	}
}

func TestDisabledShieldConfig(t *testing.T) {
	cfg := DisabledShieldConfig()
	if cfg.BlockXMLRPC || cfg.RateLimitLogin || cfg.BlockUploadsPHP || cfg.BlockUserScan || cfg.BlockSensitiveFiles {
		t.Errorf("DisabledShieldConfig must have all protections disabled, got %+v", cfg)
	}
}

func TestSaveAndGetShieldConfig(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shield-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	domain := "testsite.local"

	// 1. When file doesn't exist, should return default config
	cfg, err := GetShieldConfig(tmpDir, domain)
	if err != nil {
		t.Fatalf("GetShieldConfig failed: %v", err)
	}
	if !cfg.BlockXMLRPC {
		t.Errorf("expected default BlockXMLRPC=true")
	}

	// 2. Modify config and save
	cfg.BlockXMLRPC = false
	cfg.RateLimitLogin = false
	if err := SaveShieldConfig(tmpDir, domain, cfg); err != nil {
		t.Fatalf("SaveShieldConfig failed: %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tmpDir, "sites", domain, "shield.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Fatalf("expected shield.json to exist at %s", expectedPath)
	}

	// 3. Read back
	loaded, err := GetShieldConfig(tmpDir, domain)
	if err != nil {
		t.Fatalf("GetShieldConfig readback failed: %v", err)
	}
	if loaded.BlockXMLRPC != false || loaded.RateLimitLogin != false || loaded.BlockUploadsPHP != true {
		t.Errorf("loaded config mismatch: %+v", loaded)
	}
}

func TestApplyShieldToHtaccess(t *testing.T) {
	initialHtaccess := `# BEGIN WordPress
RewriteEngine On
RewriteRule . /index.php [L]
# END WordPress`

	cfg := DefaultShieldConfig()
	result := ApplyShieldToHtaccess(initialHtaccess, cfg)

	if !strings.Contains(result, BlockShieldStartMarker) {
		t.Errorf("expected BlockShieldStartMarker in result")
	}
	if !strings.Contains(result, `RewriteRule ^/?xmlrpc\.php$ - [F,L,NC]`) {
		t.Errorf("expected xmlrpc blocking rule in htaccess")
	}
	if !strings.Contains(result, "# BEGIN WordPress") {
		t.Errorf("expected WordPress block to be preserved")
	}

	// Disable XMLRPC
	cfg.BlockXMLRPC = false
	resultDisabledXMLRPC := ApplyShieldToHtaccess(result, cfg)
	if strings.Contains(resultDisabledXMLRPC, `xmlrpc\.php`) {
		t.Errorf("expected xmlrpc blocking rule to be removed when disabled")
	}

	// Disable all
	disabledCfg := DisabledShieldConfig()
	resultAllDisabled := ApplyShieldToHtaccess(result, disabledCfg)
	if strings.Contains(resultAllDisabled, BlockShieldStartMarker) {
		t.Errorf("expected BlockShieldStartMarker to be removed when all disabled")
	}
	if !strings.Contains(resultAllDisabled, "# BEGIN WordPress") {
		t.Errorf("expected WordPress block to still be preserved")
	}
}
