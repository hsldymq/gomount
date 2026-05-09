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
	defer client.Close()

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "No mount entry specified. Use 'gomount interactive' for interactive selection.")
		return nil
	}

	meta := getMetaInfo()

	var failCount int
	for _, name := range args {
		result, err := client.Mount([]string{name}, meta)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR %s: %v\n", name, err)
			failCount++
			continue
		}
		if result.Status == "success" {
			fmt.Printf("  %s: %s\n", name, result.Message)
		} else {
			fmt.Fprintf(os.Stderr, "  ERROR %s: %s\n", name, result.Error)
			failCount++
		}
	}

	if failCount > 0 && failCount == len(args) {
		return fmt.Errorf("%d mount(s) failed", failCount)
	}
	return nil
}
