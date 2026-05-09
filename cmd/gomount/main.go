package main

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/daemon"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	//go:embed config.example.yaml
	configExample string

	//go:embed usage.tmpl
	usageTemplate string

	configPath string
)

var rootCmd = &cobra.Command{
	Use:           "gomount",
	Short:         "便捷的挂载管理工具",
	SilenceUsage:  true,
	SilenceErrors: true,
	Long: `gomount 是一个管理多种网络共享挂载的 CLI 工具。
它提供了挂载和卸载网络共享的交互界面。`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var configExampleCmd = &cobra.Command{
	Use:     "config-example",
	Aliases: []string{"ce"},
	Short:   "输出配置文件示例",
	Long:    `输出一个包含所有挂载类型的配置文件示例，可重定向到文件使用。`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprint(os.Stdout, configExample)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "",
		fmt.Sprintf("配置文件路径 (默认: %s)", config.DefaultConfigPath()))

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(interactiveCmd)
	rootCmd.AddCommand(mountCmd)
	rootCmd.AddCommand(umountCmd)
	rootCmd.AddCommand(mkdirCmd)
	rootCmd.AddCommand(configExampleCmd)

	daemonCmd.AddCommand(daemonDownCmd)
	daemonCmd.AddCommand(daemonStatusCmd)
	rootCmd.AddCommand(daemonCmd)

	cmdNameFunc := func(name string, aliases []string) string {
		if len(aliases) == 0 {
			return name
		}
		return name + " (" + strings.Join(aliases, ", ") + ")"
	}
	cobra.AddTemplateFunc("cmdName", cmdNameFunc)
	cobra.AddTemplateFunc("cmdNamePadding", func(cmd *cobra.Command) int {
		maxLen := 0
		for _, c := range cmd.Commands() {
			if !c.IsAvailableCommand() {
				continue
			}
			l := len(cmdNameFunc(c.Name(), c.Aliases))
			if l > maxLen {
				maxLen = l
			}
		}
		return maxLen
	})
	rootCmd.SetUsageTemplate(usageTemplate)
}

func main() {
	if daemon.IsDaemon() {
		runAsDaemon()
		return
	}

	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	pflag.CommandLine.Usage = func() {}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
