package views

import (
	"fmt"
	"os"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/BitwiseSang/dots/internal/git"
	"github.com/BitwiseSang/dots/internal/ui/components"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type wizardStep int

const (
	wizardStepRepo wizardStep = iota
	wizardStepCloning
	wizardStepEditor
	wizardStepGit
	wizardStepIgnore
	wizardStepDiscover
	wizardStepComplete
)

type cloneFinishedMsg struct {
	err      error
	repoPath string
}

type WizardModel struct {
	cfg            *config.Config
	step           wizardStep
	repoInput      textinput.Model
	repoPath       string
	resolvedLocal  string
	resolvedRemote string
	isRemote       bool
	cloneErr       error
	editorOpts     []string
	editorIdx      int
	autoCommit     bool
	autoPush       bool
	gitOptIdx      int // 0: commit toggle, 1: push toggle
	selector       components.Selector
	repoSpecs      []config.DotfileSpec
	foundRepoCfg   bool
	dotIgnore      *dotfile.DotIgnore
	ignoreSelector components.Selector
	ignoreInput    textinput.Model
	ignoreAdding   bool
	ignoreMsg      string
	spinner        spinner.Model
	err            error
	linkedCount    int
	backedUpCount  int
	backupDir      string
	width          int
	height         int
	animStep       int
}

func NewWizardModel(cfg *config.Config) WizardModel {
	repoTi := textinput.New()
	repoTi.Prompt = "Repo: "
	repoTi.Placeholder = "e.g. username/repo, https://github.com/..., or ~/dotfiles"
	repoTi.PromptStyle = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
	repoTi.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	repoTi.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Muted)
	repoTi.CharLimit = 0 // 0 allows unlimited length for long filesystem paths and repository URLs
	repoTi.Focus()

	defaultRepo := "~/dotfiles"
	if cfg != nil && cfg.RepoPath != "" {
		defaultRepo = cfg.RepoPath
	}
	repoTi.SetValue(defaultRepo)

	editors := []string{"nvim", "vim", "helix", "code", "nano"}
	if envEd := os.Getenv("EDITOR"); envEd != "" {
		found := false
		for _, e := range editors {
			if e == envEd {
				found = true
				break
			}
		}
		if !found {
			editors = append([]string{envEd}, editors...)
		}
	}

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)

	sel := components.NewSelector(nil)
	sel.ActiveColor = theme.Secondary
	sel.CheckColor = theme.Success

	ignoreSel := components.NewSelector(nil)
	ignoreSel.ActiveColor = theme.Secondary
	ignoreSel.CheckColor = theme.Success

	ignoreTi := textinput.New()
	ignoreTi.Prompt = "Pattern: "
	ignoreTi.Placeholder = "e.g. *.log, temp/, backup.sh"
	ignoreTi.PromptStyle = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
	ignoreTi.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ignoreTi.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Muted)

	return WizardModel{
		cfg:            cfg,
		step:           wizardStepRepo,
		repoInput:      repoTi,
		editorOpts:     editors,
		editorIdx:      0,
		autoCommit:     true,
		autoPush:       false,
		gitOptIdx:      0,
		selector:       sel,
		ignoreSelector: ignoreSel,
		ignoreInput:    ignoreTi,
		spinner:        sp,
		animStep:       0,
	}
}

func (m WizardModel) IsTyping() bool {
	if m.step == wizardStepRepo {
		return true
	}
	if m.step == wizardStepIgnore && m.ignoreAdding {
		return true
	}
	if m.step == wizardStepDiscover && m.selector.IsFiltering() {
		return true
	}
	return false
}

func (m WizardModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, m.spinner.Tick)
}

