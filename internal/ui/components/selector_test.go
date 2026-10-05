package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectorSearchFiltering(t *testing.T) {
	items := []SelectorItem{
		{Name: "alacritty", Desc: "In sync", Path: "~/.config/alacritty"},
		{Name: "nvim", Desc: "Changed", Path: "~/.config/nvim"},
		{Name: "kitty", Desc: "Missing", Path: "~/.config/kitty"},
		{Name: "tmux", Desc: "In sync", Path: "~/.tmux.conf"},
	}

	sel := NewSelector(items)

	// Initially all 4 items visible
	if len(sel.filteredIndices) != 4 {
		t.Fatalf("expected 4 filtered indices, got %d", len(sel.filteredIndices))
	}

	// Press '/' to activate search
	sel, _ = sel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !sel.IsFiltering() {
		t.Fatalf("expected selector to be in filtering mode")
	}

	// Type 'nv'
	sel, _ = sel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	sel, _ = sel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})

	if len(sel.filteredIndices) != 1 {
		t.Fatalf("expected 1 matching item for 'nv', got %d", len(sel.filteredIndices))
	}
	if sel.Items[sel.filteredIndices[0]].Name != "nvim" {
		t.Errorf("expected 'nvim', got %s", sel.Items[sel.filteredIndices[0]].Name)
	}

	// Test 1: Press Tab to toggle selection directly while filtering
	sel, _ = sel.Update(tea.KeyMsg{Type: tea.KeyTab})
	if !sel.Items[1].Selected {
		t.Errorf("expected nvim to be selected via Tab while filtering")
	}

	// Press Enter to commit filter
	sel, _ = sel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if sel.IsFiltering() {
		t.Errorf("expected isFiltering to be false after Enter")
	}
	if !sel.HasFilter() {
		t.Errorf("expected HasFilter to be true")
	}

	// Test 2: In committed filter mode, press Space to unselect and re-select
	sel, _ = sel.Update(tea.KeyMsg{Type: tea.KeySpace})
	if sel.Items[1].Selected {
		t.Errorf("expected nvim to be unselected via Space")
	}
	sel, _ = sel.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !sel.Items[1].Selected {
		t.Errorf("expected nvim to be re-selected via Space")
	}

	// Press Esc to clear filter
	sel, _ = sel.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if sel.HasFilter() {
		t.Errorf("expected filter to be cleared after Esc")
	}
	if len(sel.filteredIndices) != 4 {
		t.Fatalf("expected all 4 items visible after clear, got %d", len(sel.filteredIndices))
	}

	// Verify nvim is STILL selected after filter cleared!
	selected := sel.SelectedItems()
	if len(selected) != 1 || selected[0].Name != "nvim" {
		t.Errorf("selection was lost after clearing filter!")
	}

	// Verify View contains items
	view := sel.View()
	if !strings.Contains(view, "alacritty") || !strings.Contains(view, "nvim") {
		t.Errorf("expected view to contain items")
	}
}

func TestSelectorCycleEnabledOnly(t *testing.T) {
	items := []SelectorItem{
		{Name: "aria2", Desc: "In sync", Disabled: true},
		{Name: "fish", Desc: "Changed", Disabled: false},
		{Name: "kitty", Desc: "In sync", Disabled: true},
		{Name: "alacritty", Desc: "Changed", Disabled: false},
		{Name: "tmux", Desc: "In sync", Disabled: true},
	}

	sel := NewSelector(items)

	// Initial cursor should start on first enabled item (fish, index 1)
	if sel.CursorIndex() != 1 {
		t.Fatalf("expected initial cursor to be 1 (fish), got %d", sel.CursorIndex())
	}

	// Press 'j' -> should skip kitty (2) and jump to alacritty (3)
	sel, _ = sel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if sel.CursorIndex() != 3 {
		t.Fatalf("expected cursor after 'j' to be 3 (alacritty), got %d", sel.CursorIndex())
	}

	// Press 'j' again -> should cycle back to fish (1)
	sel, _ = sel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if sel.CursorIndex() != 1 {
		t.Fatalf("expected cursor after second 'j' to cycle back to 1 (fish), got %d", sel.CursorIndex())
	}

	// Press 'k' -> should cycle backward to alacritty (3)
	sel, _ = sel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if sel.CursorIndex() != 3 {
		t.Fatalf("expected cursor after 'k' to cycle backward to 3 (alacritty), got %d", sel.CursorIndex())
	}

	// Case 2: All items disabled
	allDisabled := []SelectorItem{
		{Name: "a", Disabled: true},
		{Name: "b", Disabled: true},
	}
	selNone := NewSelector(allDisabled)
	if selNone.CursorIndex() != -1 {
		t.Errorf("expected cursor to be -1 when all items are disabled, got %d", selNone.CursorIndex())
	}

	// Pressing j, k, space does nothing
	selNone, _ = selNone.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if selNone.CursorIndex() != -1 {
		t.Errorf("expected cursor to stay -1 on j")
	}
	selNone, _ = selNone.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if selNone.CursorIndex() != -1 {
		t.Errorf("expected cursor to stay -1 on k")
	}
	selNone, _ = selNone.Update(tea.KeyMsg{Type: tea.KeySpace})
	if len(selNone.SelectedItems()) > 0 {
		t.Errorf("expected no items selected on space when all disabled")
	}
}
