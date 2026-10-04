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
}

func NewBackupModel(entries []dotfile.Entry, cfg *config.Config) BackupModel {
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

	return BackupModel{
		entries:  entries,
		cfg:      cfg,
		phase:    phaseSelect,
		selector: components.NewSelector(items),
		spinner:  sp,
		vp:       viewport.New(0, 0),
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
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.selector.SetSize(msg.Width, msg.Height-4)
		m.vp.Width = msg.Width - 4
		m.vp.Height = msg.Height - 6

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
		if m.gitPrompt == "commit" && m.cfg.Git.AutoPush {
			m.gitPrompt = "push"
		} else {
			m.phase = phaseDone
		}
		return m, nil

	case tea.KeyMsg:
		if msg.String() == "esc" {
			if m.phase == phaseSelect {
				return m, func() tea.Msg { return NavigateMsg{View: ViewHome} }
			} else if m.phase == phaseDiff {
				m.phase = phaseSelect
				return m, nil
			}
		}

		switch m.phase {
		case phaseSelect:
			if msg.String() == "enter" {
				selected := m.selector.SelectedIndices()
				if len(selected) > 0 {
					m.phase = phaseDiff
					m.vp.SetContent(m.generateDiff(selected))
				}
				return m, nil
			}
			m.selector, cmd = m.selector.Update(msg)
			cmds = append(cmds, cmd)

		case phaseDiff:
			if msg.String() == "enter" {
				m.phase = phaseExecute

				selected := m.selector.SelectedIndices()
				var toBackup []dotfile.Entry
				for _, idx := range selected {
					toBackup = append(toBackup, m.entries[idx])
				}

				return m, func() tea.Msg {
					res := dotfile.BackupAll(toBackup)
					return backupDoneMsg(res)
				}
			}
			m.vp, cmd = m.vp.Update(msg)
			cmds = append(cmds, cmd)

		case phaseGit:
			if msg.String() == "y" || msg.String() == "Y" {
				prompt := m.gitPrompt
				return m, func() tea.Msg {
					if prompt == "commit" {
						msg := git.CommitMessage(m.cfg.Git.CommitPrefix)
						git.Add(m.cfg.RepoPath)
						git.Commit(m.cfg.RepoPath, msg)
					} else if prompt == "push" {
						git.Push(m.cfg.RepoPath)
					}
					return gitDoneMsg{}
				}
			} else if msg.String() == "n" || msg.String() == "N" {
				m.phase = phaseDone
				return m, nil
			}

		case phaseDone:
			if msg.String() == "enter" || msg.String() == "esc" {
				return m, func() tea.Msg { return NavigateMsg{View: ViewHome} }
			}
		}
	}

	if m.phase == phaseExecute {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m BackupModel) generateDiff(indices []int) string {
	var b strings.Builder
	for _, idx := range indices {
		entry := m.entries[idx]
		d, err := dotfile.Diff(entry)
		if err != nil {
			b.WriteString(theme.ErrorStyle.Render(fmt.Sprintf("Error diffing %s: %v\n", entry.Name, err)))
			continue
		}

		if d == "" {
			b.WriteString(theme.MutedStyle.Render(fmt.Sprintf("No changes for %s\n", entry.Name)))
			continue
		}

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
	header := components.Header(m.width)
	var content string
	var statusHint string

	switch m.phase {
	case phaseSelect:
		content = lipgloss.NewStyle().Padding(2, 4).Render(m.selector.View())
		statusHint = "space toggle • a toggle all • enter proceed • esc back"

	case phaseDiff:
		content = lipgloss.NewStyle().Padding(2, 4).Render(m.vp.View())
		statusHint = "up/down scroll • enter confirm backup • esc back"

	case phaseExecute:
		content = lipgloss.NewStyle().Padding(4, 4).Render(
			fmt.Sprintf("%s Backing up %d items...", m.spinner.View(), len(m.selector.SelectedIndices())),
		)
		statusHint = "Executing..."

	case phaseGit:
		prompt := fmt.Sprintf("Commit changes to git? (y/n)")
		if m.gitPrompt == "push" {
			prompt = fmt.Sprintf("Push changes to remote? (y/n)")
		}
		content = lipgloss.NewStyle().Padding(4, 4).Render(prompt)
		statusHint = "y yes • n no"

	case phaseDone:
		var b strings.Builder
		b.WriteString("Backup complete:\n\n")
		for _, r := range m.results {
			icon := theme.SuccessStyle.Render("✓")
			status := "Success"
			if r.Err != nil {
				icon = theme.ErrorStyle.Render("✗")
				status = r.Err.Error()
			} else if r.Skipped {
				icon = theme.MutedStyle.Render("-")
				status = "Skipped"
			}

			b.WriteString(fmt.Sprintf("%s %s: %s\n", icon, r.Entry.Name, status))
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

	statusBar := components.StatusBar("Backup", statusHint, m.width)
	return lipgloss.JoinVertical(lipgloss.Top, padded, statusBar)
}
