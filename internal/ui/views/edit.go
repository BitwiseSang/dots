package views

import (
	"fmt"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/BitwiseSang/dots/internal/editor"
	"github.com/BitwiseSang/dots/internal/ui/components"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type OpenEditorMsg struct {
	Path   string
	Editor string
}

type EditModel struct {
	entries         []dotfile.Entry
	cfg             *config.Config
	cursor          int
	width           int
	height          int
	animStep        int
	searchInput     textinput.Model
	filtering       bool
	filterQuery     string
	filteredIndices []int
}

func NewEditModel(entries []dotfile.Entry, cfg *config.Config) EditModel {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "type to filter..."
	ti.PromptStyle = lipgloss.NewStyle().Foreground(theme.Pink).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Muted)
	ti.CharLimit = 32

	indices := make([]int, len(entries))
	for i := range entries {
		indices[i] = i
	}

	return EditModel{
		entries:         entries,
		cfg:             cfg,
		cursor:          0,
		animStep:        0,
		searchInput:     ti,
		filtering:       false,
		filterQuery:     "",
		filteredIndices: indices,
	}
}

func (m *EditModel) recomputeFiltered() {
	if m.filterQuery == "" {
		m.filteredIndices = make([]int, len(m.entries))
		for i := range m.entries {
			m.filteredIndices[i] = i
		}
	} else {
		m.filteredIndices = nil
		q := strings.ToLower(m.filterQuery)
		for i, e := range m.entries {
			if strings.Contains(strings.ToLower(e.Name), q) ||
				strings.Contains(strings.ToLower(e.EditPath()), q) ||
				strings.Contains(strings.ToLower(e.SystemPath), q) {
				m.filteredIndices = append(m.filteredIndices, i)
			}
		}
	}
	if m.cursor >= len(m.filteredIndices) {
		m.cursor = 0
	}
	if m.cursor < 0 && len(m.filteredIndices) > 0 {
		m.cursor = 0
	}
}

func (m EditModel) Init() tea.Cmd {
	return nil
}

func (m EditModel) IsSearching() bool {
	return m.filtering
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
		if m.filtering {
			switch msg.String() {
			case "esc":
				if m.searchInput.Value() != "" {
					m.searchInput.SetValue("")
					m.filterQuery = ""
					m.recomputeFiltered()
				}
				m.filtering = false
				m.searchInput.Blur()
				return m, nil
			case "enter":
				m.filtering = false
				m.searchInput.Blur()
				return m, nil
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
				return m, nil
			case "down":
				if m.cursor < len(m.filteredIndices)-1 {
					m.cursor++
				}
				return m, nil
			}

			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			if m.searchInput.Value() != m.filterQuery {
				m.filterQuery = m.searchInput.Value()
				m.recomputeFiltered()
			}
			return m, cmd
		}

		switch msg.String() {
		case "/":
			m.filtering = true
			m.searchInput.Focus()
			return m, textinput.Blink

		case "esc":
			if m.filterQuery != "" {
				m.filterQuery = ""
				m.searchInput.SetValue("")
				m.recomputeFiltered()
				return m, nil
			}
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewHome}
			}

		case "tab":
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewBrowse}
			}

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filteredIndices)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.filteredIndices) > 0 {
				actualIdx := m.filteredIndices[m.cursor]
				entry := m.entries[actualIdx]
				if entry.IsDir {
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
			if len(m.filteredIndices) > 0 {
				actualIdx := m.filteredIndices[m.cursor]
				entry := m.entries[actualIdx]
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
	width := m.width
	if width <= 0 {
		width = 80
	}
	height := m.height
	if height <= 0 {
		height = 24
	}

	repoPath := ""
	if m.cfg != nil {
		repoPath = m.cfg.RepoPath
	}
	header := components.Header(width, height, m.animStep, repoPath)

	var contentBuilder strings.Builder

	blockWidth := 66
	padLeft := (width - blockWidth) / 2
	if padLeft < 2 {
		padLeft = 2
	}
	indent := strings.Repeat(" ", padLeft)

	title := indent + lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Pink).
		Render("Select configurations to open in your editor:")
	contentBuilder.WriteString(title + "\n\n")

	if m.filtering || m.filterQuery != "" {
		countBadge := fmt.Sprintf("(%d/%d)", len(m.filteredIndices), len(m.entries))
		badgeStyle := lipgloss.NewStyle().Foreground(theme.Muted).Render(countBadge)
		contentBuilder.WriteString(indent + m.searchInput.View() + " " + badgeStyle + "\n\n")
	}

	if len(m.filteredIndices) == 0 {
		contentBuilder.WriteString(indent + theme.MutedStyle.Render("No matching configurations found.\n"))
	} else {
		for i, actualIdx := range m.filteredIndices {
			entry := m.entries[actualIdx]
			isSelected := i == m.cursor

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
			if i < len(m.filteredIndices)-1 {
				contentBuilder.WriteString("\n")
			}
		}
	}

	content := lipgloss.NewStyle().
		PaddingTop(1).
		Render(contentBuilder.String())

	var statusHint string
	if m.filtering {
		statusHint = "type to filter • enter done • esc clear • ↑/↓ move"
	} else if m.filterQuery != "" {
		statusHint = "enter open • / search • esc clear • tab browse • q quit"
	} else {
		statusHint = "enter open • / search • tab browse • esc home • q quit"
	}

	statusBar := components.StatusBar("Edit", statusHint, width)
	topBlock := lipgloss.JoinVertical(lipgloss.Top, header, content)
	return components.PlacePinnedStatusBar(topBlock, statusBar, height)
}