func (m WizardModel) Update(msg tea.Msg) (WizardModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case TickMsg:
		m.animStep++
		return m, nil

	case spinner.TickMsg:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case cloneFinishedMsg:
		if msg.err != nil {
			m.cloneErr = msg.err
			return m, nil
		}
		return m.loadRepoAndAdvance(msg.repoPath)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerLines := 13
		if msg.Height > 0 && msg.Height < 28 {
			headerLines = 5
		}
		avail := msg.Height - headerLines - 6
		if avail < 3 {
			avail = 3
		}
		m.selector.SetSize(msg.Width, avail)
		m.ignoreSelector.SetSize(msg.Width, avail)

	case tea.KeyMsg:
		switch m.step {
		case wizardStepRepo:
			switch msg.String() {
			case "enter":
				val := strings.TrimSpace(m.repoInput.Value())
				if val == "" {
					val = "~/dotfiles"
				}
				isRemote, remoteURL, localPath := git.ResolveRepoInput(val)
				m.repoPath = val
				m.resolvedLocal = localPath
				m.resolvedRemote = remoteURL
				m.isRemote = isRemote

				if isRemote {
					if fi, err := os.Stat(localPath); err == nil && fi.IsDir() {
						return m.loadRepoAndAdvance(localPath)
					}
					m.step = wizardStepCloning
					m.cloneErr = nil
					return m, tea.Batch(
						m.spinner.Tick,
						func() tea.Msg {
							err := git.Clone(remoteURL, localPath)
							return cloneFinishedMsg{err: err, repoPath: localPath}
						},
					)
				} else {
					if _, err := os.Stat(localPath); os.IsNotExist(err) {
						if err := git.InitRepo(localPath); err != nil {
							m.cloneErr = err
							m.step = wizardStepCloning
							return m, nil
						}
					}
					return m.loadRepoAndAdvance(localPath)
				}
			case "esc":
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHome}
				}
			}
			m.repoInput, cmd = m.repoInput.Update(msg)
			return m, cmd

		case wizardStepCloning:
			if m.cloneErr != nil {
				switch msg.String() {
				case "r":
					m.cloneErr = nil
					remoteURL := m.resolvedRemote
					localPath := m.resolvedLocal
					return m, tea.Batch(
						m.spinner.Tick,
						func() tea.Msg {
							err := git.Clone(remoteURL, localPath)
							return cloneFinishedMsg{err: err, repoPath: localPath}
						},
					)
				case "esc":
					m.step = wizardStepRepo
					return m, nil
				case "q":
					return m, func() tea.Msg {
						return NavigateMsg{View: ViewHome}
					}
				}
			} else {
				if msg.String() == "q" {
					return m, func() tea.Msg {
						return NavigateMsg{View: ViewHome}
					}
				}
			}

		case wizardStepEditor:
			switch msg.String() {
			case "q":
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHome}
				}
			case "up", "k":
				if m.editorIdx > 0 {
					m.editorIdx--
				}
			case "down", "j":
				if m.editorIdx < len(m.editorOpts)-1 {
					m.editorIdx++
				}
			case "enter":
				m.step = wizardStepGit
				return m, nil
			case "esc":
				m.step = wizardStepRepo
				return m, nil
			}

		case wizardStepGit:
			switch msg.String() {
			case "q":
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHome}
				}
			case "up", "k", "down", "j", "tab":
				m.gitOptIdx = (m.gitOptIdx + 1) % 2
			case " ":
				if m.gitOptIdx == 0 {
					m.autoCommit = !m.autoCommit
				} else {
					m.autoPush = !m.autoPush
				}
			case "enter":
				m.step = wizardStepIgnore
				return m, nil
			case "esc":
				m.step = wizardStepEditor
				return m, nil
			}

		case wizardStepIgnore:
			if m.ignoreAdding {
				switch msg.String() {
				case "esc":
					m.ignoreAdding = false
					m.ignoreInput.Blur()
					return m, nil
				case "enter":
					pat := strings.TrimSpace(m.ignoreInput.Value())
					if pat != "" {
						m.ignoreSelector.Items = append(m.ignoreSelector.Items, components.SelectorItem{
							Name:     pat,
							Selected: true,
						})
					}
					m.ignoreAdding = false
					m.ignoreInput.Blur()
					m.ignoreInput.Reset()
					return m, nil
				}
				m.ignoreInput, cmd = m.ignoreInput.Update(msg)
				return m, cmd
			}

			switch msg.String() {
			case "q":
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHome}
				}
			case "a":
				m.ignoreAdding = true
				m.ignoreInput.Focus()
				m.ignoreInput.Reset()
				return m, textinput.Blink
			case "d", "x":
				idx := m.ignoreSelector.CursorIndex()
				if idx >= 0 && idx < len(m.ignoreSelector.Items) {
					m.ignoreSelector.Items = append(m.ignoreSelector.Items[:idx], m.ignoreSelector.Items[idx+1:]...)
					if m.ignoreSelector.CursorIndex() >= len(m.ignoreSelector.Items) && len(m.ignoreSelector.Items) > 0 {
						m.ignoreSelector.SetCursor(len(m.ignoreSelector.Items) - 1)
					}
				}
				return m, nil
			case "esc":
				m.step = wizardStepGit
				return m, nil
			case "enter":
				var pats []string
				for _, it := range m.ignoreSelector.Items {
					if it.Selected {
						pats = append(pats, it.Name)
					} else {
						pats = append(pats, "!"+it.Name)
					}
				}
				if m.dotIgnore == nil {
					m.dotIgnore = dotfile.LoadDotIgnore(m.resolvedLocal)
				}
				m.dotIgnore.Patterns = pats
				if err := m.dotIgnore.Save(); err != nil {
					m.ignoreMsg = fmt.Sprintf("Failed to save .dotignore: %v", err)
				}

				m.rescanDiscoveredConfigs()
				m.step = wizardStepDiscover
				return m, nil
			}
			m.ignoreSelector, cmd = m.ignoreSelector.Update(msg)
			return m, cmd

		case wizardStepDiscover:
			if m.selector.IsFiltering() {
				m.selector, cmd = m.selector.Update(msg)
				return m, cmd
			}

			switch msg.String() {
			case "q":
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHome}
				}
			case "esc":
				if m.selector.HasFilter() {
					m.selector.ClearFilter()
					return m, nil
				}
				m.step = wizardStepIgnore
				return m, nil
			case "i":
				idx := m.selector.CursorIndex()
				if idx >= 0 && idx < len(m.repoSpecs) {
					target := m.repoSpecs[idx]
					if m.dotIgnore == nil {
						m.dotIgnore = dotfile.LoadDotIgnore(m.resolvedLocal)
					}
					if err := m.dotIgnore.AddPattern(target.RepoPath); err != nil {
						m.ignoreMsg = fmt.Sprintf("Failed to ignore '%s': %v", target.Name, err)
					} else {
						m.ignoreMsg = fmt.Sprintf("Ignored '%s' (saved to .dotignore)", target.Name)
					}

					m.repoSpecs = append(m.repoSpecs[:idx], m.repoSpecs[idx+1:]...)
					newItems := append(m.selector.Items[:idx], m.selector.Items[idx+1:]...)
					m.selector.SetItems(newItems)
					if idx < len(newItems) {
						m.selector.SetCursor(idx)
					} else if len(newItems) > 0 {
						m.selector.SetCursor(len(newItems) - 1)
					}
				}
				return m, nil
			case "enter":
				return m.finalizeSetup()
			}
			m.selector, cmd = m.selector.Update(msg)
			return m, cmd

		case wizardStepComplete:
			switch msg.String() {
			case "enter", "esc", "q":
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHome}
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m WizardModel) loadRepoAndAdvance(repoPath string) (WizardModel, tea.Cmd) {
	m.resolvedLocal = repoPath
	m.dotIgnore = dotfile.LoadDotIgnore(repoPath)

	var ignoreItems []components.SelectorItem
	for _, p := range m.dotIgnore.Patterns {
		ignoreItems = append(ignoreItems, components.SelectorItem{
			Name:     p,
			Selected: !strings.HasPrefix(p, "!"),
		})
	}
	m.ignoreSelector = components.NewSelector(ignoreItems)
	m.ignoreSelector.ActiveColor = theme.Secondary
	m.ignoreSelector.CheckColor = theme.Success

	specs, repoCfg, _ := dotfile.InspectRepository(repoPath)
	if repoCfg != nil {
		if repoCfg.Editor != "" {
			for i, ed := range m.editorOpts {
				if ed == repoCfg.Editor {
					m.editorIdx = i
					break
				}
			}
		}
		if repoCfg.Git.AutoCommit {
			m.autoCommit = true
		}
		if repoCfg.Git.AutoPush {
			m.autoPush = true
		}
	}

	m.foundRepoCfg = config.FindRepoConfigFile(repoPath) != ""
	m.repoSpecs = specs

	items := make([]components.SelectorItem, len(specs))
	for i, s := range specs {
		typeLabel := "file"
		if s.IsDir {
			typeLabel = "directory"
		}
		items[i] = components.SelectorItem{
			Name:     s.Name,
			Desc:     typeLabel,
			Path:     s.SystemPath,
			IsDir:    s.IsDir,
			Selected: true,
		}
	}

	m.selector = components.NewSelector(items)
	m.selector.ActiveColor = theme.Secondary
	m.selector.CheckColor = theme.Success

	headerLines := 13
	if m.height > 0 && m.height < 28 {
		headerLines = 5
	}
	avail := m.height - headerLines - 6
	if avail < 3 {
		avail = 3
	}
	m.selector.SetSize(m.width, avail)
	m.ignoreSelector.SetSize(m.width, avail)

	m.step = wizardStepEditor
	return m, nil
}

