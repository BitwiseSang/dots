package views

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/BitwiseSang/dots/internal/editor"
	"github.com/BitwiseSang/dots/internal/ui/components"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type OpenEditorMsg struct {
	Path   string
	Editor string
}

type editSubMode int

const (
	modeConfigList editSubMode = iota
	modeFilePicker
)

type EditModel struct {
	entries    []dotfile.Entry
	cfg        *config.Config
	cursor     int
	width      int
	height     int
	mode       editSubMode
	fp         filepicker.Model
	animStep   int
	statusText string
}

func NewEditModel(entries []dotfile.Entry, cfg *config.Config) EditModel {
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
	fp.DirAllowed = true
	fp.FileAllowed = true
	fp.AutoHeight = false
	fp.Height = 12

	fp.Styles.Cursor = lipgloss.NewStyle().Foreground(theme.Pink).Bold(true)
	fp.Styles.Directory = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
	fp.Styles.File = lipgloss.NewStyle().Foreground(theme.Text)
	fp.Styles.Selected = lipgloss.NewStyle().Foreground(theme.Success).Bold(true)

	return EditModel{
		entries:  entries,
		cfg:      cfg,
		cursor:   0,
		mode:     modeConfigList,
		fp:       fp,
		animStep: 0,
	}
}

func (m EditModel) Init() tea.Cmd {
	return m.fp.Init()
}

func (m EditModel) Update(msg tea.Msg) (EditModel, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		m.animStep++
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.fp.Height = msg.Height - 16
		if m.fp.Height < 8 {
			m.fp.Height = 8
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.mode == modeFilePicker {
				m.mode = modeConfigList
				return m, nil
			}
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewHome}
			}

		case "tab", "f":
			if m.mode == modeConfigList {
				m.mode = modeFilePicker
				return m, nil
			} else {
				m.mode = modeConfigList
				return m, nil
			}
		}

		if m.mode == modeFilePicker {
			var cmd tea.Cmd
			m.fp, cmd = m.fp.Update(msg)

			if didSelect, path := m.fp.DidSelectFile(msg); didSelect {
				ed := editor.Resolve(m.cfg.Editor)
				return m, func() tea.Msg {
					return OpenEditorMsg{Path: path, Editor: ed}
				}
			}
			return m, cmd
		}

		// modeConfigList navigation
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.entries) > 0 {
				entry := m.entries[m.cursor]
				path := entry.EditPath()
				ed := editor.Resolve(m.cfg.Editor)
				return m, func() tea.Msg {
					return OpenEditorMsg{Path: path, Editor: ed}
				}
			}
		}
	}
	return m, nil
}

// decorateFilePickerLines adds appropriate Nerd Font icons to items rendered by filepicker.
func decorateFilePickerLines(rawView string) string {
	lines := strings.Split(rawView, "\n")
	var decorated []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			decorated = append(decorated, line)
			continue
		}

		// Extract base name after cursor or indentation
		parts := strings.Fields(line)
		if len(parts) == 0 {
			decorated = append(decorated, line)
			continue
		}

		lastWord := parts[len(parts)-1]
		isDir := strings.HasSuffix(lastWord, "/")
		cleanName := strings.TrimSuffix(lastWord, "/")

		icon := theme.FileIcon(cleanName, isDir)
		iconColor := theme.Secondary
		if isDir {
			iconColor = theme.Pink
		}

		coloredIcon := lipgloss.NewStyle().Foreground(iconColor).Render(icon)

		// If line has cursor '>' or '❯'
		if strings.Contains(line, ">") || strings.Contains(line, "❯") {
			cursorStr := lipgloss.NewStyle().Foreground(theme.Pink).Bold(true).Render(" " + theme.IconCursor + " ")
			decorated = append(decorated, cursorStr+coloredIcon+line[strings.Index(line, parts[0]):])
		} else {
			decorated = append(decorated, "   "+coloredIcon+line)
		}
	}

	return strings.Join(decorated, "\n")
}

func (m EditModel) View() string {
	header := components.Header(m.width, m.animStep)

	var contentBuilder strings.Builder

	// Mode tabs
	tabList := lipgloss.NewStyle().Foreground(theme.Text).Render(" Config Dotfiles ")
	tabPicker := lipgloss.NewStyle().Foreground(theme.Text).Render(" Filesystem Browser ")

	if m.mode == modeConfigList {
		tabList = lipgloss.NewStyle().Foreground(theme.Pink).Bold(true).Underline(true).Render(" [Config Dotfiles] ")
		tabPicker = lipgloss.NewStyle().Foreground(theme.Muted).Render("  Filesystem Browser (f/tab)  ")
	} else {
		tabList = lipgloss.NewStyle().Foreground(theme.Muted).Render("  Config Dotfiles (tab)  ")
		tabPicker = lipgloss.NewStyle().Foreground(theme.Pink).Bold(true).Underline(true).Render(" [Filesystem Browser] ")
	}

	tabsBar := lipgloss.JoinHorizontal(lipgloss.Center, tabList, "  •  ", tabPicker)
	contentBuilder.WriteString(lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(tabsBar) + "\n\n")

	if m.mode == modeFilePicker {
		// Filepicker view
		currentDirBadge := lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Bold(true).
			Render(" " + theme.IconDir + " " + m.fp.CurrentDirectory)

		fpView := decorateFilePickerLines(m.fp.View())

		pickerBox := lipgloss.JoinVertical(
			lipgloss.Left,
			currentDirBadge,
			"",
			fpView,
		)

		contentBuilder.WriteString(lipgloss.NewStyle().Padding(0, 4).Render(pickerBox))
	} else {
		// Clean config list with Magenta/Pink active highlight
		for i, entry := range m.entries {
			isSelected := i == m.cursor

			cursorStr := "   "
			if isSelected {
				cursorStr = lipgloss.NewStyle().Foreground(theme.Pink).Bold(true).Render(" " + theme.IconCursor + " ")
			}

			iconStr := lipgloss.NewStyle().Foreground(theme.Secondary).Render(theme.FileIcon(entry.Name, entry.IsDir))

			nameStyle := lipgloss.NewStyle().Width(18)
			if isSelected {
				nameStyle = nameStyle.Foreground(theme.Pink).Bold(true)
			} else {
				nameStyle = nameStyle.Foreground(theme.Text)
			}
			nameStr := nameStyle.Render(entry.Name)

			pathStr := lipgloss.NewStyle().Foreground(theme.Muted).Render(entry.EditPath())

			row := lipgloss.JoinHorizontal(
				lipgloss.Left,
				cursorStr,
				iconStr,
				nameStr,
				" ",
				pathStr,
			)

			contentBuilder.WriteString(row)
			if i < len(m.entries)-1 {
				contentBuilder.WriteString("\n")
			}
		}
	}

	content := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		PaddingTop(1).
		Render(contentBuilder.String())

	contentHeight := lipgloss.Height(content) + lipgloss.Height(header)
	padHeight := m.height - contentHeight - 3
	if padHeight < 0 {
		padHeight = 0
	}
	padded := lipgloss.JoinVertical(lipgloss.Top, header, content, strings.Repeat("\n", padHeight))

	hint := "j/k move • enter edit • f/tab browse files • esc back"
	if m.mode == modeFilePicker {
		hint = "j/k move • enter open • h parent • tab switch back • esc back"
	}

	statusBar := components.StatusBar("Edit", hint, m.width)
	return lipgloss.JoinVertical(lipgloss.Top, padded, statusBar)
}

func (m *EditModel) SetMode(mode editSubMode) {
	m.mode = mode
}
