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
Chỉ cần gõ **`ols`** (hoặc `ols menu`), toàn bộ chức năng sẽ hiển thị trực quan dưới dạng menu phân nhóm bo góc hiện đại, tương thích 100% với mọi Terminal (kể cả Web Terminal của aaPanel, cPanel):

```bash
ols
```

```text
╭───────────────────────────────────────────────────────────────────╮
│        HỆ THỐNG QUẢN TRỊ WORDPRESS & OPENLITESPEED (OLS-CLI)      │
╰───────────────────────────────────────────────────────────────────╯

  [ HẠ TẦNG CỐT LÕI ]
   [1] Khởi tạo máy chủ VPS (Traefik Proxy, MariaDB 11, Redis 7, SSL)

  [ QUẢN LÝ WEBSITE ]
   [2] Thêm website WordPress mới (Tự động tải WP core & vhost OLS)
   [3] Xem danh sách website đang chạy
   [4] Khởi động lại website (Restart container OLS)
   [5] Xóa website (Xóa container, mã nguồn, database & SSL)

  [ SAO LƯU & BẢO MẬT ]
   [6] Sao lưu website (Backup 1 site hoặc tất cả website)
   [7] Khôi phục website từ bản sao lưu (Restore .tar.gz)
   [8] Quản trị Database phpMyAdmin (Bật / Tắt qua web port 8080)
   [9] Kiểm tra trạng thái các container Docker

  [ TỐI ƯU, GIÁM SÁT & DEBUG ]
  [10] Đồng bộ cấu hình các website (Sync vhost, cache & Traefik)
  [11] Bảo mật: Làm mới Salt Keys & Đổi pass Admin (WordPress.org API)
  [12] Quản lý bộ nhớ Swap RAM (Tạo Swap 2-8GB chống sập VPS)
  [13] Đánh giá tải VPS & Tính số website có thể cài thêm
  [14] Xem nhật ký lỗi & Hỗ trợ Debug (Traefik, DB, PHP Error & Quét lỗi)
  [15] Lá chắn bảo vệ OLS Shield (Chống brute-force, khóa XML-RPC & Uploads)

  [ HỆ THỐNG ]
   [0] Thoát
─────────────────────────────────────────────────────────────────────
Nhập lựa chọn của bạn [0-15]: 
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
# Sao lưu một website cụ thể
ols backup example.com
# File sao lưu lưu tại: /opt/ols/backups/example.com/YYYYMMDD_HHMMSS_example.com.tar.gz

# Hoặc tự động sao lưu toàn bộ website trên VPS
ols backup --all
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

#### Bảo mật: Làm mới Salt Keys & Đổi mật khẩu Admin (`security`)
Tự động lấy 8 Authentication Salts & Keys mới từ **WordPress.org API chính chủ** để đăng xuất toàn bộ phiên đăng nhập của hacker, đồng thời đặt lại mật khẩu admin:
```bash
# Bảo mật cho 1 site (làm mới salts và tự sinh mật khẩu mới)
ols site security example.com

# Hoặc chỉ định tài khoản và mật khẩu admin cụ thể
ols site security example.com --user admin --pass "MatKhauMoi@123"

# Hoặc áp dụng bảo mật hàng loạt cho TẤT CẢ website trên VPS
ols site security --all
```

#### Quản lý bộ nhớ Swap RAM chống sập VPS (`swap`)
Tạo Swap RAM dự phòng từ ổ cứng SSD để ngăn ngừa MariaDB / container bị OOM-Kill khi tải đột biến:
```bash
# Xem thông tin dung lượng Swap hiện tại trên VPS
ols swap --status

# Tạo hoặc đổi dung lượng Swap (ví dụ: 2GB hoặc 4GB)
ols swap 2
ols swap --size 4

