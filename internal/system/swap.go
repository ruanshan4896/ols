package system

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// SwapInfo chứa thông tin hiện tại của Swap trên máy chủ
type SwapInfo struct {
	TotalMB int
	UsedMB  int
	FreeMB  int
	Path    string
}

// GetSwapInfo đọc thông tin swap hiện tại của hệ thống Linux
func GetSwapInfo() (*SwapInfo, error) {
	cmd := exec.Command("free", "-m")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("không thể đọc thông tin bộ nhớ: %w", err)
	}

	info := &SwapInfo{}
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 4 && strings.HasPrefix(strings.ToLower(fields[0]), "swap") {
			info.TotalMB, _ = strconv.Atoi(fields[1])
			info.UsedMB, _ = strconv.Atoi(fields[2])
			info.FreeMB, _ = strconv.Atoi(fields[3])
			break
		}
	}

	// Kiểm tra đường dẫn swapfile nếu có
	swaponCmd := exec.Command("swapon", "--show")
	var swaponOut bytes.Buffer
	swaponCmd.Stdout = &swaponOut
	if err := swaponCmd.Run(); err == nil {
		linesSwapon := strings.Split(swaponOut.String(), "\n")
		if len(linesSwapon) > 1 {
			f := strings.Fields(linesSwapon[1])
			if len(f) > 0 {
				info.Path = f[0]
			}
		}
	}

	return info, nil
}

// CreateSwap tạo hoặc thay đổi dung lượng Swapfile (đơn vị: GB)
func CreateSwap(sizeGB int) error {
	if sizeGB <= 0 {
		return fmt.Errorf("dung lượng Swap phải lớn hơn 0 GB")
	}

	swapPath := "/swapfile"

	// 1. Tắt swap cũ nếu đang kích hoạt trên /swapfile
	_ = exec.Command("swapoff", swapPath).Run()

	// 2. Xóa swapfile cũ nếu có
	_ = os.Remove(swapPath)

	// 3. Khởi tạo file mới với dung lượng sizeGB
	// Thử dùng fallocate trước (nhanh nhất), nếu không hỗ trợ thì fallback sang dd
	fallocateCmd := exec.Command("fallocate", "-l", fmt.Sprintf("%dG", sizeGB), swapPath)
	if err := fallocateCmd.Run(); err != nil {
		ddCmd := exec.Command("dd", "if=/dev/zero", fmt.Sprintf("of=%s", swapPath), "bs=1M", fmt.Sprintf("count=%d", sizeGB*1024), "status=progress")
		if out, errDD := ddCmd.CombinedOutput(); errDD != nil {
			return fmt.Errorf("không thể tạo file swap: %s (%w)", string(out), errDD)
		}
	}

	// 4. Phân quyền bảo mật tuyệt đối 600 (chỉ root mới được đọc/ghi)
	if err := exec.Command("chmod", "600", swapPath).Run(); err != nil {
		return fmt.Errorf("phân quyền swapfile thất bại: %w", err)
	}

	// 5. Định dạng file thành swap
	mkswapCmd := exec.Command("mkswap", swapPath)
	if out, err := mkswapCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mkswap thất bại: %s (%w)", string(out), err)
	}

	// 6. Kích hoạt swap ngay lập tức
	swaponCmd := exec.Command("swapon", swapPath)
	if out, err := swaponCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("swapon thất bại: %s (%w)", string(out), err)
	}

	// 7. Ghi vào /etc/fstab để tự động kích hoạt lại khi khởi động lại VPS
	fstabPath := "/etc/fstab"
	fstabBytes, err := os.ReadFile(fstabPath)
	if err == nil {
		fstabContent := string(fstabBytes)
		swapEntry := fmt.Sprintf("%s none swap sw 0 0", swapPath)
		if !strings.Contains(fstabContent, swapPath) {
			fstabContent = strings.TrimRight(fstabContent, "\n") + "\n" + swapEntry + "\n"
			_ = os.WriteFile(fstabPath, []byte(fstabContent), 0644)
		}
	}

	// 8. Tối ưu swappiness = 10 (ưu tiên dùng RAM thật, chỉ dùng swap khi RAM thật sắp hết)
	_ = exec.Command("sysctl", "vm.swappiness=10").Run()
	sysctlConf := "/etc/sysctl.conf"
	if scBytes, err := os.ReadFile(sysctlConf); err == nil {
		scContent := string(scBytes)
		if strings.Contains(scContent, "vm.swappiness") {
			lines := strings.Split(scContent, "\n")
			for i, l := range lines {
				if strings.HasPrefix(strings.TrimSpace(l), "vm.swappiness") {
					lines[i] = "vm.swappiness=10"
				}
			}
			scContent = strings.Join(lines, "\n")
		} else {
			scContent = strings.TrimRight(scContent, "\n") + "\nvm.swappiness=10\n"
		}
		_ = os.WriteFile(sysctlConf, []byte(scContent), 0644)
	}

	return nil
}

// DisableSwap tắt và xóa vĩnh viễn Swap
func DisableSwap() error {
	swapPath := "/swapfile"
	_ = exec.Command("swapoff", swapPath).Run()
	_ = os.Remove(swapPath)

	// Xóa khỏi /etc/fstab
	fstabPath := "/etc/fstab"
	if fstabBytes, err := os.ReadFile(fstabPath); err == nil {
		lines := strings.Split(string(fstabBytes), "\n")
		var newLines []string
		for _, l := range lines {
			if !strings.Contains(l, swapPath) {
				newLines = append(newLines, l)
			}
		}
		_ = os.WriteFile(fstabPath, []byte(strings.Join(newLines, "\n")), 0644)
	}

	return nil
}
