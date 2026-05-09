package webdav

import (
	"context"
	"fmt"
	"os"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/drivers"
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
	return "webdav"
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

	remoteName := fmt.Sprintf("gomount_webdav_%s", entry.Name)
	remotePath := mountkit.RegisterWebDAVRemote(remoteName, entry.WebDAV.URL, entry.WebDAV.Username, entry.WebDAV.Password)

	if err := d.mountMgr.Mount(ctx, remotePath, entry.MountDirPath, entry.Name); err != nil {
		return &drivers.DriverError{
			Driver: d.Type(), Op: "mount", Entry: entry.Name,
			Err: err,
		}
	}

	return nil
}

func (d *Driver) Unmount(ctx context.Context, entry *config.MountEntry) error {
	if err := d.mountMgr.Unmount(entry.MountDirPath); err != nil {
		return &drivers.DriverError{
			Driver: d.Type(), Op: "unmount", Entry: entry.Name,
			Err: err,
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
			"type": "webdav",
			"url":  entry.WebDAV.URL,
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
	if entry.WebDAV == nil {
		return fmt.Errorf("webdav config is required for webdav driver")
	}
	if entry.WebDAV.URL == "" {
		return fmt.Errorf("webdav.url is required for webdav driver")
	}
	return nil
}
