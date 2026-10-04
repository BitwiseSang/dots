package components

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

var (
	cachedBranches = make(map[string]string)
)

func getGitBranchIn(dir string) string {
	if b, ok := cachedBranches[dir]; ok {
		return b
	}
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err == nil {
		b := strings.TrimSpace(string(out))
		if b != "" {
			cachedBranches[dir] = b
			return b
		}
	}
	cachedBranches[dir] = "main"
	return "main"
}

// Header renders the Crush-inspired top header bar with the massive gradient logo,
// diagonal hatching, version badge, dotfiles git repository breadcrumbs, and 3 animated dots.
func Header(width, step int, repoPath string) string {
	if width <= 0 {
		width = 80
	}

	if repoPath == "" {
		repoPath = "~/Documents/dotfiles"
	}
	absRepoPath := config.ExpandPath(repoPath)

	hatchStyle := lipgloss.NewStyle().Foreground(theme.Hatch)
	brandStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary)
	versionStyle := lipgloss.NewStyle().Foreground(theme.Subtle)

	// Top bar: //// dots™                      v0.1.0 ///////////////////
	badgeLeft := " " + brandStyle.Render("dots™") + " "
	badgeRight := " " + versionStyle.Render("v0.1.0") + " "

	topAvailable := width - lipgloss.Width(badgeLeft) - lipgloss.Width(badgeRight) - 8
	if topAvailable < 4 {
		topAvailable = 4
	}
	leftHatchLen := 4
	rightHatchLen := topAvailable
	if rightHatchLen < 4 {
		rightHatchLen = 4
	}

	topBar := hatchStyle.Render(strings.Repeat("/", leftHatchLen)) +
		badgeLeft +
		badgeRight +
		hatchStyle.Render(strings.Repeat("/", rightHatchLen))

	// Animated gradient logo
	gradientLogo := theme.RenderGradientLogo(step)

	// Breadcrumb: branch • dotfiles repo path
	branch := getGitBranchIn(absRepoPath)

	displayPath := absRepoPath
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(displayPath, home) {
		displayPath = "~" + displayPath[len(home):]
	}
	displayPath = filepath.Clean(displayPath)

	breadcrumb := fmt.Sprintf("%s %s %s %s %s",
		lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Render(theme.IconGit),
		lipgloss.NewStyle().Foreground(theme.Text).Render(branch),
		lipgloss.NewStyle().Foreground(theme.Muted).Render("•"),
		lipgloss.NewStyle().Foreground(theme.Subtle).Render(displayPath),
		theme.SubtleDotsCluster(),
	)

	// 3 centered animated gradient dots matching the title
	threeDots := theme.AnimatedThreeDots(step)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		topBar,
		"",
		gradientLogo,
		"",
		breadcrumb,
		"",
		threeDots,
	)

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		PaddingTop(1).
		Render(content)
}
