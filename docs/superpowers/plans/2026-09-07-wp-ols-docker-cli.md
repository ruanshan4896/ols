# WordPress & OpenLiteSpeed Multi-Site Docker Management CLI (`ols-cli`) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Xây dựng công cụ dòng lệnh Go (`ols-cli`) hoàn chỉnh để tự động hóa khởi tạo, quản trị nhiều website WordPress chạy trên Docker với OpenLiteSpeed, Redis cache, Traefik SSL và Shared MariaDB trên VPS.

**Architecture:** Mô hình Docker Compose Multi-Stack. Một stack Core tập trung (Traefik + Shared MariaDB + phpMyAdmin) trên mạng `ols-network`, và mỗi website là một stack độc lập (OpenLiteSpeed + Redis) với volume và Traefik labels riêng. Go CLI quản lý vòng đời, render template nhúng và điều phối qua Docker/SQL.

**Tech Stack:** Go 1.22+, `github.com/spf13/cobra`, `github.com/fatih/color`, `github.com/olekukonko/tablewriter`, `github.com/go-sql-driver/mysql`, `gopkg.in/yaml.v3`, Docker Compose v2, Traefik v3, OpenLiteSpeed 1.8+, MariaDB 11.4.

**Spec:** `docs/superpowers/specs/2026-09-07-wp-ols-docker-cli-design.md`

## Global Constraints
- Target root path trên VPS: `/opt/ols` (các thư mục con: `bin/`, `config/`, `core/`, `sites/`, `backups/`).
- Network Docker chung: `ols-network` (driver bridge).
- UID/GID của tiến trình OpenLiteSpeed trong container: `1001:1001`.
- Không có bất kỳ placeholder nào ("TODO", "TBD"). Toàn bộ template và logic phải hoàn chỉnh.
- Mọi thao tác lỗi trong quá trình tạo site phải tự động rollback (dọn dẹp container, DB, thư mục).

---

### Task 1: Project Scaffolding & CLI Root / Version Command

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `cmd/root.go`
- Create: `cmd/version.go`
- Test: `cmd/version_test.go`

**Interfaces:**
- Produces: `cmd.Execute() error`, `cmd.RootCmd`, `cmd.Version`

- [ ] **Step 1: Khởi tạo module Go và tạo test kiểm tra lệnh version**

Tạo file `cmd/version_test.go`:
```go
package cmd

import (
	"bytes"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetArgs([]string{"version"})

	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error executing version command: %v", err)
	}

	output := buf.String()
	expected := "ols-cli version 0.1.0\n"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}
```

- [ ] **Step 2: Chạy test để xác nhận thất bại**

Run: `go test ./cmd/... -v`
Expected: FAIL (chưa có package cmd, go.mod)

- [ ] **Step 3: Khởi tạo `go.mod`, `main.go`, `cmd/root.go`, `cmd/version.go`**

Chạy lệnh: `go mod init github.com/ols-cli/ols` và cài đặt dependency:
`go get github.com/spf13/cobra@v1.8.1 github.com/fatih/color@v1.17.0 github.com/olekukonko/tablewriter@v0.0.5 github.com/go-sql-driver/mysql@v1.8.1 gopkg.in/yaml.v3@v3.0.1`

Tạo `cmd/root.go`:
```go
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "ols-cli",
	Short: "WordPress & OpenLiteSpeed Docker Multi-Site Management CLI",
	Long:  `ols-cli là công cụ tự động hóa quản lý nhiều website WordPress độc lập với OpenLiteSpeed và Docker trên VPS.`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

Tạo `cmd/version.go`:
```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "0.1.0"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Hiển thị phiên bản của ols-cli",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "ols-cli version %s\n", Version)
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
}
```

Tạo `main.go`:
```go
package main

import "github.com/ols-cli/ols/cmd"

