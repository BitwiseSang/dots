package components

import (
	"strings"

	"github.com/BitwiseSang/dots/internal/ui/theme"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SelectorItem struct {
	Name     string
	Desc     string
	Path     string
	Selected bool
	Status   string
	IsDir    bool
}

type Selector struct {
	Items       []SelectorItem
	cursor      int
	focused     bool
	width       int
	height      int
	ActiveColor lipgloss.Color
	CheckColor  lipgloss.Color
}

func NewSelector(items []SelectorItem) Selector {
	return Selector{
		Items:       items,
		cursor:      0,
		focused:     true,
		ActiveColor: theme.Primary,
		CheckColor:  theme.Success,
	}
}

func (s Selector) Init() tea.Cmd {
	return nil
}

func (s Selector) Update(msg tea.Msg) (Selector, tea.Cmd) {
	if !s.focused {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.Items)-1 {
				s.cursor++
			}
		case " ":
			if len(s.Items) > 0 {
				s.Items[s.cursor].Selected = !s.Items[s.cursor].Selected
			}
		case "a":
			allSelected := true
			for _, item := range s.Items {
				if !item.Selected {
					allSelected = false
					break
				}
			}
			for i := range s.Items {
				s.Items[i].Selected = !allSelected
			}
		}
	}
	return s, nil
}

func (s Selector) View() string {
	if len(s.Items) == 0 {
		return theme.MutedStyle.Render("  No items available.")
	}

	var b strings.Builder
	for i, item := range s.Items {
		isCursor := i == s.cursor

		// Cursor pointer
		cursorStr := "   "
		if isCursor {
			cursorStr = lipgloss.NewStyle().Foreground(s.ActiveColor).Bold(true).Render(" " + theme.IconCursor + " ")
		}

		// Checkbox
		checkboxStr := lipgloss.NewStyle().Foreground(theme.Muted).Render("[ ]")
		if item.Selected {
			checkboxStr = lipgloss.NewStyle().Foreground(s.CheckColor).Bold(true).Render("[" + theme.IconInSync + "]")
		}

		// Icon
		iconStr := lipgloss.NewStyle().Foreground(theme.Secondary).Render(theme.FileIcon(item.Name, item.IsDir))

		// Item Name
		nameStyle := lipgloss.NewStyle().Width(16)
		if isCursor {
			nameStyle = nameStyle.Foreground(s.ActiveColor).Bold(true)
		} else if item.Selected {
			nameStyle = nameStyle.Foreground(theme.Text).Bold(true)
		} else {
			nameStyle = nameStyle.Foreground(theme.Text)
		}
		nameStr := nameStyle.Render(item.Name)

		// Status formatting with Nerd Font icons
		statusDesc := item.Desc
		statusStyle := lipgloss.NewStyle().Width(18)
		switch {
		case strings.Contains(statusDesc, "In sync"):
			statusStr := lipgloss.NewStyle().Foreground(theme.Success).Render(theme.IconInSync + " In sync")
			statusDesc = statusStyle.Render(statusStr)
		case strings.Contains(statusDesc, "Changed"):
			statusStr := lipgloss.NewStyle().Foreground(theme.Accent).Render(theme.IconChanged + " Changed")
			statusDesc = statusStyle.Render(statusStr)
		case strings.Contains(statusDesc, "Missing"):
			statusStr := lipgloss.NewStyle().Foreground(theme.Muted).Render(theme.IconMissing + " Missing")
			statusDesc = statusStyle.Render(statusStr)
		case strings.Contains(statusDesc, "Linked"):
			statusStr := lipgloss.NewStyle().Foreground(theme.Secondary).Render(theme.IconLinked + " Linked")
			statusDesc = statusStyle.Render(statusStr)
		default:
			statusDesc = statusStyle.Render(lipgloss.NewStyle().Foreground(theme.Subtle).Render(statusDesc))
		}

		// Path hint
		pathStr := ""
		if item.Path != "" {
			pathStr = lipgloss.NewStyle().Foreground(theme.Muted).Render(item.Path)
		}

		line := lipgloss.JoinHorizontal(
			lipgloss.Left,
			cursorStr,
			checkboxStr,
			" ",
			iconStr,
			nameStr,
			" ",
			statusDesc,
			" ",
			pathStr,
		)

		if isCursor {
			// Subtle accent indicator line
			line = lipgloss.NewStyle().
				Padding(0, 1).
				Render(line)
		} else {
			line = lipgloss.NewStyle().
				Padding(0, 1).
				Render(line)
		}

		b.WriteString(line)
		if i < len(s.Items)-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (s Selector) SelectedItems() []SelectorItem {
	var selected []SelectorItem
	for _, item := range s.Items {
		if item.Selected {
			selected = append(selected, item)
		}
	}
	return selected
}

func (s Selector) SelectedIndices() []int {
	var indices []int
	for i, item := range s.Items {
		if item.Selected {
			indices = append(indices, i)
		}
	}
	return indices
}

func (s *Selector) SetSize(w, h int) {
	s.width = w
	s.height = h
}

func (s *Selector) Focus() {
	s.focused = true
}

func (s *Selector) Blur() {
	s.focused = false
}

func (s Selector) CursorIndex() int {
	return s.cursor
}
