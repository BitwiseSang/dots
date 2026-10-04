package theme

import "github.com/charmbracelet/lipgloss"

const (
	Primary   = lipgloss.Color("#7C3AED") // violet
	Secondary = lipgloss.Color("#06B6D4") // cyan
	Accent    = lipgloss.Color("#F59E0B") // amber
	Success   = lipgloss.Color("#10B981") // emerald
	Error     = lipgloss.Color("#EF4444") // red
	Muted     = lipgloss.Color("#6B7280") // gray
	Surface   = lipgloss.Color("#1F2937") // dark gray
	Text      = lipgloss.Color("#F9FAFB") // off-white
	Subtle    = lipgloss.Color("#9CA3AF") // mid-gray
)

var (
	TitleStyle        = lipgloss.NewStyle().Bold(true).Foreground(Primary).Padding(0, 1)
	SubtitleStyle     = lipgloss.NewStyle().Foreground(Subtle)
	SelectedStyle     = lipgloss.NewStyle().Foreground(Primary).Bold(true)
	UnselectedStyle   = lipgloss.NewStyle().Foreground(Text)
	SuccessStyle      = lipgloss.NewStyle().Foreground(Success).Bold(true)
	ErrorStyle        = lipgloss.NewStyle().Foreground(Error).Bold(true)
	WarningStyle      = lipgloss.NewStyle().Foreground(Accent)
	MutedStyle        = lipgloss.NewStyle().Foreground(Muted)
	PanelStyle        = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Surface).Padding(1, 2)
	ActivePanelStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(Primary).Padding(1, 2)
	StatusBarStyle    = lipgloss.NewStyle().Background(Surface).Foreground(Subtle).Padding(0, 1)
	HeaderStyle       = lipgloss.NewStyle().Bold(true).Foreground(Primary).Align(lipgloss.Center)
	DiffAddStyle      = lipgloss.NewStyle().Foreground(Success)
	DiffDelStyle      = lipgloss.NewStyle().Foreground(Error)
	DiffHdrStyle      = lipgloss.NewStyle().Foreground(Secondary).Bold(true)
	CheckboxChecked   = SuccessStyle.Render("[✓]")
	CheckboxUnchecked = MutedStyle.Render("[ ]")
)

func Logo() string {
	logo := `  ╺━━━━━━━╸
    d o t s
  ╺━━━━━━━╸`
	return lipgloss.NewStyle().Foreground(Primary).Bold(true).Align(lipgloss.Center).Render(logo)
}
