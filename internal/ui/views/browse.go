package views

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/editor"
	"github.com/BitwiseSang/dots/internal/ui/components"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

type dirItem struct {
	name      string
	path      string
	isDir     bool
	isSymlink bool
	modeStr   string
	sizeStr   string
	symlinkTo string
}

type BrowseModel struct {
	cfg             *config.Config
	currentDir      string
	items           []dirItem
	filteredIndices []int
	cursor          int
	offset          int
	width           int
	height          int
	animStep        int
	searchInput     textinput.Model
	filtering       bool
	filterQuery     string
}

func readDirectory(dir string) []dirItem {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var items []dirItem
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		isSymlink := info.Mode()&os.ModeSymlink != 0
		symlinkTarget := ""
		if isSymlink {
			target, err := filepath.EvalSymlinks(filepath.Join(dir, e.Name()))
			if err == nil {
				symlinkTarget = target
			}
		}
		sizeStr := strings.Replace(humanize.Bytes(uint64(info.Size())), " ", "", 1)
		items = append(items, dirItem{
			name:      e.Name(),
			path:      filepath.Join(dir, e.Name()),
			isDir:     e.IsDir(),
			isSymlink: isSymlink,
			modeStr:   info.Mode().String(),
			sizeStr:   sizeStr,
			symlinkTo: symlinkTarget,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].isDir != items[j].isDir {
			return items[i].isDir
		}
		return strings.ToLower(items[i].name) < strings.ToLower(items[j].name)
	})

	return items
}

func NewBrowseModel(cfg *config.Config) BrowseModel {
	startDir := "."
	if home, err := os.UserHomeDir(); err == nil {
		configDir := filepath.Join(home, ".config")
		if _, err := os.Stat(configDir); err == nil {
			startDir = configDir
		} else {
			startDir = home
		}
	}

	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "type to filter directory..."
	ti.PromptStyle = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true)
	ti.TextStyle = lipgloss.NewStyle().Foreground(theme.Text)
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(theme.Muted)
	ti.CharLimit = 32

	items := readDirectory(startDir)
	indices := make([]int, len(items))
	for i := range items {
		indices[i] = i
	}

	return BrowseModel{
		cfg:             cfg,
		currentDir:      startDir,
		items:           items,
		filteredIndices: indices,
		cursor:          0,
		offset:          0,
		animStep:        0,
		searchInput:     ti,
		filtering:       false,
		filterQuery:     "",
	}
}

func (m *BrowseModel) recomputeFiltered() {
	if m.filterQuery == "" {
		m.filteredIndices = make([]int, len(m.items))
		for i := range m.items {
			m.filteredIndices[i] = i
		}
	} else {
		m.filteredIndices = nil
		q := strings.ToLower(m.filterQuery)
		for i, it := range m.items {
			if strings.Contains(strings.ToLower(it.name), q) {
				m.filteredIndices = append(m.filteredIndices, i)
			}
		}
	}
	if m.cursor >= len(m.filteredIndices) {
		m.cursor = 0
	}
	if m.cursor < 0 && len(m.filteredIndices) > 0 {
		m.cursor = 0
	}
	m.offset = 0
}

func (m BrowseModel) Init() tea.Cmd {
	return nil
}

func (m *BrowseModel) SetDirectory(path string) tea.Cmd {
	if fi, err := os.Stat(path); err == nil && fi.IsDir() {
		m.currentDir = path
	} else if err == nil {
		m.currentDir = filepath.Dir(path)
	}
	m.items = readDirectory(m.currentDir)
	m.filterQuery = ""
	m.searchInput.SetValue("")
	m.filtering = false
	m.recomputeFiltered()
	m.cursor = 0
	m.offset = 0
	return nil
}

func (m BrowseModel) Refresh() tea.Cmd {
	m.items = readDirectory(m.currentDir)
	m.recomputeFiltered()
	return nil
}

func (m BrowseModel) availHeight() int {
	headerLines := 13
	if m.height > 0 && m.height < 28 {
		headerLines = 5
	}
	overhead := headerLines + 5
	if m.filtering || m.filterQuery != "" {
		overhead += 2
	}
	avail := m.height - overhead
	if avail < 4 {
		avail = 4
	}
	return avail
}

