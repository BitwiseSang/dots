package views

import (
	"fmt"
	"strings"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/BitwiseSang/dots/internal/version"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewRenderer interface {
	View() string
}

func TestViewsResponsiveLayoutAndHeaderPermanence(t *testing.T) {
	cfg := &config.Config{
		RepoPath: "/home/user/test-repo",
		Editor:   "nano",
		Dotfiles: []config.DotfileSpec{
			{Name: "bashrc", RepoPath: "bashrc", SystemPath: "/home/user/.bashrc", Method: "copy"},
			{Name: "nvim", RepoPath: "nvim", SystemPath: "/home/user/.config/nvim", Method: "copy"},
		},
	}
	entries := []dotfile.Entry{
		{Name: "bashrc", RepoPath: "bashrc", SystemPath: "/home/user/.bashrc"},
		{Name: "nvim", RepoPath: "nvim", SystemPath: "/home/user/.config/nvim", IsDir: true},
	}

	testCases := []struct {
		width  int
		height int
		desc   string
	}{
		{width: 100, height: 35, desc: "Large standard"},
		{width: 80, height: 28, desc: "Standard full header threshold"},
		{width: 80, height: 24, desc: "Compact height"},
		{width: 80, height: 16, desc: "Ultra-compact height"},
		{width: 45, height: 28, desc: "Narrow width"},
		{width: 35, height: 15, desc: "Narrow and short"},
	}

	for _, tc := range testCases {
		winMsg := tea.WindowSizeMsg{Width: tc.width, Height: tc.height}

		home := NewHomeModel(cfg.RepoPath)
		home, _ = home.Update(winMsg)

		backup := NewBackupModel(entries, cfg)
		backup, _ = backup.Update(winMsg)

		setup := NewSetupModel(entries, cfg)
		setup, _ = setup.Update(winMsg)

		edit := NewEditModel(entries, cfg)
		edit, _ = edit.Update(winMsg)

		browse := NewBrowseModel(cfg)
		browse, _ = browse.Update(winMsg)

		addConfig := NewAddConfigModel(cfg)
		addConfig, _ = addConfig.Update(winMsg)

		remove := NewRemoveModel(entries, cfg)
		remove, _ = remove.Update(winMsg)

		wizard := NewWizardModel(cfg)
		wizard, _ = wizard.Update(winMsg)

		viewsToTest := []struct {
			name  string
			model viewRenderer
		}{
			{"Home", home},
			{"Backup", backup},
			{"Setup", setup},
			{"Edit", edit},
			{"Browse", browse},
			{"AddConfig", addConfig},
			{"Remove", remove},
			{"Wizard", wizard},
		}

		for _, v := range viewsToTest {
			t.Run(fmt.Sprintf("%s_%s_%dx%d", v.name, tc.desc, tc.width, tc.height), func(t *testing.T) {
				rendered := v.model.View()

				// 1. Output must not exceed target height (prevents terminal scrolling)
				h := lipgloss.Height(rendered)
				if h != tc.height {
					t.Errorf("[%s] Expected rendered height %d, got %d", v.name, tc.height, h)
				}

				// 2. Top bar must contain brand and dynamic version badge
				if !strings.Contains(rendered, "dots™") {
					t.Errorf("[%s] Missing 'dots™' brand at %dx%d", v.name, tc.width, tc.height)
				}
				expectedVersion := "v" + version.Version
				if !strings.Contains(rendered, expectedVersion) {
					t.Errorf("[%s] Missing version badge '%s' at %dx%d", v.name, expectedVersion, tc.width, tc.height)
				}

				// 3. Directory breadcrumb line must be present
				if !strings.Contains(rendered, "test-repo") {
					t.Errorf("[%s] Missing repo path 'test-repo' in header at %dx%d", v.name, tc.width, tc.height)
				}

				// 4. Three separator dots must be present
				if !strings.Contains(rendered, "•") && !strings.Contains(rendered, "·") {
					t.Errorf("[%s] Missing separator dots in header at %dx%d", v.name, tc.width, tc.height)
				}
			})
		}
	}
}
