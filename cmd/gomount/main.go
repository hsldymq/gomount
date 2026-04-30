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
	"github.com/hsldymq/gomount/internal/status"
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
	rootCmd.AddCommand(mountCmd)
	rootCmd.AddCommand(umountCmd)
	rootCmd.AddCommand(configExampleCmd)
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

	warnings, _ := config.CheckConfigPermissions(path)
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
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

	if err := interaction.EnsureSudoCached(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: sudo authentication failed: %v\n", err)
	}

	for i, entry := range entries {
		fmt.Printf("[%d/%d] %s\n", i+1, len(entries), entry.Name)

		if entry.IsMounted {
			fmt.Printf("  Already mounted at: %s\n\n", entry.MountDirPath)
			successCount++
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
			failCount++
			continue
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

	if err := interaction.EnsureSudoCached(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: sudo authentication failed: %v\n", err)
	}

	for i, entry := range entries {
		fmt.Printf("[%d/%d] %s\n", i+1, len(entries), entry.Name)
		fmt.Printf("  From: %s\n", entry.MountDirPath)

		if err := mgr.Unmount(ctx, entry.Name); err != nil {
			fmt.Fprintf(os.Stderr, "  Failed: %v\n\n", err)
			failCount++
			continue
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
