package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestBrowseModelExactHeight(t *testing.T) {
	cfg := &config.Config{
		Editor:   "nvim",
		RepoPath: "~/Documents/dotfiles",
	}

	for _, h := range []int{20, 24, 28, 30, 40} {
		m := NewBrowseModel(cfg)
		dir, _ := os.UserHomeDir()
		m.SetDirectory(dir)

		// Send WindowSizeMsg
		m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: h})

		view := m.View()
		viewHeight := lipgloss.Height(view)
		t.Logf("Height target: %d, actual: %d", h, viewHeight)

		if viewHeight != h {
			t.Errorf("For terminal height %d: expected view height %d, got %d", h, h, viewHeight)
		}
	}
}

func TestBrowseModelSearch(t *testing.T) {
	cfg := &config.Config{
		Editor:   "nvim",
		RepoPath: "~/Documents/dotfiles",
	}

	tmpDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(tmpDir, "alacritty.toml"), []byte("test"), 0644)
	_ = os.WriteFile(filepath.Join(tmpDir, "kitty.conf"), []byte("test"), 0644)
	_ = os.Mkdir(filepath.Join(tmpDir, "nvim"), 0755)

	m := NewBrowseModel(cfg)
	m.SetDirectory(tmpDir)

	if len(m.items) != 3 {
		t.Fatalf("expected 3 items in directory, got %d", len(m.items))
	}

	// Press '/' to search
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !m.filtering {
		t.Fatalf("expected filtering mode to be active")
	}

	// Type 'nv'
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})

	if len(m.filteredIndices) != 1 {
		t.Fatalf("expected 1 match for 'nv', got %d", len(m.filteredIndices))
	}
	if m.items[m.filteredIndices[0]].name != "nvim" {
		t.Errorf("expected 'nvim', got %s", m.items[m.filteredIndices[0]].name)
	}

	// Press Enter on nvim directory: should navigate into nvim/ and clear search!
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.currentDir != filepath.Join(tmpDir, "nvim") {
		t.Errorf("expected currentDir to be nvim, got %s", m.currentDir)
	}
	if m.filtering {
		t.Errorf("expected filtering to be false after drilling in")
	}
	if m.filterQuery != "" {
		t.Errorf("expected filterQuery to be reset after drilling in")
	}

	// Press 'h' to navigate back to parent
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if m.currentDir != tmpDir {
		t.Errorf("expected to return to tmpDir, got %s", m.currentDir)
	}
	if len(m.items) != 3 {
		t.Fatalf("expected 3 items after returning to parent, got %d", len(m.items))
	}

	// Test search view rendering
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	v := m.View()
	if !strings.Contains(v, "alacritty.toml") {
		t.Errorf("expected search view to show matching alacritty.toml")
	}
	if lipgloss.Height(v) != 24 {
		t.Errorf("expected view height to remain 24 during search, got %d", lipgloss.Height(v))
	}
}
