package main

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/drivers"
	smbDriver "github.com/hsldymq/gomount/internal/drivers/smb"
	sshfsDriver "github.com/hsldymq/gomount/internal/drivers/sshfs"
	webdavDriver "github.com/hsldymq/gomount/internal/drivers/webdav"
	"github.com/hsldymq/gomount/internal/interaction"
	"github.com/hsldymq/gomount/internal/status"
	"github.com/hsldymq/gomount/internal/tui"
	"github.com/spf13/cobra"
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

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "列出所有配置的挂载点",
	Long: `显示所有配置的挂载条目及其当前挂载状态。
显示名称、来源地址、挂载路径以及每个共享是否已挂载。`,
	RunE: runList,
}

var interactiveCmd = &cobra.Command{
	Use:     "interactive",
	Aliases: []string{"i"},
	Short:   "交互式挂载管理",
	Long: `显示交互式选择菜单，可以选择要挂载或卸载的共享。
这是默认的交互式操作模式。`,
	RunE: runInteractive,
}

var mountCmd = &cobra.Command{
	Use:     "mount [name...]",
	Aliases: []string{"m"},
	Short:   "挂载指定共享",
	Long:    `挂载指定名称的共享。可指定多个名称依次挂载。`,
	Args:    cobra.MinimumNArgs(0),
	RunE:    runMount,
}

var umountCmd = &cobra.Command{
	Use:     "unmount [name...]",
	Aliases: []string{"u"},
	Short:   "卸载指定共享",
	Long:    `卸载指定名称的已挂载共享。可指定多个名称依次卸载。`,
	Args:    cobra.MinimumNArgs(0),
	RunE:    runUmount,
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
	rootCmd.AddCommand(configExampleCmd)

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
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func createDriverManager(cfg *config.Config) *drivers.Manager {
	mgr := drivers.NewManager(cfg)
	mgr.RegisterDriver(smbDriver.NewDriver())
	mgr.RegisterDriver(sshfsDriver.NewDriver())
	mgr.RegisterDriver(webdavDriver.NewDriver())
	return mgr
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	if err := status.RefreshAllStatus(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to refresh mount status: %v\n", err)
	}

	if err := tui.DisplayList(cfg.Mounts); err != nil {
		return fmt.Errorf("failed to display list: %w", err)
	}

	return nil
}

func mountEntries(ctx context.Context, mgr *drivers.Manager, entries []*config.MountEntry) (success, fail int) {
	if len(entries) == 0 {
		return 0, 0
	}

	fmt.Printf("Mounting %d share(s)...\n", len(entries))

	if err := interaction.EnsureSudoCached(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: sudo authentication failed: %v\n", err)
	}

	for i, entry := range entries {
		fmt.Printf("  [%d/%d] %s\n", i+1, len(entries), entry.Name)

		if entry.IsMounted {
			fmt.Printf("    Already mounted at: %s\n\n", entry.MountDirPath)
			success++
			continue
		}

		switch entry.Type {
		case "smb":
			fmt.Printf("    From: //%s:%d/%s\n", entry.SMB.Addr, entry.SMB.GetPort(), entry.SMB.ShareName)
		case "sshfs":
			fmt.Printf("    From: %s:%s\n", entry.SSHFS.Host, entry.SSHFS.RemotePath)
		case "webdav":
			fmt.Printf("    From: %s\n", entry.WebDAV.URL)
		}
		fmt.Printf("    To: %s\n", entry.MountDirPath)

		if err := mgr.Mount(ctx, entry.Name); err != nil {
			fmt.Fprintf(os.Stderr, "    ERROR: %s\n\n", shortError(err))
			fail++
			continue
		}

		fmt.Println("    Successfully mounted")
		success++
		fmt.Println()
	}

	return
}

func unmountEntries(ctx context.Context, mgr *drivers.Manager, entries []*config.MountEntry) (success, fail int) {
	if len(entries) == 0 {
		return 0, 0
	}

	fmt.Printf("Unmounting %d share(s)...\n", len(entries))

	if err := interaction.EnsureSudoCached(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: sudo authentication failed: %v\n", err)
	}

	for i, entry := range entries {
		fmt.Printf("  [%d/%d] %s\n", i+1, len(entries), entry.Name)
		fmt.Printf("    At: %s\n", entry.MountDirPath)

		if !entry.IsMounted {
			fmt.Printf("    Already unmounted\n\n")
			success++
			continue
		}

		if err := mgr.Unmount(ctx, entry.Name); err != nil {
			fmt.Fprintf(os.Stderr, "    ERROR: %s\n\n", shortError(err))
			fail++
			continue
		}

		fmt.Println("    Successfully unmounted")
		success++
		fmt.Println()
	}

	return
}

func runMount(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	mgr := createDriverManager(cfg)
	ctx := context.Background()

	if err := mgr.RefreshAllStatus(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to refresh mount status: %v\n", err)
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "No mount entry specified. Use 'gomount interactive' for interactive selection.")
		return nil
	}

	var entries []*config.MountEntry
	for _, name := range args {
		entry, found := cfg.FindByName(name)
		if !found {
			return fmt.Errorf("mount entry '%s' not found", name)
		}
		entries = append(entries, entry)
	}

	success, fail := mountEntries(ctx, mgr, entries)
	if fail > 0 && success == 0 {
		return fmt.Errorf("%d mount(s) failed", fail)
	}

	return nil
}

func runUmount(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	mgr := createDriverManager(cfg)
	ctx := context.Background()

	if err := mgr.RefreshAllStatus(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to refresh mount status: %v\n", err)
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "No mount entry specified. Use 'gomount interactive' for interactive selection.")
		return nil
	}

	var entries []*config.MountEntry
	for _, name := range args {
		entry, found := cfg.FindByName(name)
		if !found {
			return fmt.Errorf("mount entry '%s' not found", name)
		}
		entries = append(entries, entry)
	}

	success, fail := unmountEntries(ctx, mgr, entries)
	if fail > 0 && success == 0 {
		return fmt.Errorf("%d unmount(s) failed", fail)
	}

	return nil
}

func runInteractive(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	mgr := createDriverManager(cfg)
	ctx := context.Background()

	if err := mgr.RefreshAllStatus(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to refresh mount status: %v\n", err)
	}

	result := tui.SelectMountAction(cfg.Mounts)
	if result.Cancelled {
		fmt.Println("Cancelled")
		return nil
	}
	if len(result.ToMount) == 0 && len(result.ToUnmount) == 0 {
		return nil
	}

	uSuccess, uFail := unmountEntries(ctx, mgr, result.ToUnmount)
	mSuccess, mFail := mountEntries(ctx, mgr, result.ToMount)

	totalSuccess := uSuccess + mSuccess
	totalFail := uFail + mFail
	if totalFail > 0 && totalSuccess == 0 {
		return fmt.Errorf("all operations failed")
	}

	return nil
}

func shortError(err error) string {
	msg := err.Error()
	for {
		idx := strings.LastIndex(msg, ": ")
		if idx == -1 {
			break
		}
		msg = strings.TrimSpace(msg[idx+2:])
	}
	return msg
}
