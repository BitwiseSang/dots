package components

import (
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

// StatusBar renders the bottom status bar.
func StatusBar(context, hint string, width int) string {
	ctxStyle := lipgloss.NewStyle().Bold(true).Padding(0, 1)
	hintStyle := lipgloss.NewStyle().Padding(0, 1)

	ctxStr := ctxStyle.Render(context)
	hintStr := hintStyle.Render(hint)

	ctxWidth := lipgloss.Width(ctxStr)
	hintWidth := lipgloss.Width(hintStr)
	spacerWidth := width - ctxWidth - hintWidth
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	bar := lipgloss.JoinHorizontal(lipgloss.Top, ctxStr, spacer, hintStr)

	return theme.StatusBarStyle.Width(width).Render(bar)
}
