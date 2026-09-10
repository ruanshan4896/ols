# Kế hoạch Triển khai Sửa lỗi Ngầm & Tối ưu hóa Toàn diện `ols-cli`

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Khắc phục toàn bộ các lỗi ngầm (Redis flushall, DB ID mismatch, exec php sai path, dump lỗi, www cctld) và tối ưu hóa tài nguyên (OPcache, Worker pool, Cache WP core, Disable WP-Cron, PMA memory limit) cho hệ thống `ols-cli`.

**Architecture:** Bổ sung cơ chế lưu trữ bền vững metadata per-site (.env), đóng gói hàm xử lý an toàn có bắt lỗi, tối ưu template vhost/docker-compose, và nâng cấp parser.

**Tech Stack:** Go 1.26, Docker, OpenLiteSpeed, MariaDB 11.4, Redis 7, Traefik v3.

---

### Task 1: Cố định Redis DB ID và thay thế `FLUSHALL` bằng `FLUSHDB`
**Files:**
- Modify: `internal/site/manager.go`
- Test: `internal/site/manager_test.go`

- [ ] **Step 1: Viết test kiểm tra lưu & đọc Redis DB ID bền vững từ .env**
- [ ] **Step 2: Chạy test để xác nhận thất bại trước khi cài đặt**
- [ ] **Step 3: Cài đặt hàm `GetOrAssignSiteRedisDB` lưu vào .env và sửa `FLUSHALL` thành `FLUSHDB`**
- [ ] **Step 4: Chạy lại test xác nhận thành công**

---

### Task 2: Sửa đường dẫn binary PHP khi đổi mật khẩu Admin trong `internal/site/security.go`
**Files:**
- Modify: `internal/site/security.go`
- Test: `internal/site/security_test.go`

- [ ] **Step 1: Viết test kiểm tra cú pháp lệnh gọi phpBin với phiên bản LSPHP chính xác**
- [ ] **Step 2: Cài đặt logic tìm binary LSPHP (`/usr/local/lsws/lsphp%s/bin/php`) trong container**
- [ ] **Step 3: Chạy test xác nhận thành công**

---

### Task 3: Bắt lỗi MariaDB Dump/Import & Bỏ qua thư mục Logs khi Backup
**Files:**
- Modify: `internal/backup/manager.go`
- Test: `internal/backup/manager_test.go`

- [ ] **Step 1: Viết test xác nhận bỏ qua logs/ khi nén backup**
- [ ] **Step 2: Cài đặt kiểm tra exit code của `mariadb-dump` và `mariadb import`, bỏ qua `logs/` trong `filepath.Walk`**
- [ ] **Step 3: Chạy test xác nhận thành công**

---

### Task 4: Nhận diện chuẩn Tên miền WWW cho ccTLD (`.com.vn`, `.co.uk`)
**Files:**
- Modify: `internal/template/render.go`
- Test: `internal/template/render_test.go`

- [ ] **Step 1: Viết test cases cho `example.com.vn`, `domain.co.uk`, `sub.example.com`**
- [ ] **Step 2: Cài đặt hàm `ShouldIncludeWWW(domain string) bool`**
- [ ] **Step 3: Chạy test xác nhận thành công**

---

### Task 5: Tinh chỉnh Resource Templates & Bổ sung `DISABLE_WP_CRON`
**Files:**
- Modify: `templates/site/docker-compose.yml.tmpl`
- Modify: `templates/site/vhost.conf.tmpl`
- Modify: `internal/site/manager.go`
- Test: `internal/template/render_test.go`, `internal/site/manager_test.go`

- [ ] **Step 1: Cập nhật template compose (children=3) và vhost (opcache=32M)**
- [ ] **Step 2: Thêm `define('DISABLE_WP_CRON', true);` vào `SanitizeWPConfig` và template khởi tạo**
- [ ] **Step 3: Chạy toàn bộ test suites xác nhận tương thích**

---

### Task 6: Giới hạn RAM cho phpMyAdmin & Tối ưu lệnh `df -Pm` trong Capacity
**Files:**
- Modify: `cmd/pma.go`
- Modify: `internal/system/capacity.go`
- Test: `internal/system/capacity_test.go`

- [ ] **Step 1: Thêm `--memory 256m --cpus 1.0` vào `pma.go`**
- [ ] **Step 2: Đổi lệnh trong `GetVPSCapacity` sang `df -Pm`**
- [ ] **Step 3: Chạy test xác nhận thành công**
