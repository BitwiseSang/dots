package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	tea "github.com/charmbracelet/bubbletea"
)

func TestRemoveModelWorkflow(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dots_ui_remove_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("HOME", tempDir)

	repoDir := filepath.Join(tempDir, "repo")
	sysDir := filepath.Join(tempDir, "sys")
	_ = os.MkdirAll(repoDir, 0755)
	_ = os.MkdirAll(sysDir, 0755)

	repoFile := filepath.Join(repoDir, "nvim.txt")
	_ = os.WriteFile(repoFile, []byte("nvim"), 0644)
	sysFile := filepath.Join(sysDir, "nvim.txt")
	_ = os.Symlink(repoFile, sysFile)

	cfg := &config.Config{
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "nvim", RepoPath: "nvim.txt", SystemPath: sysFile},
		},
	}
	entries := dotfile.LoadEntries(cfg)

	m := NewRemoveModel(entries, cfg)
	m.width = 80
	m.height = 30

	if m.phase != removePhaseSelect {
		t.Errorf("expected phaseSelect, got %v", m.phase)
	}

	// Press Enter to proceed with cursor entry
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.phase != removePhaseMode {
		t.Fatalf("expected phaseMode, got %v", m.phase)
	}
	if len(m.selectedTargets) != 1 {
		t.Fatalf("expected 1 selected target, got %d", len(m.selectedTargets))
	}

	// Switch to Repo & Symlink mode (cursor 1)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	if m.modeCursor != 1 {
		t.Errorf("expected modeCursor 1, got %d", m.modeCursor)
	}

	// Press Enter to go to confirm
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.phase != removePhaseConfirm {
		t.Fatalf("expected phaseConfirm, got %v", m.phase)
	}

	// Press 'n' to go back
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.phase != removePhaseMode {
		t.Fatalf("expected phaseMode after cancel confirm, got %v", m.phase)
	}

	// Switch back to Symlink mode (cursor 0)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if m.modeCursor != 0 {
		t.Errorf("expected modeCursor 0, got %d", m.modeCursor)
	}

	// Press Enter: Symlink mode executes directly
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.phase != removePhaseExecute {
		t.Fatalf("expected phaseExecute, got %v", m.phase)
	}
	if cmd == nil {
		t.Fatalf("expected non-nil cmd for execution")
	}

	// Execute cmd
	msg := cmd()
	m, _ = m.Update(msg)
	if m.phase != removePhaseDone {
		t.Fatalf("expected phaseDone, got %v", m.phase)
	}
	if len(m.results) != 1 || !m.results[0].SymlinkRemoved {
		t.Errorf("expected symlink removal in results: %+v", m.results)
	}

	// Press Enter in Done returns to Home
	_, navCmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if navCmd == nil {
		t.Fatalf("expected navCmd on done")
	}
	navMsg := navCmd()
	if nm, ok := navMsg.(NavigateMsg); !ok || nm.View != ViewHome {
		t.Errorf("expected NavigateMsg to ViewHome, got %v", navMsg)
	}
}

func TestCLIRemoveModel_SymlinkExecution(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "dots_cli_remove_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	t.Setenv("HOME", tempDir)

	repoDir := filepath.Join(tempDir, "repo")
	sysDir := filepath.Join(tempDir, "sys")
	_ = os.MkdirAll(repoDir, 0755)
	_ = os.MkdirAll(sysDir, 0755)

	repoFile := filepath.Join(repoDir, "fish.txt")
	_ = os.WriteFile(repoFile, []byte("fish"), 0644)
	sysFile := filepath.Join(sysDir, "fish.txt")
	_ = os.Symlink(repoFile, sysFile)

	cfg := &config.Config{
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "fish", RepoPath: "fish.txt", SystemPath: sysFile},
		},
	}
	entries := dotfile.LoadEntries(cfg)

	cliModel := NewCLIRemoveModel(entries[0], cfg)
	// Mode 0: press enter
	updatedModel, cmd := cliModel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m := updatedModel.(CLIRemoveModel)
	if m.Step != cliStepDone {
		t.Errorf("expected cliStepDone, got %v", m.Step)
	}
	if m.Result == nil || !m.Result.SymlinkRemoved {
		t.Errorf("expected symlink to be removed: %+v", m.Result)
	}
	if cmd == nil {
		t.Errorf("expected tea.Quit command")
	}
}

func TestCLIRemoveModel_CancelFlows(t *testing.T) {
	entry := dotfile.Entry{Name: "test"}
	cfg := &config.Config{}

	// Cancel at select mode
	m := NewCLIRemoveModel(entry, cfg)
	up, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !up.(CLIRemoveModel).Cancelled {
		t.Errorf("expected Cancelled to be true on 'q'")
	}

	// Cancel at confirm
	m = NewCLIRemoveModel(entry, cfg)
	up, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}) // mode 1
	up, _ = up.(CLIRemoveModel).Update(tea.KeyMsg{Type: tea.KeyEnter})
	cm := up.(CLIRemoveModel)
	if cm.Step != cliStepConfirm {
		t.Fatalf("expected cliStepConfirm, got %v", cm.Step)
	}
	up, _ = cm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if !up.(CLIRemoveModel).Cancelled {
		t.Errorf("expected Cancelled to be true on 'n' at confirm")
	}
}

func TestRemoveModelViewAppearance(t *testing.T) {
	cfg := &config.Config{
		RepoPath: "/test/repo",
		Dotfiles: []config.DotfileSpec{
			{Name: "zsh", RepoPath: "zshrc", SystemPath: "/tmp/.zshrc"},
		},
	}
	entries := dotfile.LoadEntries(cfg)
	m := NewRemoveModel(entries, cfg)
	m.width = 100
	m.height = 30

	view := m.View()

	// Must NOT contain the old large title
	if strings.Contains(view, "REMOVE CONFIGURATIONS") {
		t.Errorf("expected view not to contain 'REMOVE CONFIGURATIONS'")
	}

	// Must contain the red subtitle
	if !strings.Contains(view, "Select configurations to remove:") {
		t.Errorf("expected view to contain 'Select configurations to remove:'")
	}

	// Verify indentation: selector lines should have padLeft spaces
	expectedIndent := strings.Repeat(" ", (100-66)/2)
	foundIndented := false
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "Select configurations to remove:") && strings.HasPrefix(line, expectedIndent) {
			foundIndented = true
			break
		}
	}
	if !foundIndented {
		t.Errorf("expected subtitle to be indented with %d spaces", (100-66)/2)
	}
}

