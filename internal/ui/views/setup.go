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

func (m *SetupModel) Reset(entries []dotfile.Entry) {
	m.entries = entries
	m.phase = setupPhaseSelect
	m.results = nil
	m.backupDir = ""

	items := make([]components.SelectorItem, len(entries))
	for i, e := range entries {
		items[i] = components.SelectorItem{
			Name:  e.Name,
			Desc:  e.StatusLabel(),
			Path:  e.ResolveSystemPath(),
			IsDir: e.IsDir,
		}
	}
	m.selector = components.NewSelector(items)
	m.selector.ActiveColor = lipgloss.Color("#A855F7") // Electric Violet for Setup
	m.selector.CheckColor = theme.Accent               // Amber
	if m.width > 0 && m.height > 0 {
		m.selector.SetSize(m.width, m.height-14)
	}
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
				m.Reset(m.entries)
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

		nameStr := lipgloss.NewStyle().Width(16).Bold(true).Render(entry.Name)
		b.WriteString(fmt.Sprintf("  %s %s\n      %s%s%s\n\n",
			icon,
			nameStr,
			src,
			arrow,
			dst,
		))
	}
	return b.String()
}

func (m SetupModel) View() string {
	repoPath := ""
	if m.cfg != nil {
		repoPath = m.cfg.RepoPath
	}
	header := components.Header(m.width, m.animStep, repoPath)
	var content string
	var statusHint string

	blockWidth := 66
	padLeft := (m.width - blockWidth) / 2
	if padLeft < 2 {
		padLeft = 2
	}
	indent := strings.Repeat(" ", padLeft)

	switch m.phase {
	case setupPhaseSelect:
		title := indent + lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#A855F7")).
			Render("Select configurations to symlink into your system:")

		selLines := strings.Split(m.selector.View(), "\n")
		var indentedSel []string
		for _, l := range selLines {
			indentedSel = append(indentedSel, indent+l)
		}

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			strings.Join(indentedSel, "\n"),
		)
		statusHint = "space toggle • a toggle all • enter preview symlinks • esc back • q quit"

	case setupPhasePreview:
		previewLines := strings.Split(m.vp.View(), "\n")
		var indentedPrev []string
		for _, l := range previewLines {
			indentedPrev = append(indentedPrev, indent+l)
		}

		content = strings.Join(indentedPrev, "\n")
		statusHint = "up/down scroll • enter confirm symlink setup • esc back • q quit"

	case setupPhaseExecute:
		content = indent + lipgloss.NewStyle().Padding(3, 0).Render(
			fmt.Sprintf("%s Linking %d configurations to system...", m.spinner.View(), len(m.selector.SelectedIndices())),
		)
		statusHint = "Executing..."

	case setupPhaseDone:
		var b strings.Builder
		titleStr := indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Success).Render("Setup Complete:")
		b.WriteString(titleStr + "\n\n")

		for _, r := range m.results {
			iconStr := lipgloss.NewStyle().Width(3).Foreground(theme.Success).Render(theme.IconLinked)
			statusStr := lipgloss.NewStyle().Foreground(theme.Success).Render("Linked")
			if r.Err != nil {
				iconStr = lipgloss.NewStyle().Width(3).Foreground(theme.Error).Render(theme.IconChanged)
				statusStr = lipgloss.NewStyle().Foreground(theme.Error).Render(r.Err.Error())
			} else if r.BackedUp {
				statusStr = lipgloss.NewStyle().Foreground(theme.Success).Render(fmt.Sprintf("Linked (old backup: %s)", r.BackupPath))
			}

			nameStr := lipgloss.NewStyle().Width(18).Bold(true).Render(r.Entry.Name)
			row := indent + lipgloss.JoinHorizontal(
				lipgloss.Left,
				iconStr,
				nameStr,
				statusStr,
			)
			b.WriteString(row + "\n")
		}

		if m.backupDir != "" {
			b.WriteString(fmt.Sprintf("\n%s%s Old configurations safely archived in:\n%s   %s\n",
				indent,
				theme.IconDotsCluster,
				indent,
				lipgloss.NewStyle().Foreground(theme.Muted).Render(m.backupDir),
			))
		}

		content = b.String()
		statusHint = "enter/esc return home • q quit"
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
