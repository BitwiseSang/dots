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

	// Press Enter to go to Step 4 (Ignore)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != wizardStepIgnore {
		t.Fatalf("expected step to be wizardStepIgnore, got %v", m.step)
	}
	vIgnore := m.View()
	if !strings.Contains(vIgnore, "Ignored Files & Patterns") {
		t.Errorf("expected Ignored Files & Patterns in view, got: %s", vIgnore)
	}

	// Press Enter to go to Step 5 (Discover)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != wizardStepDiscover {
		t.Fatalf("expected step to be wizardStepDiscover, got %v", m.step)
	}

	// Verify only repo configs (nvim, fish, tmux) are present in selector
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
	// Step 3 -> Step 4 (Ignore)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != wizardStepIgnore {
		t.Fatalf("expected step to be wizardStepIgnore, got %v", m.step)
	}
	// Step 4 -> Step 5 (Discover)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.step != wizardStepDiscover {
		t.Fatalf("expected step to be wizardStepDiscover")
	}

	// In Step 5, unselect all, then select only item 0
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

func TestWizardModelIgnoreStepAndInSelectorIgnore(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	tmpRepo := filepath.Join(tmpDir, "repo")
	_ = os.MkdirAll(tmpRepo, 0755)
	_ = os.Mkdir(filepath.Join(tmpRepo, "nvim"), 0755)
	_ = os.Mkdir(filepath.Join(tmpRepo, "fish"), 0755)
	_ = os.WriteFile(filepath.Join(tmpRepo, "backup.sh"), []byte("backup"), 0755)

	cfg := &config.Config{
		Editor:   "nvim",
		RepoPath: tmpRepo,
	}

	m := NewWizardModel(cfg)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.repoInput.SetValue(tmpRepo)

	// Step 1 -> Step 2 -> Step 3 -> Step 4 (Ignore)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.step != wizardStepIgnore {
		t.Fatalf("expected step to be wizardStepIgnore")
	}

	// Press 'a' to add a pattern
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if !m.IsTyping() || !m.ignoreAdding {
		t.Fatalf("expected ignoreAdding to be true on 'a'")
	}
	m.ignoreInput.SetValue("custom_skip/")
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.ignoreAdding {
		t.Fatalf("expected ignoreAdding to be false after enter")
	}

	// Verify custom_skip/ was added to ignore items
	foundCustom := false
	for _, it := range m.ignoreSelector.Items {
		if it.Name == "custom_skip/" {
			foundCustom = true
			break
		}
	}
	if !foundCustom {
		t.Errorf("expected custom_skip/ in ignore items")
	}

	// Press Enter to confirm ignore rules and advance to Discover
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != wizardStepDiscover {
		t.Fatalf("expected step to be wizardStepDiscover, got %v", m.step)
	}

	// .dotignore should have been saved
	dotIgnorePath := filepath.Join(tmpRepo, ".dotignore")
	data, err := os.ReadFile(dotIgnorePath)
	if err != nil {
		t.Fatalf("failed to read .dotignore: %v", err)
	}
	if !strings.Contains(string(data), "custom_skip/") {
		t.Errorf("expected .dotignore to contain custom_skip/")
	}

	// In Discover step, find backup.sh and press 'i' to ignore it!
	initialCount := len(m.repoSpecs)
	backupIdx := -1
	for i, s := range m.repoSpecs {
		if strings.Contains(s.RepoPath, "backup") {
			backupIdx = i
			break
		}
	}
	if backupIdx == -1 {
		t.Fatalf("expected to find backup item in repoSpecs: %+v", m.repoSpecs)
	}

	m.selector.SetCursor(backupIdx)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}})

	// Should have removed backup from repoSpecs
	if len(m.repoSpecs) != initialCount-1 {
		t.Errorf("expected repoSpecs count to decrease by 1, got %d", len(m.repoSpecs))
	}
	if !strings.Contains(m.ignoreMsg, "Ignored") {
		t.Errorf("expected ignoreMsg to confirm ignore, got: %s", m.ignoreMsg)
	}

	// Verify .dotignore now contains backup.sh
	data2, _ := os.ReadFile(dotIgnorePath)
	if !strings.Contains(string(data2), "backup.sh") {
		t.Errorf("expected .dotignore to contain backup.sh after 'i' pressed: %s", string(data2))
	}
}

func TestWizardModelLongPathSupport(t *testing.T) {
	// macOS temp paths frequently exceed 80 characters (e.g. /var/folders/xx/.../T/TestName123/001)
	deepDir := filepath.Join(t.TempDir(), "subfolder_with_a_very_long_path_name_exceeding_eighty_characters_to_ensure_long_paths_work_correctly")
	_ = os.MkdirAll(deepDir, 0755)
	_ = os.Mkdir(filepath.Join(deepDir, "nvim"), 0755)
	_ = os.Mkdir(filepath.Join(deepDir, "fish"), 0755)
	_ = os.WriteFile(filepath.Join(deepDir, "tmux.conf"), []byte("test"), 0644)

	cfg := &config.Config{
		Editor:   "nvim",
		RepoPath: deepDir,
	}

	m := NewWizardModel(cfg)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.repoInput.SetValue(deepDir)

	// Step 1 -> Step 2
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Step 2 -> Step 3
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Step 3 -> Step 4
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	// Step 4 -> Step 5
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.step != wizardStepDiscover {
		t.Fatalf("expected step to be wizardStepDiscover, got %v", m.step)
	}
	if len(m.repoSpecs) != 3 {
		t.Fatalf("expected 3 configs discovered with long path (length %d), got %d", len(deepDir), len(m.repoSpecs))
	}
}
