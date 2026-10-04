package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/BitwiseSang/dots/internal/ui/components"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type addConfigMode int

const (
	modeDiscover addConfigMode = iota
	modeManual
	modeSuccess
)

type AddConfigModel struct {
	cfg        *config.Config
	mode       addConfigMode
	selector   components.Selector
	discovered []dotfile.DiscoveredConfig

	// Manual form fields
	nameInput   textinput.Model
	systemInput textinput.Model
	repoInput   textinput.Model
	methodIndex int // 0: rsync, 1: copy
	formIndex   int // 0: name, 1: system, 2: repo, 3: method, 4: save button

	successMsg string
	width      int
	height     int
	animStep   int
}

func NewAddConfigModel(cfg *config.Config) AddConfigModel {
	nameTi := textinput.New()
	nameTi.Prompt = "Name:        "
	nameTi.Placeholder = "e.g. alacritty"
	nameTi.PromptStyle = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
	nameTi.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	nameTi.Focus()

	sysTi := textinput.New()
	sysTi.Prompt = "System Path: "
	sysTi.Placeholder = "e.g. ~/.config/alacritty"
	sysTi.PromptStyle = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
	sysTi.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)

	repoTi := textinput.New()
	repoTi.Prompt = "Repo Path:   "
	repoTi.Placeholder = "e.g. alacritty"
	repoTi.PromptStyle = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
	repoTi.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)

	m := AddConfigModel{
		cfg:         cfg,
		mode:        modeDiscover,
		nameInput:   nameTi,
		systemInput: sysTi,
		repoInput:   repoTi,
		methodIndex: 0,
		formIndex:   0,
		animStep:    0,
	}

	m.refreshDiscover()
	return m
}

func (m *AddConfigModel) refreshDiscover() {
	discovered := dotfile.DiscoverSystemConfigs(m.cfg)
	var unmanaged []dotfile.DiscoveredConfig
	for _, d := range discovered {
		if !d.AlreadyManaged {
			unmanaged = append(unmanaged, d)
		}
	}
	m.discovered = unmanaged

	items := make([]components.SelectorItem, len(unmanaged))
	for i, d := range unmanaged {
		typeLabel := "file"
		if d.IsDir {
			typeLabel = "directory"
		}
		items[i] = components.SelectorItem{
			Name:   d.Name,
			Desc:   typeLabel,
			Path:   d.SystemPath,
			IsDir:  d.IsDir,
			Status: "unmanaged",
		}
	}
	m.selector = components.NewSelector(items)
	m.selector.ActiveColor = theme.Secondary
	m.selector.CheckColor = theme.Success
}

func (m *AddConfigModel) PreFill(path string) {
	m.mode = modeManual
	expanded := config.ExpandPath(path)
	fi, err := os.Stat(expanded)
	isDir := err == nil && fi.IsDir()

	base := filepath.Base(expanded)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	if name == "" || name == "." {
		name = base
	}

	m.nameInput.SetValue(name)
	m.systemInput.SetValue(path)
	m.repoInput.SetValue(name)
	if isDir {
		m.methodIndex = 0 // rsync
	} else {
		m.methodIndex = 1 // copy
	}
	m.formIndex = 0
	m.nameInput.Focus()
	m.systemInput.Blur()
	m.repoInput.Blur()
}

func (m AddConfigModel) Init() tea.Cmd {
	return nil
}

func (m AddConfigModel) Update(msg tea.Msg) (AddConfigModel, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		m.animStep++
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.selector.SetSize(msg.Width, msg.Height-14)

	case tea.KeyMsg:
		switch m.mode {
		case modeDiscover:
			if m.selector.IsFiltering() {
				var cmd tea.Cmd
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
			case "m", "tab":
				m.mode = modeManual
				m.formIndex = 0
				m.nameInput.Focus()
				m.systemInput.Blur()
				m.repoInput.Blur()
				return m, nil
			case "enter":
				selected := m.selector.SelectedItems()
				if len(selected) == 0 && len(m.discovered) > 0 {
					// Add the currently highlighted item if none checked
					idx := m.selector.CursorIndex()
					if idx >= 0 && idx < len(m.discovered) {
						selected = append(selected, m.selector.Items[idx])
					}
				}

				if len(selected) > 0 {
					count := 0
					for _, item := range selected {
						isDir := item.IsDir
						method := "copy"
						if isDir {
							method = "rsync"
						}
						spec := config.DotfileSpec{
							Name:       item.Name,
							SystemPath: item.Path,
							RepoPath:   item.Name,
							Method:     method,
							IsDir:      isDir,
						}
						if m.cfg.AddDotfile(spec) {
							count++
						}
					}
					_ = config.Save(m.cfg)
					m.mode = modeSuccess
					m.successMsg = fmt.Sprintf("Successfully added %d configuration(s) to dots!", count)
					return m, nil
				}
			}
			var cmd tea.Cmd
			m.selector, cmd = m.selector.Update(msg)
			return m, cmd

		case modeManual:
			switch msg.String() {
			case "esc":
				m.mode = modeDiscover
				m.refreshDiscover()
				return m, nil

			case "tab", "down":
				m.formIndex = (m.formIndex + 1) % 5
				m.updateFormFocus()
				return m, nil

			case "shift+tab", "up":
				m.formIndex = (m.formIndex + 4) % 5
				m.updateFormFocus()
				return m, nil

			case " ":
				if m.formIndex == 3 {
					m.methodIndex = (m.methodIndex + 1) % 2
					return m, nil
				}

			case "enter":
				if m.formIndex == 4 || (m.nameInput.Value() != "" && m.systemInput.Value() != "") {
					name := strings.TrimSpace(m.nameInput.Value())
					sysPath := strings.TrimSpace(m.systemInput.Value())
					repoPath := strings.TrimSpace(m.repoInput.Value())
					if repoPath == "" {
						repoPath = name
					}

					if name == "" || sysPath == "" {
						return m, nil
					}

					method := "rsync"
					if m.methodIndex == 1 {
						method = "copy"
					}
					fi, err := os.Stat(config.ExpandPath(sysPath))
					isDir := err == nil && fi.IsDir()

					spec := config.DotfileSpec{
						Name:       name,
						SystemPath: sysPath,
						RepoPath:   repoPath,
						Method:     method,
						IsDir:      isDir,
					}

					m.cfg.AddDotfile(spec)
					_ = config.Save(m.cfg)
					m.mode = modeSuccess
					m.successMsg = fmt.Sprintf("Successfully added '%s' to dots!", name)
					return m, nil
				}
			}

			// Forward typing to focused input
			var cmd tea.Cmd
			switch m.formIndex {
			case 0:
				m.nameInput, cmd = m.nameInput.Update(msg)
				if m.repoInput.Value() == "" || m.repoInput.Value() == strings.TrimSuffix(m.nameInput.Value(), msg.String()) {
					m.repoInput.SetValue(m.nameInput.Value())
				}
			case 1:
				m.systemInput, cmd = m.systemInput.Update(msg)
			case 2:
				m.repoInput, cmd = m.repoInput.Update(msg)
			}
			return m, cmd

		case modeSuccess:
			switch msg.String() {
			case "enter", "esc":
				return m, func() tea.Msg {
					return NavigateMsg{View: ViewHome}
				}
			}
		}
	}

	return m, nil
}

