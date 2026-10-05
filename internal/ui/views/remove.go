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

type removePhase int

const (
	removePhaseSelect removePhase = iota
	removePhaseMode
	removePhaseConfirm
	removePhaseGit
	removePhaseExecute
	removePhaseDone
)

type removeFinishedMsg struct {
	results []*dotfile.RemoveResult
	err     error
}

type RemoveModel struct {
	entries         []dotfile.Entry
	selectedTargets []dotfile.Entry
	cfg             *config.Config
	phase           removePhase
	selector        components.Selector
	vp              viewport.Model
	spinner         spinner.Model
	results         []*dotfile.RemoveResult
	modeCursor      int // 0 = Symlink only, 1 = Repo & Symlink
	gitPrompt       string
	commitGit       bool
	pushGit         bool
	width           int
	height          int
	animStep        int
}

func NewRemoveModel(entries []dotfile.Entry, cfg *config.Config) RemoveModel {
	items := make([]components.SelectorItem, len(entries))
	for i, e := range entries {
		items[i] = components.SelectorItem{
			Name:     e.Name,
			Desc:     e.RemoveStatusLabel(),
			Path:     e.AbsRepoPath(),
			IsDir:    e.IsDir,
			Disabled: false,
		}
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(theme.Error).Bold(true)

	sel := components.NewSelector(items)
	sel.ActiveColor = theme.Error
	sel.CheckColor = theme.Error

	return RemoveModel{
		entries:  entries,
		cfg:      cfg,
		phase:    removePhaseSelect,
		selector: sel,
		spinner:  sp,
		vp:       viewport.New(0, 0),
		animStep: 0,
	}
}

func (m RemoveModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m RemoveModel) IsSearching() bool {
	return m.phase == removePhaseSelect && m.selector.IsSearching()
}

func (m *RemoveModel) Reset(entries []dotfile.Entry) {
	m.entries = entries
	m.selectedTargets = nil
	m.phase = removePhaseSelect
	m.results = nil
	m.modeCursor = 0
	m.gitPrompt = ""
	m.commitGit = false
	m.pushGit = false

	items := make([]components.SelectorItem, len(entries))
	for i, e := range entries {
		items[i] = components.SelectorItem{
			Name:     e.Name,
			Desc:     e.RemoveStatusLabel(),
			Path:     e.AbsRepoPath(),
			IsDir:    e.IsDir,
			Disabled: false,
		}
	}
	m.selector = components.NewSelector(items)
	m.selector.ActiveColor = theme.Error
	m.selector.CheckColor = theme.Error
	if m.width > 0 && m.height > 0 {
		m.selector.SetSize(m.width, m.height-14)
	}
}

func (m RemoveModel) Update(msg tea.Msg) (RemoveModel, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		m.animStep++
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.selector.SetSize(msg.Width, msg.Height-14)
		m.vp.Width = msg.Width - 4
		m.vp.Height = msg.Height - 14

	case removeFinishedMsg:
		m.results = msg.results
		m.phase = removePhaseDone
		return m, nil

	case tea.KeyMsg:
		switch m.phase {
		case removePhaseSelect:
			if m.selector.IsSearching() {
				var cmd tea.Cmd
				m.selector, cmd = m.selector.Update(msg)
				return m, cmd
			}

			switch msg.String() {
			case "esc":
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHome}
				}
			case "enter":
				selected := m.selector.SelectedItems()
				if len(selected) == 0 {
					idx := m.selector.CursorIndex()
					if idx >= 0 && idx < len(m.entries) {
						m.selectedTargets = []dotfile.Entry{m.entries[idx]}
					}
				} else {
					var targets []dotfile.Entry
					for _, item := range selected {
						for _, e := range m.entries {
							if strings.EqualFold(e.Name, item.Name) {
								targets = append(targets, e)
								break
							}
						}
					}
					m.selectedTargets = targets
				}

				if len(m.selectedTargets) > 0 {
					m.phase = removePhaseMode
					m.modeCursor = 0
				}
				return m, nil
			default:
				var cmd tea.Cmd
				m.selector, cmd = m.selector.Update(msg)
				return m, cmd
			}

		case removePhaseMode:
			switch msg.String() {
			case "esc":
				m.phase = removePhaseSelect
				return m, nil
			case "up", "k", "1":
				m.modeCursor = 0
				return m, nil
			case "down", "j", "2":
				m.modeCursor = 1
				return m, nil
			case "enter":
				if m.modeCursor == 0 {
					// Symlink only: directly execute removal
					m.phase = removePhaseExecute
					return m, m.startRemoveExecution(dotfile.RemoveModeSymlink)
				}
				// Repo & Symlink: ask confirmation
				m.phase = removePhaseConfirm
				return m, nil
			}

		case removePhaseConfirm:
			switch strings.ToLower(msg.String()) {
			case "y":
				if git.IsRepo(m.cfg.RepoPath) {
					m.phase = removePhaseGit
					m.gitPrompt = "commit"
					m.commitGit = false
					m.pushGit = false
					return m, nil
				}
				m.phase = removePhaseExecute
				return m, m.startRemoveExecution(dotfile.RemoveModeAll)
			case "n", "esc":
				m.phase = removePhaseMode
				return m, nil
			}

		case removePhaseGit:
			switch m.gitPrompt {
			case "commit":
				switch strings.ToLower(msg.String()) {
				case "y":
					m.commitGit = true
					m.gitPrompt = "push"
					return m, nil
				case "n":
					m.commitGit = false
					m.phase = removePhaseExecute
					return m, m.startRemoveExecution(dotfile.RemoveModeAll)
				case "esc":
					m.phase = removePhaseConfirm
					return m, nil
				}
			case "push":
				switch strings.ToLower(msg.String()) {
				case "y":
					m.pushGit = true
					m.phase = removePhaseExecute
					return m, m.startRemoveExecution(dotfile.RemoveModeAll)
				case "n":
					m.pushGit = false
					m.phase = removePhaseExecute
					return m, m.startRemoveExecution(dotfile.RemoveModeAll)
				case "esc":
					m.gitPrompt = "commit"
					return m, nil
				}
			}

		case removePhaseDone:
			switch msg.String() {
			case "enter", "esc", "q":
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHome}
				}
			}
		}
	}

	return m, nil
}

