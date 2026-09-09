package template

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/ols-cli/ols/templates"
)

type CoreTemplateData struct {
	NetworkName        string
	BackendNetworkName string
	DBRootPassword     string
}

type SiteTemplateData struct {
	Domain             string
	DomainSlug         string
	PHPVersion         string
	PHPVersionShort    string
	IncludeWWW         bool
	NetworkName        string
	BackendNetworkName string
}

func RenderCoreCompose(data CoreTemplateData) (string, error) {
	return renderFile("core/docker-compose.yml.tmpl", data)
}

func RenderTraefikConfig(email string) (string, error) {
	return renderFile("core/traefik.yml.tmpl", map[string]string{"Email": email})
}

func RenderSiteCompose(data SiteTemplateData) (string, error) {
	data.PHPVersionShort = strings.ReplaceAll(data.PHPVersion, ".", "")
	data.IncludeWWW = strings.Count(data.Domain, ".") == 1
	return renderFile("site/docker-compose.yml.tmpl", data)
}

func RenderSiteVhost(domain string) (string, error) {
	return renderFile("site/vhost.conf.tmpl", map[string]string{"Domain": domain})
}

func renderFile(name string, data interface{}) (string, error) {
	content, err := templates.FS.ReadFile(name)
	if err != nil {
		return "", fmt.Errorf("read template %s: %w", name, err)
	}

	tmpl, err := template.New(name).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", name, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template %s: %w", name, err)
	}

	return buf.String(), nil
}
