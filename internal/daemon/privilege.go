package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

// IsRoot checks if current process is running as root
func IsRoot() bool {
	return os.Getuid() == 0
}

// GetOriginalUser returns the original user (before sudo)
func GetOriginalUser() (*user.User, error) {
	// Try sudo environment variables first
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		return user.Lookup(sudoUser)
	}

	// Try gomount environment variables
	if origUser := os.Getenv("GOMOUNT_ORIGINAL_USERNAME"); origUser != "" {
		return user.Lookup(origUser)
	}

	// Fall back to current user
	return user.Current()
}

// GetOriginalUID returns the original user UID
func GetOriginalUID() int {
	if sudoUID := os.Getenv("SUDO_UID"); sudoUID != "" {
		uid, _ := strconv.Atoi(sudoUID)
		return uid
	}
	if origUID := os.Getenv("GOMOUNT_ORIGINAL_UID"); origUID != "" {
		uid, _ := strconv.Atoi(origUID)
		return uid
	}
	return os.Getuid()
}

// GetOriginalGID returns the original user GID
func GetOriginalGID() int {
	if sudoGID := os.Getenv("SUDO_GID"); sudoGID != "" {
		gid, _ := strconv.Atoi(sudoGID)
		return gid
	}
	if origGID := os.Getenv("GOMOUNT_ORIGINAL_GID"); origGID != "" {
		gid, _ := strconv.Atoi(origGID)
		return gid
	}
	return os.Getgid()
}

// NeedsPrivilege checks if config requires root privileges
func NeedsPrivilege(cfg interface{}) bool {
	// This is a simplified check - in reality you'd check the config
	// For now, return true if SMB mounts exist
	return true
}

// StartWithSudo starts the current executable with sudo
func StartWithSudo(args ...string) error {
	me, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find executable: %w", err)
	}

	sudoArgs := append([]string{me}, args...)
	cmd := exec.Command("sudo", sudoArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// DropPrivileges drops root privileges to original user
func DropPrivileges() error {
	uid := GetOriginalUID()
	gid := GetOriginalGID()

	if uid == 0 || uid == os.Getuid() {
		return nil // Already root or already correct user
	}

	if err := syscall.Setgid(gid); err != nil {
		return fmt.Errorf("failed to set gid: %w", err)
	}
	if err := syscall.Setuid(uid); err != nil {
		return fmt.Errorf("failed to set uid: %w", err)
	}

	return nil
}

// RunAsUser runs a function as the specified user
func RunAsUser(uid, gid int, fn func() error) error {
	if uid == 0 {
		return fn() // No need to switch
	}

	// Save current permissions
	oldUID := os.Getuid()
	oldGID := os.Getgid()

	// Switch to target user
	if err := syscall.Setgid(gid); err != nil {
		return err
	}
	if err := syscall.Setuid(uid); err != nil {
		// Try to restore gid on failure
		syscall.Setgid(oldGID)
		return err
	}

	// Run function
	err := fn()

	// Restore permissions
	syscall.Setuid(oldUID)
	syscall.Setgid(oldGID)

	return err
}
