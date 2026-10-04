package views

import (
	"fmt"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/BitwiseSang/dots/internal/ui/components"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type setupPhase int

const (
	setupPhaseSelect setupPhase = iota
	setupPhasePreview
	setupPhaseExecute
	setupPhaseDone
)

type SetupModel struct {
	entries   []dotfile.Entry
	cfg       *config.Config
	phase     setupPhase
	selector  components.Selector
	vp        viewport.Model
	spinner   spinner.Model
	results   []dotfile.SetupResult
	backupDir string
	width     int
	height    int
	animStep  int
}

func NewSetupModel(entries []dotfile.Entry, cfg *config.Config) SetupModel {
	items := make([]components.SelectorItem, len(entries))
	for i, e := range entries {
		items[i] = components.SelectorItem{
			Name:  e.Name,
			Desc:  e.StatusLabel(),
			Path:  e.ResolveSystemPath(),
			IsDir: e.IsDir,
		}
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#A855F7")).Bold(true)

	sel := components.NewSelector(items)
	sel.ActiveColor = lipgloss.Color("#A855F7") // Electric Violet for Setup
	sel.CheckColor = theme.Accent               // Amber

	return SetupModel{
		entries:  entries,
		cfg:      cfg,
		phase:    setupPhaseSelect,
		selector: sel,
		spinner:  sp,
		vp:       viewport.New(0, 0),
		animStep: 0,
	}
}

func (m SetupModel) Init() tea.Cmd {
	return m.spinner.Tick
}

type setupDoneMsg struct {
	Results   []dotfile.SetupResult
	BackupDir string
}

func (m SetupModel) Update(msg tea.Msg) (SetupModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case TickMsg:
		m.animStep++
		var spinCmd tea.Cmd
		m.spinner, spinCmd = m.spinner.Update(msg)
		return m, spinCmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.selector.SetSize(msg.Width, msg.Height-14)
		m.vp.Width = msg.Width - 4
		m.vp.Height = msg.Height - 16

	case setupDoneMsg:
		m.results = msg.Results
		m.backupDir = msg.BackupDir
		m.phase = setupPhaseDone
		return m, nil

	case spinner.TickMsg:
		if m.phase == setupPhaseExecute {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		switch m.phase {
		case setupPhaseSelect:
			switch msg.String() {
			case "esc":
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHome}
				}
			case "enter":
				selected := m.selector.SelectedIndices()
				if len(selected) > 0 {
					m.phase = setupPhasePreview
					m.vp.SetContent(m.generatePreview(selected))
					return m, nil
				}
			}
			m.selector, cmd = m.selector.Update(msg)
			cmds = append(cmds, cmd)

		case setupPhasePreview:
			switch msg.String() {
			case "esc":
				m.phase = setupPhaseSelect
				return m, nil
			case "enter":
				m.phase = setupPhaseExecute
				var toSetup []dotfile.Entry
				for _, idx := range m.selector.SelectedIndices() {
					toSetup = append(toSetup, m.entries[idx])
				}

				return m, tea.Batch(
					m.spinner.Tick,
					func() tea.Msg {
						bDir, _ := dotfile.CreateBackupDir()
						res := dotfile.SetupAll(toSetup, bDir)
						return setupDoneMsg{Results: res, BackupDir: bDir}
					},
				)
			}
			m.vp, cmd = m.vp.Update(msg)
			cmds = append(cmds, cmd)

		case setupPhaseDone:
			if msg.String() == "enter" || msg.String() == "esc" {
				return m, func() tea.Msg { return NavigateMsg{View: ViewHome} }
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m SetupModel) generatePreview(indices []int) string {
	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#A855F7")).Render("The following symlinks will be created:")
	b.WriteString(title + "\n\n")

	for _, idx := range indices {
		entry := m.entries[idx]

		icon := theme.FileIcon(entry.Name, entry.IsDir)
		src := lipgloss.NewStyle().Foreground(theme.Accent).Render(entry.AbsRepoPath())
		dst := lipgloss.NewStyle().Foreground(theme.Primary).Render(entry.ResolveSystemPath())
		arrow := lipgloss.NewStyle().Foreground(theme.Secondary).Render(" ➜ ")

		b.WriteString(fmt.Sprintf("  %s %s\n      %s%s%s\n\n",
			icon,
			lipgloss.NewStyle().Bold(true).Render(entry.Name),
			src,
			arrow,
			dst,
		))
	}
	return b.String()
}

func (m SetupModel) View() string {
	header := components.Header(m.width, m.animStep)
	var content string
	var statusHint string

	switch m.phase {
	case setupPhaseSelect:
		title := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A855F7")).
			Render("Select configurations to symlink into your system:")
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			"  "+title,
			"",
			m.selector.View(),
		)
		statusHint = "space toggle • a toggle all • enter preview symlinks • esc back"

	case setupPhasePreview:
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			m.vp.View(),
		)
		statusHint = "up/down scroll • enter confirm symlink setup • esc back"

	case setupPhaseExecute:
		content = lipgloss.NewStyle().Padding(3, 4).Render(
			fmt.Sprintf("%s Linking %d configurations to system...", m.spinner.View(), len(m.selector.SelectedIndices())),
		)
		statusHint = "Executing..."

	case setupPhaseDone:
		var b strings.Builder
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(theme.Success).Render("Setup Complete:\n\n"))
		for _, r := range m.results {
			icon := lipgloss.NewStyle().Foreground(theme.Success).Render(theme.IconLinked)
			status := "Linked"
			if r.Err != nil {
				icon = lipgloss.NewStyle().Foreground(theme.Error).Render(theme.IconChanged)
				status = r.Err.Error()
			} else if r.BackedUp {
				status = fmt.Sprintf("Linked (old backup: %s)", r.BackupPath)
			}

			b.WriteString(fmt.Sprintf("  %s %s: %s\n", icon, lipgloss.NewStyle().Bold(true).Render(r.Entry.Name), status))
		}

		if m.backupDir != "" {
			b.WriteString(fmt.Sprintf("\n  %s Old configurations safely archived in:\n     %s\n",
				theme.IconDotsCluster,
				lipgloss.NewStyle().Foreground(theme.Muted).Render(m.backupDir),
			))
		}

		content = lipgloss.NewStyle().Padding(2, 4).Render(b.String())
		statusHint = "enter/esc return home"
	}

	contentHeight := lipgloss.Height(content) + lipgloss.Height(header)
	padHeight := m.height - contentHeight - 3
	if padHeight < 0 {
		padHeight = 0
	}
	padded := lipgloss.JoinVertical(lipgloss.Top, header, content, strings.Repeat("\n", padHeight))

	statusBar := components.StatusBar("Setup", statusHint, m.width)
	return lipgloss.JoinVertical(lipgloss.Top, padded, statusBar)
}
