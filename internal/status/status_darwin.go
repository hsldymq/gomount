//go:build darwin

package status

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CheckStatus(mountPath string) (bool, error) {
	absPath, err := filepath.Abs(mountPath)
	if err != nil {
		return false, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return false, nil
	}

	out, err := exec.Command("mount").Output()
	if err != nil {
		return false, fmt.Errorf("failed to list mounts: %w", err)
	}

	for _, line := range strings.Split(string(out), "\n") {
		if darwinMountLineMatches(line, absPath) {
			return true, nil
		}
	}
	return false, nil
}

func darwinMountLineMatches(line, mountPath string) bool {
	needle := " on " + mountPath + " ("
	return strings.Contains(line, needle)
}
