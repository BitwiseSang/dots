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

func TestWizardModelNavigationAndHeight(t *testing.T) {
	tmpRepo := t.TempDir()
	_ = os.Mkdir(filepath.Join(tmpRepo, "nvim"), 0755)
	_ = os.Mkdir(filepath.Join(tmpRepo, "fish"), 0755)
	_ = os.WriteFile(filepath.Join(tmpRepo, "tmux.conf"), []byte("test"), 0644)

	cfg := &config.Config{
		Editor:   "nvim",
		RepoPath: tmpRepo,
	}

	m := NewWizardModel(cfg)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Step 1: Repo
	if m.step != wizardStepRepo {
		t.Fatalf("expected initial step to be wizardStepRepo")
	}
	if !m.IsTyping() {
		t.Errorf("expected IsTyping to be true in wizardStepRepo")
	}

	// Set local repository
	m.repoInput.SetValue(tmpRepo)
	v := m.View()
	if !strings.Contains(v, "Will use local repository") {
		t.Errorf("expected local repository preview in view")
	}
	if lipgloss.Height(v) != 24 {
		t.Errorf("expected view height to be 24, got %d", lipgloss.Height(v))
	}

	// Press Enter to go to Step 2 (Editor)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != wizardStepEditor {
		t.Fatalf("expected step to be wizardStepEditor, got %v", m.step)
	}
	if m.IsTyping() {
		t.Errorf("expected IsTyping to be false in wizardStepEditor")
	}

	// Press Enter to go to Step 3 (Git)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != wizardStepGit {
		t.Fatalf("expected step to be wizardStepGit, got %v", m.step)
	}
	if m.IsTyping() {
		t.Errorf("expected IsTyping to be false in wizardStepGit")
	}

	// Press Enter to go to Step 4 (Discover)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != wizardStepDiscover {
		t.Fatalf("expected step to be wizardStepDiscover, got %v", m.step)
	}

	// Verify only repo configs (nvim, fish, tmux) are present in selector, NOT 110 host configs!
	if len(m.repoSpecs) != 3 {
		t.Errorf("expected exactly 3 repo configs, got %d", len(m.repoSpecs))
	}

	v4 := m.View()
	if lipgloss.Height(v4) != 24 {
		t.Errorf("expected view height to be 24 on discover step, got %d", lipgloss.Height(v4))
	}
}

func TestWizardModelCloningStep(t *testing.T) {
	cfg := &config.Config{
		Editor:   "nvim",
		RepoPath: "~/Documents/dotfiles",
	}

	m := NewWizardModel(cfg)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Enter GitHub shorthand
	m.repoInput.SetValue("BitwiseSang/nonexistent-test-repo-12345")
	m, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Must transition to wizardStepCloning
	if m.step != wizardStepCloning {
		t.Fatalf("expected step to be wizardStepCloning, got %v", m.step)
	}
	if cmd == nil {
		t.Fatalf("expected asynchronous clone command to be returned")
	}

	v := m.View()
	if !strings.Contains(v, "Cloning") {
		t.Errorf("expected cloning prompt in view")
	}
	if lipgloss.Height(v) != 24 {
		t.Errorf("expected view height to be 24 during cloning, got %d", lipgloss.Height(v))
	}

	// Test clone failure handling
	m, _ = m.Update(cloneFinishedMsg{err: os.ErrPermission, repoPath: "/tmp/foo"})
	if m.cloneErr == nil {
		t.Errorf("expected cloneErr to be stored")
	}
	vErr := m.View()
	if !strings.Contains(vErr, "Failed to clone") {
		t.Errorf("expected failure message in view")
	}
	if lipgloss.Height(vErr) != 24 {
		t.Errorf("expected view height to be 24 on clone error, got %d", lipgloss.Height(vErr))
	}
}

func TestWizardModelFinalizeRegistersAllAndLinksSelected(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	tmpRepo := filepath.Join(tmpDir, "repo")
	_ = os.MkdirAll(tmpRepo, 0755)
	_ = os.Mkdir(filepath.Join(tmpRepo, "nvim"), 0755)
	_ = os.Mkdir(filepath.Join(tmpRepo, "fish"), 0755)
	_ = os.WriteFile(filepath.Join(tmpRepo, "tmux.conf"), []byte("test"), 0644)

	cfg := &config.Config{
		Editor:   "nvim",
		RepoPath: tmpRepo,
	}

	m := NewWizardModel(cfg)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.repoInput.SetValue(tmpRepo)

	// Step 1 -> Step 2
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Step 2 -> Step 3
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Step 3 -> Step 4
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.step != wizardStepDiscover {
		t.Fatalf("expected step to be wizardStepDiscover")
	}

	// In Step 4, unselect all, then select only item 0
	for i := range m.selector.Items {
		m.selector.Items[i].Selected = false
	}
	m.selector.Items[0].Selected = true
	selectedName := m.selector.Items[0].Name

	// Press Enter to finalize
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != wizardStepComplete {
		t.Fatalf("expected step to be wizardStepComplete, got %v", m.step)
	}

	// Verify all 3 repo configs are registered in cfg.Dotfiles!
	if len(cfg.Dotfiles) != 3 {
		t.Fatalf("expected all 3 repo configs to be registered, got %d", len(cfg.Dotfiles))
	}

	// Verify only 1 config was linked
	if m.linkedCount != 1 {
		t.Errorf("expected 1 linked config, got %d", m.linkedCount)
	}

	v := m.View()
	if !strings.Contains(v, "Managing 3 total configuration(s)") {
		t.Errorf("expected view to state managing 3 configs, got: %s", v)
	}
	if !strings.Contains(v, "Successfully linked: 1") {
		t.Errorf("expected view to state 1 linked, got: %s", v)
	}
	_ = selectedName
}
