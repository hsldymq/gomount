package main

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

var mkdirDryRun bool

var mkdirCmd = &cobra.Command{
	Use:     "mkdir [name...]",
	Aliases: []string{"d"},
	Short:   "创建挂载目录",
	Long: `根据配置文件创建所有挂载条目的目录。
可指定名称创建，或不指定/使用 "*" 创建全部。
使用 --dry-run 仅预览操作。`,
	Args: cobra.MinimumNArgs(0),
	RunE: runMkdir,
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
	mkdirCmd.Flags().BoolVar(&mkdirDryRun, "dry-run", false, "仅预览将要执行的操作")
	rootCmd.AddCommand(mkdirCmd)
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
			printIndentedError(err)
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
			printIndentedError(err)
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

func printIndentedError(err error) {
	var de *drivers.DriverError
	if errors.As(err, &de) {
		fmt.Fprintf(os.Stderr, "    ERROR: %s\n", de.CoreMessage())
		if ctx := drivers.ErrorContext(err, de.Op); ctx != "" {
			fmt.Fprintf(os.Stderr, "           %s\n", ctx)
		}
	} else {
		msg := err.Error()
		if idx := strings.Index(msg, "\n"); idx != -1 {
			msg = msg[:idx]
		}
		fmt.Fprintf(os.Stderr, "    ERROR: %s\n", strings.TrimSpace(msg))
	}
	fmt.Println()
}

type mkdirResultKind int

const (
	mkdirResultCreated mkdirResultKind = iota
	mkdirResultOwnerOK
	mkdirResultOwnerMismatch
)

type mkdirResult struct {
	kind     mkdirResultKind
	entry    *config.MountEntry
	needSudo bool
	fileUid  int
	fileGid  int
}

type mkdirCollector struct {
	toCreate      []mkdirResult
	ownerOK       []*config.MountEntry
	ownerMismatch []mkdirResult
}

func (c *mkdirCollector) add(r mkdirResult) {
	switch r.kind {
	case mkdirResultCreated:
		c.toCreate = append(c.toCreate, r)
	case mkdirResultOwnerOK:
		c.ownerOK = append(c.ownerOK, r.entry)
	case mkdirResultOwnerMismatch:
		c.ownerMismatch = append(c.ownerMismatch, r)
	}
}

func resolveMkdirEntries(cfg *config.Config, args []string) ([]*config.MountEntry, error) {
	if len(args) == 1 && args[0] == "*" {
		entries := make([]*config.MountEntry, len(cfg.Mounts))
		for i := range cfg.Mounts {
			entries[i] = &cfg.Mounts[i]
		}
		return entries, nil
	}
	var entries []*config.MountEntry
	for _, name := range args {
		entry, found := cfg.FindByName(name)
		if !found {
			return nil, fmt.Errorf("mount entry '%s' not found", name)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func processMkdirEntry(entry *config.MountEntry, dryRun bool, currentUid, currentGid int) mkdirResult {
	info, err := os.Stat(entry.MountDirPath)
	if err == nil {
		return checkExistingDir(entry, info, currentUid, currentGid)
	}
	if !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "  ERROR: %s: %v\n", entry.Name, err)
		return mkdirResult{kind: mkdirResultOwnerOK, entry: entry}
	}
	return handleMissingDir(entry, dryRun, currentUid, currentGid)
}

func checkExistingDir(entry *config.MountEntry, info os.FileInfo, currentUid, currentGid int) mkdirResult {
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "  ERROR: %s: path exists but is not a directory: %s\n", entry.Name, entry.MountDirPath)
		return mkdirResult{kind: mkdirResultOwnerOK, entry: entry}
	}

	fileUid, fileGid, err := interaction.GetFileOwner(entry.MountDirPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: %s: failed to check owner: %v\n", entry.Name, err)
		return mkdirResult{kind: mkdirResultOwnerOK, entry: entry}
	}

	if fileUid == currentUid && fileGid == currentGid {
		return mkdirResult{kind: mkdirResultOwnerOK, entry: entry}
	}
	return mkdirResult{kind: mkdirResultOwnerMismatch, entry: entry, fileUid: fileUid, fileGid: fileGid, needSudo: true}
}

func handleMissingDir(entry *config.MountEntry, dryRun bool, currentUid, currentGid int) mkdirResult {
	if dryRun {
		return mkdirResult{kind: mkdirResultCreated, entry: entry, needSudo: predictNeedsSudo(entry.MountDirPath)}
	}

	if err := interaction.MkdirAll(entry.MountDirPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "  ERROR: %s: %v\n", entry.Name, err)
		return mkdirResult{kind: mkdirResultOwnerOK, entry: entry}
	}
	if err := interaction.Chown(entry.MountDirPath, currentUid, currentGid); err != nil {
		fmt.Fprintf(os.Stderr, "  Warning: %s: failed to set owner: %v\n", entry.Name, err)
	}
	return mkdirResult{kind: mkdirResultCreated, entry: entry}
}

func predictNeedsSudo(path string) bool {
	_, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return true
	}
	owned, _ := interaction.IsOwnedByCurrentUser(filepath.Dir(path))
	return !owned
}