func main() {
	cmd.Execute()
}
```

- [ ] **Step 4: Chạy lại test để xác nhận pass**

Run: `go test ./cmd/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum main.go cmd/
git commit -m "feat: initialize Go CLI scaffolding with root and version commands"
```

---

### Task 2: Config Management (`internal/config`)

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces:
  ```go
  type Config struct {
      SystemDir     string `yaml:"system_dir"`
      ACMEEmail     string `yaml:"acme_email"`
      DBRootPassword string `yaml:"db_root_password"`
      DefaultPHP    string `yaml:"default_php"`
      NetworkName   string `yaml:"network_name"`
  }
  func LoadConfig(path string) (*Config, error)
  func SaveConfig(path string, cfg *Config) error
  func DefaultConfig() *Config
  ```

- [ ] **Step 1: Viết test cho `internal/config`**

Tạo `internal/config/config_test.go`:
```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "ols.yaml")

	cfg := DefaultConfig()
	cfg.ACMEEmail = "admin@example.com"
	cfg.DBRootPassword = "secretpassword123"

	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if loaded.ACMEEmail != "admin@example.com" {
		t.Errorf("expected email admin@example.com, got %s", loaded.ACMEEmail)
	}
	if loaded.DBRootPassword != "secretpassword123" {
		t.Errorf("expected password secretpassword123, got %s", loaded.DBRootPassword)
	}
	if loaded.SystemDir != "/opt/ols" {
		t.Errorf("expected system dir /opt/ols, got %s", loaded.SystemDir)
	}
}
```

- [ ] **Step 2: Chạy test để xác nhận thất bại**

Run: `go test ./internal/config/... -v`
Expected: FAIL with "cannot find package"

- [ ] **Step 3: Triển khai mã nguồn `internal/config/config.go`**

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	SystemDir      string `yaml:"system_dir"`
	ACMEEmail      string `yaml:"acme_email"`
	DBRootPassword string `yaml:"db_root_password"`
	DefaultPHP     string `yaml:"default_php"`
	NetworkName    string `yaml:"network_name"`
}

func DefaultConfig() *Config {
	return &Config{
		SystemDir:   "/opt/ols",
		DefaultPHP:  "8.2",
		NetworkName: "ols-network",
	}
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse yaml config: %w", err)
	}

	return cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal yaml config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
```

- [ ] **Step 4: Chạy lại test để xác nhận pass**

Run: `go test ./internal/config/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat: add configuration manager for ols-cli"
```

---

### Task 3: Embedded Templates Engine (`templates/` & `internal/template`)

**Files:**
- Create: `templates/templates.go`
- Create: `templates/core/docker-compose.yml.tmpl`
- Create: `templates/core/traefik.yml.tmpl`
- Create: `templates/site/docker-compose.yml.tmpl`
- Create: `templates/site/vhost.conf.tmpl`
- Create: `templates/site/env.tmpl`
- Create: `internal/template/render.go`
- Test: `internal/template/render_test.go`

**Interfaces:**
- Produces:
  ```go
  type CoreTemplateData struct {
      NetworkName    string
      DBRootPassword string
  }
  type SiteTemplateData struct {
      Domain      string
      DomainSlug  string
      PHPVersion  string
      RedisPass   string
      NetworkName string
  }
  func RenderCoreCompose(data CoreTemplateData) (string, error)
  func RenderTraefikConfig(email string) (string, error)
  func RenderSiteCompose(data SiteTemplateData) (string, error)
  func RenderSiteVhost(domain string) (string, error)
  func RenderSiteEnv(data map[string]string) (string, error)
  ```

- [ ] **Step 1: Viết test render các template trong `internal/template/render_test.go`**

```go
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
```

- [ ] **Step 2: Chạy test để xác nhận thất bại**

Run: `go test ./internal/template/... -v`
Expected: FAIL

- [ ] **Step 3: Tạo các template nhúng và hàm render**

Tạo `templates/templates.go`:
```go
package templates

import "embed"

//go:embed core/* site/*
var FS embed.FS
```

Tạo `templates/core/docker-compose.yml.tmpl`:
```yaml
services:
  traefik:
    image: traefik:v3.1
    container_name: ols-traefik
    restart: always
    ports:
      - "80:80"
      - "443:443/tcp"
      - "443:443/udp"
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./traefik/traefik.yml:/etc/traefik/traefik.yml:ro
      - ./traefik/acme.json:/etc/traefik/acme.json
    networks:
      - {{ .NetworkName }}

  mariadb:
    image: mariadb:11.4
    container_name: ols-mariadb
    restart: always
    environment:
      MARIADB_ROOT_PASSWORD: "{{ .DBRootPassword }}"
    volumes:
      - ./mariadb/data:/var/lib/mysql
    networks:
      - {{ .NetworkName }}
    healthcheck:
      test: ["CMD", "healthcheck.sh", "--connect", "--innodb_initialized"]
      interval: 5s
      timeout: 3s
      retries: 10

networks:
  {{ .NetworkName }}:
    external: true
```

Tạo `templates/core/traefik.yml.tmpl`:
```yaml
global:
  checkNewVersion: false
  sendAnonymousUsage: false

entryPoints:
  web:
    address: ":80"
    http:
      redirections:
        entryPoint:
          to: websecure
          scheme: https
  websecure:
    address: ":443"
    http3: {}

providers:
  docker:
    exposedByDefault: false
    network: ols-network

certificatesResolvers:
  letsencrypt:
    acme:
      email: "{{ .Email }}"
      storage: "/etc/traefik/acme.json"
      httpChallenge:
        entryPoint: web
```

