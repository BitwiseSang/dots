package views

import (
	"os"
	"strings"
	"testing"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/ui/components"
	"github.com/BitwiseSang/dots/internal/ui/theme"
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
		m.fp.Styles.FileSize = lipgloss.NewStyle().Width(9).Align(lipgloss.Right).Foreground(theme.Subtle)
		dir, _ := os.UserHomeDir()
		m.fp.CurrentDirectory = dir

		// Send WindowSizeMsg
		m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: h})
		cmd := m.fp.Init()
		if cmd != nil {
			msg := cmd()
			m.fp, _ = m.fp.Update(msg)
		}

		repoPath := cfg.RepoPath
		header := components.Header(m.width, m.height, m.animStep, repoPath)

		blockWidth := 66
		padLeft := (m.width - blockWidth) / 2
		if padLeft < 2 {
			padLeft = 2
		}
		indent := strings.Repeat(" ", padLeft)

		title := indent + lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Secondary).
			Render("Browse filesystem to open configurations:")

		currentDirBadge := indent + lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Bold(true).
			Render(theme.IconDirModern + "  " + shortenHome(m.fp.CurrentDirectory))

		rawFpLines := strings.Split(strings.TrimRight(m.fp.View(), "\n"), "\n")
		var indentedFp []string
		for _, l := range rawFpLines {
			indentedFp = append(indentedFp, indent+l)
		}

		content := lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			currentDirBadge,
			"",
			strings.Join(indentedFp, "\n"),
		)

		statusBar := components.StatusBar("Browse", "enter open • o nvim • h parent • tab edit • esc home • q quit", m.width)
		topBlock := lipgloss.JoinVertical(lipgloss.Top, header, content)
		fullView := components.PlacePinnedStatusBar(topBlock, statusBar, m.height)

		viewHeight := lipgloss.Height(fullView)
		t.Logf("Height target: %d, actual: %d, headerHeight: %d, contentHeight: %d",
			h, viewHeight, lipgloss.Height(header), lipgloss.Height(content))

		if viewHeight != h {
			t.Errorf("For terminal height %d: expected view height %d, got %d", h, h, viewHeight)
		}
	}
}
