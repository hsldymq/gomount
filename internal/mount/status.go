package mount

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/moby/sys/mountinfo"
)

// CheckStatus 检查指定路径是否已挂载
func CheckStatus(mountPath string) (bool, error) {
	// Normalize the path
	absPath, err := filepath.Abs(mountPath)
	if err != nil {
		return false, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if mount point exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return false, nil
	}

	// Use moby/sys/mountinfo to check mount status
	mounted, err := mountinfo.Mounted(absPath)
	if err != nil {
		return false, fmt.Errorf("failed to check mount status: %w", err)
	}

	return mounted, nil
}

// CheckEntryStatus 检查单个挂载条目的挂载状态
func CheckEntryStatus(entry *config.MountEntry) (bool, error) {
	return CheckStatus(entry.MountDirPath)
}

// RefreshAllStatus 更新配置中所有条目的挂载状态
func RefreshAllStatus(cfg *config.Config) error {
	for i := range cfg.Mounts {
		mounted, err := CheckEntryStatus(&cfg.Mounts[i])
		if err != nil {
			// Log error but continue checking other entries
			fmt.Fprintf(os.Stderr, "Warning: failed to check status for %s: %v\n", cfg.Mounts[i].Name, err)
		}
		cfg.Mounts[i].IsMounted = mounted
	}
	return nil
}