func (m *WizardModel) rescanDiscoveredConfigs() {
	specs, repoCfg, _ := dotfile.InspectRepository(m.resolvedLocal)
	m.foundRepoCfg = config.FindRepoConfigFile(m.resolvedLocal) != ""
	if repoCfg != nil && len(repoCfg.Dotfiles) > 0 {
		specs = repoCfg.Dotfiles
	}
	m.repoSpecs = specs

	items := make([]components.SelectorItem, len(specs))
	for i, s := range specs {
		typeLabel := "file"
		if s.IsDir {
			typeLabel = "directory"
		}
		items[i] = components.SelectorItem{
			Name:     s.Name,
			Desc:     typeLabel,
			Path:     s.SystemPath,
			IsDir:    s.IsDir,
			Selected: true,
		}
	}

	m.selector = components.NewSelector(items)
	m.selector.ActiveColor = theme.Secondary
	m.selector.CheckColor = theme.Success

	headerLines := 13
	if m.height > 0 && m.height < 28 {
		headerLines = 5
	}
	avail := m.height - headerLines - 6
	if avail < 3 {
		avail = 3
	}
	m.selector.SetSize(m.width, avail)
}

func (m WizardModel) finalizeSetup() (WizardModel, tea.Cmd) {
	repoLocal := m.resolvedLocal
	if repoLocal == "" {
		_, _, repoLocal = git.ResolveRepoInput(m.repoInput.Value())
	}
	savedRepoPath := config.NormalizeRepoPath(repoLocal)

	newCfg := &config.Config{
		Editor:   m.editorOpts[m.editorIdx],
		RepoPath: savedRepoPath,
		Git: config.GitConfig{
			AutoCommit:   m.autoCommit,
			AutoPush:     m.autoPush,
			CommitPrefix: "Config backup",
		},
	}

	// Register all repository configurations into dots
	newCfg.Dotfiles = append([]config.DotfileSpec(nil), m.repoSpecs...)

	// Update any system paths if customized in the selector
	for i, spec := range newCfg.Dotfiles {
		for _, item := range m.selector.Items {
			if strings.EqualFold(spec.Name, item.Name) {
				newCfg.Dotfiles[i].SystemPath = item.Path
				break
			}
		}
	}

	if err := config.Save(newCfg); err != nil {
		m.err = err
	}

	if m.cfg != nil {
		*m.cfg = *newCfg
	}

	// Execute setup (symlink) on selected configurations
	selected := m.selector.SelectedItems()
	selectedMap := make(map[string]bool)
	for _, item := range selected {
		selectedMap[strings.ToLower(item.Name)] = true
	}

	var linkedCount, backedUpCount int
	backupDir, _ := dotfile.CreateBackupDir()
	for _, spec := range newCfg.Dotfiles {
		if selectedMap[strings.ToLower(spec.Name)] {
			entry := dotfile.NewEntry(spec, newCfg.RepoPath)
			backedUp, _, err := dotfile.Setup(entry, backupDir)
			if err == nil {
				linkedCount++
				if backedUp {
					backedUpCount++
				}
			}
		}
	}
	m.linkedCount = linkedCount
	m.backedUpCount = backedUpCount
	m.backupDir = backupDir

	m.step = wizardStepComplete
	return m, nil
}

