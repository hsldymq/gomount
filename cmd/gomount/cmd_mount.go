package main

import (
	"fmt"
	"os"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/spf13/cobra"
)

var mountCmd = &cobra.Command{
	Use:     "mount [name...]",
	Aliases: []string{"m"},
	Short:   "挂载指定共享",
	Long:    `挂载指定名称的共享。可指定多个名称依次挂载。`,
	Args:    cobra.MinimumNArgs(0),
	RunE:    runMount,
}

func runMount(cmd *cobra.Command, args []string) error {
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
		resp, err := client.Mount(name)
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
		return fmt.Errorf("%d mount(s) failed", failCount)
	}
	return nil
}
