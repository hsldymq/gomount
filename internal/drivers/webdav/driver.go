package webdav

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/drivers"
	"github.com/hsldymq/gomount/internal/interaction"
	"github.com/hsldymq/gomount/internal/status"
)

type Driver struct{}

func NewDriver() *Driver {
	return &Driver{}
}

func (d *Driver) Type() string {
	return "webdav"
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

	confFile, secretsPath, cleanup, err := d.createCredentialFiles(entry)
	if err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    err,
		}
	}
	if cleanup != nil {
		defer cleanup()
	}

	cmd := d.buildMountCommand(entry, confFile)

	if interaction.NeedsPrivilege() {
		if secretsPath != "" {
			if err := chownToRoot(secretsPath); err != nil {
				return &drivers.DriverError{
					Driver: d.Type(),
					Op:     "mount",
					Entry:  entry.Name,
					Err:    fmt.Errorf("failed to chown secrets file: %w", err),
				}
			}
			if err := chownToRoot(confFile); err != nil {
				return &drivers.DriverError{
					Driver: d.Type(),
					Op:     "mount",
					Entry:  entry.Name,
					Err:    fmt.Errorf("failed to chown conf file: %w", err),
				}
			}
		}
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
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "mount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("mount.davfs failed: %w", err),
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

	if err := interaction.RunCommand(cmd); err != nil {
		return &drivers.DriverError{
			Driver: d.Type(),
			Op:     "unmount",
			Entry:  entry.Name,
			Err:    fmt.Errorf("umount failed: %w", err),
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
			"type": "webdav",
			"url":  entry.WebDAV.URL,
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
	if entry.WebDAV == nil {
		return fmt.Errorf("webdav config is required for webdav driver")
	}
	if entry.WebDAV.URL == "" {
		return fmt.Errorf("webdav.url is required for webdav driver")
	}
	return nil
}

func (d *Driver) createCredentialFiles(entry *config.MountEntry) (string, string, func(), error) {
	if entry.WebDAV.Username == "" && entry.WebDAV.Password == "" {
		return "", "", nil, nil
	}

	secretsTmp, err := os.CreateTemp("", "gomount_webdav_secrets_*.txt")
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to create secrets file: %w", err)
	}

	if err := secretsTmp.Chmod(0600); err != nil {
		os.Remove(secretsTmp.Name())
		secretsTmp.Close()
		return "", "", nil, fmt.Errorf("failed to set secrets file permissions: %w", err)
	}

	content := fmt.Sprintf("%s %s %s\n", entry.WebDAV.URL, entry.WebDAV.Username, entry.WebDAV.Password)
	if _, err := secretsTmp.WriteString(content); err != nil {
		os.Remove(secretsTmp.Name())
		secretsTmp.Close()
		return "", "", nil, fmt.Errorf("failed to write secrets: %w", err)
	}
	secretsPath := secretsTmp.Name()
	secretsTmp.Close()

	confTmp, err := os.CreateTemp("", "gomount_webdav_conf_*.conf")
	if err != nil {
		os.Remove(secretsPath)
		return "", "", nil, fmt.Errorf("failed to create conf file: %w", err)
	}

	if err := confTmp.Chmod(0600); err != nil {
		os.Remove(secretsPath)
		os.Remove(confTmp.Name())
		confTmp.Close()
		return "", "", nil, fmt.Errorf("failed to set conf file permissions: %w", err)
	}

	if _, err := confTmp.WriteString(fmt.Sprintf("secrets %s\n", secretsPath)); err != nil {
		os.Remove(secretsPath)
		os.Remove(confTmp.Name())
		confTmp.Close()
		return "", "", nil, fmt.Errorf("failed to write conf: %w", err)
	}
	confPath := confTmp.Name()
	confTmp.Close()

	cleanup := func() {
		os.Remove(secretsPath)
		os.Remove(confPath)
	}

	return confPath, secretsPath, cleanup, nil
}

func chownToRoot(path string) error {
	cmd := exec.Command("sudo", "chown", "root:root", path)
	return cmd.Run()
}

func (d *Driver) buildMountCommand(entry *config.MountEntry, confFile string) *exec.Cmd {
	var options []string

	options = append(options,
		fmt.Sprintf("uid=%d", os.Getuid()),
		fmt.Sprintf("gid=%d", os.Getgid()),
	)

	if confFile != "" {
		options = append(options, fmt.Sprintf("conf=%s", confFile))
	}

	if entry.Options != nil {
		if fileMode, ok := entry.Options["file_mode"]; ok {
			options = append(options, fmt.Sprintf("file_mode=%v", fileMode))
		}
		if dirMode, ok := entry.Options["dir_mode"]; ok {
			options = append(options, fmt.Sprintf("dir_mode=%v", dirMode))
		}
	}

	args := []string{
		entry.WebDAV.URL,
		entry.MountDirPath,
	}

	if len(options) > 0 {
		args = append(args, "-o", strings.Join(options, ","))
	}

	return exec.Command("mount.davfs", args...)
}
