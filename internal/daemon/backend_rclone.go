//go:build linux || (darwin && cgo && cmount)

package daemon

import (
	"context"
	"fmt"
	"path"

	"github.com/rclone/rclone/backend/s3"
	"github.com/rclone/rclone/backend/webdav"
	"github.com/rclone/rclone/cmd/mountlib"
	"github.com/rclone/rclone/fs"
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
	return daemonapi.ManagedTypes()
}

func (m *RcloneMounter) Mount(entry daemonapi.EntrySnapshot) (MountSession, error) {
	_, mountFn := mountlib.ResolveMountMethod("")
	if mountFn == nil {
		return nil, fmt.Errorf("rclone mount backend is not registered")
	}
	var f fs.Fs
	var err error
	switch entry.Type {
	case "webdav":
		f, err = newWebDAVFs(entry)
	case "aliyun_oss":
		f, err = newAliyunOSSFs(entry)
	default:
		return nil, fmt.Errorf("unsupported rclone backend type %q", entry.Type)
	}
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

func newWebDAVFs(entry daemonapi.EntrySnapshot) (fs.Fs, error) {
	values := configmap.Simple{"type": "webdav", "url": entry.Source.URL, "vendor": "other"}
	if entry.Source.Username != "" {
		values["user"] = entry.Source.Username
	}
	if entry.Source.Password != "" {
		values["pass"] = obscure.MustObscure(entry.Source.Password)
	}
	backendConfig, err := backendConfigWithDefaults("webdav", values)
	if err != nil {
		return nil, err
	}
	return webdav.NewFs(context.Background(), entry.Name, entry.Source.Path, backendConfig)
}

func newAliyunOSSFs(entry daemonapi.EntrySnapshot) (fs.Fs, error) {
	values := configmap.Simple{
		"type": "s3", "provider": "Alibaba", "endpoint": entry.Source.Endpoint,
		"access_key_id": entry.Source.AccessKeyID, "secret_access_key": entry.Source.AccessKeySecret,
		"env_auth": "false",
	}
	if entry.Source.SecurityToken != "" {
		values["session_token"] = entry.Source.SecurityToken
	}
	backendConfig, err := backendConfigWithDefaults("s3", values)
	if err != nil {
		return nil, err
	}
	return s3.NewFs(context.Background(), entry.Name, path.Join(entry.Source.Bucket, entry.Source.Path), backendConfig)
}

// backendConfigWithDefaults builds the same layered configuration map used by
// rclone's normal remote loader. A Simple map alone leaves typed options at
// their Go zero values because backend defaults live in registry metadata.
func backendConfigWithDefaults(backendType string, values configmap.Simple) (configmap.Mapper, error) {
	info, err := fs.Find(backendType)
	if err != nil {
		return nil, fmt.Errorf("find rclone backend %q: %w", backendType, err)
	}
	return fs.ConfigMap(info.Prefix, info.Options, "", values), nil
}

type rcloneMountSession struct{ mountPoint *mountlib.MountPoint }

func (s *rcloneMountSession) Unmount() error     { return s.mountPoint.Unmount() }
func (s *rcloneMountSession) Done() <-chan error { return s.mountPoint.ErrChan }