func (m RemoveModel) startRemoveExecution(mode dotfile.RemoveMode) tea.Cmd {
	targets := m.selectedTargets
	cfg := m.cfg
	commit := m.commitGit
	push := m.pushGit

	return func() tea.Msg {
		results, err := dotfile.RemoveAll(targets, mode, cfg, commit, "", push)
		return removeFinishedMsg{results: results, err: err}
	}
}

func (m RemoveModel) View() string {
	header := components.Header(m.width, m.height, m.animStep, m.cfg.RepoPath)
	blockWidth := 60
	padLeft := (m.width - blockWidth) / 2
	if padLeft < 2 {
		padLeft = 2
	}
	indent := strings.Repeat(" ", padLeft)

	var sb strings.Builder
	sb.WriteString(header)
	sb.WriteString("\n")

	switch m.phase {
	case removePhaseSelect:
		title := theme.TitleStyle.Foreground(theme.Error).Render("REMOVE CONFIGURATIONS")
		sub := theme.SubtitleStyle.Render("Select configurations to remove:")
		sb.WriteString(indent + title + "\n")
		sb.WriteString(indent + sub + "\n\n")
		sb.WriteString(m.selector.View())

	case removePhaseMode:
		title := theme.TitleStyle.Foreground(theme.Error).Render("SELECT REMOVAL MODE")
		targetNames := make([]string, len(m.selectedTargets))
		for i, t := range m.selectedTargets {
			targetNames[i] = t.Name
		}
		targetsStr := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true).Render(strings.Join(targetNames, ", "))
		sb.WriteString(indent + title + "\n")
		sb.WriteString(indent + theme.SubtitleStyle.Render("Target configurations: ") + targetsStr + "\n\n")

		// Mode 1: Symlink only
		cur1 := "  "
		style1 := theme.UnselectedStyle
		if m.modeCursor == 0 {
			cur1 = lipgloss.NewStyle().Foreground(theme.Error).Bold(true).Render(theme.IconCursor + " ")
			style1 = theme.RemoveActiveStyle
		}
		sb.WriteString(indent + cur1 + style1.Render("[1] Remove symlinks only") + "\n")
		sb.WriteString(indent + "    " + theme.MutedStyle.Render("Unlinks system files. Keeps files in repository and tracks them in dots.") + "\n\n")

		// Mode 2: Repo & Symlink
		cur2 := "  "
		style2 := theme.UnselectedStyle
		if m.modeCursor == 1 {
			cur2 = lipgloss.NewStyle().Foreground(theme.Error).Bold(true).Render(theme.IconCursor + " ")
			style2 = theme.RemoveActiveStyle
		}
		sb.WriteString(indent + cur2 + style2.Render("[2] Remove repository files and symlinks") + "\n")
		sb.WriteString(indent + "    " + theme.MutedStyle.Render("Deletes files from repo, unlinks system symlinks, and untracks from dots.") + "\n\n")

	case removePhaseConfirm:
		title := theme.ErrorStyle.Render("⚠️  CONFIRM DELETION")
		sb.WriteString(indent + title + "\n")
		sb.WriteString(indent + theme.WarningStyle.Render("This will permanently delete the following configurations from your repository:") + "\n\n")

		for _, t := range m.selectedTargets {
			icon, iconColor := theme.FileIconStyled(t.Name, t.IsDir)
			coloredIcon := lipgloss.NewStyle().Foreground(iconColor).Bold(true).Render(icon + " ")
			sb.WriteString(indent + "  " + coloredIcon + lipgloss.NewStyle().Bold(true).Foreground(theme.Text).Render(t.Name) + "\n")
			sb.WriteString(indent + "    " + theme.MutedStyle.Render("Repo:   "+t.AbsRepoPath()) + "\n")
			sb.WriteString(indent + "    " + theme.MutedStyle.Render("System: "+t.ResolveSystemPath()) + "\n\n")
		}

		prompt := fmt.Sprintf("%s Are you sure you want to proceed with permanent deletion? (y/n)", theme.IconQuit)
		sb.WriteString(indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Error).Render(prompt) + "\n")

	case removePhaseGit:
		title := theme.TitleStyle.Foreground(theme.Secondary).Render("GIT INTEGRATION")
		sb.WriteString(indent + title + "\n\n")

		if m.gitPrompt == "commit" {
			prompt := fmt.Sprintf("%s Commit removal changes to git repository? (y/n)", theme.IconGit)
			sb.WriteString(indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(prompt) + "\n")
		} else if m.gitPrompt == "push" {
			prompt := fmt.Sprintf("%s Push committed changes to remote repository? (y/n)", theme.IconGit)
			sb.WriteString(indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(prompt) + "\n")
		}

	case removePhaseExecute:
		title := theme.TitleStyle.Foreground(theme.Error).Render("REMOVING CONFIGURATIONS")
		sb.WriteString(indent + title + "\n\n")
		sb.WriteString(indent + m.spinner.View() + " Removing configurations...\n")

	case removePhaseDone:
		title := theme.SuccessStyle.Render("REMOVAL COMPLETE")
		sb.WriteString(indent + title + "\n\n")

		for _, res := range m.results {
			icon, iconColor := theme.FileIconStyled(res.Entry.Name, res.Entry.IsDir)
			coloredIcon := lipgloss.NewStyle().Foreground(iconColor).Bold(true).Render(icon + " ")
			sb.WriteString(indent + coloredIcon + lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(res.Entry.Name) + "\n")

			if res.Err != nil {
				sb.WriteString(indent + "  " + theme.ErrorStyle.Render("✗ Error: ") + theme.ErrorStyle.Render(res.Err.Error()) + "\n")
				continue
			}

			if res.SymlinkRemoved {
				sb.WriteString(indent + "  " + theme.SuccessStyle.Render("✓ ") + theme.MutedStyle.Render("Unlinked: "+res.Entry.ResolveSystemPath()) + "\n")
			} else if res.SymlinkSkipped {
				sb.WriteString(indent + "  " + theme.WarningStyle.Render("· ") + theme.MutedStyle.Render(res.Message) + "\n")
			}

			if res.RepoFilesRemoved {
				sb.WriteString(indent + "  " + theme.SuccessStyle.Render("✓ ") + theme.MutedStyle.Render("Deleted repo files: "+res.Entry.AbsRepoPath()) + "\n")
			}
			if res.ConfigRemoved {
				sb.WriteString(indent + "  " + theme.SuccessStyle.Render("✓ ") + theme.MutedStyle.Render("Untracked from dots configuration") + "\n")
			}
			if res.GitCommitted {
				sb.WriteString(indent + "  " + theme.SuccessStyle.Render("✓ ") + theme.MutedStyle.Render("Committed deletion to git") + "\n")
			}
			if res.GitPushed {
				sb.WriteString(indent + "  " + theme.SuccessStyle.Render("✓ ") + theme.MutedStyle.Render("Pushed changes to remote") + "\n")
			}
			sb.WriteString("\n")
		}
	}

	content := sb.String()

	statusHint := ""
	switch m.phase {
	case removePhaseSelect:
		if m.selector.IsSearching() {
			statusHint = "enter confirm search • esc cancel search"
		} else {
			statusHint = "↑/↓ navigate • space select • enter next • / search • esc home • q quit"
		}
	case removePhaseMode:
		statusHint = "↑/↓/1/2 select mode • enter confirm • esc back • q quit"
	case removePhaseConfirm:
		statusHint = "y confirm delete • n/esc cancel • q quit"
	case removePhaseGit:
		statusHint = "y yes • n no • esc back • q quit"
	case removePhaseDone:
		statusHint = "enter/esc return home • q quit"
	}

	statusBar := components.StatusBar("Remove", statusHint, m.width)
	return components.PlacePinnedStatusBar(content, statusBar, m.height)
}
