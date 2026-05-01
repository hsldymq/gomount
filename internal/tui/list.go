package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/hsldymq/gomount/internal/config"
)

func entryAddr(entry config.MountEntry) string {
	switch {
	case entry.SMB != nil:
		return fmt.Sprintf("//%s:%d/%s", entry.SMB.Addr, entry.SMB.GetPort(), entry.SMB.ShareName)
	case entry.SSHFS != nil:
		return fmt.Sprintf("%s:%s", entry.SSHFS.Host, entry.SSHFS.RemotePath)
	case entry.WebDAV != nil:
		return entry.WebDAV.URL
	}
	return ""
}

func DisplayList(mounts []config.MountEntry) error {
	data := make([][]string, 0, len(mounts))
	for _, entry := range mounts {
		status := StatusBadgeUnmounted.Render("UNMOUNTED")
		if entry.IsMounted {
			status = StatusBadgeMounted.Render("MOUNTED")
		}
		data = append(data, []string{
			entry.Name,
			entryAddr(entry),
			entry.MountDirPath,
			entry.Type,
			status,
		})
	}

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(primaryColor).PaddingLeft(1).PaddingRight(1)
	cellStyle := lipgloss.NewStyle().Foreground(subtleColor).PaddingLeft(1).PaddingRight(1)

	t := table.New().
		Headers("NAME", "ADDRESS", "MOUNT PATH", "TYPE", "STATUS").
		Rows(data...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return cellStyle
		})

	fmt.Println(t.Render())
	return nil
}
