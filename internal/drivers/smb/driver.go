package smb

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/drivers"
	"github.com/hsldymq/gomount/internal/interaction"
	"github.com/hsldymq/gomount/internal/sshtunnel"
	"github.com/hsldymq/gomount/internal/status"
)

type Driver struct{}

func NewDriver() *Driver {
	return &Driver{}
}

func (d *Driver) Type() string {
	return "smb"
}

func (d *Driver) NeedsSudo() bool {
	return true
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

	smbAddr := entry.SMB.Addr
	smbPort := entry.SMB.GetPort()
	if entry.SSHTunnel != nil {
		remoteAddr := fmt.Sprintf("%s:%d", entry.SMB.Addr, smbPort)
		localPort, err := sshtunnel.EstablishForMount(ctx, entry.Name, entry.SSHTunnel.Host, remoteAddr)
		if err != nil {
			return &drivers.DriverError{
				Driver: d.Type(),
				Op:     "mount",
				Entry:  entry.Name,
				Err:    &drivers.TunnelError{Host: entry.SSHTunnel.Host, Err: err},
			}
		}
		smbAddr = "localhost"
		smbPort = localPort
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

	cmd := d.buildMountCommand(entry, credsFile, smbAddr, smbPort)

	if interaction.NeedsPrivilege() {
		cmd, err = interaction.WrapWithSudo(cmd)
		if err != nil {
			return &drivers.DriverError{
				Driver: d.Type(),
				Op:     "mount",
				Entry:  entry.Name,
				Err:    err,
			}
		}
	}

	if err := interaction.RunCommand(cmd); err != nil {
		if entry.SSHTunnel != nil {
			sshtunnel.Teardown(entry.Name)
		}
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    &drivers.CommandError{Cmd: "mount.cifs", Err: err},
		}
	}

	return nil
}

func (d *Driver) Unmount(ctx context.Context, entry *config.MountEntry) error {
	cmd := exec.CommandContext(ctx, "umount", entry.MountDirPath)

	var err error
	if interaction.NeedsPrivilege() {
		cmd, err = interaction.WrapWithSudo(cmd)
		if err != nil {
			return &drivers.DriverError{
				Driver: d.Type(),
				Op:     "unmount",
				Entry:  entry.Name,
				Err:    err,
			}
		}
	}

	if err := interaction.RunCommandSilent(cmd); err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "unmount",
			Entry:  entry.Name,
			Err:    err,
		}
	}

	if entry.SSHTunnel != nil {
		_ = sshtunnel.Teardown(entry.Name)
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

func (d *Driver) buildMountCommand(entry *config.MountEntry, credsFile, addr string, port int) *exec.Cmd {
	smbAddr := fmt.Sprintf("//%s/%s", addr, entry.SMB.ShareName)

	options := fmt.Sprintf("credentials=%s,port=%d,file_mode=0755,dir_mode=0755,uid=%d,gid=%d",
		credsFile,
		port,
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
