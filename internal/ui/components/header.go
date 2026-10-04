package components

import (
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/lipgloss"
)

// Header renders the top header bar with the app name.
func Header(width int) string {
	logo := theme.Logo()
	tagline := theme.MutedStyle.Render("dotfile manager")

	content := lipgloss.JoinVertical(lipgloss.Center, logo, tagline)

	headerStyle := lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Padding(1, 0)

	return headerStyle.Render(content)
}
