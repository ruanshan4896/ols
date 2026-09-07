# Design Spec: WordPress & OpenLiteSpeed Multi-Site Docker Management CLI (`ols-cli`)

- **Author**: Antigravity & User
- **Date**: 2026-09-07
- **Status**: Draft (Under Review)
- **Target OS**: Linux (Ubuntu 20.04+, Debian 11+, AlmaLinux 8+)

---

## 1. Mục tiêu & Phạm vi Dự án (Goals & Scope)

### 1.1. Mục tiêu
Xây dựng một công cụ dòng lệnh (CLI tool) viết bằng **Go (Golang)** mang tên `ols-cli` để tự động hóa hoàn toàn việc khởi tạo, cấu hình, và quản trị nhiều website **WordPress** chạy trên máy chủ ảo (VPS) bằng Docker.
Mỗi website phải được vận hành độc lập, sử dụng web server **OpenLiteSpeed (OLS)** siêu tốc kết hợp với **LSCache** và **Redis Object Cache**, được cấp phát chứng chỉ SSL tự động qua **Traefik v3**, đồng thời tối ưu hóa tài nguyên RAM trên VPS bằng một máy chủ **MariaDB tập trung (Shared MariaDB)**.

### 1.2. Yêu cầu phi chức năng (Non-Functional Requirements)
* **Tính cô lập (Isolation):** Mỗi website chạy một cặp container riêng (OpenLiteSpeed + Redis riêng), sử dụng Document Root riêng biệt (`/opt/ols/sites/<domain>/html`). Sự cố ở một website không làm ảnh hưởng đến các website khác trên cùng VPS.
* **Tiết kiệm tài nguyên:** Sử dụng chung một container MariaDB 11.x thay vì mỗi website một container DB, giúp VPS cấu hình thấp (1GB - 2GB RAM) có thể chạy ổn định từ 5 - 15 websites.
* **Độc lập runtime (Self-contained binary):** Công cụ CLI được biên dịch ra một file binary duy nhất (`ols-cli`), nhúng sẵn toàn bộ template cấu hình, không yêu cầu cài đặt Go, Python hay runtime phụ thuộc trên VPS đích.
* **Dễ bảo trì & Trực quan:** Sử dụng mô hình **Docker Compose Multi-Stack**. Mỗi website sở hữu một file `docker-compose.yml` riêng biệt, người quản trị có thể dùng lệnh `ols-cli` hoặc can thiệp trực tiếp bằng `docker compose` khi cần.

---

## 2. Kiến trúc Hệ thống (Architecture Overview)

### 2.1. Sơ đồ Luồng Lưu lượng (Traffic & Network Diagram)

