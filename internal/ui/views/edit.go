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
	"github.com/charmbracelet/bubbles/key"
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
	entries  []dotfile.Entry
	cfg      *config.Config
	cursor   int
	width    int
	height   int
	mode     editSubMode
	fp       filepicker.Model
	animStep int
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
	fp.DirAllowed = false // DirAllowed=false allows Enter to drill/traverse into directories
	fp.FileAllowed = true
	fp.AutoHeight = false
	fp.Height = 14

	// Exclude esc from Back so esc strictly navigates to Home
	fp.KeyMap.Back = key.NewBinding(key.WithKeys("h", "backspace", "left"))

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
	var cmds []tea.Cmd

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
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewHome}
			}

		case "tab", "f":
			if m.mode == modeConfigList {
				m.mode = modeFilePicker
				return m, m.fp.Init()
			} else {
				m.mode = modeConfigList
				return m, nil
			}
		}

		if m.mode == modeConfigList {
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
					if entry.IsDir {
						m.mode = modeFilePicker
						m.fp.CurrentDirectory = entry.EditPath()
						return m, m.fp.Init()
					}
					path := entry.EditPath()
					ed := editor.Resolve(m.cfg.Editor)
					return m, func() tea.Msg {
						return OpenEditorMsg{Path: path, Editor: ed}
					}
				}
			case "o", "O":
				if len(m.entries) > 0 {
					entry := m.entries[m.cursor]
					path := entry.EditPath()
					ed := editor.Resolve(m.cfg.Editor)
					return m, func() tea.Msg {
						return OpenEditorMsg{Path: path, Editor: ed}
					}
				}
			}
			return m, nil
		}

		if m.mode == modeFilePicker {
			if msg.String() == "o" || msg.String() == "O" {
				ed := editor.Resolve(m.cfg.Editor)
				dirToOpen := m.fp.CurrentDirectory
				return m, func() tea.Msg {
					return OpenEditorMsg{Path: dirToOpen, Editor: ed}
				}
			}
		}
	}

	// Always forward messages (including readDirMsg from filepicker.Init) to filepicker when in picker mode
	if m.mode == modeFilePicker {
		var cmd tea.Cmd
		m.fp, cmd = m.fp.Update(msg)
		cmds = append(cmds, cmd)

		if didSelect, path := m.fp.DidSelectFile(msg); didSelect {
			ed := editor.Resolve(m.cfg.Editor)
			return m, func() tea.Msg {
				return OpenEditorMsg{Path: path, Editor: ed}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// decorateFilePickerLines adds appropriate Nerd Font icons and colors to items rendered by filepicker.
func decorateFilePickerLines(rawView string, indent string) string {
	lines := strings.Split(rawView, "\n")
	var decorated []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			decorated = append(decorated, "")
			continue
		}

		if strings.Contains(line, "Bummer") || strings.Contains(line, "No Files") {
			decorated = append(decorated, indent+theme.MutedStyle.Render("  "+line))
			continue
		}

		parts := strings.Fields(line)
		if len(parts) == 0 {
			decorated = append(decorated, indent+line)
			continue
		}

		lastWord := parts[len(parts)-1]
		isDir := strings.HasSuffix(lastWord, "/")
		cleanName := strings.TrimSuffix(lastWord, "/")

		icon, iconColor := theme.FileIconStyled(cleanName, isDir)
		coloredIcon := lipgloss.NewStyle().Width(3).Foreground(iconColor).Render(icon + " ")

		if strings.Contains(line, ">") || strings.Contains(line, "❯") {
			cursorStr := lipgloss.NewStyle().Width(3).Foreground(theme.Pink).Bold(true).Render(theme.IconCursor + " ")
			decorated = append(decorated, indent+cursorStr+coloredIcon+line[strings.Index(line, parts[0]):])
		} else {
			decorated = append(decorated, indent+"   "+coloredIcon+line)
		}
	}

	return strings.Join(decorated, "\n")
}

func (m EditModel) View() string {
	repoPath := ""
	if m.cfg != nil {
		repoPath = m.cfg.RepoPath
	}
	header := components.Header(m.width, m.animStep, repoPath)

	var contentBuilder strings.Builder

	// Calculate horizontal indent for left-aligned block centering
	blockWidth := 66
	padLeft := (m.width - blockWidth) / 2
	if padLeft < 2 {
		padLeft = 2
	}
	indent := strings.Repeat(" ", padLeft)

	// Clean, consistent page titles matching Setup and Backup
	titleText := "Select configurations to open in your editor:"
	if m.mode == modeFilePicker {
		titleText = "Browse filesystem to open configurations:"
	}
	title := indent + lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Pink).
		Render(titleText)
	contentBuilder.WriteString(title + "\n\n")

	if m.mode == modeFilePicker {
		currentDirBadge := indent + lipgloss.NewStyle().
			Foreground(theme.Secondary).
			Bold(true).
			Render(theme.IconDirModern+" "+shortenHome(m.fp.CurrentDirectory))

		fpView := decorateFilePickerLines(m.fp.View(), indent)

		pickerBox := lipgloss.JoinVertical(
			lipgloss.Left,
			currentDirBadge,
			"",
			fpView,
		)

		contentBuilder.WriteString(pickerBox)
	} else {
		for i, entry := range m.entries {
			isSelected := i == m.cursor

			cursorStyle := lipgloss.NewStyle().Width(3)
			cursorStr := cursorStyle.Render(" ")
			if isSelected {
				cursorStr = cursorStyle.Foreground(theme.Pink).Bold(true).Render(theme.IconCursor + " ")
			}

			icon, iconColor := theme.FileIconStyled(entry.Name, entry.IsDir)
			iconStr := lipgloss.NewStyle().Width(3).Foreground(iconColor).Render(icon + " ")

			nameStyle := lipgloss.NewStyle().Width(18)
			if isSelected {
				nameStyle = nameStyle.Foreground(theme.Pink).Bold(true)
			} else {
				nameStyle = nameStyle.Foreground(theme.Text)
			}
			nameStr := nameStyle.Render(entry.Name)

			pathStr := lipgloss.NewStyle().Foreground(theme.Muted).Render(shortenHome(entry.EditPath()))

			row := indent + lipgloss.JoinHorizontal(
				lipgloss.Left,
				cursorStr,
				iconStr,
				nameStr,
				pathStr,
			)

			contentBuilder.WriteString(row)
			if i < len(m.entries)-1 {
				contentBuilder.WriteString("\n")
			}
		}
	}

	content := lipgloss.NewStyle().
		PaddingTop(1).
		Render(contentBuilder.String())

	contentHeight := lipgloss.Height(content) + lipgloss.Height(header)
	padHeight := m.height - contentHeight - 3
	if padHeight < 0 {
		padHeight = 0
	}
	padded := lipgloss.JoinVertical(lipgloss.Top, header, content, strings.Repeat("\n", padHeight))

	hint := "j/k move • enter edit/browse • o open dir • f/tab browse files • esc back • q quit"
	if m.mode == modeFilePicker {
		hint = "j/k move • enter open/traverse • o open dir in nvim • h parent • tab list • esc back • q quit"
	}

	statusBar := components.StatusBar("Edit", hint, m.width)
	return lipgloss.JoinVertical(lipgloss.Top, padded, statusBar)
}

func (m *EditModel) SetMode(mode editSubMode) tea.Cmd {
	m.mode = mode
	if mode == modeFilePicker {
		return m.fp.Init()
	}
	return nil
}