func (m *AddConfigModel) updateFormFocus() {
	m.nameInput.Blur()
	m.systemInput.Blur()
	m.repoInput.Blur()

	switch m.formIndex {
	case 0:
		m.nameInput.Focus()
	case 1:
		m.systemInput.Focus()
	case 2:
		m.repoInput.Focus()
	}
}

func (m AddConfigModel) View() string {
	repoPath := ""
	if m.cfg != nil {
		repoPath = m.cfg.RepoPath
	}
	header := components.Header(m.width, m.height, m.animStep, repoPath)

	blockWidth := 66
	padLeft := (m.width - blockWidth) / 2
	if padLeft < 2 {
		padLeft = 2
	}
	indent := strings.Repeat(" ", padLeft)

	var content string
	var statusHint string

	switch m.mode {
	case modeDiscover:
		tabHeader := indent + lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render("[Discovered Configurations]") +
			"   " + lipgloss.NewStyle().Foreground(theme.Muted).Render("[Tab] Switch to Manual Form") + "\n\n"

		title := indent + lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Secondary).
			Render("Select unmanaged configurations to track in dots:")

		selLines := strings.Split(m.selector.View(), "\n")
		var indentedSel []string
		for _, l := range selLines {
			indentedSel = append(indentedSel, indent+l)
		}

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			tabHeader,
			title,
			"",
			strings.Join(indentedSel, "\n"),
		)

		if m.selector.IsFiltering() {
			statusHint = "tab toggle • enter add • esc clear • ↑/↓ move"
		} else {
			statusHint = "space toggle • enter add • / search • tab manual • esc home • q quit"
		}

	case modeManual:
		tabHeader := indent + lipgloss.NewStyle().Foreground(theme.Muted).Render("[Esc] Back to Discovered") +
			"   " + lipgloss.NewStyle().Bold(true).Foreground(theme.Secondary).Render("[Manual Configuration Form]") + "\n\n"

		title := indent + lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Secondary).
			Render("Define a new configuration to track:")

		methods := []string{"rsync (directory)", "copy (single file)"}
		methodStr := methods[m.methodIndex]
		methodStyle := lipgloss.NewStyle().Foreground(theme.Accent).Bold(true)
		if m.formIndex == 3 {
			methodStyle = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Underline(true)
		}

		saveBtn := lipgloss.NewStyle().Foreground(theme.Muted).Render("[ Save Configuration ]")
		if m.formIndex == 4 {
			saveBtn = lipgloss.NewStyle().Foreground(theme.Success).Bold(true).Render("[ Save Configuration ] ↵")
		}

		formRows := []string{
			indent + m.nameInput.View(),
			indent + m.systemInput.View(),
			indent + m.repoInput.View(),
			indent + lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Render("Sync Method: ") + methodStyle.Render(methodStr) + " (Space to toggle)",
			"",
			indent + saveBtn,
		}

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			tabHeader,
			title,
			"",
			strings.Join(formRows, "\n"),
		)
		statusHint = "tab/↑↓ navigate fields • space toggle method • enter save • esc back • q quit"

	case modeSuccess:
		title := indent + lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Success).
			Render(theme.IconInSync + " Configuration Added Successfully!")

		desc := indent + lipgloss.NewStyle().
			Foreground(theme.Text).
			Render(m.successMsg) + "\n\n" +
			indent + lipgloss.NewStyle().Foreground(theme.Muted).Render("Configuration saved to ~/.config/dots/config.toml")

		content = lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			desc,
		)
		statusHint = "enter return home • q quit"
	}

	statusBar := components.StatusBar("Add Config", statusHint, m.width)
	topBlock := lipgloss.JoinVertical(lipgloss.Top, header, content)
	return components.PlacePinnedStatusBar(topBlock, statusBar, m.height)
}
