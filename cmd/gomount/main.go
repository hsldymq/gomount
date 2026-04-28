package main

import (
	"context"
	"fmt"
	"os"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/drivers"
	smbDriver "github.com/hsldymq/gomount/internal/drivers/smb"
	sshfsDriver "github.com/hsldymq/gomount/internal/drivers/sshfs"
	webdavDriver "github.com/hsldymq/gomount/internal/drivers/webdav"
	"github.com/hsldymq/gomount/internal/interaction"
	"github.com/hsldymq/gomount/internal/mount"
	"github.com/hsldymq/gomount/internal/tui"
	"github.com/spf13/cobra"
)

var (
	configPath string
)

var rootCmd = &cobra.Command{
	Use:   "gomount",
	Short: "便捷的 SMB/CIFS 挂载管理工具",
	Long: `gomount 是一个用于管理 Linux 系统 SMB/CIFS 共享的 CLI 工具。
它提供了挂载和卸载 SMB 共享的交互界面，
可配置选项存储在 YAML 文件中。`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"l"},
	Short:   "列出所有配置的挂载点",
	Long: `显示所有配置的 SMB 挂载条目及其当前挂载状态。
显示名称、SMB 地址、挂载路径以及每个共享是否已挂载。`,
	RunE: runList,
}

var mountCmd = &cobra.Command{
	Use:     "mount [name]",
	Aliases: []string{"m"},
	Short:   "挂载 SMB 共享",
	Long: `挂载 SMB 共享。如果提供了名称，则挂载该特定共享。
如果未提供名称，则显示交互式选择菜单。`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMount,
}

var umountCmd = &cobra.Command{
	Use:     "umount [name]",
	Aliases: []string{"u", "unmount"},
	Short:   "卸载 SMB 共享",
	Long: `卸载 SMB 共享。如果提供了名称，则卸载该特定共享。
如果未提供名称，则显示已挂载共享的交互式选择菜单。`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUmount,
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "",
		fmt.Sprintf("配置文件路径 (默认: %s)", config.DefaultConfigPath()))

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(mountCmd)
	rootCmd.AddCommand(umountCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// loadConfig 从指定或默认路径加载配置
func loadConfig() (*config.Config, error) {
	path := configPath
	if path == "" {
		path = config.DefaultConfigPath()
	}

	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	warnings, _ := config.CheckConfigPermissions(path)
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}

	return cfg, nil
}

// createDriverManager 创建并初始化驱动管理器
func createDriverManager(cfg *config.Config) *drivers.Manager {
	mgr := drivers.NewManager(cfg)
	mgr.RegisterDriver(smbDriver.NewDriver())
	mgr.RegisterDriver(sshfsDriver.NewDriver())
	mgr.RegisterDriver(webdavDriver.NewDriver())
	return mgr
}

// prepareMountEntry 准备挂载条目，如果需要则提示输入密码
func prepareMountEntry(entry *config.MountEntry) error {
	if entry.SMB != nil && !entry.HasPassword() {
		fmt.Printf("Mounting: %s\n", entry.Name)
		fmt.Printf("SMB Address: %s:%d\n", entry.SMB.Addr, entry.SMB.GetPort())
		fmt.Printf("Username: %s\n", entry.SMB.Username)
		fmt.Println()

		password, err := interaction.PromptPassword("Enter password: ", true)
		if err != nil {
			return fmt.Errorf("failed to read password: %w", err)
		}
		entry.SMB.Password = password
	}

	return nil
}

// getDriverType 检测条目使用的驱动类型
func getDriverType(entry *config.MountEntry) string {
	if entry.Type != "" {
		return entry.Type
	}
	if entry.SSHFS != nil && entry.SSHFS.Host != "" {
		return "sshfs"
	}
	if entry.WebDAV != nil && entry.WebDAV.URL != "" {
		return "webdav"
	}
	return "smb"
}

// runList 实现列表命令
func runList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if err := mount.RefreshAllStatus(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to refresh mount status: %v\n", err)
	}

	if err := tui.DisplayList(cfg.Mounts); err != nil {
		return fmt.Errorf("failed to display list: %w", err)
	}

	return nil
}

