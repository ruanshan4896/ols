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

## 2. Hướng dẫn Triển khai lên VPS mới (Từng bước chi tiết)

Bạn có thể triển khai công cụ lên bất kỳ VPS Linux mới nào (Ubuntu, Debian, AlmaLinux, RockyLinux...) theo các bước sau:

### Bước 1: Chuẩn bị VPS
Đảm bảo VPS đã cài đặt Docker và Docker Compose. Nếu dùng VPS có aaPanel, bạn chỉ cần cài Docker từ App Store của aaPanel (không cài LAMP/LNMP để tránh chiếm dụng port 80/443).

### Bước 2: Tải công cụ lên VPS
**Cách A — Copy trực tiếp từ máy tính lên VPS:**
Từ máy tính của bạn, mở PowerShell / Terminal và copy file binary đã biên dịch sẵn lên VPS:
```bash
# Đối với VPS x86_64 thông dụng (Intel / AMD):
scp bin/ols-cli-linux-amd64 root@<IP_VPS>:/usr/local/bin/ols

# Hoặc đối với VPS ARM (Oracle Ampere, AWS Graviton):
scp bin/ols-cli-linux-arm64 root@<IP_VPS>:/usr/local/bin/ols
```

**Cách B — Tải lên qua giao diện File Manager của aaPanel:**
1. Mở aaPanel > **Files** > đi đến thư mục `/usr/local/bin`.
2. Bấm **Upload** file `bin/ols-cli-linux-amd64` từ máy tính lên.
3. Đổi tên file vừa tải lên thành **`ols`**.

### Bước 3: Cấp quyền thực thi và khởi tạo hạ tầng
Đăng nhập SSH vào VPS bằng tài khoản root:
```bash
# 1. Cấp quyền thực thi và tạo alias
chmod +x /usr/local/bin/ols
ln -sf /usr/local/bin/ols /usr/local/bin/ols-cli

# 2. Khởi tạo toàn bộ hạ tầng VPS (Traefik, MariaDB 11.4, Shared Redis, Docker Network)
ols init --email email-cua-ban@gmail.com
```

### Bước 4: Bắt đầu sử dụng qua Menu trực quan
Gõ lệnh sau để mở giao diện quản lý:
```bash
ols
```

---

## 3. Hướng dẫn Sử dụng

### 3.1. Giao diện Menu tương tác (Khuyến nghị sử dụng)
Chỉ cần gõ **`ols`** (hoặc `ols menu`), toàn bộ chức năng sẽ hiển thị trực quan dưới dạng menu số để bạn thao tác nhanh:

```bash
ols
```

```text
==================================================================
      HỆ THỐNG QUẢN TRỊ WORDPRESS & OPENLITESPEED (OLS-CLI)
==================================================================
  [1] Khởi tạo hạ tầng máy chủ VPS (Traefik, MariaDB, Redis, SSL)
  [2] Thêm website WordPress mới (Tự động tải mã nguồn & cấu hình)
  [3] Xem danh sách website đang chạy
  [4] Khởi động lại website (Restart)
  [5] Xóa website
  [6] Sao lưu website (Backup)
  [7] Khôi phục website từ bản sao lưu (Restore)
  [8] Quản trị Database phpMyAdmin (Bật / Tắt)
  [9] Kiểm tra trạng thái các container Docker
  [10] Đồng bộ cấu hình các website (Sync & Upgrade Config)
  [0] Thoát
==================================================================
👉 Nhập lựa chọn của bạn [0-10]: 
```

---

### 3.2. Chế độ dòng lệnh (CLI - Dùng cho Script tự động hóa)

#### Khởi tạo Hạ tầng VPS (`init`)
```bash
ols init --email your-email@example.com
```

#### Tạo website WordPress mới
```bash
# Tạo site với cấu hình mặc định (PHP 8.2 + WordPress)
ols site create example.com

# Tạo site với phiên bản PHP tùy chọn (hỗ trợ 8.1, 8.2, 8.3)
ols site create myblog.vn --php 8.3
```

#### Liệt kê danh sách website
```bash
ols site list
```

#### Khởi động lại website
```bash
ols site restart example.com
```

#### Xóa website
```bash
ols site delete example.com --force
```

#### Sao lưu website (Backup)
```bash
ols backup example.com
# File sao lưu lưu tại: /opt/ols/backups/example.com/YYYYMMDD_HHMMSS_example.com.tar.gz
```

