package views

import (
	"fmt"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/BitwiseSang/dots/internal/git"
	"github.com/BitwiseSang/dots/internal/ui/components"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type backupPhase int

const (
	phaseSelect backupPhase = iota
	phaseDiff
	phaseExecute
	phaseGit
	phaseDone
)

type BackupModel struct {
	entries   []dotfile.Entry
	cfg       *config.Config
	phase     backupPhase
	selector  components.Selector
	vp        viewport.Model
	spinner   spinner.Model
	results   []dotfile.BackupResult
	width     int
	height    int
	gitPrompt string
	animStep  int
}

func NewBackupModel(entries []dotfile.Entry, cfg *config.Config) BackupModel {
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
	sp.Style = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)

	sel := components.NewSelector(items)
	sel.ActiveColor = theme.Secondary // Electric Cyan for Backup
	sel.CheckColor = theme.Success    // Emerald

	return BackupModel{
		entries:  entries,
		cfg:      cfg,
		phase:    phaseSelect,
		selector: sel,
		spinner:  sp,
		vp:       viewport.New(0, 0),
		animStep: 0,
	}
}

func (m *BackupModel) Reset(entries []dotfile.Entry) {
	m.entries = entries
	m.phase = phaseSelect
	m.results = nil
	m.gitPrompt = ""

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
	m.selector.ActiveColor = theme.Secondary
	m.selector.CheckColor = theme.Success
	if m.width > 0 && m.height > 0 {
		m.selector.SetSize(m.width, m.height-14)
	}
}

func (m BackupModel) Init() tea.Cmd {
	return m.spinner.Tick
}

type backupDoneMsg []dotfile.BackupResult
type gitDoneMsg struct{}