// runMount 实现挂载命令
func runMount(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	mgr := createDriverManager(cfg)
	ctx := context.Background()

	if err := mgr.RefreshAllStatus(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to refresh mount status: %v\n", err)
	}

	var entries []*config.MountEntry

	if len(args) == 0 {
		selected, cancelled := tui.SelectMountEntry(cfg.Mounts)
		if cancelled {
			fmt.Println("Cancelled")
			return nil
		}
		if selected == nil || len(selected) == 0 {
			fmt.Println("No shares selected")
			return nil
		}
		entries = selected
	} else {
		name := args[0]
		entry, found := cfg.FindByName(name)
		if !found {
			return fmt.Errorf("mount entry '%s' not found", name)
		}
		entries = []*config.MountEntry{entry}
	}

	var successCount, failCount int
	fmt.Printf("Mounting %d share(s)...\n\n", len(entries))

	for i, entry := range entries {
		fmt.Printf("[%d/%d] %s\n", i+1, len(entries), entry.Name)

		if entry.IsMounted {
			fmt.Printf("  Already mounted at: %s\n\n", entry.MountDirPath)
			successCount++
			continue
		}

		if err := prepareMountEntry(entry); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed to prepare: %v\n\n", err)
			failCount++
			continue
		}

		fmt.Printf("  From: //%s:%d/%s\n", entry.SMB.Addr, entry.SMB.GetPort(), entry.SMB.ShareName)
		fmt.Printf("  To: %s\n", entry.MountDirPath)

		driverType := getDriverType(entry)

		if err := mgr.Mount(ctx, entry.Name); err != nil {
			// 对于 SMB 驱动，尝试使用 sudo
			if driverType == "smb" && interaction.NeedsPrivilege() {
				fmt.Println("  Privilege escalation required...")
				if err := mountWithSudo(entry); err != nil {
					fmt.Fprintf(os.Stderr, "  Failed: %v\n\n", err)
					failCount++
					continue
				}
			} else {
				fmt.Fprintf(os.Stderr, "  Failed: %v\n\n", err)
				failCount++
				continue
			}
		}

		fmt.Println("  Successfully mounted")
		successCount++
		fmt.Println()
	}

	fmt.Println("==========================================")
	fmt.Printf("Mount complete: %d succeeded, %d failed\n", successCount, failCount)
	fmt.Println("==========================================")

	if failCount > 0 && successCount == 0 {
		return fmt.Errorf("%d mount(s) failed", failCount)
	}

	return nil
}

// runUmount 实现卸载命令
func runUmount(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	mgr := createDriverManager(cfg)
	ctx := context.Background()

	if err := mgr.RefreshAllStatus(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to refresh mount status: %v\n", err)
	}

	var entries []*config.MountEntry

	if len(args) == 0 {
		selected, cancelled := tui.SelectUnmountEntry(cfg.Mounts)
		if cancelled {
			fmt.Println("Cancelled")
			return nil
		}
		if selected == nil || len(selected) == 0 {
			fmt.Println("No mounted shares available")
			return nil
		}
		entries = selected
	} else {
		name := args[0]
		entry, found := cfg.FindByName(name)
		if !found {
			return fmt.Errorf("mount entry '%s' not found", name)
		}

		if !entry.IsMounted {
			return fmt.Errorf("not mounted: %s", entry.Name)
		}
		entries = []*config.MountEntry{entry}
	}

	var successCount, failCount int
	fmt.Printf("Unmounting %d share(s)...\n\n", len(entries))

	for i, entry := range entries {
		fmt.Printf("[%d/%d] %s\n", i+1, len(entries), entry.Name)
		fmt.Printf("  From: %s\n", entry.MountDirPath)

		driverType := getDriverType(entry)

		if err := mgr.Unmount(ctx, entry.Name); err != nil {
			// 对于 SMB 驱动，尝试使用 sudo
			if driverType == "smb" && interaction.NeedsPrivilege() {
				fmt.Println("  Privilege escalation required...")
				if err := umountWithSudo(entry.MountDirPath); err != nil {
					fmt.Fprintf(os.Stderr, "  Failed: %v\n\n", err)
					failCount++
					continue
				}
			} else {
				fmt.Fprintf(os.Stderr, "  Failed: %v\n\n", err)
				failCount++
				continue
			}
		}

		fmt.Println("  Successfully unmounted")
		successCount++
		fmt.Println()
	}

	fmt.Println("==========================================")
	fmt.Printf("Unmount complete: %d succeeded, %d failed\n", successCount, failCount)
	fmt.Println("==========================================")

	if failCount > 0 && successCount == 0 {
		return fmt.Errorf("%d unmount(s) failed", failCount)
	}

	return nil
}

// mountWithSudo 尝试使用权限提升进行挂载（仅用于 SMB 驱动）
func mountWithSudo(entry *config.MountEntry) error {
	credsFile, err := mount.CreateCredentialFile(entry)
	if err != nil {
		return err
	}
	defer os.Remove(credsFile)

	cmd := mount.BuildMountCommand(entry, credsFile)

	if err := interaction.RunWithSudo(cmd); err != nil {
		return fmt.Errorf("mount with sudo failed: %w", err)
	}

	return nil
}

// umountWithSudo 尝试使用权限提升进行卸载（仅用于 SMB 驱动）
func umountWithSudo(mountPath string) error {
	cmd := mount.BuildUmountCommand(mountPath)

	if err := interaction.RunWithSudo(cmd); err != nil {
		return fmt.Errorf("unmount with sudo failed: %w", err)
	}

	return nil
}
