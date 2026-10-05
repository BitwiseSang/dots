package views

import (
	"fmt"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/BitwiseSang/dots/internal/git"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type cliStep int

const (
	cliStepSelectMode cliStep = iota
	cliStepConfirm
	cliStepGitCommit
	cliStepGitMessage
	cliStepGitPush
	cliStepExecute
	cliStepDone
)

type CLIRemoveModel struct {
	Entry       dotfile.Entry
	Cfg         *config.Config
	Step        cliStep
	ModeCursor  int // 0 = symlink, 1 = repo & symlink
	Cancelled   bool
	CommitGit   bool
	PushGit     bool
	CommitInput textinput.Model
	Result      *dotfile.RemoveResult
	Err         error
	Width       int
}

func NewCLIRemoveModel(entry dotfile.Entry, cfg *config.Config) CLIRemoveModel {
	ti := textinput.New()
	ti.Placeholder = fmt.Sprintf("feat(config): remove %s config", entry.Name)
	ti.SetValue(fmt.Sprintf("feat(config): remove %s config", entry.Name))
	ti.CharLimit = 120
	ti.Width = 50

	return CLIRemoveModel{
		Entry:       entry,
		Cfg:         cfg,
		Step:        cliStepSelectMode,
		ModeCursor:  0,
		CommitInput: ti,
		Width:       80,
	}
}

func (m CLIRemoveModel) Init() tea.Cmd {
	return nil
}

func (m CLIRemoveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		return m, nil

	case tea.KeyMsg:
		// Universal abort keys
		if msg.Type == tea.KeyCtrlC {
			m.Cancelled = true
			return m, tea.Quit
		}

		switch m.Step {
		case cliStepSelectMode:
			switch msg.String() {
			case "q", "esc":
				m.Cancelled = true
				return m, tea.Quit
			case "up", "k", "1":
				m.ModeCursor = 0
				return m, nil
			case "down", "j", "2":
				m.ModeCursor = 1
				return m, nil
			case "enter":
				if m.ModeCursor == 0 {
					// Symlink only
					m.Step = cliStepExecute
					return m.executeRemoval(dotfile.RemoveModeSymlink)
				}
				// Repo & Symlink: ask confirm
				m.Step = cliStepConfirm
				return m, nil
			}

		case cliStepConfirm:
			switch strings.ToLower(msg.String()) {
			case "y":
				if git.IsRepo(m.Cfg.RepoPath) {
					m.Step = cliStepGitCommit
					return m, nil
				}
				m.Step = cliStepExecute
				return m.executeRemoval(dotfile.RemoveModeAll)
			case "n", "esc", "q", "enter":
				m.Cancelled = true
				return m, tea.Quit
			}

		case cliStepGitCommit:
			switch strings.ToLower(msg.String()) {
			case "y", "enter":
				m.CommitGit = true
				m.Step = cliStepGitMessage
				m.CommitInput.Focus()
				return m, textinput.Blink
			case "n":
				m.CommitGit = false
				m.Step = cliStepExecute
				return m.executeRemoval(dotfile.RemoveModeAll)
			case "esc", "q":
				m.Step = cliStepConfirm
				return m, nil
			}

		case cliStepGitMessage:
			switch msg.Type {
			case tea.KeyEnter:
				m.CommitInput.Blur()
				// Check for remote
				m.Step = cliStepGitPush
				return m, nil
			case tea.KeyEsc:
				m.Step = cliStepGitCommit
				return m, nil
			default:
				var cmd tea.Cmd
				m.CommitInput, cmd = m.CommitInput.Update(msg)
				return m, cmd
			}

		case cliStepGitPush:
			switch strings.ToLower(msg.String()) {
			case "y":
				m.PushGit = true
				m.Step = cliStepExecute
				return m.executeRemoval(dotfile.RemoveModeAll)
			case "n", "enter":
				m.PushGit = false
				m.Step = cliStepExecute
				return m.executeRemoval(dotfile.RemoveModeAll)
			case "esc":
				m.Step = cliStepGitMessage
				m.CommitInput.Focus()
				return m, textinput.Blink
			}

		case cliStepDone:
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m CLIRemoveModel) executeRemoval(mode dotfile.RemoveMode) (tea.Model, tea.Cmd) {
	commitMsg := strings.TrimSpace(m.CommitInput.Value())
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("feat(config): remove %s config", m.Entry.Name)
	}
	res, err := dotfile.Remove(m.Entry, mode, m.Cfg, m.CommitGit, commitMsg, m.PushGit)
	m.Result = res
	m.Err = err
	m.Step = cliStepDone
	return m, tea.Quit
}

func (m CLIRemoveModel) View() string {
	if m.Cancelled {
		return theme.MutedStyle.Render("Removal cancelled.") + "\n"
	}

	if m.Step == cliStepDone {
		var sb strings.Builder
		if m.Err != nil {
			sb.WriteString(theme.ErrorStyle.Render(fmt.Sprintf("✗ Error: %v\n", m.Err)))
			return sb.String()
		}

		sb.WriteString(theme.TitleStyle.Foreground(theme.Success).Render(fmt.Sprintf("✓ Removed %s successfully", m.Entry.Name)) + "\n")
		if m.Result != nil {
			if m.Result.SymlinkRemoved {
				sb.WriteString(theme.SuccessStyle.Render("  ✓ ") + theme.MutedStyle.Render(fmt.Sprintf("Unlinked system symlink: %s", m.Entry.ResolveSystemPath())) + "\n")
			} else if m.Result.SymlinkSkipped {
				sb.WriteString(theme.WarningStyle.Render("  · ") + theme.MutedStyle.Render(m.Result.Message) + "\n")
			}
			if m.Result.RepoFilesRemoved {
				sb.WriteString(theme.SuccessStyle.Render("  ✓ ") + theme.MutedStyle.Render(fmt.Sprintf("Deleted repository files: %s", m.Entry.AbsRepoPath())) + "\n")
			}
			if m.Result.ConfigRemoved {
				sb.WriteString(theme.SuccessStyle.Render("  ✓ ") + theme.MutedStyle.Render("Untracked from dots configuration") + "\n")
			}
			if m.Result.GitCommitted {
				sb.WriteString(theme.SuccessStyle.Render("  ✓ ") + theme.MutedStyle.Render(fmt.Sprintf("Git commit created: %s", m.CommitInput.Value())) + "\n")
			}
			if m.Result.GitPushed {
				sb.WriteString(theme.SuccessStyle.Render("  ✓ ") + theme.MutedStyle.Render("Pushed changes to remote repository") + "\n")
			}
		}
		return sb.String()
	}

	var sb strings.Builder

	icon, iconColor := theme.FileIconStyled(m.Entry.Name, m.Entry.IsDir)
	coloredIcon := lipgloss.NewStyle().Foreground(iconColor).Bold(true).Render(icon + " ")
	header := coloredIcon + lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(m.Entry.Name)

	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(theme.Error).Render("Dots Remove") + " — " + header + "\n")
	sb.WriteString(theme.MutedStyle.Render(fmt.Sprintf("  • System path:     %s", m.Entry.ResolveSystemPath())) + "\n")
	sb.WriteString(theme.MutedStyle.Render(fmt.Sprintf("  • Repository path: %s", m.Entry.AbsRepoPath())) + "\n\n")

	switch m.Step {
	case cliStepSelectMode:
		sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Select removal mode:") + "\n")

		cur0 := "  "
		style0 := theme.UnselectedStyle
		if m.ModeCursor == 0 {
			cur0 = lipgloss.NewStyle().Foreground(theme.Error).Bold(true).Render(theme.IconCursor + " ")
			style0 = theme.RemoveActiveStyle
		}
		sb.WriteString(cur0 + style0.Render("[1] Remove symlink only") + "\n")
		sb.WriteString("      " + theme.MutedStyle.Render("Unlinks system file. Preserves files in repository and tracks them in dots.") + "\n\n")

		cur1 := "  "
		style1 := theme.UnselectedStyle
		if m.ModeCursor == 1 {
			cur1 = lipgloss.NewStyle().Foreground(theme.Error).Bold(true).Render(theme.IconCursor + " ")
			style1 = theme.RemoveActiveStyle
		}
		sb.WriteString(cur1 + style1.Render("[2] Remove repository files and symlinks") + "\n")
		sb.WriteString("      " + theme.MutedStyle.Render("Deletes files from repository, unlinks symlink, and untracks from dots.") + "\n\n")

		sb.WriteString(theme.MutedStyle.Render("[↑/↓/1/2] Select mode  [Enter] Confirm  [q/Esc] Cancel") + "\n")

	case cliStepConfirm:
		sb.WriteString(theme.ErrorStyle.Render("⚠️  WARNING: ") + theme.WarningStyle.Render(fmt.Sprintf("This will permanently delete %s from your repository and unlink %s.", m.Entry.AbsRepoPath(), m.Entry.ResolveSystemPath())) + "\n\n")
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(theme.Error).Render("Are you sure you want to proceed? [y/N]: "))

	case cliStepGitCommit:
		prompt := fmt.Sprintf("%s Commit removal changes to git repository? [Y/n]: ", theme.IconGit)
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(prompt))

	case cliStepGitMessage:
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render("Commit message: ") + "\n")
		sb.WriteString("  " + m.CommitInput.View() + "\n\n")
		sb.WriteString(theme.MutedStyle.Render("[Enter] Confirm  [Esc] Back") + "\n")

	case cliStepGitPush:
		prompt := fmt.Sprintf("%s Push committed changes to remote repository? [y/N]: ", theme.IconGit)
		sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(prompt))
	}

	return sb.String()
}