Tạo `templates/site/docker-compose.yml.tmpl`:
```yaml
services:
  ols:
    image: litespeedtech/openlitespeed:1.8.2-lsphp{{ .PHPVersionShort }}
    container_name: ols_{{ .DomainSlug }}
    restart: always
    environment:
      - TZ=Asia/Ho_Chi_Minh
    volumes:
      - ./html:/var/www/vhosts/localhost/html
      - ./ols/conf/vhost.conf:/usr/local/lsws/conf/vhosts/localhost/vhconf.conf
    networks:
      - {{ .NetworkName }}
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.{{ .DomainSlug }}.rule=Host(`{{ .Domain }}`) || Host(`www.{{ .Domain }}`)"
      - "traefik.http.routers.{{ .DomainSlug }}.entrypoints=websecure"
      - "traefik.http.routers.{{ .DomainSlug }}.tls.certresolver=letsencrypt"
      - "traefik.http.services.{{ .DomainSlug }}.loadbalancer.server.port=80"
    depends_on:
      - redis

  redis:
    image: redis:7-alpine
    container_name: redis_{{ .DomainSlug }}
    restart: always
    command: ["redis-server", "--maxmemory", "64mb", "--maxmemory-policy", "allkeys-lru", "--requirepass", "{{ .RedisPass }}"]
    networks:
      - {{ .NetworkName }}

networks:
  {{ .NetworkName }}:
    external: true
```

Tạo `templates/site/vhost.conf.tmpl`:
```text
docRoot                   $VH_ROOT/html/
enableGzip                1
enableBr                  1

errorlog $VH_ROOT/logs/error.log {
  useServer               1
  logLevel                DEBUG
  rollingSize             10M
}

accesslog $VH_ROOT/logs/access.log {
  useServer               0
  rollingSize             10M
  keepDays                30
  compressArchive         1
}

index  {
  useServer               0
  indexFiles              index.php, index.html
}

rewrite  {
  enable                  1
  autoLoadHtaccess        1
  logLevel                0
}

module cache {
  ls_enabled              1
}
```

Tạo `internal/template/render.go`:
```go
package template

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/ols-cli/ols/templates"
)

type CoreTemplateData struct {
	NetworkName    string
	DBRootPassword string
}

type SiteTemplateData struct {
	Domain          string
	DomainSlug      string
	PHPVersion      string
	PHPVersionShort string
	RedisPass       string
	NetworkName     string
}

func RenderCoreCompose(data CoreTemplateData) (string, error) {
	return renderFile("core/docker-compose.yml.tmpl", data)
}

func RenderTraefikConfig(email string) (string, error) {
	return renderFile("core/traefik.yml.tmpl", map[string]string{"Email": email})
}

func RenderSiteCompose(data SiteTemplateData) (string, error) {
	data.PHPVersionShort = strings.ReplaceAll(data.PHPVersion, ".", "")
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
```

- [ ] **Step 4: Chạy lại test để xác nhận pass**

Run: `go test ./internal/template/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add templates/ internal/template/
git commit -m "feat: implement embedded templates engine for core and site stacks"
```

---

### Task 4: MariaDB Database Manager (`internal/mariadb`)

**Files:**
- Create: `internal/mariadb/client.go`
- Test: `internal/mariadb/client_test.go`

**Interfaces:**
- Produces:
  ```go
  type Client struct { ... }
  func NewClient(host string, port int, user, password string) (*Client, error)
  func (c *Client) CreateDatabaseAndUser(dbName, username, password string) error
  func (c *Client) DropDatabaseAndUser(dbName, username string) error
  func (c *Client) Ping() error
  func (c *Client) Close() error
  ```

- [ ] **Step 1: Viết test unit kiểm tra format câu lệnh SQL an toàn**

Tạo `internal/mariadb/client_test.go`:
```go
package mariadb

import "testing"

func TestSanitizeIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"wp_site_a", "wp_site_a"},
		{"wp-site.a", "wp_site_a"},
		{"drop;database", "dropdatabase"},
	}

	for _, tt := range tests {
		actual := SanitizeIdentifier(tt.input)
		if actual != tt.expected {
			t.Errorf("SanitizeIdentifier(%q) = %q; want %q", tt.input, actual, tt.expected)
		}
	}
}
```

- [ ] **Step 2: Chạy test để xác nhận thất bại**

Run: `go test ./internal/mariadb/... -v`
Expected: FAIL

- [ ] **Step 3: Triển khai `internal/mariadb/client.go`**

