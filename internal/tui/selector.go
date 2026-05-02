package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hsldymq/gomount/internal/config"
)

type SelectionState int

const (
	SelNone SelectionState = iota
	SelMount
	SelUnmount
)

func checkboxText(entry config.MountEntry, index int, selectedMap map[int]SelectionState) string {
	if !entry.IsMounted {
		if state, ok := selectedMap[index]; ok && state == SelMount {
			return CheckMarkPendingStyle.Render("[✓]")
		}
		return "[ ]"
	}
	if _, exists := selectedMap[index]; !exists {
		return CheckMarkMountedStyle.Render("[✓]")
	}
	return CheckMarkUnmountStyle.Render("[ ]")
}

// SelectionResult 是选择的结果
type SelectionResult struct {
	Entries []*config.MountEntry
	Cancel  bool
}

type MountActionResult struct {
	ToMount   []*config.MountEntry
	ToUnmount []*config.MountEntry
	Cancelled bool
}

// SelectorModel 交互式选择的 BubbleTea 模型
type SelectorModel struct {
	Title       string
	Mounts      []config.MountEntry
	Cursor      int
	Scroll      int
	Selected    []*config.MountEntry
	SelectedMap map[int]SelectionState
	Cancelled   bool
	Confirmed   bool
	Height      int
	Width       int
	ShowStatus  bool
	NoFallback  bool
}

// NewSelectorModel 创建新的选择器模型
func NewSelectorModel(title string, mounts []config.MountEntry, showStatus bool) SelectorModel {
	return SelectorModel{
		Title:       title,
		Mounts:      mounts,
		Cursor:      0,
		Scroll:      0,
		SelectedMap: make(map[int]SelectionState),
		ShowStatus:  showStatus,
	}
}

// Init 初始化选择器模型
func (m SelectorModel) Init() tea.Cmd {
	return nil
}

// Update 处理选择器模型的消息
func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.Cancelled = true
			return m, tea.Quit

		case "enter":
			m.Selected = nil
			for i := range m.Mounts {
				if m.SelectedMap[i] != SelNone {
					m.Selected = append(m.Selected, &m.Mounts[i])
				}
			}
			if !m.NoFallback && len(m.Selected) == 0 && len(m.Mounts) > 0 {
				m.Selected = []*config.MountEntry{&m.Mounts[m.Cursor]}
			}
			m.Confirmed = true
			return m, tea.Quit

		case " ":
			if len(m.Mounts) > 0 {
				if m.Mounts[m.Cursor].IsMounted {
					if _, exists := m.SelectedMap[m.Cursor]; exists {
						delete(m.SelectedMap, m.Cursor)
					} else {
						m.SelectedMap[m.Cursor] = SelNone
					}
				} else {
					current := m.SelectedMap[m.Cursor]
					if current == SelMount {
						m.SelectedMap[m.Cursor] = SelNone
					} else {
						m.SelectedMap[m.Cursor] = SelMount
					}
				}
			}

		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			}

		case "down", "j":
			if m.Cursor < len(m.Mounts)-1 {
				m.Cursor++
			}

		case "home", "g":
			m.Cursor = 0

		case "end", "G":
			m.Cursor = len(m.Mounts) - 1
		}

		m.adjustScroll()

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}

	return m, nil
}

// adjustScroll 调整滚动位置以保持光标可见
func (m *SelectorModel) adjustScroll() {
	// 为标题（4行）和帮助（3行）预留空间
	visibleItems := m.Height - 7
	if visibleItems < 1 {
		visibleItems = 1
	}

	if m.Cursor < m.Scroll {
		m.Scroll = m.Cursor
	} else if m.Cursor >= m.Scroll+visibleItems {
		m.Scroll = m.Cursor - visibleItems + 1
	}

	// 限制滚动范围
	if m.Scroll < 0 {
		m.Scroll = 0
	}
	maxScroll := len(m.Mounts) - visibleItems
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.Scroll > maxScroll {
		m.Scroll = maxScroll
	}
}

// View 渲染选择器模型
func (m SelectorModel) View() string {
	if m.Cancelled || m.Confirmed {
		return ""
	}

	var b strings.Builder

	// 标题
	if m.Title != "" {
		b.WriteString(TitleStyle.Render(m.Title))
		b.WriteString("\n")
	}

	b.WriteString(m.renderItems())

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("↑/k: up | ↓/j: down | space: select | enter: confirm | q/esc: cancel"))

	return b.String()
}

