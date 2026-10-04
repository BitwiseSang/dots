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
	Selected bool
	Status   string
}

type Selector struct {
	Items   []SelectorItem
	cursor  int
	focused bool
	width   int
	height  int
}

func NewSelector(items []SelectorItem) Selector {
	return Selector{
		Items:   items,
		cursor:  0,
		focused: true,
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
		return theme.MutedStyle.Render("No items available.")
	}

	var b strings.Builder
	for i, item := range s.Items {
		cursor := "  "
		if i == s.cursor {
			cursor = theme.SelectedStyle.Render("> ")
		}

		checkbox := theme.CheckboxUnchecked
		if item.Selected {
			checkbox = theme.CheckboxChecked
		}

		name := lipgloss.NewStyle().Width(20).Render(item.Name)
		desc := item.Desc

		line := lipgloss.JoinHorizontal(lipgloss.Left, cursor, checkbox, " ", name, " ", desc)

		if i == s.cursor {
			line = theme.SelectedStyle.Render(line)
		} else {
			line = theme.UnselectedStyle.Render(line)
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

func (s *Selector) ToggleAll() {
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