func (m BrowseModel) Update(msg tea.Msg) (BrowseModel, tea.Cmd) {
	switch msg := msg.(type) {
	case TickMsg:
		m.animStep++
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.filtering {
			switch msg.String() {
			case "esc":
				if m.searchInput.Value() != "" {
					m.searchInput.SetValue("")
					m.filterQuery = ""
					m.recomputeFiltered()
				}
				m.filtering = false
				m.searchInput.Blur()
				return m, nil
			case "enter":
				m.filtering = false
				m.searchInput.Blur()
				if len(m.filteredIndices) > 0 {
					item := m.items[m.filteredIndices[m.cursor]]
					if item.isDir {
						m.currentDir = item.path
						m.items = readDirectory(m.currentDir)
						m.filterQuery = ""
						m.searchInput.SetValue("")
						m.recomputeFiltered()
						m.cursor = 0
						m.offset = 0
						return m, nil
					}
					ed := editor.Resolve(m.cfg.Editor)
					targetPath := item.path
					return m, func() tea.Msg {
						return OpenEditorMsg{Path: targetPath, Editor: ed}
					}
				}
				return m, nil
			case "up":
				if m.cursor > 0 {
					m.cursor--
				}
				return m, nil
			case "down":
				if m.cursor < len(m.filteredIndices)-1 {
					m.cursor++
				}
				return m, nil
			}

			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			if m.searchInput.Value() != m.filterQuery {
				m.filterQuery = m.searchInput.Value()
				m.recomputeFiltered()
			}
			return m, cmd
		}

		switch msg.String() {
		case "/":
			m.filtering = true
			m.searchInput.Focus()
			return m, textinput.Blink

		case "esc":
			if m.filterQuery != "" {
				m.filterQuery = ""
				m.searchInput.SetValue("")
				m.recomputeFiltered()
				return m, nil
			}
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewHome}
			}

		case "tab":
			return m, func() tea.Msg {
				return NavigateMsg{View: ViewEdit}
			}

		case "h", "backspace", "left":
			parent := filepath.Dir(m.currentDir)
			if parent != "" && parent != m.currentDir {
				m.currentDir = parent
				m.items = readDirectory(m.currentDir)
				m.filterQuery = ""
				m.searchInput.SetValue("")
				m.recomputeFiltered()
				m.cursor = 0
				m.offset = 0
			}
			return m, nil

		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filteredIndices)-1 {
				m.cursor++
			}

		case "enter":
			if len(m.filteredIndices) > 0 {
				item := m.items[m.filteredIndices[m.cursor]]
				if item.isDir {
					m.currentDir = item.path
					m.items = readDirectory(m.currentDir)
					m.filterQuery = ""
					m.searchInput.SetValue("")
					m.recomputeFiltered()
					m.cursor = 0
					m.offset = 0
					return m, nil
				}
				ed := editor.Resolve(m.cfg.Editor)
				targetPath := item.path
				return m, func() tea.Msg {
					return OpenEditorMsg{Path: targetPath, Editor: ed}
				}
			}

		case "o", "O":
			ed := editor.Resolve(m.cfg.Editor)
			dirToOpen := m.currentDir
			return m, func() tea.Msg {
				return OpenEditorMsg{Path: dirToOpen, Editor: ed}
			}
		}
	}

	return m, nil
}

func (m BrowseModel) View() string {
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

	title := indent + lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Secondary).
		Render("Browse filesystem to open configurations:")

	currentDirBadge := indent + lipgloss.NewStyle().
		Foreground(theme.Secondary).
		Bold(true).
		Render(theme.IconDirModern + "  " + shortenHome(m.currentDir))

	var searchBarRow string
	if m.filtering || m.filterQuery != "" {
		countBadge := fmt.Sprintf("(%d/%d)", len(m.filteredIndices), len(m.items))
		badgeStyle := lipgloss.NewStyle().Foreground(theme.Muted).Render(countBadge)
		searchBarRow = indent + m.searchInput.View() + " " + badgeStyle + "\n\n"
	}

	avail := m.availHeight()
	offset := m.offset
	if m.cursor < offset {
		offset = m.cursor
	} else if m.cursor >= offset+avail {
		offset = m.cursor - avail + 1
	}
	if offset < 0 {
		offset = 0
	}

	var fileRows []string
	if len(m.filteredIndices) == 0 {
		fileRows = append(fileRows, indent+theme.MutedStyle.Render("   No matching files or directories found."))
	} else {
		end := offset + avail
		if end > len(m.filteredIndices) {
			end = len(m.filteredIndices)
		}

		for i := offset; i < end; i++ {
			item := m.items[m.filteredIndices[i]]
			isCursor := i == m.cursor

			cursorStr := "  "
			if isCursor {
				cursorStr = lipgloss.NewStyle().Foreground(theme.Secondary).Bold(true).Render("> ")
			}

			modeStr := lipgloss.NewStyle().Width(10).Foreground(theme.Muted).Render(item.modeStr)
			if isCursor {
				modeStr = lipgloss.NewStyle().Width(10).Foreground(theme.Secondary).Bold(true).Render(item.modeStr)
			}

			sizeStr := fmt.Sprintf("%9s", item.sizeStr)
			if isCursor {
				sizeStr = lipgloss.NewStyle().Width(9).Align(lipgloss.Right).Foreground(theme.Secondary).Bold(true).Render(item.sizeStr)
			} else {
				sizeStr = lipgloss.NewStyle().Width(9).Align(lipgloss.Right).Foreground(theme.Subtle).Render(item.sizeStr)
			}

			nameStyle := lipgloss.NewStyle()
			if isCursor {
				nameStyle = nameStyle.Foreground(theme.Secondary).Bold(true)
			} else if item.isDir {
				nameStyle = nameStyle.Foreground(theme.Secondary).Bold(true)
			} else {
				nameStyle = nameStyle.Foreground(theme.Text)
			}

			displayName := item.name
			if item.isDir {
				displayName += "/"
			}
			if item.isSymlink && item.symlinkTo != "" {
				displayName += " → " + item.symlinkTo
			}
			nameStr := nameStyle.Render(displayName)

			row := indent + cursorStr + modeStr + sizeStr + " " + nameStr
			fileRows = append(fileRows, row)
		}
	}

	var content string
	if searchBarRow != "" {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			currentDirBadge,
			"",
			searchBarRow+strings.Join(fileRows, "\n"),
		)
	} else {
		content = lipgloss.JoinVertical(
			lipgloss.Left,
			title,
			"",
			currentDirBadge,
			"",
			strings.Join(fileRows, "\n"),
		)
	}

	var statusHint string
	if m.filtering {
		statusHint = "type to filter • enter done • esc clear • ↑/↓ move"
	} else if m.filterQuery != "" {
		statusHint = "enter open • / search • esc clear • h parent • tab edit • q quit"
	} else {
		statusHint = "enter open • / search • o nvim • h parent • tab edit • esc home • q quit"
	}

	statusBar := components.StatusBar("Browse", statusHint, m.width)
	topBlock := lipgloss.JoinVertical(lipgloss.Top, header, content)
	return components.PlacePinnedStatusBar(topBlock, statusBar, m.height)
}
