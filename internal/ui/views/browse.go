package views

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/editor"
	"github.com/BitwiseSang/dots/internal/ui/components"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type BrowseModel struct {
	cfg      *config.Config
	fp       filepicker.Model
	width    int
	height   int
	animStep int
}

func NewBrowseModel(cfg *config.Config) BrowseModel {
	fp := filepicker.New()
	startDir := "."
	if home, err := os.UserHomeDir(); err == nil {
		configDir := filepath.Join(home, ".config")
		if _, err := os.Stat(configDir); err == nil {
			startDir = configDir
		} else {
			startDir = home
		}
	}
	fp.CurrentDirectory = startDir
	fp.ShowHidden = true
	fp.DirAllowed = false // Enter drills into directories; 'o' opens directory in editor
	fp.FileAllowed = true
	fp.AutoHeight = false
	fp.Height = 10

	// Exclude esc from Back so esc strictly navigates to Home
	fp.KeyMap.Back = key.NewBinding(key.WithKeys("h", "backspace", "left"))

	// Style filepicker to look clean and native like ls -lA with a single cohesive cursor
	fp.Styles.Cursor = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
	fp.Styles.Directory = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
	fp.Styles.File = lipgloss.NewStyle().Foreground(theme.Text)
	fp.Styles.Permission = lipgloss.NewStyle().Foreground(theme.Muted)
	fp.Styles.FileSize = lipgloss.NewStyle().Foreground(theme.Subtle)
	fp.Styles.Selected = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)

	return BrowseModel{
		cfg:      cfg,
		fp:       fp,
		animStep: 0,
	}
}

func (m BrowseModel) Init() tea.Cmd {
	return m.fp.Init()
}

func (m *BrowseModel) SetDirectory(path string) tea.Cmd {
	m.fp.CurrentDirectory = path
	return m.fp.Init()
}

func (m BrowseModel) Refresh() tea.Cmd {
	return m.fp.Init()
}

func (m BrowseModel) Update(msg tea.Msg) (BrowseModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case TickMsg:
		m.animStep++
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Header (13) + Title (2) + Badge (2) + StatusBar (1) = 18 lines overhead
		avail := msg.Height - 18
		if avail < 5 {
			avail = 5
		}
		m.fp.Height = avail

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewHome}
			}
		case "tab":
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewEdit}
			}
		case "o", "O":
			ed := editor.Resolve(m.cfg.Editor)
			dirToOpen := m.fp.CurrentDirectory
			return m, func() tea.Msg {
				return OpenEditorMsg{Path: dirToOpen, Editor: ed}
			}
		}
	}

	var cmd tea.Cmd
	m.fp, cmd = m.fp.Update(msg)
	cmds = append(cmds, cmd)

	if didSelect, path := m.fp.DidSelectFile(msg); didSelect {
		ed := editor.Resolve(m.cfg.Editor)
		return m, func() tea.Msg {
			return OpenEditorMsg{Path: path, Editor: ed}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m BrowseModel) View() string {
	repoPath := ""
	if m.cfg != nil {
		repoPath = m.cfg.RepoPath
	}
	header := components.Header(m.width, m.animStep, repoPath)

	blockWidth := 66
	padLeft := (m.width - blockWidth) / 2
	if padLeft < 2 {
		padLeft = 2
	}
	indent := strings.Repeat(" ", padLeft)

	title := indent + lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Secondary).
		Render("Browse filesystem to open configurations:")

	currentDirBadge := indent + lipgloss.NewStyle().
		Foreground(theme.Secondary).
		Bold(true).
		Render(theme.IconDirModern+" "+shortenHome(m.fp.CurrentDirectory))

	// Indent raw filepicker lines directly without hardcoding icons or double arrows
	rawFpLines := strings.Split(m.fp.View(), "\n")
	var indentedFp []string
	for _, l := range rawFpLines {
		indentedFp = append(indentedFp, indent+l)
	}

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		currentDirBadge,
		"",
		strings.Join(indentedFp, "\n"),
	)

	contentHeight := lipgloss.Height(content) + lipgloss.Height(header)
	padHeight := m.height - contentHeight - 1
	if padHeight < 0 {
		padHeight = 0
	}
	padded := lipgloss.JoinVertical(lipgloss.Top, header, content, strings.Repeat("\n", padHeight))

	statusBar := components.StatusBar("Browse", "j/k move • enter open/traverse • o open dir in nvim • h parent • tab edit dotfiles • esc back • q quit", m.width)
	return lipgloss.JoinVertical(lipgloss.Top, padded, statusBar)
}
