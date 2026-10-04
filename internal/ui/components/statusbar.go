package components

import (
	"strings"

	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

// ShortcutItem represents an interactive keybinding and its description
type ShortcutItem struct {
	Key  string
	Desc string
}

// RenderShortcuts parses a list of shortcuts or string hints and renders them Crush-style:
// "key desc • key desc"
func RenderShortcuts(hints string) string {
	parts := strings.Split(hints, "•")
	var formatted []string

	keyStyle := lipgloss.NewStyle().Foreground(theme.Subtle)
	descStyle := lipgloss.NewStyle().Faint(true).Foreground(theme.Muted)
	bulletStyle := lipgloss.NewStyle().Faint(true).Foreground(theme.Muted)

	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}

		// Split on first space between key and action
		words := strings.SplitN(trimmed, " ", 2)
		if len(words) == 2 {
			k := keyStyle.Render(words[0])
			d := descStyle.Render(words[1])
			formatted = append(formatted, k+" "+d)
		} else {
			formatted = append(formatted, descStyle.Render(trimmed))
		}
	}

	return strings.Join(formatted, bulletStyle.Render(" • "))
}

// StatusBar renders the bottom status bar with the Crush-style shortcuts strip.
// It is strictly limited to MaxHeight(1) and styled with Faint/subtle intensity for a smaller visual footprint.
func StatusBar(context, hint string, width int) string {
	if width <= 0 {
		width = 80
	}

	ctxBadge := ""
	if context != "" {
		ctxColor := theme.Secondary
		switch context {
		case "Home":
			ctxColor = theme.Primary
		case "Backup":
			ctxColor = theme.Secondary
		case "Setup":
			ctxColor = lipgloss.Color("#A855F7")
		case "Edit":
			ctxColor = theme.Pink
		case "Browse":
			ctxColor = theme.Secondary
		}

		ctxBadge = lipgloss.NewStyle().
			Foreground(ctxColor).
			Render(context) + " " + lipgloss.NewStyle().Foreground(theme.Muted).Render("•") + " "
	}

	shortcuts := RenderShortcuts(hint)
	barContent := ctxBadge + shortcuts

	return lipgloss.NewStyle().
		Width(width).
		MaxHeight(1).
		Faint(true).
		Align(lipgloss.Center).
		Padding(0, 1).
		Render(barContent)
}

// PlacePinnedStatusBar joins the top content block and status bar, inserting blank lines if targetHeight > used,
// ensuring the output NEVER exceeds targetHeight (preventing terminal scrolling and header clipping).
func PlacePinnedStatusBar(topBlock, statusBar string, targetHeight int) string {
	if targetHeight <= 0 {
		return lipgloss.JoinVertical(lipgloss.Top, topBlock, statusBar)
	}
	topHeight := lipgloss.Height(topBlock)
	statusHeight := lipgloss.Height(statusBar)
	totalUsed := topHeight + statusHeight

	if totalUsed >= targetHeight {
		return lipgloss.JoinVertical(lipgloss.Top, topBlock, statusBar)
	}

	gap := targetHeight - totalUsed
	gapLines := strings.Repeat("\n", gap-1)
	return lipgloss.JoinVertical(lipgloss.Top, topBlock, gapLines, statusBar)
}