```go
package mariadb

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type Client struct {
	db *sql.DB
}

func SanitizeIdentifier(name string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	cleaned := re.ReplaceAllString(name, "_")
	return strings.Trim(cleaned, "_")
}

func NewClient(host string, port int, user, password string) (*Client, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=5s", user, password, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql connection: %w", err)
	}

	return &Client{db: db}, nil
}

func (c *Client) Ping() error {
	return c.db.Ping()
}

func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) CreateDatabaseAndUser(dbName, username, password string) error {
	cleanDB := SanitizeIdentifier(dbName)
	cleanUser := SanitizeIdentifier(username)

	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;", cleanDB)); err != nil {
		return fmt.Errorf("create database: %w", err)
	}

	if _, err := tx.Exec(fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'%%' IDENTIFIED BY '%s';", cleanUser, password)); err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	if _, err := tx.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%%';", cleanDB, cleanUser)); err != nil {
		return fmt.Errorf("grant privileges: %w", err)
	}

	if _, err := tx.Exec("FLUSH PRIVILEGES;"); err != nil {
		return fmt.Errorf("flush privileges: %w", err)
	}

	return tx.Commit()
}

func (c *Client) DropDatabaseAndUser(dbName, username string) error {
	cleanDB := SanitizeIdentifier(dbName)
	cleanUser := SanitizeIdentifier(username)

	_, _ = c.db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", cleanDB))
	_, _ = c.db.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'%%';", cleanUser))
	_, _ = c.db.Exec("FLUSH PRIVILEGES;")
	return nil
}
```

- [ ] **Step 4: Chạy lại test để xác nhận pass**

Run: `go test ./internal/mariadb/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/mariadb/
git commit -m "feat: add MariaDB database and user management module"
```

---

### Task 5: Docker & Compose Wrapper (`internal/docker`)

**Files:**
- Create: `internal/docker/client.go`
- Test: `internal/docker/client_test.go`

**Interfaces:**
- Produces:
  ```go
  type DockerManager struct { ... }
  func NewDockerManager() *DockerManager
  func (d *DockerManager) EnsureNetwork(name string) error
  func (d *DockerManager) ComposeUp(dir string) error
  func (d *DockerManager) ComposeDown(dir string, removeVolumes bool) error
  func (d *DockerManager) ComposeRestart(dir string) error
  func (d *DockerManager) ExecInContainer(container string, cmd ...string) (string, error)
  func (d *DockerManager) IsContainerRunning(container string) bool
  ```

- [ ] **Step 1: Viết test cho Docker CLI argument builders**

Tạo `internal/docker/client_test.go`:
```go
package docker

import "testing"

func TestBuildComposeArgs(t *testing.T) {
	args := buildComposeArgs("/opt/ols/core", "up", "-d")
	expected := []string{"compose", "--project-directory", "/opt/ols/core", "up", "-d"}
	if len(args) != len(expected) {
		t.Fatalf("expected len %d, got %d", len(expected), len(args))
	}
	for i := range args {
		if args[i] != expected[i] {
			t.Errorf("arg[%d] expected %s, got %s", i, expected[i], args[i])
		}
	}
}
```

- [ ] **Step 2: Chạy test để xác nhận thất bại**

Run: `go test ./internal/docker/... -v`
Expected: FAIL

- [ ] **Step 3: Triển khai `internal/docker/client.go`**

```go
package docker

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

type DockerManager struct{}

func NewDockerManager() *DockerManager {
	return &DockerManager{}
}

func buildComposeArgs(dir string, subCmd ...string) []string {
	args := []string{"compose", "--project-directory", dir}
	return append(args, subCmd...)
}

func (d *DockerManager) EnsureNetwork(name string) error {
	checkCmd := exec.Command("docker", "network", "inspect", name)
	if err := checkCmd.Run(); err == nil {
		return nil // Network already exists
	}

	createCmd := exec.Command("docker", "network", "create", name)
	if out, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create docker network %s: %s (%w)", name, string(out), err)
	}
	return nil
}

func (d *DockerManager) ComposeUp(dir string) error {
	args := buildComposeArgs(dir, "up", "-d", "--remove-orphans")
	cmd := exec.Command("docker", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compose up in %s: %s (%w)", dir, string(out), err)
	}
	return nil
}

func (d *DockerManager) ComposeDown(dir string, removeVolumes bool) error {
	subCmd := []string{"down"}
	if removeVolumes {
		subCmd = append(subCmd, "-v")
	}
	args := buildComposeArgs(dir, subCmd...)
	cmd := exec.Command("docker", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compose down in %s: %s (%w)", dir, string(out), err)
	}
	return nil
}

func (d *DockerManager) ComposeRestart(dir string) error {
	args := buildComposeArgs(dir, "restart")
	cmd := exec.Command("docker", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compose restart in %s: %s (%w)", dir, string(out), err)
	}
	return nil
}

func (d *DockerManager) ExecInContainer(container string, command ...string) (string, error) {
	args := append([]string{"exec", container}, command...)
	cmd := exec.Command("docker", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (d *DockerManager) IsContainerRunning(container string) bool {
	cmd := exec.Command("docker", "inspect", "-f", "{{.State.Running}}", container)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return false
	}
	return strings.TrimSpace(out.String()) == "true"
}
```

