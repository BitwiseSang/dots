package views

import (
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/BitwiseSang/dots/internal/editor"
	"github.com/BitwiseSang/dots/internal/ui/components"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type OpenEditorMsg struct {
	Path   string
	Editor string
}

type EditModel struct {
	entries []dotfile.Entry
	cfg     *config.Config
	cursor  int
	width   int
	height  int
}

func NewEditModel(entries []dotfile.Entry, cfg *config.Config) EditModel {
	return EditModel{
		entries: entries,
		cfg:     cfg,
		cursor:  0,
	}
}

func (m EditModel) Init() tea.Cmd {
	return nil
}

func (m EditModel) Update(msg tea.Msg) (EditModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewHome}
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.entries) > 0 {
				entry := m.entries[m.cursor]
				path := entry.EditPath()
				ed := editor.Resolve(m.cfg.Editor)
				return m, func() tea.Msg {
					return OpenEditorMsg{Path: path, Editor: ed}
				}
			}
		}
	}
	return m, nil
}

func (m EditModel) View() string {
	header := components.Header(m.width)

	var listBuilder strings.Builder
	for i, entry := range m.entries {
		isSelected := i == m.cursor

		cursor := "  "
		style := theme.UnselectedStyle
		if isSelected {
			cursor = theme.SelectedStyle.Render("> ")
			style = theme.SelectedStyle
		}

		icon := "📄"
		if entry.IsDir {
			icon = "📁"
		}

		name := lipgloss.NewStyle().Width(20).Render(entry.Name)
		path := lipgloss.NewStyle().Foreground(theme.Muted).Render(entry.EditPath())

		line := lipgloss.JoinHorizontal(lipgloss.Left, cursor, icon, " ", name, " ", path)
		listBuilder.WriteString(style.Render(line))

		if i < len(m.entries)-1 {
			listBuilder.WriteString("\n")
		}
	}

	content := lipgloss.NewStyle().
		Padding(2, 4).
		Render(listBuilder.String())

	contentHeight := lipgloss.Height(content) + lipgloss.Height(header)
	padHeight := m.height - contentHeight - 1
	if padHeight < 0 {
		padHeight = 0
	}
	padded := lipgloss.JoinVertical(lipgloss.Top, header, content, strings.Repeat("\n", padHeight))

	statusBar := components.StatusBar("Edit", "j/k navigate • enter edit • esc back", m.width)
	return lipgloss.JoinVertical(lipgloss.Top, padded, statusBar)
}
