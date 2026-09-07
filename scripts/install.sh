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