#### Khôi phục website (Restore)
```bash
ols restore example.com /opt/ols/backups/example.com/20260907_143000_example.com.tar.gz
```

#### Quản trị Database qua phpMyAdmin
```bash
# Bật phpMyAdmin trên port 8080
ols pma enable --port 8080

# Tắt phpMyAdmin khi không sử dụng để tiết kiệm RAM
ols pma disable
```

#### Đồng bộ & Nâng cấp cấu hình website (`sync`)
Khi bạn cập nhật template vhost mới, cài thêm extension hoặc cập nhật Traefik:
```bash
# Đồng bộ một website cụ thể
ols sync example.com

# Hoặc tự động đồng bộ toàn bộ website trên VPS
ols sync --all
```

---

## 4. Tính Cô Lập & Bảo Mật Giữa Các Website (Isolation Security)

Hệ thống được thiết kế theo mô hình **Multi-Tenant Isolation** đạt chuẩn an toàn cao:

1. **Cô lập Tiến trình & Container (Process Isolation):**
   * Mỗi website chạy trong một Docker container OpenLiteSpeed hoàn toàn độc lập (`ols_<slug>`).
   * Nếu một website bị lỗi PHP fatal, crash, hoặc bị tấn công ddos/malware, tiến trình chỉ dừng lại trong phạm vi container đó, hoàn toàn không làm gián đoạn các website khác trên máy chủ.

2. **Cô lập Hệ thống Tệp (Filesystem Isolation):**
   * Mỗi container chỉ mount duy nhất thư mục web của chính nó: `/opt/ols/sites/<domain>/html`.
   * Nhờ cơ chế Linux Mount Namespaces của Docker, website A **tuyệt đối không thể đọc hay can thiệp** vào mã nguồn, tệp tin của website B.

3. **Cô lập Cơ sở Dữ liệu (Database Isolation):**
   * Sử dụng MariaDB 11.4 tập trung, nhưng mỗi website sở hữu một Database riêng (`wp_<slug>`) và một User riêng (`usr_<slug>`).
   * Phân quyền nghiêm ngặt: `GRANT ALL ON wp_<slug>.* TO 'usr_<slug>'@'%'`. User của site A hoàn toàn không có quyền truy cập hay đọc dữ liệu của site B. Mật khẩu được sinh ngẫu nhiên 32 ký tự bảo mật cao.

4. **Cô lập Bộ nhớ Đệm Redis (Object Cache Isolation):**
   * Sử dụng cụm Redis 7 dùng chung (`ols-redis`) với cấu hình bộ nhớ LRU an toàn.
   * Mỗi website được gán tiền tố key riêng biệt thông qua `define('WP_CACHE_KEY_SALT', '<slug>:')` trong `wp-config.php`, ngăn chặn hoàn toàn việc nhầm lẫn hoặc đè cache giữa các website.

5. **Phân quyền người dùng an toàn (Least Privilege):**
   * Bên trong container, OpenLiteSpeed thực thi dưới tài khoản hệ thống `nobody:nogroup` (UID `65534`). Không chạy mã PHP dưới quyền `root`, triệt tiêu nguy cơ chiếm quyền điều khiển VPS từ mã nguồn WordPress.

---

## 5. Cấu trúc Thư mục Hệ thống
```text
/opt/ols/
├── bin/
│   └── ols                           # Binary CLI
├── config/
│   └── ols.yaml                      # Cấu hình hệ thống & mật khẩu root MariaDB
├── core/
│   ├── docker-compose.yml            # Traefik v3 + MariaDB 11.4 + Redis 7
│   ├── traefik/                      # Cấu hình SSL Let's Encrypt acme.json
│   └── mariadb/                      # Dữ liệu MariaDB
├── sites/
│   └── example.com/
│       ├── docker-compose.yml        # Container OpenLiteSpeed độc lập
│       ├── ols/conf/vhost.conf       # Cấu hình Virtual Host & WordPress Rewrite Rules
│       ├── logs/                     # Access log và Error log riêng của site
│       └── html/                     # Document root WordPress (chown nobody 65534:65534)
└── backups/
    └── example.com/                  # Các bản sao lưu nén tar.gz (mã nguồn + database)
```

---

## 6. Giấy phép
Dự án được phát hành theo giấy phép [MIT](LICENSE).