```
                       [Internet / Clients]
                               │
                               ▼ (Port 80 HTTP, 443 HTTPS & HTTP/3 QUIC)
┌────────────────────────────────────────────────────────────────────────┐
│ VPS Host                                                               │
│ Docker Network: `ols-network` (Bridge)                                │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Traefik v3 Reverse Proxy Container                               │  │
│  │ - Port binding: 0.0.0.0:80, 0.0.0.0:443 (TCP/UDP)                 │  │
│  │ - ACME Let's Encrypt automated HTTP-01 / TLS-ALPN-01 resolver     │  │
│  │ - Dynamic Docker Provider (tự lắng nghe Docker socket)           │  │
│  │ - Tự động redirect 80 -> 443                                     │  │
│  └──────────────────┬───────────────────────────┬───────────────────┘  │
│                     │                           │                      │
│      SNI: site-a.com│              SNI: site-b.com│                      │
│                     ▼                           ▼                      │
│  ┌──────────────────────────────┐  ┌──────────────────────────────┐    │
│  │ Site A Stack                 │  │ Site B Stack                 │    │
│  │ ┌──────────────────────────┐ │  │ ┌──────────────────────────┐ │    │
│  │ │ Container: ols_site_a    │ │  │ │ Container: ols_site_b    │ │    │
│  │ │ Image: OpenLiteSpeed     │ │  │ │ Image: OpenLiteSpeed     │ │    │
│  │ │ PHP: 8.2 (LSPHP)         │ │  │ │ PHP: 8.3 (LSPHP)         │ │    │
│  │ │ LSCache enabled          │ │  │ │ LSCache enabled          │ │    │
│  │ └────────────┬─────────────┘ │  │ └────────────┬─────────────┘ │    │
│  │              │               │  │              │               │    │
│  │              ▼               │  │              ▼               │    │
│  │ ┌──────────────────────────┐ │  │ ┌──────────────────────────┐ │    │
│  │ │ Container: redis_site_a  │ │  │ │ Container: redis_site_b  │ │    │
│  │ │ Image: redis:7-alpine    │ │  │ │ Image: redis:7-alpine    │ │    │
│  │ │ Max memory: 64MB         │ │  │ │ Max memory: 64MB         │ │    │
│  │ └──────────────────────────┘ │  │ └──────────────────────────┘ │    │
│  └──────────────┬───────────────┘  └──────────────┬───────────────┘    │
│                 │                                 │                    │
│                 │ (TCP 3306 nội bộ Docker)        │                    │
│                 ▼                                 ▼                    │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Shared MariaDB 11.x Container (`ols-mariadb`)                    │  │
│  │ - Không bind port 3306 ra host (Chỉ truy cập trong `ols-network`)│  │
│  │ - DB: `wp_site_a` (User: `usr_site_a`, Pass: random 24 chars)    │  │
│  │ - DB: `wp_site_b` (User: `usr_site_b`, Pass: random 24 chars)    │  │
│  │ - Tối ưu InnoDB Buffer Pool theo dung lượng RAM của VPS          │  │
│  └──────────────────────────────────────────────────────────────────┘  │
│                                                                        │
│  ┌──────────────────────────────────────────────────────────────────┐  │
│  │ Tùy chọn: phpMyAdmin Container (`ols-pma`)                       │  │
│  │ - Bật/tắt theo yêu cầu (`ols-cli pma enable/disable`)            │  │
│  │ - Bảo vệ bằng Traefik Basic Auth                                 │  │
│  └──────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Cấu trúc Thư mục trên VPS (`/opt/ols`)

Mọi dữ liệu, cấu hình và mã nguồn được chuẩn hóa trong thư mục gốc `/opt/ols`:

```text
/opt/ols/
├── bin/
│   └── ols-cli                       # Binary thực thi Go CLI
├── config/
│   └── ols.yaml                      # Cấu hình hệ thống (DB root pass, ACME email, v.v.)
├── core/
│   ├── docker-compose.yml            # Stack Core: Traefik + MariaDB + phpMyAdmin
│   ├── traefik/
│   │   ├── traefik.yml               # Cấu hình tĩnh Traefik
│   │   └── acme.json                 # File lưu trữ chứng chỉ SSL Let's Encrypt (chmod 600)
│   └── mariadb/
│       ├── conf.d/my.cnf             # Tối ưu RAM & buffer pool
│       └── data/                     # Volume dữ liệu database
├── sites/
│   └── <domain>/
│       ├── docker-compose.yml        # Stack riêng của website
│       ├── .env                      # Môi trường riêng (DB creds, domain, slugs)
│       ├── ols/
│       │   └── conf/vhost.conf       # Cấu hình vhost OpenLiteSpeed (Rewrite rule WP, cache)
│       └── html/                     # Thư mục mã nguồn WordPress (chown 1001:1001)
└── backups/
    └── <domain>/
        └── <timestamp>_<domain>.tar.gz # File backup (nén html/ + database sql dump)