- [ ] **Step 4: Chạy lại test để xác nhận pass**

Run: `go test ./internal/docker/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/docker/
git commit -m "feat: implement Docker and Docker Compose orchestrator wrapper"
```

---

### Task 6: System Init Command (`ols-cli init`)

**Files:**
- Create: `cmd/init.go`
- Create: `internal/util/random.go`
- Test: `internal/util/random_test.go`

**Interfaces:**
- Produces: `cmd.initCmd` with flag `--email`

- [ ] **Step 1: Viết test cho tiện ích sinh mật khẩu ngẫu nhiên an toàn**

Tạo `internal/util/random_test.go`:
```go
package util

import "testing"

func TestGeneratePassword(t *testing.T) {
	pass1, err := GenerateRandomString(24)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pass1) != 24 {
		t.Errorf("expected len 24, got %d", len(pass1))
	}

	pass2, _ := GenerateRandomString(24)
	if pass1 == pass2 {
		t.Errorf("passwords should be unique")
	}
}
```

- [ ] **Step 2: Chạy test xác nhận thất bại**

Run: `go test ./internal/util/... -v`
Expected: FAIL

- [ ] **Step 3: Triển khai `internal/util/random.go` và `cmd/init.go`**

Tạo `internal/util/random.go`:
```go
package util

import (
	"crypto/rand"
	"math/big"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRandomString(length int) (string, error) {
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))
	for i := range result {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", err
		}
		result[i] = charset[num.Int64()]
	}
	return string(result), nil
}
```

Tạo `cmd/init.go`:
```go
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/docker"
	"github.com/ols-cli/ols/internal/template"
	"github.com/ols-cli/ols/internal/util"
	"github.com/spf13/cobra"
)

var initEmail string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Khởi tạo hệ thống máy chủ: Traefik, MariaDB và Docker network",
	RunE: func(cmd *cobra.Command, args []string) error {
		if initEmail == "" {
			return fmt.Errorf("vui lòng cung cấp email Let's Encrypt qua flag --email")
		}

		color.Cyan("=== Bắt đầu khởi tạo hệ thống ols-cli ===")

		cfg := config.DefaultConfig()
		cfg.ACMEEmail = initEmail

		rootPass, err := util.GenerateRandomString(32)
		if err != nil {
			return fmt.Errorf("sinh mật khẩu root mariadb: %w", err)
		}
		cfg.DBRootPassword = rootPass

		// 1. Tạo các thư mục
		coreDir := filepath.Join(cfg.SystemDir, "core")
		dirs := []string{
			filepath.Join(cfg.SystemDir, "bin"),
			filepath.Join(cfg.SystemDir, "config"),
			filepath.Join(cfg.SystemDir, "sites"),
			filepath.Join(cfg.SystemDir, "backups"),
			coreDir,
			filepath.Join(coreDir, "traefik"),
			filepath.Join(coreDir, "mariadb", "conf.d"),
			filepath.Join(coreDir, "mariadb", "data"),
		}
		for _, d := range dirs {
			if err := os.MkdirAll(d, 0755); err != nil {
				return fmt.Errorf("tạo thư mục %s: %w", d, err)
			}
		}

		// 2. Tạo file acme.json với quyền 600
		acmePath := filepath.Join(coreDir, "traefik", "acme.json")
		if _, err := os.Stat(acmePath); os.IsNotExist(err) {
			if err := os.WriteFile(acmePath, []byte("{}"), 0600); err != nil {
				return fmt.Errorf("tạo acme.json: %w", err)
			}
		}

		// 3. Render traefik.yml
		traefikYaml, err := template.RenderTraefikConfig(cfg.ACMEEmail)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(coreDir, "traefik", "traefik.yml"), []byte(traefikYaml), 0644); err != nil {
			return err
		}

		// 4. Render core docker-compose.yml
		coreCompose, err := template.RenderCoreCompose(template.CoreTemplateData{
			NetworkName:    cfg.NetworkName,
			DBRootPassword: cfg.DBRootPassword,
		})
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(coreDir, "docker-compose.yml"), []byte(coreCompose), 0644); err != nil {
			return err
		}

		// 5. Lưu config
		configPath := filepath.Join(cfg.SystemDir, "config", "ols.yaml")
		if err := config.SaveConfig(configPath, cfg); err != nil {
			return err
		}

		// 6. Khởi động Docker Network & Stack Core
		dm := docker.NewDockerManager()
		color.Yellow("-> Tạo mạng Docker %s...", cfg.NetworkName)
		if err := dm.EnsureNetwork(cfg.NetworkName); err != nil {
			return err
		}

		color.Yellow("-> Khởi chạy container Traefik và Shared MariaDB...")
		if err := dm.ComposeUp(coreDir); err != nil {
			return err
		}

		color.Green("✓ Hệ thống đã khởi tạo thành công!")
		color.Green("  - Thư mục quản trị: %s", cfg.SystemDir)
		color.Green("  - Traefik Reverse Proxy đang lắng nghe Port 80, 443")
		color.Green("  - MariaDB Root Password đã được lưu an toàn tại: %s", configPath)

		_ = time.Second
		return nil
	},
}

func init() {
	initCmd.Flags().StringVar(&initEmail, "email", "", "Email quản trị viên dùng đăng ký SSL Let's Encrypt")
	RootCmd.AddCommand(initCmd)
}
```

