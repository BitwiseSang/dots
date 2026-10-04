package ui

import (
	"os/exec"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/BitwiseSang/dots/internal/ui/views"
	tea "github.com/charmbracelet/bubbletea"
)

type editorFinishedMsg struct {
	err error
}

type AppModel struct {
	currentView views.ViewType
	home        views.HomeModel
	backup      views.BackupModel
	setup       views.SetupModel
	edit        views.EditModel
	cfg         *config.Config
	entries     []dotfile.Entry
	width       int
	height      int
	quitting    bool
}

func NewApp(cfg *config.Config, initialView views.ViewType) AppModel {
	entries := dotfile.LoadEntries(cfg)

	return AppModel{
		currentView: initialView,
		home:        views.NewHomeModel(),
		backup:      views.NewBackupModel(entries, cfg),
		setup:       views.NewSetupModel(entries, cfg),
		edit:        views.NewEditModel(entries, cfg),
		cfg:         cfg,
		entries:     entries,
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.home.Init(),
		m.backup.Init(),
		m.setup.Init(),
		m.edit.Init(),
	)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		var childCmd tea.Cmd
		m.home, childCmd = m.home.Update(msg)
		cmds = append(cmds, childCmd)
		m.backup, childCmd = m.backup.Update(msg)
		cmds = append(cmds, childCmd)
		m.setup, childCmd = m.setup.Update(msg)
		cmds = append(cmds, childCmd)
		m.edit, childCmd = m.edit.Update(msg)
		cmds = append(cmds, childCmd)

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || (msg.String() == "q" && m.currentView == views.ViewHome) {
			m.quitting = true
			return m, tea.Quit
		}

	case views.NavigateMsg:
		m.currentView = msg.View
		return m, nil

	case views.OpenEditorMsg:
		c := exec.Command(msg.Editor, msg.Path)
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			return editorFinishedMsg{err}
		})

	case editorFinishedMsg:
		m.entries = dotfile.LoadEntries(m.cfg)
		m.backup = views.NewBackupModel(m.entries, m.cfg)
		m.setup = views.NewSetupModel(m.entries, m.cfg)
		m.edit = views.NewEditModel(m.entries, m.cfg)

		sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
		m.backup, _ = m.backup.Update(sizeMsg)
		m.setup, _ = m.setup.Update(sizeMsg)
		m.edit, _ = m.edit.Update(sizeMsg)

		return m, nil
	}

	switch m.currentView {
	case views.ViewHome:
		m.home, cmd = m.home.Update(msg)
		cmds = append(cmds, cmd)
	case views.ViewBackup:
		m.backup, cmd = m.backup.Update(msg)
		cmds = append(cmds, cmd)
	case views.ViewSetup:
		m.setup, cmd = m.setup.Update(msg)
		cmds = append(cmds, cmd)
	case views.ViewEdit:
		m.edit, cmd = m.edit.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m AppModel) View() string {
	if m.quitting {
		return ""
	}

	switch m.currentView {
	case views.ViewHome:
		return m.home.View()
	case views.ViewBackup:
		return m.backup.View()
	case views.ViewSetup:
		return m.setup.View()
	case views.ViewEdit:
		return m.edit.View()
	default:
		return m.home.View()
	}
}
