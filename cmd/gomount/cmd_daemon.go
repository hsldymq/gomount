package main

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"strconv"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/daemon"
	"github.com/hsldymq/gomount/internal/drivers"
	smbDriver "github.com/hsldymq/gomount/internal/drivers/smb"
	sshfsDriver "github.com/hsldymq/gomount/internal/drivers/sshfs"
	webdavDriver "github.com/hsldymq/gomount/internal/drivers/webdav"
	"github.com/hsldymq/gomount/internal/mountkit"
	"github.com/hsldymq/gomount/internal/status"
)

func runAsDaemon() {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "daemon: failed to load config: %v\n", err)
		os.Exit(1)
	}

	mountMgr := mountkit.NewManager()
	mgr := createDriverManagerWithMountKit(cfg, mountMgr)

	daemonCfg := daemon.DaemonConfig{}
	if cfg.Daemon != nil {
		daemonCfg.Port = cfg.Daemon.Port
	}

	handlers := &daemon.Handlers{
		Mount: func(ctx context.Context, name string) (string, error) {
			if err := mgr.Mount(ctx, name); err != nil {
				return "", err
			}
			return "mounted", nil
		},
		Unmount: func(ctx context.Context, name string) (string, error) {
			if err := mgr.Unmount(ctx, name); err != nil {
				return "", err
			}
			return "unmounted", nil
		},
		UnmountAll: func() {
			mountMgr.UnmountAll()
		},
		List: func() []daemon.MountEntryStatus {
			var result []daemon.MountEntryStatus
			for i := range cfg.Mounts {
				m := &cfg.Mounts[i]
				m.GetMountPath()
				mounted, _ := status.CheckStatus(m.MountDirPath)
				result = append(result, daemon.MountEntryStatus{
					Name:      m.Name,
					Type:      m.Type,
					MountPath: m.MountDirPath,
					Mounted:   mounted,
				})
			}
			return result
		},
		Shutdown: func() {
			// Shutdown will be triggered by the WebSocket stop command
		},
	}

	if err := daemon.RunDaemon(handlers, daemonCfg); err != nil {
		fmt.Fprintf(os.Stderr, "daemon: %v\n", err)
		os.Exit(1)
	}
}

func ensureDaemon(cfg *config.Config) (*daemon.WSClient, error) {
	info, err := daemon.ReadDaemonInfo()
	if err == nil {
		client, err := daemon.NewWSClient(info.Port)
		if err == nil {
			return client, nil
		}
	}

	daemonCfg := daemon.DaemonConfig{}
	if cfg.Daemon != nil {
		daemonCfg.Port = cfg.Daemon.Port
	}

	if err := daemon.StartDaemon(configPath, daemonCfg); err != nil {
		return nil, fmt.Errorf("failed to start daemon: %w", err)
	}

	info, err = daemon.ReadDaemonInfo()
	if err != nil {
		return nil, err
	}

	return daemon.NewWSClient(info.Port)
}

func getMetaInfo() daemon.MetaInfo {
	currentUser, _ := user.Current()
	uid, _ := strconv.Atoi(currentUser.Uid)
	gid, _ := strconv.Atoi(currentUser.Gid)

	home, _ := os.UserHomeDir()

	return daemon.MetaInfo{
		UID:      uid,
		GID:      gid,
		Username: currentUser.Username,
		Home:     home,
	}
}

func createDriverManagerWithMountKit(cfg *config.Config, mountMgr *mountkit.Manager) *drivers.Manager {
	mgr := drivers.NewManager(cfg)
	mgr.RegisterDriver(smbDriver.NewDriver(mountMgr))
	mgr.RegisterDriver(sshfsDriver.NewDriver(mountMgr))
	mgr.RegisterDriver(webdavDriver.NewDriver(mountMgr))
	return mgr
}
