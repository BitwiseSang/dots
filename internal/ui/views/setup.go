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
}

func NewSetupModel(entries []dotfile.Entry, cfg *config.Config) SetupModel {
	items := make([]components.SelectorItem, len(entries))
	for i, e := range entries {
		items[i] = components.SelectorItem{
			Name: e.Name,
			Desc: e.StatusLabel(),
		}
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(theme.Accent)

	return SetupModel{
		entries:  entries,
		cfg:      cfg,
		phase:    setupPhaseSelect,
		selector: components.NewSelector(items),
		spinner:  sp,
		vp:       viewport.New(0, 0),
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.selector.SetSize(msg.Width, msg.Height-4)
		m.vp.Width = msg.Width - 4
		m.vp.Height = msg.Height - 6

	case setupDoneMsg:
		m.results = msg.Results
		m.backupDir = msg.BackupDir
		m.phase = setupPhaseDone
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "esc" {
			if m.phase == setupPhaseSelect {
				return m, func() tea.Msg { return NavigateMsg{View: ViewHome} }
			} else if m.phase == setupPhasePreview {
				m.phase = setupPhaseSelect
				return m, nil
			}
		}

		switch m.phase {
		case setupPhaseSelect:
			if msg.String() == "enter" {
				selected := m.selector.SelectedIndices()
				if len(selected) > 0 {
					m.phase = setupPhasePreview
					m.vp.SetContent(m.generatePreview(selected))
				}
				return m, nil
			}
			m.selector, cmd = m.selector.Update(msg)
			cmds = append(cmds, cmd)

		case setupPhasePreview:
			if msg.String() == "enter" {
				m.phase = setupPhaseExecute

				selected := m.selector.SelectedIndices()
				var toSetup []dotfile.Entry
				for _, idx := range selected {
					toSetup = append(toSetup, m.entries[idx])
				}

				return m, func() tea.Msg {
					bDir, _ := dotfile.CreateBackupDir()
					res := dotfile.SetupAll(toSetup, bDir)
					return setupDoneMsg{Results: res, BackupDir: bDir}
				}
			}
			m.vp, cmd = m.vp.Update(msg)
			cmds = append(cmds, cmd)

		case setupPhaseDone:
			if msg.String() == "enter" || msg.String() == "esc" {
				return m, func() tea.Msg { return NavigateMsg{View: ViewHome} }
			}
		}
	}

	if m.phase == setupPhaseExecute {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m SetupModel) generatePreview(indices []int) string {
	var b strings.Builder
	b.WriteString("The following items will be symlinked:\n\n")

	for _, idx := range indices {
		entry := m.entries[idx]

		src := lipgloss.NewStyle().Foreground(theme.Accent).Render(entry.RepoPath)
		dst := lipgloss.NewStyle().Foreground(theme.Primary).Render(entry.SystemPath)

		b.WriteString(fmt.Sprintf("• %s\n  %s → %s\n\n", entry.Name, src, dst))
	}
	return b.String()
}

func (m SetupModel) View() string {
	header := components.Header(m.width)
	var content string
	var statusHint string

	switch m.phase {
	case setupPhaseSelect:
		content = lipgloss.NewStyle().Padding(2, 4).Render(m.selector.View())
		statusHint = "space toggle • a toggle all • enter proceed • esc back"

	case setupPhasePreview:
		content = lipgloss.NewStyle().Padding(2, 4).Render(m.vp.View())
		statusHint = "up/down scroll • enter confirm setup • esc back"

	case setupPhaseExecute:
		content = lipgloss.NewStyle().Padding(4, 4).Render(
			fmt.Sprintf("%s Setting up %d items...", m.spinner.View(), len(m.selector.SelectedIndices())),
		)
		statusHint = "Executing..."

	case setupPhaseDone:
		var b strings.Builder
		b.WriteString("Setup complete:\n\n")
		for _, r := range m.results {
			icon := theme.SuccessStyle.Render("✓")
			status := "Success"
			if r.Err != nil {
				icon = theme.ErrorStyle.Render("✗")
				status = r.Err.Error()
			} else if r.BackedUp {
				status = fmt.Sprintf("Success (backed up to %s)", r.BackupPath)
			}

			b.WriteString(fmt.Sprintf("%s %s: %s\n", icon, r.Entry.Name, status))
		}

		if m.backupDir != "" {
			b.WriteString(fmt.Sprintf("\nBackups stored in: %s\n", m.backupDir))
		}

		content = lipgloss.NewStyle().Padding(2, 4).Render(b.String())
		statusHint = "enter/esc return to home"
	}

	contentHeight := lipgloss.Height(content) + lipgloss.Height(header)
	padHeight := m.height - contentHeight - 1
	if padHeight < 0 {
		padHeight = 0
	}
	padded := lipgloss.JoinVertical(lipgloss.Top, header, content, strings.Repeat("\n", padHeight))

	statusBar := components.StatusBar("Setup", statusHint, m.width)
	return lipgloss.JoinVertical(lipgloss.Top, padded, statusBar)
}
