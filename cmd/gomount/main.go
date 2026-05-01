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
	configPath string
)

var rootCmd = &cobra.Command{
	Use:           "gomount",
	Short:         "便捷的 SMB/CIFS 挂载管理工具",
	SilenceUsage:  true,
	SilenceErrors: true,
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
	Use:     "mount [name...]",
	Aliases: []string{"m"},
	Short:   "挂载指定共享",
	Long:    `挂载指定名称的共享。可指定多个名称依次挂载。`,
	Args:    cobra.MinimumNArgs(0),
	RunE:    runMount,
}

var umountCmd = &cobra.Command{
	Use:     "umount [name...]",
	Aliases: []string{"u"},
	Short:   "卸载指定共享",
	Long:    `卸载指定名称的已挂载共享。可指定多个名称依次卸载。`,
	Args:    cobra.MinimumNArgs(0),
	RunE:    runUmount,
}

var menuCmd = &cobra.Command{
	Use:     "menu",
	Aliases: []string{"i"},
	Short:   "交互式挂载管理",
	Long: `显示交互式选择菜单，可以选择要挂载或卸载的共享。
这是默认的交互式操作模式。`,
	RunE: runMenu,
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

//go:embed usage.tmpl
var usageTemplate string

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "",
		fmt.Sprintf("配置文件路径 (默认: %s)", config.DefaultConfigPath()))

	rootCmd.AddCommand(menuCmd)
	rootCmd.AddCommand(listCmd)
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

func loadConfig() (*config.Config, error) {
	path := configPath
	if path == "" {
		path = config.DefaultConfigPath()
	}

	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return cfg, nil
}

func createDriverManager(cfg *config.Config) *drivers.Manager {
	mgr := drivers.NewManager(cfg)
	mgr.RegisterDriver(smbDriver.NewDriver())
	mgr.RegisterDriver(sshfsDriver.NewDriver())
	mgr.RegisterDriver(webdavDriver.NewDriver())
	return mgr
}

func runList(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
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

	fmt.Printf("Mounting %d share(s)...\n\n", len(entries))

	if err := interaction.EnsureSudoCached(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: sudo authentication failed: %v\n", err)
	}

	for i, entry := range entries {
		fmt.Printf("[%d/%d] %s\n", i+1, len(entries), entry.Name)

		if entry.IsMounted {
			fmt.Printf("  Already mounted at: %s\n\n", entry.MountDirPath)
			success++
			continue
		}

		switch entry.Type {
		case "smb":
			fmt.Printf("  From: //%s:%d/%s\n", entry.SMB.Addr, entry.SMB.GetPort(), entry.SMB.ShareName)
		case "sshfs":
			fmt.Printf("  From: %s:%s\n", entry.SSHFS.Host, entry.SSHFS.RemotePath)
		case "webdav":
			fmt.Printf("  From: %s\n", entry.WebDAV.URL)
		}
		fmt.Printf("  To: %s\n", entry.MountDirPath)

		if err := mgr.Mount(ctx, entry.Name); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed: %v\n\n", err)
			fail++
			continue
		}

		fmt.Println("  Successfully mounted")
		success++
		fmt.Println()
	}

	return
}

func unmountEntries(ctx context.Context, mgr *drivers.Manager, entries []*config.MountEntry) (success, fail int) {
	if len(entries) == 0 {
		return 0, 0
	}

	fmt.Printf("Unmounting %d share(s)...\n\n", len(entries))

	if err := interaction.EnsureSudoCached(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: sudo authentication failed: %v\n", err)
	}

	for i, entry := range entries {
		fmt.Printf("[%d/%d] %s\n", i+1, len(entries), entry.Name)

		if !entry.IsMounted {
			fmt.Printf("  Already unmounted\n\n")
			success++
			continue
		}

		fmt.Printf("  From: %s\n", entry.MountDirPath)

		if err := mgr.Unmount(ctx, entry.Name); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed: %v\n\n", err)
			fail++
			continue
		}

		fmt.Println("  Successfully unmounted")
		success++
		fmt.Println()
	}

	return
}

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

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "No mount entry specified. Use 'gomount menu' for interactive selection.")
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
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	mgr := createDriverManager(cfg)
	ctx := context.Background()

	if err := mgr.RefreshAllStatus(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to refresh mount status: %v\n", err)
	}

	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "No mount entry specified. Use 'gomount menu' for interactive selection.")
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

func runMenu(cmd *cobra.Command, args []string) error {
	cfg, err := loadConfig()
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

const configExample = `# gomount 配置文件示例
# 复制到 ~/.config/gomount_config.yaml 并根据需要修改
# 使用: gomount config-example > ~/.config/gomount_config.yaml

mounts:
  # SMB/CIFS 挂载
  - name: nas
    type: smb
    smb:
      addr: 192.168.1.100           # SMB 服务器地址
      # port: 445                   # 可选，默认 445
      share_name: shared_folder
      username: user
      # password: pass              # 可选，不填则挂载时交互式输入
    mount_dir_path: /mnt/nas

  # SSHFS 挂载
  # 连接细节（用户名、端口、密钥等）由 ~/.ssh/config 管理
  - name: dev-server
    type: sshfs
    sshfs:
      host: dev.example.com         # ~/.ssh/config 别名或直接 hostname
      remote_path: /home/user/projects
    mount_dir_path: /mnt/dev

  # SSHFS 通过跳板机（在 ~/.ssh/config 中配置 ProxyJump）
  # ~/.ssh/config 示例:
  #   Host internal-server
  #     HostName 192.168.1.50
  #     User developer
  #     ProxyJump bastion.example.com
  #     IdentityFile ~/.ssh/id_rsa
  #
  # - name: internal-server
  #   type: sshfs
  #   sshfs:
  #     host: internal-server
  #     remote_path: /data/shared
  #   mount_dir_path: /mnt/internal

  # WebDAV 挂载
  - name: cloud
    type: webdav
    webdav:
      url: https://cloud.example.com/remote.php/dav/files/user/
      username: user
      # password: pass
    mount_dir_path: /mnt/cloud

# 提示:
#   - mount_dir_path 支持 ~ 展开为用户主目录
#   - 建议设置配置文件权限: chmod 600 ~/.config/gomount_config.yaml
`
