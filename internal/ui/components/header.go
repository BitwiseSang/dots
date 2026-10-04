package components

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

var (
	cachedBranch string
)

func getGitBranch() string {
	if cachedBranch != "" {
		return cachedBranch
	}
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err == nil {
		cachedBranch = strings.TrimSpace(string(out))
		return cachedBranch
	}
	cachedBranch = "main"
	return cachedBranch
}

// Header renders the Crush-inspired top header bar with the massive gradient logo,
// diagonal hatching, version badge, and git breadcrumbs.
func Header(width, step int) string {
	if width <= 0 {
		width = 80
	}

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

	// Breadcrumb: main • ~/Documents/dotfiles
	branch := getGitBranch()
	cwd, _ := os.Getwd()
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(cwd, home) {
		cwd = "~" + cwd[len(home):]
	}

	breadcrumb := fmt.Sprintf("%s %s %s %s %s",
		lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Render(theme.IconGit),
		lipgloss.NewStyle().Foreground(theme.Text).Render(branch),
		lipgloss.NewStyle().Foreground(theme.Muted).Render("•"),
		lipgloss.NewStyle().Foreground(theme.Subtle).Render(cwd),
		theme.SubtleDotsCluster(),
	)

	separator := theme.DotsSeparator(width)

	content := lipgloss.JoinVertical(
		lipgloss.Center,
		topBar,
		"",
		gradientLogo,
		"",
		breadcrumb,
		"",
		separator,
	)

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		PaddingTop(1).
		Render(content)
}
