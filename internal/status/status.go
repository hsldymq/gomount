package status

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/moby/sys/mountinfo"
)

func CheckStatus(mountPath string) (bool, error) {
	absPath, err := filepath.Abs(mountPath)
	if err != nil {
		return false, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return false, nil
	}

	mounted, err := mountinfo.Mounted(absPath)
	if err != nil {
		return false, fmt.Errorf("failed to check mount status: %w", err)
	}

	return mounted, nil
}

func RefreshAllStatus(cfg *config.Config) error {
	for i := range cfg.Mounts {
		mounted, err := CheckStatus(cfg.Mounts[i].MountDirPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to check status for %s: %v\n", cfg.Mounts[i].Name, err)
		}
		cfg.Mounts[i].IsMounted = mounted
	}
	return nil
}
