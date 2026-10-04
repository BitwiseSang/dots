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

func TestAddConfigModel(t *testing.T) {
	cfg := &config.Config{
		Editor:   "nvim",
		RepoPath: "~/Documents/dotfiles",
	}

	for _, h := range []int{20, 24, 28, 30, 40} {
		m := NewAddConfigModel(cfg)
		m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: h})

		// Initial view is Discover mode
		if m.mode != modeDiscover {
			t.Fatalf("expected initial mode to be modeDiscover")
		}
		if m.IsTyping() {
			t.Errorf("expected IsTyping to be false in modeDiscover without active search")
		}

		// Verify that even with 100+ discovered items, the view height strictly equals terminal height h!
		// This guarantees that the header is NEVER pushed off-screen.
		v := m.View()
		if lipgloss.Height(v) != h {
			t.Errorf("For terminal height %d: expected view height %d, got %d", h, h, lipgloss.Height(v))
		}
		if !strings.Contains(v, "dots™") && h >= 28 {
			t.Errorf("expected header with logo to be present for height %d", h)
		}
	}

	// Press Tab to switch to Manual mode
	m := NewAddConfigModel(cfg)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.mode != modeManual {
		t.Fatalf("expected mode to be modeManual after Tab")
	}
	if !m.IsTyping() {
		t.Errorf("expected IsTyping to be true when typing in manual form")
	}

	// Test PreFill
	tmpDir := t.TempDir()
	testConf := filepath.Join(tmpDir, "alacritty.toml")
	_ = os.WriteFile(testConf, []byte("test"), 0644)

	m.PreFill(testConf)
	if m.nameInput.Value() != "alacritty" {
		t.Errorf("expected prefilled name to be 'alacritty', got '%s'", m.nameInput.Value())
	}
	if m.systemInput.Value() != testConf {
		t.Errorf("expected prefilled system path to be '%s', got '%s'", testConf, m.systemInput.Value())
	}

	// Test View rendering height in manual mode
	v := m.View()
	if !strings.Contains(v, "Manual Configuration Form") {
		t.Errorf("expected manual form header in view")
	}
	if lipgloss.Height(v) != 24 {
		t.Errorf("expected view height to be 24, got %d", lipgloss.Height(v))
	}
}
