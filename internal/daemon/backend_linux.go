//go:build linux

package daemon

import (
	"context"
	"fmt"

	"github.com/rclone/rclone/backend/webdav"
	_ "github.com/rclone/rclone/cmd/mount"
	"github.com/rclone/rclone/cmd/mountlib"
	"github.com/rclone/rclone/fs/config/configfile"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/rclone/rclone/fs/config/obscure"
	"github.com/rclone/rclone/vfs/vfscommon"

	"github.com/hsldymq/gomount/internal/daemonapi"
)

type RcloneMounter struct{}

func NewRcloneMounter() Mounter {
	configfile.Install()
	return &RcloneMounter{}
}

func SupportedManagedTypes() []string {
	return []string{"webdav"}
}

func (m *RcloneMounter) Mount(entry daemonapi.EntrySnapshot) (MountSession, error) {
	_, mountFn := mountlib.ResolveMountMethod("mount")
	if mountFn == nil {
		return nil, fmt.Errorf("rclone mount backend is not registered")
	}
	backendConfig := configmap.Simple{
		"type":   "webdav",
		"url":    entry.Source.URL,
		"vendor": "other",
	}
	if entry.Source.Username != "" {
		backendConfig["user"] = entry.Source.Username
	}
	if entry.Source.Password != "" {
		backendConfig["pass"] = obscure.MustObscure(entry.Source.Password)
	}
	f, err := webdav.NewFs(context.Background(), entry.Name, entry.Source.Path, backendConfig)
	if err != nil {
		return nil, err
	}
	mountPoint := mountlib.NewMountPoint(mountFn, entry.MountDirPath, f, &mountlib.Opt, &vfscommon.Opt)
	_, err = mountPoint.Mount()
	if err != nil {
		return nil, err
	}
	return &rcloneMountSession{mountPoint: mountPoint}, nil
}

type rcloneMountSession struct {
	mountPoint *mountlib.MountPoint
}

func (s *rcloneMountSession) Unmount() error {
	return s.mountPoint.Unmount()
}

func (s *rcloneMountSession) Done() <-chan error {
	return s.mountPoint.ErrChan
}
