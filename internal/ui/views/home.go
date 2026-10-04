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
)

type NavigateMsg struct {
	View ViewType
}

type menuItem struct {
	title string
	desc  string
	view  ViewType
}

type HomeModel struct {
	items  []menuItem
	cursor int
	width  int
	height int
}

func NewHomeModel() HomeModel {
	return HomeModel{
		items: []menuItem{
			{title: "📦 Backup", desc: "Copy system configs into repo", view: ViewBackup},
			{title: "🔗 Setup", desc: "Symlink repo configs to system", view: ViewSetup},
			{title: "✏️  Edit", desc: "Open configs in your editor", view: ViewEdit},
		},
		cursor: 0,
	}
}

func (m HomeModel) Init() tea.Cmd {
	return nil
}

func (m HomeModel) Update(msg tea.Msg) (HomeModel, tea.Cmd) {
	switch msg := msg.(type) {
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
		}
	}
	return m, nil
}

func (m HomeModel) View() string {
	header := components.Header(m.width)

	var menuBuilder strings.Builder
	for i, item := range m.items {
		isSelected := i == m.cursor

		titleStyle := lipgloss.NewStyle().Bold(true).Width(15)
		descStyle := lipgloss.NewStyle().Foreground(theme.Muted)
		if isSelected {
			titleStyle = titleStyle.Foreground(theme.Primary)
			descStyle = descStyle.Foreground(theme.Text)
		}

		cursor := "  "
		if isSelected {
			cursor = theme.SelectedStyle.Render("> ")
		}

		title := titleStyle.Render(item.title)
		desc := descStyle.Render(item.desc)

		row := lipgloss.JoinHorizontal(lipgloss.Left, cursor, title, " ", desc)

		rowContainer := lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Muted).
			Width(60)

		if isSelected {
			rowContainer = rowContainer.BorderForeground(theme.Primary)
		}

		menuBuilder.WriteString(rowContainer.Render(row))
		if i < len(m.items)-1 {
			menuBuilder.WriteString("\n")
		}
	}

	menu := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		PaddingTop(2).
		Render(menuBuilder.String())

	content := lipgloss.JoinVertical(lipgloss.Center, header, menu)

	contentHeight := lipgloss.Height(content)
	padHeight := m.height - contentHeight - 1
	if padHeight < 0 {
		padHeight = 0
	}
	paddedContent := lipgloss.JoinVertical(lipgloss.Top, content, strings.Repeat("\n", padHeight))

	statusBar := components.StatusBar("Home", "j/k navigate • enter select • q quit", m.width)

	return lipgloss.JoinVertical(lipgloss.Top, paddedContent, statusBar)
}