```

---

## 4. Đặc tả Cấu hình Container & Dịch vụ

### 4.1. Tầng Core (`/opt/ols/core/docker-compose.yml`)
* **Dịch vụ Traefik:**
  * Image: `traefik:v3.1`
  * Network: `ols-network`
  * Volumes:
    * `/var/run/docker.sock:/var/run/docker.sock:ro`
    * `/opt/ols/core/traefik/traefik.yml:/etc/traefik/traefik.yml:ro`
    * `/opt/ols/core/traefik/acme.json:/etc/traefik/acme.json`
  * Ports:
    * `80:80`
    * `443:443/tcp`
    * `443:443/udp` (HTTP/3)
* **Dịch vụ MariaDB:**
  * Image: `mariadb:11.4`
  * Network: `ols-network`
  * Volumes: `/opt/ols/core/mariadb/data:/var/lib/mysql`
  * Environment: `MARIADB_ROOT_PASSWORD=${DB_ROOT_PASSWORD}`
  * Healthcheck: `mysqladmin ping -h localhost -u root -p$$MARIADB_ROOT_PASSWORD`

### 4.2. Tầng Từng Website (`/opt/ols/sites/<domain>/docker-compose.yml`)
* **Dịch vụ OpenLiteSpeed:**
  * Image: `litespeedtech/openlitespeed:1.8.2-lsphp${PHP_VERSION}` (8.1, 8.2, 8.3)
  * Container Name: `ols_${DOMAIN_SLUG}`
  * Volumes:
    * `./html:/var/www/vhosts/localhost/html`
    * `./ols/conf/vhost.conf:/usr/local/lsws/conf/vhosts/localhost/vhconf.conf`
  * Labels Traefik:
    * `traefik.enable=true`
    * `traefik.http.routers.${DOMAIN_SLUG}.rule=Host(\`${DOMAIN}\`) || Host(\`www.${DOMAIN}\`)`
    * `traefik.http.routers.${DOMAIN_SLUG}.entrypoints=websecure`
    * `traefik.http.routers.${DOMAIN_SLUG}.tls.certresolver=letsencrypt`
    * `traefik.http.services.${DOMAIN_SLUG}.loadbalancer.server.port=80`
* **Dịch vụ Redis:**
  * Image: `redis:7-alpine`
  * Container Name: `redis_${DOMAIN_SLUG}`
  * Command: `redis-server --maxmemory 64mb --maxmemory-policy allkeys-lru --requirepass ${REDIS_PASS}`

---

## 5. Đặc tả Công cụ Go CLI (`ols-cli`)

### 5.1. Cấu trúc Source Code
```text
ols/
├── cmd/
│   ├── root.go             # Cobra Root command
│   ├── init.go             # ols-cli init
│   ├── site.go             # ols-cli site [create|delete|list|start|stop|restart]
│   ├── backup.go           # ols-cli site [backup|restore]
│   ├── pma.go              # ols-cli pma [enable|disable]
│   └── version.go          # ols-cli version
├── internal/
│   ├── config/             # Đọc/ghi /opt/ols/config/ols.yaml
│   ├── docker/             # Wrapper gọi docker compose & docker command
│   ├── mariadb/            # Client thực thi SQL tạo DB, User, Grant, Drop
│   ├── site/               # Lifecycle website (create, delete, list, rollback)
│   ├── wp/                 # Tải WP core, sinh wp-config.php, inject Redis
│   └── ui/                 # Format terminal, màu sắc, table view
├── templates/              # //go:embed
│   ├── core/
│   │   ├── docker-compose.yml.tmpl
│   │   └── traefik.yml.tmpl
│   └── site/
│       ├── docker-compose.yml.tmpl
│       ├── vhost.conf.tmpl
│       └── env.tmpl
├── go.mod
├── go.sum
└── main.go
```

### 5.2. Danh sách Lệnh và Flags

| Lệnh CLI | Tham số / Flags | Mô tả chi tiết |
|---|---|---|
| `ols-cli init` | `--email <email>` (bắt buộc cho SSL) | Kiểm tra Docker, tạo thư mục `/opt/ols`, tạo mạng `ols-network`, khởi chạy Traefik & MariaDB. |
| `ols-cli site create <domain>` | `--php=8.2`, `--redis=true`, `--wp=true` | Tạo website mới: database, vhost, wp-config, Redis container, phân quyền và khởi chạy stack. |
| `ols-cli site list` | Không | Hiển thị bảng toàn bộ website: Tên domain, Trạng thái (Running/Stopped), PHP version, Redis, SSL status. |
| `ols-cli site delete <domain>` | `--force` | Dừng và xóa container của site, xóa database + user MariaDB, xóa thư mục mã nguồn. |
| `ols-cli site start <domain>` | Không | Khởi động lại container của website (`docker compose start`). |
| `ols-cli site stop <domain>` | Không | Dừng container của website (`docker compose stop`). |
| `ols-cli site restart <domain>` | Không | Khởi động lại container của website (`docker compose restart`). |
| `ols-cli site backup <domain>` | Không | Xuất database thành `.sql` và nén toàn bộ mã nguồn vào `/opt/ols/backups/<domain>/`. |
| `ols-cli site restore <domain>` | `<backup_file.tar.gz>` | Khôi phục mã nguồn và database từ file backup. |
| `ols-cli pma enable` | `--port 8080` | Kích hoạt phpMyAdmin truy cập trực quan database. |
| `ols-cli pma disable` | Không | Dừng và tắt phpMyAdmin để giải phóng tài nguyên. |

### 5.3. Quy trình Tự phục hồi Lỗi (Automated Rollback on Error)
Khi chạy `site create <domain>`, nếu xảy ra lỗi ở bất kỳ bước nào (ví dụ: tạo database thất bại, không tải được WordPress, hoặc container OLS không khởi động được trong 30 giây):
1. Dừng và xóa các container `ols_<domain>` và `redis_<domain>` vừa tạo.
2. Xóa database `wp_<domain_slug>` và user `usr_<domain_slug>` trên MariaDB.
3. Xóa thư mục `/opt/ols/sites/<domain>/`.
4. In ra thông báo lỗi chi tiết trên terminal, đảm bảo không để lại tài nguyên rác (dangling resources).

---

## 6. Kế hoạch Kiểm thử & Xác minh (Verification Plan)

### 6.1. Unit Tests
* Kiểm thử render các template (`core/docker-compose.yml`, `site/docker-compose.yml`, `vhost.conf`, `.env`) với các bộ dữ liệu khác nhau.
* Kiểm thử hàm chuẩn hóa domain (`domain_slug`), hàm sinh mật khẩu ngẫu nhiên an toàn.
* Kiểm thử hàm kiểm tra tính hợp lệ của domain name và phiên bản PHP.

### 6.2. Integration & E2E Tests
1. **Kiểm thử `ols-cli init`**:
   * Chạy lệnh init -> Kiểm tra mạng `ols-network` tồn tại.
   * Kiểm tra container `ols-traefik` và `ols-mariadb` có trạng thái `Up (healthy)`.
   * Kiểm tra file `/opt/ols/config/ols.yaml` được tạo với mật khẩu root MariaDB.
2. **Kiểm thử `ols-cli site create`**:
   * Tạo site `test.example.com` với `--php=8.2 --redis=true`.
   * Kiểm tra 2 container `ols_test_example_com` và `redis_test_example_com` đang chạy.
   * Kiểm tra truy vấn MariaDB xem database và user đã được tạo với quyền hợp lệ.
   * Kiểm tra file `wp-config.php` có đầy đủ thông tin DB và cấu hình Redis.
   * Kiểm tra curl HTTP trả về mã 200 từ container OLS.
3. **Kiểm thử `ols-cli site backup` & `restore`**:
   * Chạy backup -> Xác nhận file `.tar.gz` chứa đầy đủ file source và file `dump.sql`.
   * Sửa đổi dữ liệu, chạy restore -> Xác nhận dữ liệu được phục hồi về trạng thái ban đầu.
4. **Kiểm thử `ols-cli site delete`**:
   * Chạy delete -> Xác nhận container, database, user và thư mục site đã biến mất hoàn toàn.