func (m WizardModel) View() string {
	header := components.Header(m.width, m.height, m.animStep, m.repoInput.Value())

	blockWidth := 66
	padLeft := (m.width - blockWidth) / 2
	if padLeft < 2 {
		padLeft = 2
	}
	indent := strings.Repeat(" ", padLeft)

	var content string
	var statusHint string

	switch m.step {
	case wizardStepRepo:
		stepTitle := indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render("[1/5] Setup Dotfiles Repository") + "\n\n"
		desc := indent + lipgloss.NewStyle().Foreground(theme.Text).Render("Enter your GitHub repository shorthand, git URL, or local path:") + "\n"
		hint := indent + lipgloss.NewStyle().Foreground(theme.Muted).Render("• GitHub Shorthand: username/repository (e.g. BitwiseSang/dotfiles)\n" +
			indent + "• Git URL:          https://github.com/user/repo.git\n" +
			indent + "• Local Directory:  ~/Documents/dotfiles or ~/dotfiles\n\n")

		val := strings.TrimSpace(m.repoInput.Value())
		isRemote, remoteURL, localPath := git.ResolveRepoInput(val)
		var preview string
		if isRemote {
			preview = indent + lipgloss.NewStyle().Foreground(theme.Accent).Render(fmt.Sprintf("➜ Will clone %s into %s", remoteURL, shortenHome(localPath))) + "\n"
		} else {
			preview = indent + lipgloss.NewStyle().Foreground(theme.Success).Render(fmt.Sprintf("➜ Will use local repository at %s", shortenHome(localPath))) + "\n"
		}

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			stepTitle,
			desc,
			hint,
			indent+m.repoInput.View()+"\n",
			preview,
		)
		statusHint = "type repo path • enter next • esc cancel • q quit"

	case wizardStepCloning:
		stepTitle := indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render("[1/5] Cloning Repository") + "\n\n"

		if m.cloneErr != nil {
			errBox := indent + lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render("✗ Failed to clone repository:") + "\n\n" +
				indent + lipgloss.NewStyle().Foreground(theme.Muted).Render(m.cloneErr.Error()) + "\n\n" +
				indent + lipgloss.NewStyle().Foreground(theme.Secondary).Render("Press [r] to retry or [esc] to change repository path.")
			content = lipgloss.JoinVertical(lipgloss.Left, stepTitle, errBox)
			statusHint = "r retry • esc back • q quit"
		} else {
			spinnerText := indent + m.spinner.View() + " " +
				lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Render(fmt.Sprintf("Cloning %s into %s...", m.resolvedRemote, shortenHome(m.resolvedLocal))) + "\n\n"
			hint := indent + lipgloss.NewStyle().Foreground(theme.Muted).Render("Please wait while your configurations are fetched from the remote repository...")
			content = lipgloss.JoinVertical(lipgloss.Left, stepTitle, spinnerText, hint)
			statusHint = "cloning in progress... • q quit"
		}

	case wizardStepEditor:
		stepTitle := indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render("[2/5] Preferred Text Editor") + "\n\n"
		desc := indent + lipgloss.NewStyle().Foreground(theme.Text).Render("Select which editor dots will launch when editing configurations:") + "\n\n"

		var optRows []string
		for i, ed := range m.editorOpts {
			isCursor := i == m.editorIdx
			cursorStr := "   "
			if isCursor {
				cursorStr = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Render(" ❯ ")
			}
			edStyle := lipgloss.NewStyle().Foreground(theme.Text)
			if isCursor {
				edStyle = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
			}
			optRows = append(optRows, indent+cursorStr+edStyle.Render(ed))
		}

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			stepTitle,
			desc,
			strings.Join(optRows, "\n"),
		)
		statusHint = "↑/↓ select editor • enter next • esc back • q quit"

	case wizardStepGit:
		stepTitle := indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render("[3/5] Git Automation Preferences") + "\n\n"
		desc := indent + lipgloss.NewStyle().Foreground(theme.Text).Render("Configure automatic Git actions after backing up dotfiles:") + "\n\n"

		commitCheck := "[ ]"
		if m.autoCommit {
			commitCheck = "[" + theme.IconInSync + "]"
		}
		pushCheck := "[ ]"
		if m.autoPush {
			pushCheck = "[" + theme.IconInSync + "]"
		}

		commitCursor := "   "
		if m.gitOptIdx == 0 {
			commitCursor = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Render(" ❯ ")
		}
		pushCursor := "   "
		if m.gitOptIdx == 1 {
			pushCursor = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Render(" ❯ ")
		}

		row1 := indent + commitCursor + lipgloss.NewStyle().Foreground(theme.Success).Bold(m.autoCommit).Render(commitCheck) + " Automatically commit changes after backup"
		row2 := indent + pushCursor + lipgloss.NewStyle().Foreground(theme.Success).Bold(m.autoPush).Render(pushCheck) + " Automatically push commits to remote repository"

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			stepTitle,
			desc,
			row1,
			"",
			row2,
		)
		statusHint = "↑/↓ navigate • space toggle • enter next • esc back • q quit"

	case wizardStepIgnore:
		stepTitle := indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render("[4/5] Ignored Files & Patterns (.dotignore)") + "\n\n"
		desc := indent + lipgloss.NewStyle().Foreground(theme.Text).Render("Configure files and directories to ignore when managing your dotfiles:") + "\n"
		hint := indent + lipgloss.NewStyle().Foreground(theme.Muted).Render("(Checked patterns are ignored and saved to .dotignore in your repository)\n\n")

		if m.ignoreAdding {
			inputBox := indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render("Add new ignore pattern:") + "\n" +
				indent + m.ignoreInput.View() + "\n\n" +
				indent + theme.MutedStyle.Render("Press Enter to add pattern, Esc to cancel")
			content = lipgloss.JoinVertical(lipgloss.Left, stepTitle, desc, hint, inputBox)
			statusHint = "type pattern • enter confirm • esc cancel"
		} else {
			headerLines := components.HeaderHeight(m.width, m.height)
			avail := m.height - headerLines - 6
			if avail < 3 {
				avail = 3
			}
			m.ignoreSelector.SetSize(m.width, avail)

			selLines := strings.Split(m.ignoreSelector.View(), "\n")
			var indentedSel []string
			for _, l := range selLines {
				indentedSel = append(indentedSel, indent+l)
			}

			content = lipgloss.JoinVertical(
				lipgloss.Left,
				stepTitle,
				desc,
				hint,
				strings.Join(indentedSel, "\n"),
			)
			statusHint = "space toggle • a add pattern • d delete • enter next • esc back • q quit"
		}

	case wizardStepDiscover:
		stepTitle := indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render("[5/5] Repository Configurations")
		descText := "Select which configurations from your repository you want to link:"
		if m.foundRepoCfg {
			descText = "Detected existing configuration file in repository. Select configs to manage:"
		}
		desc := indent + lipgloss.NewStyle().Foreground(theme.Text).Render(descText)

		var banner string
		if m.ignoreMsg != "" {
			banner = indent + lipgloss.NewStyle().Foreground(theme.Accent).Bold(true).Render("⚠️  "+m.ignoreMsg) + "\n"
		}

		headerLines := components.HeaderHeight(m.width, m.height)
		avail := m.height - headerLines - 6
		if avail < 3 {
			avail = 3
		}
		m.selector.SetSize(m.width, avail)

		selLines := strings.Split(m.selector.View(), "\n")
		var indentedSel []string
		for _, l := range selLines {
			indentedSel = append(indentedSel, indent+l)
		}

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			stepTitle,
			"",
			desc,
			banner,
			strings.Join(indentedSel, "\n"),
		)
		if m.selector.IsFiltering() {
			statusHint = "tab toggle • enter finish • esc clear • ↑/↓ move"
		} else {
			statusHint = "space toggle • a all • i ignore item • / search • enter finish setup • esc back • q quit"
		}

	case wizardStepComplete:
		title := indent + lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Success).
			Render(theme.IconInSync + " Setup Completed Successfully!") + "\n\n"

		details := []string{
			fmt.Sprintf("• Managing %d total configuration(s) from %s", len(m.cfg.Dotfiles), m.cfg.RepoPath),
			fmt.Sprintf("• Successfully linked: %d configuration(s)", m.linkedCount),
		}
		if m.backedUpCount > 0 {
			details = append(details, fmt.Sprintf("• Backed up %d existing file(s) to %s", m.backedUpCount, m.backupDir))
		}
		if unlinked := len(m.cfg.Dotfiles) - m.linkedCount; unlinked > 0 {
			details = append(details, fmt.Sprintf("• %d unlinked configuration(s) can be enabled anytime from the Setup menu", unlinked))
		}

		desc := indent + lipgloss.NewStyle().
			Foreground(theme.Text).
			Render("Your dotfiles manager is fully configured and ready to use.") + "\n\n" +
			indent + strings.Join(details, "\n"+indent) + "\n\n" +
			indent + lipgloss.NewStyle().Foreground(theme.Muted).Render("Configuration saved to ~/.config/dots/config.toml")

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			desc,
		)
		statusHint = "enter launch dots • q quit"
	}

	statusBar := components.StatusBar("Setup Wizard", statusHint, m.width)
	topBlock := lipgloss.JoinVertical(lipgloss.Top, header, content)
	return components.PlacePinnedStatusBar(topBlock, statusBar, m.height)
}
