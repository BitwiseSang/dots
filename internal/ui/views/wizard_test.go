package views

import (
	"strings"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func TestWizardModelNavigationAndHeight(t *testing.T) {
	cfg := &config.Config{
		Editor:   "nvim",
		RepoPath: "~/Documents/dotfiles",
	}

	m := NewWizardModel(cfg)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Step 1: Repo
	if m.step != wizardStepRepo {
		t.Fatalf("expected initial step to be wizardStepRepo")
	}

	// Set GitHub shorthand
	m.repoInput.SetValue("BitwiseSang/dotfiles")
	v := m.View()
	if !strings.Contains(v, "Will clone https://github.com/BitwiseSang/dotfiles.git") {
		t.Errorf("expected GitHub shorthand clone preview in view")
	}
	if lipgloss.Height(v) != 24 {
		t.Errorf("expected view height to be 24, got %d", lipgloss.Height(v))
	}

	// Press Enter to go to Step 2 (Editor)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != wizardStepEditor {
		t.Fatalf("expected step to be wizardStepEditor, got %v", m.step)
	}

	// Press Enter to go to Step 3 (Git)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != wizardStepGit {
		t.Fatalf("expected step to be wizardStepGit, got %v", m.step)
	}

	// Press Enter to go to Step 4 (Discover)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.step != wizardStepDiscover {
		t.Fatalf("expected step to be wizardStepDiscover, got %v", m.step)
	}

	v4 := m.View()
	if lipgloss.Height(v4) != 24 {
		t.Errorf("expected view height to be 24 on discover step, got %d", lipgloss.Height(v4))
	}
}
