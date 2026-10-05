package views

import (
	"strings"

	"github.com/BitwiseSang/dots/internal/ui/components"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ViewType int

const (
	ViewHome ViewType = iota
	ViewBackup
	ViewSetup
	ViewEdit
	ViewBrowse
	ViewAddConfig
	ViewWizard
)

type NavigateMsg struct {
	View ViewType
	Path string
}

type TickMsg struct{}

type menuItem struct {
	key   string
	icon  string
	title string
	desc  string
	view  ViewType
}

type HomeModel struct {
	items    []menuItem
	cursor   int
	width    int
	height   int
	animStep int
	repoPath string
}

func NewHomeModel(repoPath string) HomeModel {
	return HomeModel{
		items: []menuItem{
			{key: "b", icon: theme.IconBackup, title: "Backup", desc: "Sync system configs into repo", view: ViewBackup},
			{key: "s", icon: theme.IconSetup, title: "Setup", desc: "Symlink repo configs to system", view: ViewSetup},
			{key: "e", icon: theme.IconEdit, title: "Edit", desc: "Open configs in your editor", view: ViewEdit},
			{key: "a", icon: "", title: "Add", desc: "Track a new configuration", view: ViewAddConfig},
			{key: "f", icon: theme.IconBrowse, title: "Browse", desc: "Explore filesystem & track configs", view: ViewBrowse},
		},
		cursor:   0,
		repoPath: repoPath,
	}
}

func (m *HomeModel) SetCursor(idx int) {
	if idx >= 0 && idx < len(m.items) {
		m.cursor = idx
	}
}

func (m *HomeModel) SetCursorForView(v ViewType) {
	for i, item := range m.items {
		if item.view == v {
			m.cursor = i
			return
		}
	}
}

func (m HomeModel) Cursor() int {
	return m.cursor
}

func (m HomeModel) Init() tea.Cmd {
	return nil
}

func (m HomeModel) Update(msg tea.Msg) (HomeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		m.animStep++
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			return m, func() tea.Msg {
				return NavigateMsg{View: m.items[m.cursor].view}
			}
		case "b":
			m.SetCursorForView(ViewBackup)
			return m, func() tea.Msg { return NavigateMsg{View: ViewBackup} }
		case "s":
			m.SetCursorForView(ViewSetup)
			return m, func() tea.Msg { return NavigateMsg{View: ViewSetup} }
		case "e":
			m.SetCursorForView(ViewEdit)
			return m, func() tea.Msg { return NavigateMsg{View: ViewEdit} }
		case "a":
			m.SetCursorForView(ViewAddConfig)
			return m, func() tea.Msg { return NavigateMsg{View: ViewAddConfig} }
		case "f":
			m.SetCursorForView(ViewBrowse)
			return m, func() tea.Msg { return NavigateMsg{View: ViewBrowse} }
		}
	}
	return m, nil
}

func (m HomeModel) View() string {
	header := components.Header(m.width, m.height, m.animStep, m.repoPath)

	// Block width: cursor(3) + key(4) + icon(3) + title(12) + desc(38) = 60
	blockWidth := 60
	padLeft := (m.width - blockWidth) / 2
	if padLeft < 2 {
		padLeft = 2
	}
	indent := strings.Repeat(" ", padLeft)

	var menuBuilder strings.Builder
	for i, item := range m.items {
		isSelected := i == m.cursor

		// Unified color on selection: theme.Primary (#8B5CF6)
		activeColor := theme.Primary

		// Cursor indicator - fixed width 3
		cursorStyle := lipgloss.NewStyle().Width(3)
		cursorStr := cursorStyle.Render(" ")
		if isSelected {
			cursorStr = cursorStyle.Foreground(activeColor).Bold(true).Render(theme.IconCursor + " ")
		}

		// Hotkey badge: [b] - fixed width 4
		keyStyle := lipgloss.NewStyle().Width(4)
		keyBadge := keyStyle.Foreground(theme.Muted).Render("[" + item.key + "] ")
		if isSelected {
			keyBadge = keyStyle.Foreground(activeColor).Bold(true).Render("[" + item.key + "] ")
		}

		// Icon - fixed width 3
		iconColor := theme.Muted
		if isSelected {
			iconColor = activeColor
		}
		iconStr := lipgloss.NewStyle().Width(3).Foreground(iconColor).Bold(true).Render(item.icon + " ")

		// Title - fixed width 12
		titleStyle := lipgloss.NewStyle().Bold(true).Width(12)
		if isSelected {
			titleStyle = titleStyle.Foreground(activeColor)
		} else {
			titleStyle = titleStyle.Foreground(theme.Text)
		}
		titleStr := titleStyle.Render(item.title)

		// Description - matching cohesive active color on select
		descStyle := lipgloss.NewStyle().Foreground(theme.Muted)
		if isSelected {
			descStyle = descStyle.Foreground(activeColor)
		}
		descStr := descStyle.Render(item.desc)

		row := indent + lipgloss.JoinHorizontal(
			lipgloss.Left,
			cursorStr,
			keyBadge,
			iconStr,
			titleStr,
			descStr,
		)

		menuBuilder.WriteString(row)
		if i < len(m.items)-1 {
			menuBuilder.WriteString("\n\n")
		}
	}

	menu := lipgloss.NewStyle().
		PaddingTop(1).
		PaddingBottom(1).
		Render(menuBuilder.String())

	statusBar := components.StatusBar("Home", "enter select • b backup • s setup • e edit • a add • f browse • q quit", m.width)
	topBlock := lipgloss.JoinVertical(lipgloss.Left, header, "", menu)
	return components.PlacePinnedStatusBar(topBlock, statusBar, m.height)
}