- [ ] **Step 4: Chạy test xác nhận pass**

Run: `go test ./internal/util/... ./cmd/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/util/ cmd/init.go
git commit -m "feat: implement ols-cli init command to bootstrap core infrastructure"
```

---

### Task 7: Site Lifecycle Manager (`internal/site` & `cmd/site.go`)

**Files:**
- Create: `internal/site/manager.go`
- Create: `cmd/site.go`
- Test: `internal/site/manager_test.go`

**Interfaces:**
- Produces:
  ```go
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
  func (m *Manager) CreateSite(opts CreateSiteOptions) error
  func (m *Manager) DeleteSite(domain string, force bool) error
  func (m *Manager) ListSites() ([]SiteInfo, error)
  func (m *Manager) StartSite(domain string) error
  func (m *Manager) StopSite(domain string) error
  func (m *Manager) RestartSite(domain string) error
  ```

- [ ] **Step 1: Viết test cho domain validation và domain slug conversion**

Tạo `internal/site/manager_test.go`:
```go
package site

import "testing"

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
```

- [ ] **Step 2: Chạy test xác nhận thất bại**

Run: `go test ./internal/site/... -v`
Expected: FAIL

- [ ] **Step 3: Triển khai `internal/site/manager.go` và `cmd/site.go`**

Tạo `internal/site/manager.go`:
```go
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
```

Tạo `cmd/site.go`:
```go
package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/ols-cli/ols/internal/config"
	"github.com/ols-cli/ols/internal/site"
	"github.com/spf13/cobra"
)

var (
	sitePHP   string
	siteRedis bool
	siteWP    bool
	siteForce bool
)

var siteCmd = &cobra.Command{
	Use:   "site",
	Short: "Quản lý vòng đời website (create, delete, list, restart)",
}

var siteCreateCmd = &cobra.Command{
	Use:   "create [domain]",
	Short: "Tạo một website WordPress mới với OpenLiteSpeed",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return fmt.Errorf("chưa khởi tạo hệ thống (vui lòng chạy 'ols-cli init' trước): %w", err)
		}

		color.Cyan("-> Đang tạo website %s...", domain)
		mgr := site.NewManager(cfg)
		if err := mgr.CreateSite(site.CreateSiteOptions{
			Domain:     domain,
			PHPVersion: sitePHP,
			WithRedis:  siteRedis,
			InstallWP:  siteWP,
		}); err != nil {
			return err
		}

		color.Green("✓ Website %s đã được tạo và khởi chạy thành công!", domain)
		return nil
	},
}

var siteListCmd = &cobra.Command{
	Use:   "list",
	Short: "Liệt kê tất cả các website đang quản lý",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return fmt.Errorf("chưa khởi tạo hệ thống: %w", err)
		}

		mgr := site.NewManager(cfg)
		sites, err := mgr.ListSites()
		if err != nil {
			return err
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Domain", "Status"})
		for _, s := range sites {
			statusStr := s.Status
			if s.Status == "Running" {
				statusStr = color.GreenString("Running")
			} else {
				statusStr = color.RedString("Stopped")
			}
			table.Append([]string{s.Domain, statusStr})
		}
		table.Render()
		return nil
	},
}

var siteDeleteCmd = &cobra.Command{
	Use:   "delete [domain]",
	Short: "Xóa website, container và database tương ứng",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		mgr := site.NewManager(cfg)
		if err := mgr.DeleteSite(domain, siteForce); err != nil {
			return err
		}

		color.Green("✓ Đã xóa hoàn toàn website %s!", domain)
		return nil
	},
}

var siteRestartCmd = &cobra.Command{
	Use:   "restart [domain]",
	Short: "Khởi động lại container của website",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		mgr := site.NewManager(cfg)
		if err := mgr.RestartSite(domain); err != nil {
			return err
		}

		color.Green("✓ Đã khởi động lại website %s!", domain)
		return nil
	},
}

func init() {
	siteCreateCmd.Flags().StringVar(&sitePHP, "php", "8.2", "Phiên bản PHP (8.1, 8.2, 8.3)")
	siteCreateCmd.Flags().BoolVar(&siteRedis, "redis", true, "Kích hoạt Redis Object Cache riêng")
	siteCreateCmd.Flags().BoolVar(&siteWP, "wp", true, "Tự động tải và cấu hình WordPress")
	siteDeleteCmd.Flags().BoolVar(&siteForce, "force", false, "Xóa không cần hỏi lại")

	siteCmd.AddCommand(siteCreateCmd)
	siteCmd.AddCommand(siteListCmd)
	siteCmd.AddCommand(siteDeleteCmd)
	siteCmd.AddCommand(siteRestartCmd)
	RootCmd.AddCommand(siteCmd)
}
```

