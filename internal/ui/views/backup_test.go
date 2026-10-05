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
			{Name: "a_clean", RepoPath: "clean.conf", SystemPath: filepath.Join(sysDir, "clean.conf"), Method: "copy"},
			{Name: "b_changed", RepoPath: "changed.conf", SystemPath: filepath.Join(sysDir, "changed.conf"), Method: "copy"},
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

	// Initial cursor should start on first enabled item (changed, index 1)
	if model.selector.CursorIndex() != 1 {
		t.Fatalf("expected initial cursor to be 1 (changed), got %d", model.selector.CursorIndex())
	}

	// Press space on enabled item -> should select it
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

func TestSetupModelDisabledLinkedItems(t *testing.T) {
	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	sysDir := filepath.Join(tmpDir, "sys")
	_ = os.MkdirAll(repoDir, 0755)
	_ = os.MkdirAll(sysDir, 0755)

	// Linked config
	linkedRepo := filepath.Join(repoDir, "linked.conf")
	_ = os.WriteFile(linkedRepo, []byte("repo\n"), 0644)
	linkedSys := filepath.Join(sysDir, "linked.conf")
	_ = os.Symlink(linkedRepo, linkedSys)

	// Unlinked config
	unlinkedRepo := filepath.Join(repoDir, "unlinked.conf")
	_ = os.WriteFile(unlinkedRepo, []byte("repo\n"), 0644)
	unlinkedSys := filepath.Join(sysDir, "unlinked.conf")
	_ = os.WriteFile(unlinkedSys, []byte("sys\n"), 0644)

	cfg := &config.Config{
		RepoPath: repoDir,
		Dotfiles: []config.DotfileSpec{
			{Name: "linked", RepoPath: "linked.conf", SystemPath: linkedSys, Method: "copy"},
			{Name: "unlinked", RepoPath: "unlinked.conf", SystemPath: unlinkedSys, Method: "copy"},
		},
	}
	entries := dotfile.LoadEntries(cfg)

	setupModel := NewSetupModel(entries, cfg)

	// linked should be disabled, unlinked should be enabled
	if !setupModel.selector.Items[0].Disabled {
		t.Errorf("expected linked config to be disabled in Setup")
	}
	if setupModel.selector.Items[1].Disabled {
		t.Errorf("expected unlinked config to not be disabled in Setup")
	}

	// Cursor should start on unlinked config (index 1)
	if setupModel.selector.CursorIndex() != 1 {
		t.Errorf("expected cursor to start on unlinked item (1), got %d", setupModel.selector.CursorIndex())
	}
}

func TestAddConfigTabHeader(t *testing.T) {
	cfg := &config.Config{RepoPath: "/tmp"}
	addModel := NewAddConfigModel(cfg)
	addModel, _ = addModel.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	view := addModel.View()
	if strings.Contains(view, "[Discovered Configurations]") {
		t.Errorf("expected '[Discovered Configurations]' to be removed from Add view")
	}
	if !strings.Contains(view, "[Tab] Switch to Manual Form") {
		t.Errorf("expected '[Tab] Switch to Manual Form' to be present in Add view")
	}
}
