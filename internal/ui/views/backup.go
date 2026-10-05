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
	entries         []dotfile.Entry
	backedUpTargets []dotfile.Entry
	cfg             *config.Config
	phase           backupPhase
	selector        components.Selector
	vp              viewport.Model
	spinner         spinner.Model
	results         []dotfile.BackupResult
	width           int
	height          int
	gitPrompt       string
	gitCommitted    bool
	gitCommitMsg    string
	gitCommitErr    error
	gitPushed       bool
	gitPushErr      error
	animStep        int
}

func NewBackupModel(entries []dotfile.Entry, cfg *config.Config) BackupModel {
	items := make([]components.SelectorItem, len(entries))
	for i, e := range entries {
		bStatus := e.BackupStatus()
		disabled := (bStatus == dotfile.StatusInSync || bStatus == dotfile.StatusMissing)
		items[i] = components.SelectorItem{
			Name:     e.Name,
			Desc:     e.BackupStatusLabel(),
			Path:     e.ResolveSystemPath(),
			IsDir:    e.IsDir,
			Disabled: disabled,
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
	m.backedUpTargets = nil
	m.phase = phaseSelect
	m.results = nil
	m.gitPrompt = ""
	m.gitCommitted = false
	m.gitCommitMsg = ""
	m.gitCommitErr = nil
	m.gitPushed = false
	m.gitPushErr = nil

	items := make([]components.SelectorItem, len(entries))
	for i, e := range entries {
		bStatus := e.BackupStatus()
		disabled := (bStatus == dotfile.StatusInSync || bStatus == dotfile.StatusMissing)
		items[i] = components.SelectorItem{
			Name:     e.Name,
			Desc:     e.BackupStatusLabel(),
			Path:     e.ResolveSystemPath(),
			IsDir:    e.IsDir,
			Disabled: disabled,
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

func (m BackupModel) IsSearching() bool {
	return m.phase == phaseSelect && m.selector.IsSearching()
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
		headerLines := 13
		if msg.Height > 0 && msg.Height < 28 {
			headerLines = 5
		}
		m.selector.SetSize(msg.Width, msg.Height-headerLines-3)
		padLeft := (msg.Width - 66) / 2
		if padLeft < 2 {
			padLeft = 2
		}
		m.vp.Width = msg.Width - (padLeft * 2)
		m.vp.Height = msg.Height - headerLines - 4

	case backupDoneMsg:
		m.results = msg

		repoPath := config.ExpandPath(m.cfg.RepoPath)
		isGitRepo := git.IsRepo(repoPath)

		var targetRelPaths []string
		var targetNames []string
		for _, entry := range m.backedUpTargets {
			targetRelPaths = append(targetRelPaths, entry.RepoPath)
			targetNames = append(targetNames, entry.Name)
		}

		hasGitChanges := false
		if isGitRepo {
			for _, p := range targetRelPaths {
				if changed, _ := git.HasChangesForPath(repoPath, p); changed {
					hasGitChanges = true
					break
				}
			}
		}

		if isGitRepo && hasGitChanges {
			if m.cfg.Git.AutoCommit {
				commitMsg := git.CommitMessageWithEntries(m.cfg.Git.CommitPrefix, targetNames)
				if err := git.AddPaths(repoPath, targetRelPaths); err != nil {
					m.gitCommitErr = err
					m.phase = phaseDone
					return m, nil
				}
				if err := git.Commit(repoPath, commitMsg); err != nil {
					m.gitCommitErr = err
					m.phase = phaseDone
					return m, nil
				}
				m.gitCommitted = true
				m.gitCommitMsg = commitMsg

				if m.cfg.Git.AutoPush {
					if err := git.Push(repoPath); err != nil {
						m.gitPushErr = err
					} else {
						m.gitPushed = true
					}
				}
				m.phase = phaseDone
				return m, nil
			}

			m.phase = phaseGit
			m.gitPrompt = "commit"
			return m, nil
		}

		m.phase = phaseDone
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
			if m.selector.IsFiltering() {
				m.selector, cmd = m.selector.Update(msg)
				return m, cmd
			}
			switch msg.String() {
			case "esc":
				if m.selector.HasFilter() {
					m.selector.ClearFilter()
					return m, nil
				}
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
				cursor := m.selector.CursorIndex()
				if cursor >= 0 && cursor < len(m.entries) {
					m.phase = phaseDiff
					m.vp.SetContent(m.generateDiff([]int{cursor}))
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
				var targets []dotfile.Entry
				for _, idx := range m.selector.SelectedIndices() {
					if !m.selector.Items[idx].Disabled {
						targets = append(targets, m.entries[idx])
					}
				}
				if len(targets) == 0 {
					cursor := m.selector.CursorIndex()
					if cursor >= 0 && cursor < len(m.entries) && !m.selector.Items[cursor].Disabled {
						targets = append(targets, m.entries[cursor])
					}
				}
				if len(targets) == 0 {
					m.phase = phaseSelect
					return m, nil
				}

				m.backedUpTargets = targets
				m.phase = phaseExecute
				return m, tea.Batch(
					m.spinner.Tick,
					func() tea.Msg {
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
				repoPath := config.ExpandPath(m.cfg.RepoPath)
				if m.gitPrompt == "commit" {
					var targetRelPaths []string
					var targetNames []string
					for _, entry := range m.backedUpTargets {
						targetRelPaths = append(targetRelPaths, entry.RepoPath)
						targetNames = append(targetNames, entry.Name)
					}
					commitMsg := git.CommitMessageWithEntries(m.cfg.Git.CommitPrefix, targetNames)
					if err := git.AddPaths(repoPath, targetRelPaths); err != nil {
						m.gitCommitErr = err
						m.phase = phaseDone
						return m, nil
					}
					if err := git.Commit(repoPath, commitMsg); err != nil {
						m.gitCommitErr = err
						m.phase = phaseDone
						return m, nil
					}
					m.gitCommitted = true
					m.gitCommitMsg = commitMsg

					if m.cfg.Git.AutoPush {
						if err := git.Push(repoPath); err != nil {
							m.gitPushErr = err
						} else {
							m.gitPushed = true
						}
						m.phase = phaseDone
						return m, nil
					}
					m.gitPrompt = "push"
					return m, nil
				} else if m.gitPrompt == "push" {
					repoPath := config.ExpandPath(m.cfg.RepoPath)
					if err := git.Push(repoPath); err != nil {
						m.gitPushErr = err
					} else {
						m.gitPushed = true
					}
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
	width := m.width
	if width <= 0 {
		width = 80
	}
	height := m.height
	if height <= 0 {
		height = 24
	}

	repoPath := ""
	if m.cfg != nil {
		repoPath = m.cfg.RepoPath
	}
	header := components.Header(width, height, m.animStep, repoPath)
	var content string
	var statusHint string

	blockWidth := 66
	padLeft := (width - blockWidth) / 2
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
		if m.selector.IsFiltering() {
			statusHint = "tab toggle • enter done • esc clear • ↑/↓ move"
		} else if m.selector.HasFilter() {
			statusHint = "space toggle • / search • esc clear • enter diff • q quit"
		} else {
			statusHint = "space toggle • a all • / search • enter diff • esc home • q quit"
		}

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

		hasChanges := false
		for _, idx := range m.selector.SelectedIndices() {
			if !m.selector.Items[idx].Disabled {
				hasChanges = true
				break
			}
		}
		if !hasChanges {
			cursor := m.selector.CursorIndex()
			if cursor >= 0 && cursor < len(m.entries) && !m.selector.Items[cursor].Disabled {
				hasChanges = true
			}
		}

		if hasChanges {
			statusHint = "up/down scroll • enter confirm backup • esc back • q quit"
		} else {
			statusHint = "up/down scroll • esc back • q quit"
		}

	case phaseExecute:
		count := len(m.backedUpTargets)
		spinnerText := lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Render(
			fmt.Sprintf("%s Backing up %d items into repository...", m.spinner.View(), count),
		)
		content = "\n\n" + indent + spinnerText + "\n"
		statusHint = "Executing..."

	case phaseGit:
		prompt := fmt.Sprintf("%s Commit changes to git repository? (y/n)", theme.IconGit)
		if m.gitPrompt == "push" {
			prompt = fmt.Sprintf("%s Push committed changes to remote? (y/n)", theme.IconGit)
		}
		promptStyled := lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render(prompt)
		content = "\n\n" + indent + promptStyled + "\n"
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
				statusStr = lipgloss.NewStyle().Foreground(theme.Muted).Render("Skipped (missing)")
			} else if r.Entry.IsLinked() {
				iconStr = lipgloss.NewStyle().Width(3).Foreground(theme.Success).Render(theme.IconInSync)
				statusStr = lipgloss.NewStyle().Foreground(theme.Success).Render("Synced")
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

		b.WriteString("\n")
		if m.gitCommitted {
			gitIcon := lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render(theme.IconGit + " ")
			b.WriteString(indent + gitIcon + lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render("Git Commit: ") +
				lipgloss.NewStyle().Foreground(theme.Text).Render(m.gitCommitMsg) + "\n")
		}
		if m.gitPushed {
			pushIcon := lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render("✓ ")
			b.WriteString(indent + pushIcon + lipgloss.NewStyle().Foreground(theme.Success).Render("Pushed to remote repository\n"))
		} else if m.gitPushErr != nil {
			errIcon := lipgloss.NewStyle().Foreground(theme.Error).Bold(true).Render("✗ ")
			b.WriteString(indent + errIcon + lipgloss.NewStyle().Foreground(theme.Error).Render(fmt.Sprintf("Push failed: %v\n", m.gitPushErr)))
		} else if m.gitCommitErr != nil {
			errIcon := lipgloss.NewStyle().Foreground(theme.Error).Bold(true).Render("✗ ")
			b.WriteString(indent + errIcon + lipgloss.NewStyle().Foreground(theme.Error).Render(fmt.Sprintf("Git commit failed: %v\n", m.gitCommitErr)))
		} else if !m.gitCommitted {
			repoPath := config.ExpandPath(m.cfg.RepoPath)
			if git.IsRepo(repoPath) {
				hasChanges := false
				for _, entry := range m.backedUpTargets {
					if changed, _ := git.HasChangesForPath(repoPath, entry.RepoPath); changed {
						hasChanges = true
						break
					}
				}
				if !hasChanges {
					b.WriteString(indent + lipgloss.NewStyle().Foreground(theme.Muted).Render("✓ Selected configs are clean in git\n"))
				} else {
					b.WriteString(indent + lipgloss.NewStyle().Foreground(theme.Muted).Render("ℹ Git commit was skipped by user\n"))
				}
			}
		}

		content = b.String()
		statusHint = "enter/esc return home • q quit"
	}

	statusBar := components.StatusBar("Backup", statusHint, width)
	topBlock := lipgloss.JoinVertical(lipgloss.Top, header, content)
	return components.PlacePinnedStatusBar(topBlock, statusBar, height)
}
