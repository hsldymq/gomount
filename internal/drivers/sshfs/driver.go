package sshfs

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/daemon"
	"github.com/hsldymq/gomount/internal/drivers"
	"github.com/hsldymq/gomount/internal/interaction"
	"github.com/hsldymq/gomount/internal/mountkit"
	"github.com/hsldymq/gomount/internal/status"
)

type Driver struct {
	mountMgr *mountkit.Manager
}

func NewDriver(mgr *mountkit.Manager) *Driver {
	return &Driver{mountMgr: mgr}
}

func (d *Driver) Type() string {
	return "sshfs"
}

func (d *Driver) NeedsSudo() bool {
	return false
}

func (d *Driver) Mount(ctx context.Context, entry *config.MountEntry) error {
	mounted, err := status.CheckStatus(entry.MountDirPath)
	if err != nil {
		return &drivers.DriverError{
			Driver: d.Type(), Op: "mount", Entry: entry.Name,
			Err: fmt.Errorf("failed to check mount status: %w", err),
		}
	}
	if mounted {
		return &drivers.DriverError{
			Driver: d.Type(), Op: "mount", Entry: entry.Name,
			Err: fmt.Errorf("already mounted at %s", entry.MountDirPath),
		}
	}

	if err := os.MkdirAll(entry.MountDirPath, 0755); err != nil {
		return &drivers.DriverError{
			Driver: d.Type(), Op: "mount", Entry: entry.Name,
			Err: fmt.Errorf("failed to create mount directory: %w", err),
		}
	}

	if daemon.IsCommandAvailable("sshfs") {
		return d.mountSSHFS(entry)
	}
	return d.mountRcloneSFTP(ctx, entry)
}

func (d *Driver) mountSSHFS(entry *config.MountEntry) error {
	cmd := d.buildMountCommand(entry)
	if err := interaction.RunCommand(cmd); err != nil {
		return &drivers.DriverError{
			Driver: d.Type(), Op: "mount", Entry: entry.Name,
			Err: &drivers.CommandError{Cmd: "sshfs", Err: err},
		}
	}
	return nil
}

func (d *Driver) mountRcloneSFTP(ctx context.Context, entry *config.MountEntry) error {
	remoteName := fmt.Sprintf("gomount_sshfs_%s", entry.Name)
	remotePath := mountkit.RegisterSFTPRemote(remoteName, entry.SSHFS.Host, entry.SSHFS.RemotePath, "ssh")

	if err := d.mountMgr.Mount(ctx, remotePath, entry.MountDirPath, entry.Name); err != nil {
		return &drivers.DriverError{
			Driver: d.Type(), Op: "mount", Entry: entry.Name,
			Err: err,
		}
	}

	return nil
}

func (d *Driver) Unmount(ctx context.Context, entry *config.MountEntry) error {
	if d.mountMgr.IsMounted(entry.MountDirPath) {
		if err := d.mountMgr.Unmount(entry.MountDirPath); err != nil {
			return &drivers.DriverError{
				Driver: d.Type(), Op: "unmount", Entry: entry.Name,
				Err: err,
			}
		}
	} else {
		cmd := exec.CommandContext(ctx, "fusermount", "-u", entry.MountDirPath)
		if err := interaction.RunCommandSilent(cmd); err != nil {
			return &drivers.DriverError{
				Driver: d.Type(), Op: "unmount", Entry: entry.Name,
				Err: &drivers.CommandError{Cmd: "fusermount", Err: err},
			}
		}
	}
	return nil
}

func (d *Driver) Status(ctx context.Context, entry *config.MountEntry) (*drivers.MountStatus, error) {
	mounted, err := status.CheckStatus(entry.MountDirPath)
	if err != nil {
		return nil, &drivers.DriverError{
			Driver: d.Type(), Op: "status", Entry: entry.Name,
			Err: fmt.Errorf("failed to check mount status: %w", err),
		}
	}

	s := &drivers.MountStatus{
		Mounted: mounted,
		Details: map[string]string{
			"type":       "sshfs",
			"host":       entry.SSHFS.Host,
			"remotePath": entry.SSHFS.RemotePath,
		},
	}

	if mounted {
		s.Message = fmt.Sprintf("Mounted at %s", entry.MountDirPath)
	} else {
		s.Message = "Not mounted"
	}

	return s, nil
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