# Tắt và xóa Swap khi không cần thiết
ols swap --off
```

#### Đánh giá năng lực & Tính số website có thể cài thêm (`capacity`)
Tự động đo đạc tài nguyên thực tế của VPS (RAM vật lý, Swap, Dung lượng đĩa, mức tiêu thụ RAM trung bình của các website đang chạy) và dự báo số website có thể thêm mới an toàn:
```bash
# Đo lường và hiển thị báo cáo năng lực chịu tải VPS
ols capacity
```

#### Xem nhật ký lỗi & Hỗ trợ Debug (`logs`)
Dễ dàng tra cứu nhật ký và lọc nhanh các cảnh báo lỗi (ERROR/FATAL) của Traefik SSL, MariaDB Database, Redis Cache hoặc từng website mà không cần lục tìm file log phức tạp:
```bash
# Quét nhanh tất cả các lỗi gần nhất trên toàn hệ thống
ols logs --scan

# Xem 50 dòng log gần nhất của Traefik SSL & Gateway
ols logs traefik

# Xem log cơ sở dữ liệu MariaDB hoặc Redis
ols logs mariadb
ols logs redis

# Xem lỗi PHP Fatal / 500 của một website cụ thể (tùy chọn -n số dòng)
ols logs example.com -n 100

# Xóa sạch toàn bộ log cũ của các container và website, reset về 0 byte
ols logs --clear
```

#### Quản lý lá chắn bảo mật OLS Shield (`shield`)
Phòng thủ đa tầng ngay tại OpenLiteSpeed và Traefik, ngăn chặn botnet dò pass, khóa XML-RPC và cấm thực thi webshell trong thư mục upload:
```bash
# Xem trạng thái lá chắn trên toàn bộ các website
ols shield status

# Xem chi tiết cấu hình lá chắn của một website cụ thể
ols shield example.com

# Kích hoạt chế độ phòng thủ toàn diện (Under Attack) cho một site
ols shield enable example.com

# Kích hoạt chế độ phòng thủ toàn diện cho TẤT CẢ website trên VPS
ols shield enable --all

# Tắt tạm thời các lớp phòng thủ (chế độ debug gỡ lỗi)
ols shield disable example.com
ols shield disable --all
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

6. **Cô lập Mạng 2 Lớp (Dual-Network Architecture):**
   * Tách biệt hoàn toàn thành 2 mạng Docker bridge độc lập:
     - **Frontend Network (`ols-network`):** Chỉ chứa `ols-traefik` tiếp nhận request từ bên ngoài và chuyển tiếp traffic HTTP tới các container website (`ols_<slug>`).
     - **Backend Network (`ols-backend-network`):** Chứa `ols-mariadb` và `ols-redis`. Tuyệt đối không expose ra ngoài Internet và không nằm chung mạng frontend với Traefik.
   * Các container website (`ols_<slug>`) kết nối vào cả hai mạng (nhận traffic từ Traefik qua frontend và truy vấn Database/Redis qua backend). Traefik được chỉ định rõ `traefik.docker.network=ols-network` để đảm bảo định tuyến chính xác.
   * Container quản trị phpMyAdmin (`ols-pma`) chỉ kết nối vào Backend Network khi được bật, triệt tiêu tối đa nguy cơ rò rỉ database.

7. **Tự động Ép HTTPS & Bộ Security Headers chuẩn A+:**
   * Traefik Gateway tự động chuyển hướng 301 toàn bộ traffic HTTP (port 80) sang HTTPS (port 443).
   * Tự động gắn các security headers tiêu chuẩn: `Strict-Transport-Security` (HSTS 1 năm, preload), `X-Frame-Options: SAMEORIGIN` (chống Clickjacking), `X-Content-Type-Options: nosniff`, và `Referrer-Policy: strict-origin-when-cross-origin`.

8. **Tối ưu Cache Tĩnh & Nhận diện Real IP Cloudflare:**
   * Cấu hình OpenLiteSpeed `expiresByType` lưu đệm CSS, JS, WebP/ảnh tĩnh 30 ngày và Web Font 1 năm, loại bỏ hoàn toàn hiện tượng `EXPIRED` trên Cloudflare edge và giải phóng CPU máy chủ VPS.
   * Chặn đứng mã HTTP 403 các cuộc tấn công brute-force bot vào `xmlrpc.php` và các file nhạy cảm (`.env`, `.git`, `readme.html`).
   * Tích hợp dải IP tin cậy của Cloudflare vào Traefik và kích hoạt `useIpInProxyHeader` trên OpenLiteSpeed để bảo toàn 100% Real IP của độc giả trong access log và plugins bảo mật.

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
