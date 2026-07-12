package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const internetCheckTimeout = 5 * time.Second

// ServerInfo holds host machine stats for the settings dashboard.
type ServerInfo struct {
	CPUCount       int    `json:"cpu_count"`
	RAMTotalBytes  uint64 `json:"ram_total_bytes"`
	RAMFreeBytes   uint64 `json:"ram_free_bytes"`
	DiskFreeBytes  uint64 `json:"disk_free_bytes"`
	DataDir        string `json:"data_dir"`
	InternetOK     bool   `json:"internet_ok"`
	InternetError  string `json:"internet_error,omitempty"`
	InternetHost   string `json:"internet_host"`
	CheckedAt      string `json:"checked_at"`
}

func (a *App) ServerInfo(ctx context.Context, dataDir string) ServerInfo {
	info := ServerInfo{
		CPUCount:     runtime.NumCPU(),
		DataDir:      dataDir,
		InternetHost: "google.com",
		CheckedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	if total, free, err := readMemoryBytes(); err == nil {
		info.RAMTotalBytes = total
		info.RAMFreeBytes = free
	}
	if free, err := diskFreeBytes(dataDir); err == nil {
		info.DiskFreeBytes = free
	}
	ok, errMsg := checkInternet(ctx)
	info.InternetOK = ok
	info.InternetError = errMsg
	return info
}

func readMemoryBytes() (total, free uint64, err error) {
	if runtime.GOOS != "linux" {
		return 0, 0, fmt.Errorf("memory stats unavailable on %s", runtime.GOOS)
	}
	raw, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	var memTotal, memAvailable uint64
	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, parseErr := strconv.ParseUint(fields[1], 10, 64)
		if parseErr != nil {
			continue
		}
		val *= 1024
		switch fields[0] {
		case "MemTotal:":
			memTotal = val
		case "MemAvailable:":
			memAvailable = val
		}
	}
	if memTotal == 0 {
		return 0, 0, fmt.Errorf("could not read MemTotal")
	}
	if memAvailable == 0 {
		for _, line := range strings.Split(string(raw), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 || fields[0] != "MemFree:" {
				continue
			}
			val, parseErr := strconv.ParseUint(fields[1], 10, 64)
			if parseErr == nil {
				memAvailable = val * 1024
			}
			break
		}
	}
	return memTotal, memAvailable, nil
}

func diskFreeBytes(path string) (uint64, error) {
	if path == "" {
		return 0, fmt.Errorf("empty path")
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return stat.Bavail * uint64(stat.Bsize), nil
}

func checkInternet(ctx context.Context) (bool, string) {
	ctx, cancel := context.WithTimeout(ctx, internetCheckTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://www.google.com", nil)
	if err != nil {
		return false, err.Error()
	}
	client := &http.Client{Timeout: internetCheckTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
	}
	return true, ""
}
