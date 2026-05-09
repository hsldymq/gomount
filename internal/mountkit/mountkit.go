package mountkit

import (
	"context"
	"fmt"
	"sync"

	"github.com/rclone/rclone/cmd/mountlib"
	"github.com/rclone/rclone/fs/cache"
	"github.com/rclone/rclone/vfs/vfscommon"

	_ "github.com/rclone/rclone/backend/all"
	_ "github.com/rclone/rclone/cmd/cmount"
	_ "github.com/rclone/rclone/cmd/mount"
	_ "github.com/rclone/rclone/fs/config/configfile"
)

func init() {
	initRcloneConfig()
}

type MountInfo struct {
	EntryName  string
	MountPoint *mountlib.MountPoint
}

type Manager struct {
	mu     sync.Mutex
	mounts map[string]*MountInfo
}

func NewManager() *Manager {
	return &Manager{
		mounts: make(map[string]*MountInfo),
	}
}

func (m *Manager) Mount(ctx context.Context, remotePath string, mountpoint string, entryName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.mounts[mountpoint]; exists {
		return fmt.Errorf("already mounted at %s", mountpoint)
	}

	f, err := cache.Get(ctx, remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote fs: %w", err)
	}

	_, mountFn := mountlib.ResolveMountMethod("")
	if mountFn == nil {
		return fmt.Errorf("no mount implementation available")
	}

	mountOpt := mountlib.Opt
	mountOpt.Daemon = false
	mountOpt.AllowOther = false

	vfsOpt := vfscommon.Opt
	vfsOpt.CacheMode = vfscommon.CacheModeWrites

	mnt := mountlib.NewMountPoint(mountFn, mountpoint, f, &mountOpt, &vfsOpt)
	if _, err := mnt.Mount(); err != nil {
		return fmt.Errorf("mount failed: %w", err)
	}

	m.mounts[mountpoint] = &MountInfo{
		EntryName:  entryName,
		MountPoint: mnt,
	}

	go func() {
		_ = mnt.Wait()
		m.mu.Lock()
		delete(m.mounts, mountpoint)
		m.mu.Unlock()
	}()

	return nil
}

func (m *Manager) Unmount(mountpoint string) error {
	m.mu.Lock()
	info, exists := m.mounts[mountpoint]
	m.mu.Unlock()

	if !exists {
		return fmt.Errorf("not mounted at %s", mountpoint)
	}

	return info.MountPoint.Unmount()
}

func (m *Manager) UnmountAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, info := range m.mounts {
		_ = info.MountPoint.Unmount()
	}
	m.mounts = make(map[string]*MountInfo)
}

func (m *Manager) IsMounted(mountpoint string) bool {
	m.mu.Lock()
	_, exists := m.mounts[mountpoint]
	m.mu.Unlock()
	return exists
}

func (m *Manager) List() map[string]*MountInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]*MountInfo, len(m.mounts))
	for k, v := range m.mounts {
		result[k] = v
	}
	return result
}
