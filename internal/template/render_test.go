package template

import (
	"strings"
	"testing"
)

func TestRenderSiteCompose(t *testing.T) {
	data := SiteTemplateData{
		Domain:             "myblog.com",
		DomainSlug:         "myblog_com",
		PHPVersion:         "8.2",
		NetworkName:        "ols-network",
		BackendNetworkName: "ols-backend-network",
	}

	out, err := RenderSiteCompose(data)
	if err != nil {
		t.Fatalf("render site compose failed: %v", err)
	}

	if !strings.Contains(out, "ols_myblog_com") {
		t.Errorf("expected container name ols_myblog_com in output, got: %s", out)
	}
	if !strings.Contains(out, "Host(`myblog.com`) || Host(`www.myblog.com`)") {
		t.Errorf("expected traefik host rule in output, got: %s", out)
	}
	if !strings.Contains(out, "litespeedtech/openlitespeed:1.8.2-lsphp82") {
		t.Errorf("expected php 8.2 image, got: %s", out)
	}
	if !strings.Contains(out, "ols-network") || !strings.Contains(out, "ols-backend-network") {
		t.Errorf("expected dual networks in output, got: %s", out)
	}
	if !strings.Contains(out, "traefik.docker.network=ols-network") {
		t.Errorf("expected traefik.docker.network label in output, got: %s", out)
	}
	if !strings.Contains(out, "myblog_com-redirect") || !strings.Contains(out, "redirectscheme.scheme=https") {
		t.Errorf("expected https redirect middleware in output, got: %s", out)
	}
	if !strings.Contains(out, "myblog_com-sec-headers") || !strings.Contains(out, "stsseconds=31536000") {
		t.Errorf("expected security headers middleware in output, got: %s", out)
	}
	if !strings.Contains(out, "Permissions-Policy=camera=()") {
		t.Errorf("expected Permissions-Policy in output, got: %s", out)
	}
}

func TestRenderCoreCompose(t *testing.T) {
	data := CoreTemplateData{
		NetworkName:        "ols-network",
		BackendNetworkName: "ols-backend-network",
		DBRootPassword:     "rootpassword123",
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
	if !strings.Contains(out, "ols-backend-network") {
		t.Errorf("expected ols-backend-network in output, got: %s", out)
	}
}

func TestRenderSiteVhost(t *testing.T) {
	out, err := RenderSiteVhost("example.com")
	if err != nil {
		t.Fatalf("render site vhost failed: %v", err)
	}

	if !strings.Contains(out, "useIpInProxyHeader        1") {
		t.Errorf("expected useIpInProxyHeader 1 in vhost, got: %s", out)
	}
	if !strings.Contains(out, "expires  {") || !strings.Contains(out, "enableExpires           1") {
		t.Errorf("expected expires block in vhost, got: %s", out)
	}
	if !strings.Contains(out, "xmlrpc") {
		t.Errorf("expected xmlrpc blocking rule in vhost, got: %s", out)
	}
}

func TestRenderTraefikConfig(t *testing.T) {
	out, err := RenderTraefikConfig("admin@example.com")
	if err != nil {
		t.Fatalf("render traefik config failed: %v", err)
	}

	if !strings.Contains(out, "admin@example.com") {
		t.Errorf("expected email in traefik config, got: %s", out)
	}
	if !strings.Contains(out, "forwardedHeaders:") || !strings.Contains(out, "trustedIPs:") {
		t.Errorf("expected trustedIPs in traefik config, got: %s", out)
	}
	if !strings.Contains(out, "173.245.48.0/20") {
		t.Errorf("expected cloudflare ip range in traefik config, got: %s", out)
	}
}
