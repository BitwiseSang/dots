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

	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(theme.Text)
	descStyle := lipgloss.NewStyle().Foreground(theme.Subtle)
	bulletStyle := lipgloss.NewStyle().Foreground(theme.Muted)

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

	return strings.Join(formatted, bulletStyle.Render("  •  "))
}

// StatusBar renders the bottom status bar with the Crush-style shortcuts strip.
func StatusBar(context, hint string, width int) string {
	if width <= 0 {
		width = 80
	}

	ctxBadge := ""
	if context != "" {
		ctxBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Secondary).
			Render(context) + " " + lipgloss.NewStyle().Foreground(theme.Muted).Render("•") + " "
	}

	shortcuts := RenderShortcuts(hint)
	barContent := ctxBadge + shortcuts

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Padding(0, 1).
		Render(barContent)
}