func (m BackupModel) Update(msg tea.Msg) (BackupModel, tea.Cmd) {
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
		padLeft := (msg.Width - 66) / 2
		if padLeft < 2 {
			padLeft = 2
		}
		m.vp.Width = msg.Width - (padLeft * 2)
		m.vp.Height = msg.Height - 16

	case backupDoneMsg:
		m.results = msg

		hasSuccess := false
		for _, r := range m.results {
			if r.Err == nil && !r.Skipped {
				hasSuccess = true
				break
			}
		}

		if hasSuccess && m.cfg.Git.AutoCommit {
			m.phase = phaseGit
			m.gitPrompt = "commit"
		} else {
			m.phase = phaseDone
		}
		return m, nil

	case gitDoneMsg:
		m.phase = phaseDone
		return m, nil

	case spinner.TickMsg:
		if m.phase == phaseExecute {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyMsg:
		switch m.phase {
		case phaseSelect:
			switch msg.String() {
			case "esc":
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHome}
				}
			case "enter":
				selected := m.selector.SelectedIndices()
				if len(selected) > 0 {
					m.phase = phaseDiff
					m.vp.SetContent(m.generateDiff(selected))
					return m, nil
				}
			}
			m.selector, cmd = m.selector.Update(msg)
			cmds = append(cmds, cmd)

		case phaseDiff:
			switch msg.String() {
			case "esc":
				m.phase = phaseSelect
				return m, nil
			case "enter":
				m.phase = phaseExecute
				return m, tea.Batch(
					m.spinner.Tick,
					func() tea.Msg {
						var targets []dotfile.Entry
						for _, idx := range m.selector.SelectedIndices() {
							targets = append(targets, m.entries[idx])
						}
						results := dotfile.BackupAll(targets)
						return backupDoneMsg(results)
					},
				)
			}
			m.vp, cmd = m.vp.Update(msg)
			cmds = append(cmds, cmd)

		case phaseGit:
			switch strings.ToLower(msg.String()) {
			case "y":
				if m.gitPrompt == "commit" {
					repoPath := config.ExpandPath(m.cfg.RepoPath)
					msg := git.CommitMessage(m.cfg.Git.CommitPrefix)
					if err := git.Add(repoPath); err != nil {
						m.phase = phaseDone
						return m, nil
					}
					if err := git.Commit(repoPath, msg); err != nil {
						m.phase = phaseDone
						return m, nil
					}
					if m.cfg.Git.AutoPush {
						_ = git.Push(repoPath)
						m.phase = phaseDone
						return m, nil
					}
					m.gitPrompt = "push"
					return m, nil
				} else if m.gitPrompt == "push" {
					repoPath := config.ExpandPath(m.cfg.RepoPath)
					_ = git.Push(repoPath)
					m.phase = phaseDone
					return m, nil
				}
			case "n", "esc":
				m.phase = phaseDone
				return m, nil
			}

		case phaseDone:
			switch msg.String() {
			case "enter", "esc":
				m.Reset(m.entries)
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHome}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m BackupModel) generateDiff(indices []int) string {
	var b strings.Builder
	for _, idx := range indices {
		entry := m.entries[idx]
		d, err := dotfile.Diff(entry)
		if err != nil {
			errStr := fmt.Sprintf("%s Error diffing %s: %v", theme.IconChanged, entry.Name, err)
			b.WriteString(theme.ErrorStyle.Render(errStr) + "\n")
			continue
		}

		if d == "" {
			iconStr := lipgloss.NewStyle().Width(3).Foreground(theme.Success).Render(theme.IconInSync)
			textStr := lipgloss.NewStyle().Foreground(theme.Muted).Render("No changes for " + entry.Name)
			b.WriteString(iconStr + textStr + "\n")
			continue
		}

		headerStr := lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render(fmt.Sprintf("─── %s ───", entry.Name))
		b.WriteString(headerStr + "\n")

		lines := strings.Split(d, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "+") {
				b.WriteString(theme.DiffAddStyle.Render(line) + "\n")
			} else if strings.HasPrefix(line, "-") {
				b.WriteString(theme.DiffDelStyle.Render(line) + "\n")
			} else if strings.HasPrefix(line, "@") {
				b.WriteString(theme.DiffHdrStyle.Render(line) + "\n")
			} else {
				b.WriteString(line + "\n")
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}

func (m BackupModel) View() string {
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
	case phaseSelect:
		title := indent + lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Secondary).
			Render("Select configurations to back up into repository:")

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
		statusHint = "space toggle • a toggle all • enter view diff • esc back • q quit"

	case phaseDiff:
		diffTitle := indent + lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Secondary).
			Render("Diff Preview (System ↔ Repository):")

		prevLines := strings.Split(m.vp.View(), "\n")
		var indentedPrev []string
		for _, l := range prevLines {
			indentedPrev = append(indentedPrev, indent+l)
		}

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			diffTitle,
			"",
			strings.Join(indentedPrev, "\n"),
		)
		statusHint = "up/down scroll • enter confirm backup • esc back • q quit"

	case phaseExecute:
		content = indent + lipgloss.NewStyle().Padding(3, 0).Render(
			fmt.Sprintf("%s Backing up %d items into repository...", m.spinner.View(), len(m.selector.SelectedIndices())),
		)
		statusHint = "Executing..."

	case phaseGit:
		prompt := fmt.Sprintf("%s Commit changes to git repository? (y/n)", theme.IconGit)
		if m.gitPrompt == "push" {
			prompt = fmt.Sprintf("%s Push committed changes to remote? (y/n)", theme.IconGit)
		}
		content = indent + lipgloss.NewStyle().Padding(3, 0).Render(
			lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render(prompt),
		)
		statusHint = "y yes • n no • q quit"

	case phaseDone:
		var b strings.Builder
		titleStr := indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Success).Render("Backup Complete:")
		b.WriteString(titleStr + "\n\n")

		for _, r := range m.results {
			iconStr := lipgloss.NewStyle().Width(3).Foreground(theme.Success).Render(theme.IconInSync)
			statusStr := lipgloss.NewStyle().Foreground(theme.Success).Render("Synced")
			if r.Err != nil {
				iconStr = lipgloss.NewStyle().Width(3).Foreground(theme.Error).Render(theme.IconChanged)
				statusStr = lipgloss.NewStyle().Foreground(theme.Error).Render(r.Err.Error())
			} else if r.Skipped {
				iconStr = lipgloss.NewStyle().Width(3).Foreground(theme.Muted).Render("-")
				statusStr = lipgloss.NewStyle().Foreground(theme.Muted).Render("Skipped")
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
		content = b.String()
		statusHint = "enter/esc return home • q quit"
	}

	contentHeight := lipgloss.Height(content) + lipgloss.Height(header)
	padHeight := m.height - contentHeight - 1
	if padHeight < 0 {
		padHeight = 0
	}
	padded := lipgloss.JoinVertical(lipgloss.Top, header, content, strings.Repeat("\n", padHeight))

	statusBar := components.StatusBar("Backup", statusHint, m.width)
	return lipgloss.JoinVertical(lipgloss.Top, padded, statusBar)
}
