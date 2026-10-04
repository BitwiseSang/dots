package theme

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	Primary   = lipgloss.Color("#8B5CF6") // violet
	Secondary = lipgloss.Color("#06B6D4") // cyan
	Accent    = lipgloss.Color("#F59E0B") // amber
	Success   = lipgloss.Color("#10B981") // emerald
	Error     = lipgloss.Color("#EF4444") // red
	Muted     = lipgloss.Color("#4B5563") // muted gray
	Surface   = lipgloss.Color("#1F2937") // dark surface
	Text      = lipgloss.Color("#F9FAFB") // off-white
	Subtle    = lipgloss.Color("#9CA3AF") // mid-gray
	Pink      = lipgloss.Color("#EC4899") // magenta/pink
	Hatch     = lipgloss.Color("#374151") // diagonal hatch color
)

var (
	TitleStyle       = lipgloss.NewStyle().Bold(true).Foreground(Primary)
	SubtitleStyle    = lipgloss.NewStyle().Foreground(Subtle)
	SelectedStyle    = lipgloss.NewStyle().Foreground(Primary).Bold(true)
	UnselectedStyle  = lipgloss.NewStyle().Foreground(Text)
	SuccessStyle     = lipgloss.NewStyle().Foreground(Success).Bold(true)
	ErrorStyle       = lipgloss.NewStyle().Foreground(Error).Bold(true)
	WarningStyle     = lipgloss.NewStyle().Foreground(Accent)
	MutedStyle       = lipgloss.NewStyle().Foreground(Muted)
	StatusBarStyle   = lipgloss.NewStyle().Background(Surface).Foreground(Subtle).Padding(0, 1)
	HeaderStyle      = lipgloss.NewStyle().Bold(true).Foreground(Primary).Align(lipgloss.Center)
	DiffAddStyle     = lipgloss.NewStyle().Foreground(Success)
	DiffDelStyle     = lipgloss.NewStyle().Foreground(Error)
	DiffHdrStyle     = lipgloss.NewStyle().Foreground(Secondary).Bold(true)

	// Context-specific active selection styles
	BackupActiveStyle = lipgloss.NewStyle().Foreground(Secondary).Bold(true)
	SetupActiveStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#A855F7")).Bold(true)
	EditActiveStyle   = lipgloss.NewStyle().Foreground(Pink).Bold(true)

	// Checkbox icons using Nerd Fonts
	CheckboxChecked   = lipgloss.NewStyle().Foreground(Success).Bold(true).Render("[]")
	CheckboxUnchecked = lipgloss.NewStyle().Foreground(Muted).Render("[ ]")
)

// Massive block ASCII logo for DOTS
var MassiveLogoLines = []string{
	`██████╗   ██████╗  ████████╗ ███████╗`,
	`██╔══██╗ ██╔═══██╗ ╚══██╔══╝ ██╔════╝`,
	`██║  ██║ ██║   ██║    ██║    ███████╗`,
	`██║  ██║ ██║   ██║    ██║    ╚════██║`,
	`██████╔╝ ╚██████╔╝    ██║    ███████║`,
	`╚═════╝   ╚═════╝     ╚═╝    ╚══════╝`,
}

// Color stops for the flowing gradient: Pink -> Purple -> Cyan -> Blue -> Pink
var gradientStops = [][3]float64{
	{236, 72, 153}, // #EC4899 Pink
	{168, 85, 247}, // #A855F7 Purple
	{6, 182, 212},  // #06B6D4 Cyan
	{59, 130, 246}, // #3B82F6 Blue
	{16, 185, 129}, // #10B981 Emerald
	{236, 72, 153}, // Loop back to Pink
}

// InterpolateRGB returns an interpolated color string along the gradient cycle given phase t in [0, 1).
func InterpolateRGB(t float64) string {
	t = math.Mod(t, 1.0)
	if t < 0 {
		t += 1.0
	}

	n := float64(len(gradientStops) - 1)
	segment := t * n
	idx := int(math.Floor(segment))
	frac := segment - float64(idx)

	if idx >= len(gradientStops)-1 {
		idx = len(gradientStops) - 2
		frac = 1.0
	}

	c1 := gradientStops[idx]
	c2 := gradientStops[idx+1]

	r := uint8(math.Round(c1[0] + frac*(c2[0]-c1[0])))
	g := uint8(math.Round(c1[1] + frac*(c2[1]-c1[1])))
	b := uint8(math.Round(c1[2] + frac*(c2[2]-c1[2])))

	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// RenderGradientLogo renders the massive DOTS logo with a horizontal flowing gradient shifted by step.
func RenderGradientLogo(step int) string {
	var sb strings.Builder
	for lineIdx, line := range MassiveLogoLines {
		runes := []rune(line)
		totalRunes := len(runes)
		for colIdx, r := range runes {
			if r == ' ' {
				sb.WriteRune(' ')
				continue
			}
			// Calculate phase along the width and animated by step
			phase := float64(colIdx)/float64(totalRunes) + float64(step)*0.02
			colorHex := InterpolateRGB(phase)
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(colorHex)).Bold(true).Render(string(r)))
		}
		if lineIdx < len(MassiveLogoLines)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// Logo returns a static fallback version of the massive logo
func Logo() string {
	return RenderGradientLogo(0)
}

// DotsSeparator renders a line of spaced dots across the specified width.
func DotsSeparator(width int) string {
	if width <= 0 {
		width = 40
	}
	count := width / 3
	if count > 40 {
		count = 40
	}
	if count < 5 {
		count = 5
	}
	dots := strings.Repeat(" • ", count)
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#4B5563")).
		Width(width).
		Align(lipgloss.Center).
		Render(strings.TrimSpace(dots))
}

// SubtleDotsCluster renders the ':::' dot cluster inspired by Crush CLI.
func SubtleDotsCluster() string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(":::")
}
