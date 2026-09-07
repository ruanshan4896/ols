# WordPress & OpenLiteSpeed Multi-Site Docker Management CLI (`ols-cli`)

[![Go Report Card](https://goreportcard.com/badge/github.com/ols-cli/ols)](https://goreportcard.com/report/github.com/ols-cli/ols)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

**`ols-cli`** là công cụ dòng lệnh (CLI) được viết bằng **Go**, giúp tự động hóa hoàn toàn việc khởi tạo và quản trị nhiều website **WordPress** độc lập trên máy chủ ảo (VPS) bằng Docker.

Mỗi website được vận hành với:
* **Web Server OpenLiteSpeed siêu tốc** + **LSPHP** (8.1 / 8.2 / 8.3 tùy chọn).
* **Redis Object Cache riêng biệt** cho từng site.
* **Traefik v3 Reverse Proxy** tự động cấp phát và gia hạn chứng chỉ **SSL Let's Encrypt** (hỗ trợ HTTP/3 QUIC).
* **Shared MariaDB 11.x tập trung** giúp tối ưu hóa bộ nhớ RAM cho các VPS tài nguyên khiêm tốn.
* **Cơ chế sao lưu / khôi phục nhanh** trọn gói mã nguồn và database.
* **phpMyAdmin tùy chọn** bật/tắt linh hoạt.

---

## 1. Yêu cầu Hệ thống trên VPS
* **Hệ điều hành:** Linux (Ubuntu 20.04+, Debian 11+, AlmaLinux 8+)
* **Docker & Docker Compose:** Đã được cài đặt (`docker compose version`)
* **RAM:** Tối thiểu 1GB (khuyến nghị từ 2GB trở lên nếu chạy nhiều site kèm Redis)
* **Port trống:** 80 (HTTP), 443 (HTTPS)

---

## 2. Cài đặt nhanh

Tải binary `ols-cli` mới nhất và cài đặt vào hệ thống:

```bash
# Cài đặt tự động
curl -sSL https://raw.githubusercontent.com/ols-cli/ols/main/scripts/install.sh | sudo bash
```

Hoặc biên dịch trực tiếp từ mã nguồn:
```bash
git clone https://github.com/ols-cli/ols.git
cd ols
make build-linux
sudo cp bin/ols-cli-linux-amd64 /usr/local/bin/ols-cli
sudo chmod +x /usr/local/bin/ols-cli
```

---

## 3. Hướng dẫn Sử dụng

### 3.1. Khởi tạo Hạ tầng VPS (`init`)
Chạy lệnh này một lần duy nhất khi vừa thiết lập VPS:
```bash
sudo ols-cli init --email your-email@example.com
```
Lệnh này sẽ:
* Tạo cấu trúc thư mục chuẩn tại `/opt/ols/`
* Khởi tạo mạng Docker `ols-network`
* Tự sinh mật khẩu root MariaDB an toàn lưu tại `/opt/ols/config/ols.yaml`
* Khởi chạy Traefik v3 Reverse Proxy và Shared MariaDB 11.x

### 3.2. Quản lý Website

#### Tạo website WordPress mới
```bash
# Tạo site với cấu hình mặc định (PHP 8.2 + Redis + WordPress)
sudo ols-cli site create example.com

# Tạo site với phiên bản PHP tùy chọn
sudo ols-cli site create myblog.vn --php 8.3 --redis=true
```

#### Liệt kê danh sách website
```bash
sudo ols-cli site list
```

#### Khởi động lại hoặc tắt/bật website
```bash
sudo ols-cli site restart example.com
```

#### Xóa website
```bash
sudo ols-cli site delete example.com --force
```

### 3.3. Sao lưu & Khôi phục (Backup & Restore)

#### Sao lưu website
```bash
sudo ols-cli backup example.com
# File sao lưu được lưu tại: /opt/ols/backups/example.com/YYYYMMDD_HHMMSS_example.com.tar.gz
```

#### Khôi phục website
```bash
sudo ols-cli restore example.com /opt/ols/backups/example.com/20260907_143000_example.com.tar.gz
```

### 3.4. Quản trị Database qua phpMyAdmin
```bash
# Bật phpMyAdmin trên port 8080
sudo ols-cli pma enable --port 8080

# Tắt phpMyAdmin khi không sử dụng để tiết kiệm RAM
sudo ols-cli pma disable
```

---

## 4. Cấu trúc Thư mục Hệ thống
```text
/opt/ols/
├── bin/
│   └── ols-cli                       # Binary CLI
├── config/
│   └── ols.yaml                      # Cấu hình chung VPS
├── core/
│   ├── docker-compose.yml            # Traefik + MariaDB
│   ├── traefik/                      # SSL Let's Encrypt acme.json
│   └── mariadb/                      # Dữ liệu MariaDB
├── sites/
│   └── example.com/
│       ├── docker-compose.yml        # Stack OLS + Redis của website
│       ├── ols/conf/vhost.conf       # Cấu hình OpenLiteSpeed vhost & rewrite rules
│       └── html/                     # Document root WordPress (chown 1001:1001)
└── backups/
    └── example.com/                  # Các bản sao lưu nén tar.gz
```

---

## 5. Giấy phép
Dự án được phát hành theo giấy phép [MIT](LICENSE).
