package views

import (
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	tea "github.com/charmbracelet/bubbletea"
)

func TestEditModelSearchFiltering(t *testing.T) {
	cfg := &config.Config{
		Editor:   "nvim",
		RepoPath: "~/Documents/dotfiles",
	}
	entries := []dotfile.Entry{
		{Name: "alacritty", SystemPath: "/home/user/.config/alacritty"},
		{Name: "nvim", SystemPath: "/home/user/.config/nvim", IsDir: true},
		{Name: "kitty", SystemPath: "/home/user/.config/kitty"},
		{Name: "tmux", SystemPath: "/home/user/.tmux.conf"},
	}

	m := NewEditModel(entries, cfg)
	if len(m.filteredIndices) != 4 {
		t.Fatalf("expected 4 entries initially, got %d", len(m.filteredIndices))
	}

	// Press '/' to start filtering
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.filtering {
		t.Fatalf("expected filtering mode to be active")
	}

	// Type 'ki'
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	if len(m.filteredIndices) != 1 {
		t.Fatalf("expected 1 match for 'ki', got %d", len(m.filteredIndices))
	}
	if m.entries[m.filteredIndices[0]].Name != "kitty" {
		t.Errorf("expected 'kitty', got %s", m.entries[m.filteredIndices[0]].Name)
	}

	// Commit filter with Enter
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.filtering {
		t.Errorf("expected filtering mode to be false after Enter")
	}
	if m.filterQuery != "ki" {
		t.Errorf("expected filterQuery to be 'ki', got %s", m.filterQuery)
	}

	// Clear filter with Esc
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery to be empty after Esc")
	}
	if len(m.filteredIndices) != 4 {
		t.Fatalf("expected all 4 entries visible, got %d", len(m.filteredIndices))
	}
}