// renderItems 渲染可选择的条目
func (m SelectorModel) renderItems() string {
	if len(m.Mounts) == 0 {
		return DimStyle.Render("No mount entries available")
	}

	var b strings.Builder

	visibleItems := m.Height - 7
	if visibleItems < 1 {
		visibleItems = 1
	}

	end := min(m.Scroll+visibleItems, len(m.Mounts))

	for i := m.Scroll; i < end; i++ {
		entry := m.Mounts[i]
		checkbox := checkboxText(entry, i, m.SelectedMap)
		segments := entrySegments(entry)

		if i == m.Cursor {
			bg := lipgloss.Color("237")
			cursorSeg := CursorArrowStyle.Background(bg).Render("▸")
			checkboxSeg := lipgloss.NewStyle().Background(bg).Render(" " + checkbox + " ")
			restSeg := renderSegments(segments, bg)
			b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left, cursorSeg, checkboxSeg, restSeg))
		} else {
			var parts []string
			parts = append(parts, " ", " ", checkbox, " ")
			for _, seg := range segments {
				parts = append(parts, seg.Style.Render(seg.Text))
			}
			b.WriteString(strings.Join(parts, ""))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderItem 渲染单个条目
type lineSegment struct {
	Text  string
	Style lipgloss.Style
}

func entrySegments(entry config.MountEntry) []lineSegment {
	var segments []lineSegment

	var addrInfo string
	switch {
	case entry.SMB != nil:
		addrInfo = fmt.Sprintf("//%s:%d/%s", entry.SMB.Addr, entry.SMB.GetPort(), entry.SMB.ShareName)
	case entry.SSHFS != nil:
		addrInfo = fmt.Sprintf("%s:%s", entry.SSHFS.Host, entry.SSHFS.RemotePath)
	case entry.WebDAV != nil:
		addrInfo = entry.WebDAV.URL
	}

	typeLabel := fmt.Sprintf("(%s)", entry.Type)
	if entry.SSHTunnel != nil {
		typeLabel = fmt.Sprintf("(%s, tunnel: %s)", entry.Type, entry.SSHTunnel.Host)
	}

	segments = append(segments,
		lineSegment{Text: entry.Name},
		lineSegment{Text: " ", Style: lipgloss.NewStyle()},
		lineSegment{Text: "›", Style: SeparatorStyle},
		lineSegment{Text: " ", Style: lipgloss.NewStyle()},
		lineSegment{Text: addrInfo},
		lineSegment{Text: " ", Style: lipgloss.NewStyle()},
		lineSegment{Text: typeLabel},
	)

	return segments
}

func renderSegments(segments []lineSegment, bg lipgloss.Color) string {
	rendered := make([]string, len(segments))
	for i, seg := range segments {
		style := seg.Style.Background(bg)
		rendered[i] = style.Render(seg.Text)
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, rendered...)
}

// SelectEntry 显示选择器并返回选中的条目（支持多选）
func SelectEntry(title string, mounts []config.MountEntry, showStatus bool) ([]*config.MountEntry, bool) {
	model := NewSelectorModel(title, mounts, showStatus)
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return nil, false
	}

	m, ok := finalModel.(SelectorModel)
	if !ok {
		return nil, false
	}

	if m.Cancelled {
		return nil, true
	}

	return m.Selected, false
}

// SelectMountEntry 显示挂载选择器（支持多选）
func SelectMountEntry(mounts []config.MountEntry) ([]*config.MountEntry, bool) {
	return SelectEntry("Select shares to mount:", mounts, true)
}

// SelectUnmountEntry 显示卸载选择器（支持多选）
func SelectUnmountEntry(mounts []config.MountEntry) ([]*config.MountEntry, bool) {
	// 只显示已挂载的条目
	var mounted []config.MountEntry
	for _, m := range mounts {
		if m.IsMounted {
			mounted = append(mounted, m)
		}
	}

	if len(mounted) == 0 {
		return nil, false
	}

	return SelectEntry("Select shares to unmount:", mounted, true)
}

func SelectMountAction(mounts []config.MountEntry) *MountActionResult {
	model := NewSelectorModel("Select shares to mount/unmount:", mounts, true)
	model.NoFallback = true
	program := tea.NewProgram(model)

	finalModel, err := program.Run()
	if err != nil {
		return &MountActionResult{Cancelled: true}
	}

	m, ok := finalModel.(SelectorModel)
	if !ok {
		return &MountActionResult{Cancelled: true}
	}

	if m.Cancelled {
		return &MountActionResult{Cancelled: true}
	}

	result := &MountActionResult{}
	for i := range m.Mounts {
		if m.Mounts[i].IsMounted {
			if _, exists := m.SelectedMap[i]; exists {
				result.ToUnmount = append(result.ToUnmount, &m.Mounts[i])
			}
		} else if m.SelectedMap[i] == SelMount {
			result.ToMount = append(result.ToMount, &m.Mounts[i])
		}
	}

	return result
}
