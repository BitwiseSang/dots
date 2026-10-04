package components

import (
	"fmt"
	"strings"

	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/bubbles/textinput"
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
	Items           []SelectorItem
	cursor          int
	focused         bool
	width           int
	height          int
	ActiveColor     lipgloss.Color
	CheckColor      lipgloss.Color
	searchInput     textinput.Model
	filtering       bool
	filterQuery     string
	filteredIndices []int
}

func NewSelector(items []SelectorItem) Selector {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "type to filter..."
	ti.PromptStyle = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Muted)
	ti.CharLimit = 32

	indices := make([]int, len(items))
	for i := range items {
		indices[i] = i
	}

	return Selector{
		Items:           items,
		cursor:          0,
		focused:         true,
		ActiveColor:     theme.Primary,
		CheckColor:      theme.Success,
		searchInput:     ti,
		filtering:       false,
		filterQuery:     "",
		filteredIndices: indices,
	}
}

func (s *Selector) recomputeFiltered() {
	if s.filterQuery == "" {
		s.filteredIndices = make([]int, len(s.Items))
		for i := range s.Items {
			s.filteredIndices[i] = i
		}
	} else {
		s.filteredIndices = nil
		q := strings.ToLower(s.filterQuery)
		for i, it := range s.Items {
			if strings.Contains(strings.ToLower(it.Name), q) ||
				strings.Contains(strings.ToLower(it.Path), q) ||
				strings.Contains(strings.ToLower(it.Desc), q) {
				s.filteredIndices = append(s.filteredIndices, i)
			}
		}
	}
	if s.cursor >= len(s.filteredIndices) {
		s.cursor = 0
	}
	if s.cursor < 0 && len(s.filteredIndices) > 0 {
		s.cursor = 0
	}
}

func (s Selector) Init() tea.Cmd {
	return nil
}

func (s Selector) Update(msg tea.Msg) (Selector, tea.Cmd) {
	if !s.focused {
		return s, nil
	}

	if s.filtering {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "esc":
				if s.searchInput.Value() != "" {
					s.searchInput.SetValue("")
					s.filterQuery = ""
					s.recomputeFiltered()
				}
				s.filtering = false
				s.searchInput.Blur()
				return s, nil
			case "enter":
				s.filtering = false
				s.searchInput.Blur()
				return s, nil
			case "tab":
				if len(s.filteredIndices) > 0 {
					actualIdx := s.filteredIndices[s.cursor]
					s.Items[actualIdx].Selected = !s.Items[actualIdx].Selected
				}
				return s, nil
			case "up":
				if s.cursor > 0 {
					s.cursor--
				}
				return s, nil
			case "down":
				if s.cursor < len(s.filteredIndices)-1 {
					s.cursor++
				}
				return s, nil
			}
		}

		var cmd tea.Cmd
		s.searchInput, cmd = s.searchInput.Update(msg)
		if s.searchInput.Value() != s.filterQuery {
			s.filterQuery = s.searchInput.Value()
			s.recomputeFiltered()
		}
		return s, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "/":
			s.filtering = true
			s.searchInput.Focus()
			return s, textinput.Blink
		case "esc":
			if s.filterQuery != "" {
				s.filterQuery = ""
				s.searchInput.SetValue("")
				s.recomputeFiltered()
				return s, nil
			}
		case "up", "k":
			if s.cursor > 0 {
				s.cursor--
			}
		case "down", "j":
			if s.cursor < len(s.filteredIndices)-1 {
				s.cursor++
			}
		case " ":
			if len(s.filteredIndices) > 0 {
				actualIdx := s.filteredIndices[s.cursor]
				s.Items[actualIdx].Selected = !s.Items[actualIdx].Selected
			}
		case "a":
			if len(s.filteredIndices) > 0 {
				allSelected := true
				for _, idx := range s.filteredIndices {
					if !s.Items[idx].Selected {
						allSelected = false
						break
					}
				}
				for _, idx := range s.filteredIndices {
					s.Items[idx].Selected = !allSelected
				}
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

	if s.filtering || s.filterQuery != "" {
		countBadge := fmt.Sprintf("(%d/%d)", len(s.filteredIndices), len(s.Items))
		badgeStyle := lipgloss.NewStyle().Foreground(theme.Muted).Render(countBadge)
		b.WriteString("   " + s.searchInput.View() + " " + badgeStyle + "\n\n")
	}

	if len(s.filteredIndices) == 0 {
		b.WriteString(theme.MutedStyle.Render("   No matching configurations found."))
		return b.String()
	}

	for i, actualIdx := range s.filteredIndices {
		item := s.Items[actualIdx]
		isCursor := i == s.cursor

		// Cursor pointer - fixed width 3
		cursorStyle := lipgloss.NewStyle().Width(3)
		cursorStr := cursorStyle.Render(" ")
		if isCursor {
			cursorStr = cursorStyle.Foreground(s.ActiveColor).Bold(true).Render(theme.IconCursor + " ")
		}

		// Checkbox - fixed width 4. When highlighted by cursor, brackets match ActiveColor
		checkboxStyle := lipgloss.NewStyle().Width(4)
		var checkboxStr string
		if isCursor {
			if item.Selected {
				checkboxStr = checkboxStyle.Foreground(s.ActiveColor).Bold(true).Render("[" + theme.IconInSync + "] ")
			} else {
				checkboxStr = checkboxStyle.Foreground(s.ActiveColor).Render("[ ] ")
			}
		} else {
			if item.Selected {
				checkboxStr = checkboxStyle.Foreground(s.CheckColor).Bold(true).Render("[" + theme.IconInSync + "] ")
			} else {
				checkboxStr = checkboxStyle.Foreground(theme.Muted).Render("[ ] ")
			}
		}

		// Item Name - fixed width 18
		nameStyle := lipgloss.NewStyle().Width(18)
		if isCursor {
			nameStyle = nameStyle.Foreground(s.ActiveColor).Bold(true)
		} else if item.Selected {
			nameStyle = nameStyle.Foreground(theme.Text).Bold(true)
		} else {
			nameStyle = nameStyle.Foreground(theme.Text)
		}
		nameStr := nameStyle.Render(item.Name)

		// Status formatting with fixed width 18
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
			nameStr,
			" ",
			statusDesc,
			" ",
			pathStr,
		)

		b.WriteString(line)
		if i < len(s.filteredIndices)-1 {
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

func (s Selector) IsFiltering() bool {
	return s.filtering
}

func (s Selector) HasFilter() bool {
	return s.filterQuery != ""
}

func (s *Selector) ClearFilter() {
	s.filterQuery = ""
	s.searchInput.SetValue("")
	s.filtering = false
	s.searchInput.Blur()
	s.recomputeFiltered()
}

func (s Selector) FilterQuery() string {
	return s.filterQuery
}