func runMkdir(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return err
	}

	if len(args) == 0 {
		if !interaction.Confirm(fmt.Sprintf("将创建所有 %d 个挂载条目的目录，是否继续?", len(cfg.Mounts))) {
			fmt.Println("Cancelled")
			return nil
		}
		args = []string{"*"}
	}

	entries, err := resolveMkdirEntries(cfg, args)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		fmt.Println("No mount entries found.")
		return nil
	}

	currentUser, err := interaction.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}
	currentUid, _ := strconv.Atoi(currentUser.Uid)
	currentGid, _ := strconv.Atoi(currentUser.Gid)

	var c mkdirCollector
	for _, entry := range entries {
		c.add(processMkdirEntry(entry, mkdirDryRun, currentUid, currentGid))
	}

	if mkdirDryRun {
		printMkdirDryRun(c.toCreate, c.ownerOK, c.ownerMismatch, currentUid, currentGid)
		return nil
	}

	printMkdirResult(c.toCreate, c.ownerOK)
	promptOwnerFix(c.ownerMismatch, currentUid, currentGid)
	return nil
}

func promptOwnerFix(mismatches []mkdirResult, currentUid, currentGid int) {
	if len(mismatches) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("以下目录的所有者与当前用户不一致:")
	for _, m := range mismatches {
		fmt.Printf("  %-30s  当前: uid=%d gid=%d  期望: uid=%d gid=%d\n",
			m.entry.MountDirPath, m.fileUid, m.fileGid, currentUid, currentGid)
	}
	fmt.Println()

	if !interaction.Confirm("是否修正所有者?") {
		return
	}

	if err := interaction.EnsureSudoCached(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: sudo authentication failed: %v\n", err)
	}
	fixed := 0
	for _, m := range mismatches {
		if err := interaction.Chown(m.entry.MountDirPath, currentUid, currentGid); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR: %s: %v\n", m.entry.MountDirPath, err)
			continue
		}
		fixed++
	}
	fmt.Printf("已修正 %d 个目录的所有者\n", fixed)
}

func mkdirNameWidth(entries ...[]*config.MountEntry) int {
	maxLen := 0
	for _, list := range entries {
		for _, e := range list {
			l := len(e.Name) + 1
			if l > maxLen {
				maxLen = l
			}
		}
	}
	return maxLen
}

func printMkdirDryRun(toCreate []mkdirResult, ownerOK []*config.MountEntry, ownerMismatch []mkdirResult, currentUid, currentGid int) {
	allEntries := make([]*config.MountEntry, 0, len(toCreate)+len(ownerOK)+len(ownerMismatch))
	for _, r := range toCreate {
		allEntries = append(allEntries, r.entry)
	}
	allEntries = append(allEntries, ownerOK...)
	for _, r := range ownerMismatch {
		allEntries = append(allEntries, r.entry)
	}
	w := mkdirNameWidth(allEntries)
	fmtStr := fmt.Sprintf("  %%-%ds %%s", w)

	if len(toCreate) > 0 {
		fmt.Println("将创建以下目录:")
		for _, c := range toCreate {
			sudoTag := ""
			if c.needSudo {
				sudoTag = " (需要 sudo)"
			}
			fmt.Printf(fmtStr+"%s\n", c.entry.Name+":", c.entry.MountDirPath, sudoTag)
		}
		fmt.Println()
	}

	if len(ownerOK) > 0 {
		fmt.Println("已存在，所有者一致:")
		for _, e := range ownerOK {
			fmt.Printf(fmtStr+"\n", e.Name+":", e.MountDirPath)
		}
		fmt.Println()
	}

	if len(ownerMismatch) > 0 {
		fmt.Println("已存在，所有者不一致:")
		for _, m := range ownerMismatch {
			fmt.Printf(fmtStr+"  当前所有者: uid=%d gid=%d\n",
				m.entry.Name+":", m.entry.MountDirPath, m.fileUid, m.fileGid)
		}
		fmt.Println()
		fmt.Println("将修正所有者的目录:")
		for _, m := range ownerMismatch {
			sudoTag := ""
			if m.needSudo {
				sudoTag = " (需要 sudo)"
			}
			fmt.Printf("  %s → uid=%d gid=%d%s\n", m.entry.MountDirPath, currentUid, currentGid, sudoTag)
		}
	}
}

func printMkdirResult(toCreate []mkdirResult, ownerOK []*config.MountEntry) {
	allEntries := make([]*config.MountEntry, 0, len(toCreate)+len(ownerOK))
	for _, r := range toCreate {
		allEntries = append(allEntries, r.entry)
	}
	allEntries = append(allEntries, ownerOK...)
	w := mkdirNameWidth(allEntries)
	fmtStr := fmt.Sprintf("  %%-%ds %%s", w)

	if len(toCreate) > 0 {
		fmt.Printf("已创建 %d 个目录:\n", len(toCreate))
		for _, c := range toCreate {
			fmt.Printf(fmtStr+"\n", c.entry.Name+":", c.entry.MountDirPath)
		}
		fmt.Println()
	}
	if len(ownerOK) > 0 {
		fmt.Printf("已跳过 %d 个已存在的目录:\n", len(ownerOK))
		for _, e := range ownerOK {
			fmt.Printf(fmtStr+"\n", e.Name+":", e.MountDirPath)
		}
	}
}
