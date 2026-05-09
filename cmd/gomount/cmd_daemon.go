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
	"github.com/spf13/cobra"
)

var daemonCmd = &cobra.Command{
	Use:     "daemon",
	Aliases: []string{"d"},
	Short:   "管理后台守护进程",
}

var daemonDownCmd = &cobra.Command{
	Use:   "down",
	Short: "停止守护进程",
	RunE: func(cmd *cobra.Command, args []string) error {
		return daemon.StopDaemon()
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看守护进程状态",
	RunE: func(cmd *cobra.Command, args []string) error {
		info, err := daemon.ReadDaemonInfo()
		if err != nil {
			fmt.Println("Daemon is not running")
			return nil
		}
		fmt.Printf("Daemon running (PID: %d, Port: %d)\n", info.PID, info.Port)
		return nil
	},
}

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
	daemonCfg := daemon.DaemonConfig{}
	if cfg.Daemon != nil {
		daemonCfg.Port = cfg.Daemon.Port
	}
	port := daemonCfg.GetPort()

	client, err := daemon.NewWSClient(port)
	if err == nil {
		return client, nil
	}

	if err := daemon.StartDaemon(configPath, daemonCfg); err != nil {
		return nil, fmt.Errorf("failed to start daemon: %w", err)
	}

	return daemon.NewWSClient(port)
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
