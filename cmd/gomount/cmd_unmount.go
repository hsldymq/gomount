package main

import (
	"fmt"
	"os"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/spf13/cobra"
)

var umountCmd = &cobra.Command{
	Use:     "unmount [name...]",
	Aliases: []string{"u"},
	Short:   "卸载指定共享",
	Long:    `卸载指定名称的已挂载共享。可指定多个名称依次卸载。`,
	Args:    cobra.MinimumNArgs(0),
	RunE:    runUmount,
}

func runUmount(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	client, err := ensureDaemon(cfg)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "No mount entry specified. Use 'gomount interactive' for interactive selection.")
		return nil
	}

	var failCount int
	for _, name := range args {
		resp, err := client.Unmount(name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR %s: %v\n", name, err)
			failCount++
			continue
		}
		if resp.Success {
			fmt.Printf("  %s: %s\n", name, resp.Message)
		} else {
			fmt.Fprintf(os.Stderr, "  ERROR %s: %s\n", name, resp.Message)
			failCount++
		}
	}

	if failCount > 0 && failCount == len(args) {
		return fmt.Errorf("%d unmount(s) failed", failCount)
	}
	return nil
}
