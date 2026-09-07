package site

import (
	"testing"

	"github.com/ols-cli/ols/internal/config"
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
