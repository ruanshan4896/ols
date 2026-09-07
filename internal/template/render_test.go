package template

import (
	"strings"
	"testing"
)

func TestRenderSiteCompose(t *testing.T) {
	data := SiteTemplateData{
		Domain:      "myblog.com",
		DomainSlug:  "myblog_com",
		PHPVersion:  "8.2",
		RedisPass:   "pass123",
		NetworkName: "ols-network",
	}

	out, err := RenderSiteCompose(data)
	if err != nil {
		t.Fatalf("render site compose failed: %v", err)
	}

	if !strings.Contains(out, "ols_myblog_com") {
		t.Errorf("expected container name ols_myblog_com in output, got: %s", out)
	}
	if !strings.Contains(out, "Host(`myblog.com`)") {
		t.Errorf("expected traefik host rule in output, got: %s", out)
	}
	if !strings.Contains(out, "litespeedtech/openlitespeed:1.8.2-lsphp82") {
		t.Errorf("expected php 8.2 image, got: %s", out)
	}
}

func TestRenderCoreCompose(t *testing.T) {
	data := CoreTemplateData{
		NetworkName:    "ols-network",
		DBRootPassword: "rootpassword123",
	}

	out, err := RenderCoreCompose(data)
	if err != nil {
		t.Fatalf("render core compose failed: %v", err)
	}

	if !strings.Contains(out, "ols-traefik") {
		t.Errorf("expected ols-traefik in output, got: %s", out)
	}
	if !strings.Contains(out, "rootpassword123") {
		t.Errorf("expected db root password in output, got: %s", out)
	}
}
