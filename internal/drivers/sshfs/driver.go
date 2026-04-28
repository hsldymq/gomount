package sshfs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/drivers"
	"github.com/hsldymq/gomount/internal/status"
)

type Driver struct{}

func NewDriver() *Driver {
	return &Driver{}
}

func (d *Driver) Type() string {
	return "sshfs"
}

func (d *Driver) Mount(ctx context.Context, entry *config.MountEntry) error {
	mounted, err := status.CheckStatus(entry.MountDirPath)
	if err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("failed to check mount status: %w", err),
		}
	}
	if mounted {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("already mounted at %s", entry.MountDirPath),
		}
	}

	if err := os.MkdirAll(entry.MountDirPath, 0755); err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("failed to create mount directory: %w", err),
		}
	}

	cmd := d.buildMountCommand(entry)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("sshfs failed: %w\nOutput: %s", err, string(output)),
		}
	}

	return nil
}

func (d *Driver) Unmount(ctx context.Context, entry *config.MountEntry) error {
	cmd := exec.CommandContext(ctx, "umount", entry.MountDirPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "unmount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("umount failed: %w\nOutput: %s", err, string(output)),
		}
	}
	return nil
}

func (d *Driver) Status(ctx context.Context, entry *config.MountEntry) (*drivers.MountStatus, error) {
	mounted, err := status.CheckStatus(entry.MountDirPath)
	if err != nil {
		return nil, &drivers.DriverError{
			Driver: d.Type(),
			Op:     "status",
			Entry:  entry.Name,
			Err:    fmt.Errorf("failed to check mount status: %w", err),
		}
	}

	status := &drivers.MountStatus{
		Mounted: mounted,
		Details: map[string]string{
			"type":       "sshfs",
			"host":       entry.SSHFS.Host,
			"remotePath": entry.SSHFS.RemotePath,
		},
	}

	if mounted {
		status.Message = fmt.Sprintf("Mounted at %s", entry.MountDirPath)
	} else {
		status.Message = "Not mounted"
	}

	return status, nil
}

func (d *Driver) Validate(entry *config.MountEntry) error {
	if entry.SSHFS == nil {
		return fmt.Errorf("sshfs config is required for SSHFS driver")
	}
	if entry.SSHFS.Host == "" {
		return fmt.Errorf("sshfs.host is required for SSHFS driver")
	}
	if entry.SSHFS.RemotePath == "" {
		return fmt.Errorf("sshfs.remote_path is required for SSHFS driver")
	}
	return nil
}

func (d *Driver) buildMountCommand(entry *config.MountEntry) *exec.Cmd {
	remotePath := fmt.Sprintf("%s:%s", entry.SSHFS.Host, entry.SSHFS.RemotePath)

	options := []string{"follow_symlinks", "cache_timeout=600"}

	args := []string{
		"-o", strings.Join(options, ","),
		remotePath,
		entry.MountDirPath,
	}

	return exec.Command("sshfs", args...)
}