- [ ] **Step 4: Chạy lại test để xác nhận pass**

Run: `go test ./internal/site/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/site/ cmd/site.go
git commit -m "feat: implement site lifecycle management commands"
```

---

### Task 8: Backup & Restore Manager (`internal/backup` & `cmd/backup.go`)

**Files:**
- Create: `internal/backup/manager.go`
- Create: `cmd/backup.go`
- Test: `internal/backup/manager_test.go`

**Interfaces:**
- Produces:
  ```go
  func (m *BackupManager) BackupSite(domain string) (string, error)
  func (m *BackupManager) RestoreSite(domain string, backupFile string) error
  ```

- [ ] **Step 1: Viết test cho hàm tạo tên file backup có timestamp chuẩn**

Tạo `internal/backup/manager_test.go`:
```go
package backup

import (
	"strings"
	"testing"
	"time"
)

func TestFormatBackupFilename(t *testing.T) {
	fixedTime := time.Date(2026, 9, 7, 14, 30, 0, 0, time.UTC)
	name := FormatBackupFilename("mysite.com", fixedTime)
	expected := "20260907_143000_mysite.com.tar.gz"
	if name != expected {
		t.Errorf("expected %s, got %s", expected, name)
	}
	if !strings.HasSuffix(name, ".tar.gz") {
		t.Errorf("expected tar.gz extension")
	}
}
```

- [ ] **Step 2: Chạy test xác nhận thất bại**

Run: `go test ./internal/backup/... -v`
Expected: FAIL

- [ ] **Step 3: Triển khai `internal/backup/manager.go` và `cmd/backup.go`**

Tạo `internal/backup/manager.go`:
```go
package backup

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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
```

Tạo `cmd/backup.go`:
```go
package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/backup"
	"github.com/ols-cli/ols/internal/config"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup [domain]",
	Short: "Sao lưu toàn bộ website (mã nguồn và database) thành file nén .tar.gz",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		bm := backup.NewBackupManager(cfg)
		color.Cyan("-> Đang sao lưu website %s...", domain)
		path, err := bm.BackupSite(domain)
		if err != nil {
			return fmt.Errorf("sao lưu thất bại: %w", err)
		}

		color.Green("✓ Sao lưu thành công! File lưu trữ: %s", path)
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore [domain] [backup_file.tar.gz]",
	Short: "Khôi phục website từ bản sao lưu",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		backupFile := args[1]
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		bm := backup.NewBackupManager(cfg)
		color.Cyan("-> Đang khôi phục website %s từ %s...", domain, backupFile)
		if err := bm.RestoreSite(domain, backupFile); err != nil {
			return fmt.Errorf("khôi phục thất bại: %w", err)
		}

		color.Green("✓ Khôi phục thành công website %s!", domain)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(backupCmd)
	RootCmd.AddCommand(restoreCmd)
}
```

- [ ] **Step 4: Chạy lại test xác nhận pass**

Run: `go test ./internal/backup/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/backup/ cmd/backup.go
git commit -m "feat: implement site backup and restore commands"
```

---

### Task 9: phpMyAdmin Control Command (`cmd/pma.go`)

