package components

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/BitwiseSang/dots/internal/version"
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

// HeaderHeight returns the exact line height of the header rendered for a given terminal width and height.
func HeaderHeight(width, height int) int {
	if height > 0 && height < 20 {
		return 3
	}
	if (height > 0 && height < 28) || (width > 0 && width < 48) {
		return 5
	}
	return 13
}

// Header renders the Crush-inspired top header bar with the gradient logo,
// diagonal hatching, version badge, dotfiles git repository breadcrumbs, and 3 animated dots.
// It is strictly responsive:
// - height >= 28 & width >= 48: Full header with 6-line gradient ASCII logo, top bar, breadcrumb, and 3 dots (13 lines).
// - 20 <= height < 28 or width < 48: Compact header without ASCII logo, keeping top bar, breadcrumb, and 3 dots (5 lines).
// - height < 20: Ultra-compact header keeping top bar, breadcrumb, and 3 dots without empty spacer lines (3 lines).
// The top bar, the directory line, and the 3 separator dots are GUARANTEED to be present at all screen sizes.
func Header(width, height, step int, repoPath string) string {
	if width <= 0 {
		width = 80
	}

	hatchStyle := lipgloss.NewStyle().Foreground(theme.Hatch)
	brandStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary)
	versionStyle := lipgloss.NewStyle().Foreground(theme.Subtle)

	// Top bar: //// dots™                      v0.1.0 ///////////////////
	badgeLeft := " " + brandStyle.Render("dots™") + " "
	badgeRight := " " + versionStyle.Render("v"+version.Version) + " "

	topAvailable := width - lipgloss.Width(badgeLeft) - lipgloss.Width(badgeRight) - 8
	leftHatchLen := 4
	rightHatchLen := topAvailable
	if rightHatchLen < 2 {
		rightHatchLen = 2
	}
	if width < 36 {
		leftHatchLen = 1
		rightHatchLen = 1
	}

	topBar := hatchStyle.Render(strings.Repeat("/", leftHatchLen)) +
		badgeLeft +
		badgeRight +
		hatchStyle.Render(strings.Repeat("/", rightHatchLen))

	// Breadcrumb: branch • dotfiles repo path
	displayPath := "uninitialized"
	branch := "-"
	if repoPath != "" {
		absRepoPath := config.ExpandPath(repoPath)
		branch = getGitBranchIn(absRepoPath)
		displayPath = absRepoPath
		if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(displayPath, home) {
			displayPath = "~" + displayPath[len(home):]
		}
		displayPath = filepath.Clean(displayPath)
	}

	// Truncate displayPath if narrow width to guarantee breadcrumb NEVER wraps
	reservedBreadcrumb := len(branch) + 12
	if width >= 50 {
		reservedBreadcrumb += 8 // for subtle dots cluster
	}
	availPathWidth := width - reservedBreadcrumb
	if availPathWidth < 6 {
		availPathWidth = 6
	}
	if len(displayPath) > availPathWidth {
		base := filepath.Base(displayPath)
		prefix := ".../"
		if strings.HasPrefix(displayPath, "~") {
			prefix = "~/.../"
		}
		if len(prefix)+len(base) <= availPathWidth {
			displayPath = prefix + base
		} else if availPathWidth > 4 && len(base) > availPathWidth-3 {
			displayPath = "..." + base[len(base)-(availPathWidth-3):]
		}
	}

	var dotsCluster string
	if width >= 50 {
		dotsCluster = " " + theme.SubtleDotsCluster()
	}

	breadcrumb := fmt.Sprintf("%s %s %s %s%s",
		lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Render(theme.IconGit),
		lipgloss.NewStyle().Foreground(theme.Text).Render(branch),
		lipgloss.NewStyle().Foreground(theme.Muted).Render("•"),
		lipgloss.NewStyle().Foreground(theme.Subtle).Render(displayPath),
		dotsCluster,
	)

	// 3 centered animated gradient dots matching the title
	threeDots := theme.AnimatedThreeDots(step)

	expectedHeight := HeaderHeight(width, height)
	var content string

	switch expectedHeight {
	case 3:
		// Ultra-compact (3 lines): top bar, directory line, three separator dots
		content = lipgloss.JoinVertical(
			lipgloss.Center,
			topBar,
			breadcrumb,
			threeDots,
		)
	case 5:
		// Compact (5 lines): top bar, blank, directory line, blank, three separator dots
		content = lipgloss.JoinVertical(
			lipgloss.Center,
			topBar,
			"",
			breadcrumb,
			"",
			threeDots,
		)
	default:
		// Full (13 lines): top bar, blank, 6-line gradient logo, blank, directory line, blank, three separator dots
		gradientLogo := theme.RenderGradientLogo(step)
		content = lipgloss.JoinVertical(
			lipgloss.Center,
			topBar,
			"",
			gradientLogo,
			"",
			breadcrumb,
			"",
			threeDots,
		)
	}

	style := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center)

	if expectedHeight == 13 {
		style = style.PaddingTop(1)
	}

	return style.Render(content)
}
