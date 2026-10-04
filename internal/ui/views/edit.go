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
	entries  []dotfile.Entry
	cfg      *config.Config
	cursor   int
	width    int
	height   int
	animStep int
}

func NewEditModel(entries []dotfile.Entry, cfg *config.Config) EditModel {
	return EditModel{
		entries:  entries,
		cfg:      cfg,
		cursor:   0,
		animStep: 0,
	}
}

func (m EditModel) Init() tea.Cmd {
	return nil
}

func (m EditModel) Update(msg tea.Msg) (EditModel, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		m.animStep++
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewHome}
			}

		case "tab", "f":
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewBrowse}
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
				if entry.IsDir {
					// Directory config -> transition to Browse rooted at that directory
					return m, func() tea.Msg {
						return NavigateMsg{View: ViewBrowse, Path: entry.EditPath()}
					}
				}
				path := entry.EditPath()
				ed := editor.Resolve(m.cfg.Editor)
				return m, func() tea.Msg {
					return OpenEditorMsg{Path: path, Editor: ed}
				}
			}
		case "o", "O":
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
	repoPath := ""
	if m.cfg != nil {
		repoPath = m.cfg.RepoPath
	}
	header := components.Header(m.width, m.height, m.animStep, repoPath)

	var contentBuilder strings.Builder

	blockWidth := 66
	padLeft := (m.width - blockWidth) / 2
	if padLeft < 2 {
		padLeft = 2
	}
	indent := strings.Repeat(" ", padLeft)

	title := indent + lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Pink).
		Render("Select configurations to open in your editor:")
	contentBuilder.WriteString(title + "\n\n")

	for i, entry := range m.entries {
		isSelected := i == m.cursor

		// Unified Pink active color when selected
		activeColor := theme.Pink

		cursorStyle := lipgloss.NewStyle().Width(3)
		cursorStr := cursorStyle.Render(" ")
		if isSelected {
			cursorStr = cursorStyle.Foreground(activeColor).Bold(true).Render(theme.IconCursor + " ")
		}

		nameStyle := lipgloss.NewStyle().Width(18)
		if isSelected {
			nameStyle = nameStyle.Foreground(activeColor).Bold(true)
		} else {
			nameStyle = nameStyle.Foreground(theme.Text)
		}
		nameStr := nameStyle.Render(entry.Name)

		pathStyle := lipgloss.NewStyle().Foreground(theme.Muted)
		if isSelected {
			pathStyle = pathStyle.Foreground(theme.Subtle)
		}
		pathStr := pathStyle.Render(shortenHome(entry.EditPath()))

		row := indent + lipgloss.JoinHorizontal(
			lipgloss.Left,
			cursorStr,
			nameStr,
			pathStr,
		)

		contentBuilder.WriteString(row)
		if i < len(m.entries)-1 {
			contentBuilder.WriteString("\n")
		}
	}

	content := lipgloss.NewStyle().
		PaddingTop(1).
		Render(contentBuilder.String())

	statusBar := components.StatusBar("Edit", "enter open • tab browse • esc home • q quit", m.width)
	topBlock := lipgloss.JoinVertical(lipgloss.Top, header, content)
	return components.PlacePinnedStatusBar(topBlock, statusBar, m.height)
}
