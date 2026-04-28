package smb

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/drivers"
	"github.com/hsldymq/gomount/internal/mount"
)

type Driver struct{}

func NewDriver() *Driver {
	return &Driver{}
}

func (d *Driver) Type() string {
	return "smb"
}

func (d *Driver) Mount(ctx context.Context, entry *config.MountEntry) error {
	mounted, err := mount.CheckStatus(entry.MountDirPath)
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

	credsFile, err := d.createCredentialFile(entry)
	if err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    err,
		}
	}
	defer os.Remove(credsFile)

	cmd := d.buildMountCommand(entry, credsFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("mount.cifs failed: %w\nOutput: %s", err, string(output)),
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
	mounted, err := mount.CheckStatus(entry.MountDirPath)
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
			"type": "smb",
			"addr": entry.SMB.Addr,
		},
	}

	if mounted {
		status.Message = fmt.Sprintf("Mounted at %s", entry.MountDirPath)
		status.Details["share"] = entry.SMB.ShareName
	} else {
		status.Message = "Not mounted"
	}

	return status, nil
}

func (d *Driver) Validate(entry *config.MountEntry) error {
	if entry.SMB == nil {
		return fmt.Errorf("smb config is required for SMB driver")
	}
	if entry.SMB.Addr == "" {
		return fmt.Errorf("smb.addr is required for SMB driver")
	}
	if entry.SMB.ShareName == "" {
		return fmt.Errorf("smb.share_name is required for SMB driver")
	}
	if entry.SMB.Username == "" {
		return fmt.Errorf("smb.username is required for SMB driver")
	}
	return nil
}

func (d *Driver) createCredentialFile(entry *config.MountEntry) (string, error) {
	tmpFile, err := os.CreateTemp("", "gomount_creds_*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create credentials file: %w", err)
	}
	defer tmpFile.Close()

	if err := tmpFile.Chmod(0600); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to set credentials file permissions: %w", err)
	}

	content := fmt.Sprintf("username=%s\npassword=%s\ndomain=%s\n",
		entry.SMB.Username,
		entry.SMB.Password,
		"",
	)

	if _, err := tmpFile.WriteString(content); err != nil {
		os.Remove(tmpFile.Name())
		return "", fmt.Errorf("failed to write credentials: %w", err)
	}

	return tmpFile.Name(), nil
}

func (d *Driver) buildMountCommand(entry *config.MountEntry, credsFile string) *exec.Cmd {
	smbAddr := fmt.Sprintf("//%s:%d/%s", entry.SMB.Addr, entry.SMB.GetPort(), entry.SMB.ShareName)

	options := fmt.Sprintf("credentials=%s,file_mode=0755,dir_mode=0755,uid=%d,gid=%d",
		credsFile,
		os.Getuid(),
		os.Getgid(),
	)

	args := []string{
		smbAddr,
		entry.MountDirPath,
		"-o", options,
	}

	return exec.Command("mount.cifs", args...)
}
