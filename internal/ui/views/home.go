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
)

type NavigateMsg struct {
	View ViewType
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
}

func NewHomeModel() HomeModel {
	return HomeModel{
		items: []menuItem{
			{key: "b", icon: theme.IconBackup, title: "Backup", desc: "Sync system configs into repo", view: ViewBackup},
			{key: "s", icon: theme.IconSetup, title: "Setup", desc: "Symlink repo configs to system", view: ViewSetup},
			{key: "e", icon: theme.IconEdit, title: "Edit", desc: "Open configs in your editor", view: ViewEdit},
			{key: "f", icon: theme.IconBrowse, title: "Browse", desc: "Explore filesystem with file picker", view: ViewBrowse},
		},
		cursor: 0,
	}
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
			return m, func() tea.Msg { return NavigateMsg{View: ViewBackup} }
		case "s":
			return m, func() tea.Msg { return NavigateMsg{View: ViewSetup} }
		case "e":
			return m, func() tea.Msg { return NavigateMsg{View: ViewEdit} }
		case "f":
			return m, func() tea.Msg { return NavigateMsg{View: ViewBrowse} }
		}
	}
	return m, nil
}

func (m HomeModel) View() string {
	header := components.Header(m.width, m.animStep)

	var menuBuilder strings.Builder
	for i, item := range m.items {
		isSelected := i == m.cursor

		// Cursor indicator
		cursorStr := "   "
		if isSelected {
			cursorStr = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Render(" " + theme.IconCursor + " ")
		}

		// Hotkey badge: [b]
		keyBadge := lipgloss.NewStyle().Foreground(theme.Muted).Render("[" + item.key + "]")
		if isSelected {
			keyBadge = lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render("[" + item.key + "]")
		}

		// Icon
		iconColor := theme.Secondary
		if isSelected {
			iconColor = theme.Pink
		}
		iconStr := lipgloss.NewStyle().Foreground(iconColor).Bold(true).Render(item.icon)

		// Title
		titleStyle := lipgloss.NewStyle().Bold(true).Width(12)
		if isSelected {
			titleStyle = titleStyle.Foreground(theme.Primary)
		} else {
			titleStyle = titleStyle.Foreground(theme.Text)
		}
		titleStr := titleStyle.Render(item.title)

		// Description
		descStyle := lipgloss.NewStyle().Foreground(theme.Muted)
		if isSelected {
			descStyle = descStyle.Foreground(theme.Subtle)
		}
		descStr := descStyle.Render(item.desc)

		row := lipgloss.JoinHorizontal(
			lipgloss.Left,
			cursorStr,
			keyBadge,
			" ",
			iconStr,
			" ",
			titleStr,
			" ",
			descStr,
		)

		menuBuilder.WriteString(row)
		if i < len(m.items)-1 {
			menuBuilder.WriteString("\n\n")
		}
	}

	menu := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		PaddingTop(1).
		PaddingBottom(1).
		Render(menuBuilder.String())

	content := lipgloss.JoinVertical(lipgloss.Center, header, menu)

	contentHeight := lipgloss.Height(content)
	padHeight := m.height - contentHeight - 3
	if padHeight < 0 {
		padHeight = 0
	}
	paddedContent := lipgloss.JoinVertical(lipgloss.Top, content, strings.Repeat("\n", padHeight))

	statusBar := components.StatusBar("Home", "j/k move • enter select • b backup • s setup • e edit • f browse • q quit", m.width)

	return lipgloss.JoinVertical(lipgloss.Top, paddedContent, statusBar)
}
