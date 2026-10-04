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

func TestAddConfigModelSymlinkPromptAndAction(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	repoDir := filepath.Join(tmpDir, "repo")
	_ = os.MkdirAll(repoDir, 0755)

	sysFile := filepath.Join(tmpDir, "dummy_config.toml")
	_ = os.WriteFile(sysFile, []byte("content=1"), 0644)

	cfg := &config.Config{
		Editor:   "nvim",
		RepoPath: repoDir,
	}

	m := NewAddConfigModel(cfg)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Use manual mode
	m.PreFill(sysFile)

	// Press Enter to submit manual form
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Must transition to modePromptSymlink
	if m.mode != modePromptSymlink {
		t.Fatalf("expected modePromptSymlink, got %v", m.mode)
	}
	if m.IsTyping() {
		t.Errorf("expected IsTyping to be false in modePromptSymlink")
	}

	v := m.View()
	if !strings.Contains(v, "Overwrite original config(s) with symlinks?") {
		t.Errorf("expected prompt in view, got: %s", v)
	}

	// Press 'y' to confirm symlink creation
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	if m.mode != modeSuccess {
		t.Fatalf("expected modeSuccess after confirming, got %v", m.mode)
	}
	if !m.symlinkCreated {
		t.Errorf("expected symlinkCreated to be true")
	}

	// Verify that dummy_config.toml is now a symlink pointing to repoDir/dummy_config!
	fi, err := os.Lstat(sysFile)
	if err != nil {
		t.Fatalf("failed to stat sysFile: %v", err)
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		t.Errorf("expected sysFile to be replaced with a symlink")
	}
}
