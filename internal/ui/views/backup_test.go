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

func TestBackupModelDisabledItemsAndSelection(t *testing.T) {
	tmpDir := t.TempDir()
	sysDir := filepath.Join(tmpDir, "sys")
	repoDir := filepath.Join(tmpDir, "repo")
	_ = os.MkdirAll(sysDir, 0755)
	_ = os.MkdirAll(repoDir, 0755)

	// Clean file (in sync)
	_ = os.WriteFile(filepath.Join(sysDir, "clean.conf"), []byte("same\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoDir, "clean.conf"), []byte("same\n"), 0644)

	// Changed file
	_ = os.WriteFile(filepath.Join(sysDir, "changed.conf"), []byte("new\n"), 0644)
	_ = os.WriteFile(filepath.Join(repoDir, "changed.conf"), []byte("old\n"), 0644)

	cfg := &config.Config{
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "clean", RepoPath: "clean.conf", SystemPath: filepath.Join(sysDir, "clean.conf"), Method: "copy"},
			{Name: "changed", RepoPath: "changed.conf", SystemPath: filepath.Join(sysDir, "changed.conf"), Method: "copy"},
		},
	}
	entries := dotfile.LoadEntries(cfg)

	model := NewBackupModel(entries, cfg)
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// clean should be disabled, changed should be enabled
	if !model.selector.Items[0].Disabled {
		t.Errorf("expected clean item to be disabled")
	}
	if model.selector.Items[1].Disabled {
		t.Errorf("expected changed item to not be disabled")
	}

	// Pressing space on clean item (cursor at 0) should NOT select it
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	if model.selector.Items[0].Selected {
		t.Errorf("expected disabled item to remain unselected on space")
	}

	// Move to changed item (cursor at 1) and press space -> should select it
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !model.selector.Items[1].Selected {
		t.Errorf("expected enabled item to be selected on space")
	}

	// Press 'a' (toggle all) -> should only toggle enabled items!
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if model.selector.Items[0].Selected {
		t.Errorf("expected disabled item to never be selected by 'a'")
	}
}

func TestBackupModelPhaseGitAlignment(t *testing.T) {
	cfg := &config.Config{RepoPath: "/tmp"}
	model := NewBackupModel(nil, cfg)
	model, _ = model.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	model.phase = phaseGit
	model.gitPrompt = "commit"

	view := model.View()
	lines := strings.Split(view, "\n")
	foundPrompt := false
	for _, l := range lines {
		if strings.Contains(l, "Commit changes to git repository?") {
			foundPrompt = true
			if !strings.HasPrefix(l, " ") {
				t.Errorf("expected git prompt to be indented, but was: %q", l)
			}
			break
		}
	}
	if !foundPrompt {
		t.Errorf("did not find git prompt in view")
	}
}
