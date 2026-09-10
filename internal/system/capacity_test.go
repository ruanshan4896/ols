package system

import (
	"testing"
)

func TestParseFreeOutput(t *testing.T) {
	raw := `               total        used        free      shared  buff/cache   available
Mem:            1980        1050         230          45         700         850
Swap:           2048         120        1928
`
	mem, swap := ParseFreeOutput(raw)
	if mem.TotalMB != 1980 {
		t.Errorf("expected total 1980, got %d", mem.TotalMB)
	}
	if mem.AvailableMB != 850 {
		t.Errorf("expected available 850, got %d", mem.AvailableMB)
	}
	if swap.TotalMB != 2048 {
		t.Errorf("expected swap total 2048, got %d", swap.TotalMB)
	}
	if swap.FreeMB != 1928 {
		t.Errorf("expected swap free 1928, got %d", swap.FreeMB)
	}
}

func TestParseDfOutput(t *testing.T) {
	raw := `Filesystem     1M-blocks  Used Available Use% Mounted on
/dev/sda1          40960 10240     30720  25% /
`
	disk := ParseDfOutput(raw)
	if disk.TotalGB != 40.0 {
		t.Errorf("expected total 40GB, got %f", disk.TotalGB)
	}
	if disk.AvailableGB != 30.0 {
		t.Errorf("expected available 30GB, got %f", disk.AvailableGB)
	}
}

func TestParseDfOutput_WrappedLines(t *testing.T) {
	// Kiểm tra trường hợp tên filesystem LVM dài bị ngắt dòng trong df truyền thống
	raw := `Filesystem           1M-blocks      Used Available Use% Mounted on
/dev/mapper/ubuntu--vg-ubuntu--lv
                         40960     10240     30720  25% /
`
	disk := ParseDfOutput(raw)
	if disk.TotalGB != 40.0 {
		t.Errorf("expected total 40GB, got %f", disk.TotalGB)
	}
	if disk.AvailableGB != 30.0 {
		t.Errorf("expected available 30GB, got %f", disk.AvailableGB)
	}
}

func TestParseDockerStats(t *testing.T) {
	raw := `ols-mariadb	180MiB / 1.95GiB
ols-redis	35MiB / 1.95GiB
ols-traefik	30MiB / 1.95GiB
ols_site1_com	80MiB / 384MiB
ols_site2_com	90MiB / 384MiB
ols_site3_com	70MiB / 384MiB
`
	count, avg, total := ParseDockerStats(raw)
	if count != 3 {
		t.Fatalf("expected 3 site containers, got %d", count)
	}
	if total != 240 {
		t.Errorf("expected total 240MB, got %d", total)
	}
	if avg != 80 {
		t.Errorf("expected avg 80MB, got %d", avg)
	}
}

func TestParseLoadAvg(t *testing.T) {
	raw := "0.25 0.35 0.40 1/140 12345"
	cpu := ParseLoadAvg(raw, 2)

	if cpu.Cores != 2 {
		t.Errorf("expected 2 cores, got %d", cpu.Cores)
	}
	if cpu.Load1 != 0.25 || cpu.Load5 != 0.35 || cpu.Load15 != 0.40 {
		t.Errorf("unexpected load values: %v, %v, %v", cpu.Load1, cpu.Load5, cpu.Load15)
	}
	// LoadPerCore = 0.40 / 2 = 0.20 -> Rất tốt
	if cpu.LoadPerCore != 0.20 {
		t.Errorf("expected LoadPerCore 0.20, got %f", cpu.LoadPerCore)
	}
	if cpu.Status != "Rất tốt (CPU rảnh rỗi)" {
		t.Errorf("unexpected status: %s", cpu.Status)
	}
}

func TestParseUptime(t *testing.T) {
	raw := " 14:20:00 up 5 days,  3:12,  2 users,  load average: 3.50, 2.10, 1.80"
	cpu := ParseUptime(raw, 2)

	if cpu.Load1 != 3.50 || cpu.Load5 != 2.10 || cpu.Load15 != 1.80 {
		t.Errorf("unexpected load values: %v, %v, %v", cpu.Load1, cpu.Load5, cpu.Load15)
	}
	// LoadPerCore = 1.80 / 2 = 0.90 -> Đang tải
	if cpu.LoadPerCore != 0.90 {
		t.Errorf("expected LoadPerCore 0.90, got %f", cpu.LoadPerCore)
	}
	if cpu.Status != "Đang tải (Nhiều tác vụ hoặc bot truy cập)" {
		t.Errorf("unexpected status: %s", cpu.Status)
	}
}

func TestCalculateCapacity(t *testing.T) {
	mem := MemInfo{
		TotalMB:     2048,
		UsedMB:      1150,
		FreeMB:      200,
		AvailableMB: 850,
	}
	swap := SwapInfo{
		TotalMB: 2048,
		FreeMB:  2000,
		UsedMB:  48,
	}
	disk := DiskInfo{
		TotalGB:     40.0,
		UsedGB:      10.0,
		AvailableGB: 30.0,
	}
	cpu := CPUInfo{
		Cores:       2,
		Load15:      0.40,
		LoadPerCore: 0.20,
		Status:      "Rất tốt (CPU rảnh rỗi)",
	}

	report := CalculateCapacity(mem, swap, disk, cpu, 8, 80)

	// SafeBuffer = 2048 * 0.15 = 307MB
	// UsableRAM = 850 - 307 = 543MB
	// EstSafe = 543 / 80 = 6 sites
	if report.EstSafeSites < 5 || report.EstSafeSites > 8 {
		t.Errorf("expected EstSafeSites around 6, got %d", report.EstSafeSites)
	}
	if report.EstWithSwapSites <= report.EstSafeSites {
		t.Errorf("expected EstWithSwapSites to be greater than EstSafeSites, got withSwap=%d, safe=%d", report.EstWithSwapSites, report.EstSafeSites)
	}
	if report.RunningSitesCount != 8 {
		t.Errorf("expected 8 running sites, got %d", report.RunningSitesCount)
	}
	if report.CPUCores != 2 {
		t.Errorf("expected 2 cpu cores, got %d", report.CPUCores)
	}
}

func TestCalculateCapacityOverloadedCPU(t *testing.T) {
	mem := MemInfo{TotalMB: 4096, AvailableMB: 2500}
	swap := SwapInfo{TotalMB: 2048, FreeMB: 2000}
	disk := DiskInfo{TotalGB: 50.0, AvailableGB: 30.0}
	cpu := CPUInfo{
		Cores:       2,
		Load15:      3.60,
		LoadPerCore: 1.80, // >= 1.5 -> Overload
		Status:      "Quá tải (CPU nghẽn, phản hồi có thể chậm)",
	}

	report := CalculateCapacity(mem, swap, disk, cpu, 10, 80)
	if report.SystemStatus != "Cảnh báo (CPU đang quá tải, phản hồi có thể chậm)" {
		t.Errorf("expected system status warning for overloaded CPU, got: %s", report.SystemStatus)
	}
}
