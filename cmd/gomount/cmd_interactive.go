package main

import (
	"fmt"
	"os"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/tui"
	"github.com/spf13/cobra"
)

var interactiveCmd = &cobra.Command{
	Use:     "interactive",
	Aliases: []string{"i"},
	Short:   "交互式挂载管理",
	Long: `显示交互式选择菜单，可以选择要挂载或卸载的共享。
这是默认的交互式操作模式。`,
	RunE: runInteractive,
}

func runInteractive(cmd *cobra.Command, args []string) error {
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

	result := tui.SelectMountAction(entries)
	if result.Cancelled {
		fmt.Println("Cancelled")
		return nil
	}
	if len(result.ToMount) == 0 && len(result.ToUnmount) == 0 {
		return nil
	}

	var failCount int
	for _, entry := range result.ToUnmount {
		resp, err := client.Unmount(entry.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR %s: %v\n", entry.Name, err)
			failCount++
		} else if !resp.Success {
			fmt.Fprintf(os.Stderr, "  ERROR %s: %s\n", entry.Name, resp.Message)
			failCount++
		} else {
			fmt.Printf("  %s: unmounted\n", entry.Name)
		}
	}

	for _, entry := range result.ToMount {
		resp, err := client.Mount(entry.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR %s: %v\n", entry.Name, err)
			failCount++
		} else if !resp.Success {
			fmt.Fprintf(os.Stderr, "  ERROR %s: %s\n", entry.Name, resp.Message)
			failCount++
		} else {
			fmt.Printf("  %s: mounted\n", entry.Name)
		}
	}

	if failCount > 0 {
		return fmt.Errorf("%d operation(s) failed", failCount)
	}
	return nil
}
