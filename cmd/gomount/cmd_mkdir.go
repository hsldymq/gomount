package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/hsldymq/gomount/internal/config"
	"github.com/hsldymq/gomount/internal/interaction"
	"github.com/spf13/cobra"
)

var mkdirDryRun bool

var mkdirCmd = &cobra.Command{
	Use:     "mkdir [name...]",
	Aliases: []string{"c"},
	Short:   "创建挂载目录",
	Long: `根据配置文件创建所有挂载条目的目录。
可指定名称创建，或不指定/使用 "*" 创建全部。
使用 --dry-run 仅预览操作。`,
	Args: cobra.MinimumNArgs(0),
	RunE: runMkdir,
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

func init() {
	mkdirCmd.Flags().BoolVar(&mkdirDryRun, "dry-run", false, "仅预览将要执行的操作")
}
