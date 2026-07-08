package status

import (
	"fmt"
	"os"

	"github.com/hsldymq/gomount/internal/config"
)

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