**Files:**
- Create: `cmd/pma.go`
- Test: `cmd/pma_test.go`

**Interfaces:**
- Produces: `cmd.pmaCmd` (`enable` / `disable`)

- [ ] **Step 1: Viết test cho cấu trúc lệnh phpMyAdmin**

Tạo `cmd/pma_test.go`:
```go
package cmd

import (
	"bytes"
	"testing"
)

func TestPMACmdHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetArgs([]string{"pma", "--help"})
	if err := RootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

- [ ] **Step 2: Chạy test xác nhận thất bại**

Run: `go test ./cmd/pma_test.go -v`
Expected: FAIL (unknown command "pma")

- [ ] **Step 3: Triển khai `cmd/pma.go`**

```go
package cmd

import (
	"fmt"
	"os/exec"

	"github.com/fatih/color"
	"github.com/ols-cli/ols/internal/config"
	"github.com/spf13/cobra"
)

var pmaPort int

var pmaCmd = &cobra.Command{
	Use:   "pma",
	Short: "Quản lý container phpMyAdmin (enable/disable)",
}

var pmaEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Khởi chạy phpMyAdmin để truy cập giao diện quản trị database",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.LoadConfig("/opt/ols/config/ols.yaml")
		if err != nil {
			return err
		}

		color.Cyan("-> Đang khởi chạy phpMyAdmin trên port %d...", pmaPort)
		runCmd := exec.Command("docker", "run", "-d",
			"--name", "ols-pma",
			"--restart", "always",
			"--network", cfg.NetworkName,
			"-p", fmt.Sprintf("%d:80", pmaPort),
			"-e", "PMA_HOST=ols-mariadb",
			"phpmyadmin:latest",
		)
		if out, err := runCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("không thể khởi chạy phpmyadmin: %s (%w)", string(out), err)
		}

		color.Green("✓ phpMyAdmin đã sẵn sàng tại http://<IP_VPS>:%d", pmaPort)
		return nil
	},
}

var pmaDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Dừng và tắt phpMyAdmin để tiết kiệm RAM",
	RunE: func(cmd *cobra.Command, args []string) error {
		_ = exec.Command("docker", "rm", "-f", "ols-pma").Run()
		color.Green("✓ Đã tắt phpMyAdmin thành công!")
		return nil
	},
}

func init() {
	pmaEnableCmd.Flags().IntVar(&pmaPort, "port", 8080, "Port truy cập phpMyAdmin")
	pmaCmd.AddCommand(pmaEnableCmd)
	pmaCmd.AddCommand(pmaDisableCmd)
	RootCmd.AddCommand(pmaCmd)
}
```

- [ ] **Step 4: Chạy lại test xác nhận pass**

Run: `go test ./cmd/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/pma.go cmd/pma_test.go
git commit -m "feat: implement phpMyAdmin enable and disable management"
```

---

### Task 10: Build Automation, Makefile & Cross-compilation

**Files:**
- Create: `Makefile`
- Create: `scripts/install.sh`
- Test: Build verification (`make build`)

- [ ] **Step 1: Viết `Makefile` hỗ trợ build cho Linux VPS và test toàn bộ project**

```makefile
BINARY_NAME=ols-cli
BUILD_DIR=bin

.PHONY: all build build-linux test clean

all: test build

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) main.go

build-linux:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 main.go

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)
```

- [ ] **Step 2: Viết script cài đặt 1 dòng trên VPS `scripts/install.sh`**

```bash
#!/bin/bash
set -e

echo "=== Đang cài đặt ols-cli cho Linux VPS ==="

# Kiểm tra kiến trúc CPU
ARCH=$(uname -m)
case $ARCH in
    x86_64) BINARY="ols-cli-linux-amd64" ;;
    aarch64|arm64) BINARY="ols-cli-linux-arm64" ;;
    *) echo "Kiến trúc CPU $ARCH không được hỗ trợ."; exit 1 ;;
esac

mkdir -p /opt/ols/bin
# Copy binary vào /opt/ols/bin/ols-cli
cp bin/$BINARY /opt/ols/bin/ols-cli
chmod +x /opt/ols/bin/ols-cli

# Tạo symlink toàn cục
ln -sf /opt/ols/bin/ols-cli /usr/local/bin/ols-cli

echo "✓ Cài đặt thành công! Bạn có thể sử dụng lệnh: ols-cli"
```

- [ ] **Step 3: Chạy test `make build` và verify**

Run: `go vet ./... && go test ./...`
Expected: ALL PASS

- [ ] **Step 4: Commit**

```bash
git add Makefile scripts/
git commit -m "ci: add Makefile and install script for Linux VPS deployment"
```
