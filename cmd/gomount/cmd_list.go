package main

import (
	"fmt"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/daemon"
	"github.com/hsldymq/gomount/internal/tui"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "列出所有配置的挂载点",
	Long: `显示所有配置的挂载条目及其当前挂载状态。
显示名称、来源地址、挂载路径以及每个共享是否已挂载。`,
	RunE: runList,
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	client, err := ensureDaemon(cfg)
	if err != nil {
		return err
	}

	resp, err := client.List()
	if err != nil {
		return fmt.Errorf("failed to list mounts: %w", err)
	}

	entries := make([]config.MountEntry, len(resp.Mounts))
	for i, m := range resp.Mounts {
		entries[i] = config.MountEntry{
			Name:         m.Name,
			Type:         m.Type,
			MountDirPath: m.MountPath,
			IsMounted:    m.Mounted,
		}
	}

	if err := tui.DisplayList(entries); err != nil {
		return fmt.Errorf("failed to display list: %w", err)
	}

	return nil
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "管理后台守护进程",
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
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
