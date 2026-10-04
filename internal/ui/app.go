package ui

import (
	"os/exec"
	"time"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/BitwiseSang/dots/internal/ui/views"
	"github.com/charmbracelet/harmonica"
	tea "github.com/charmbracelet/bubbletea"
)

type editorFinishedMsg struct {
	err error
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Millisecond*60, func(t time.Time) tea.Msg {
		return views.TickMsg{}
	})
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

	// Harmonica spring simulation for smooth view transition animations
	spring    harmonica.Spring
	springPos float64
	springVel float64
}

func NewApp(cfg *config.Config, initialView views.ViewType) AppModel {
	entries := dotfile.LoadEntries(cfg)

	// Harmonic spring for smooth damping transitions
	spring := harmonica.NewSpring(harmonica.FPS(30), 6.0, 0.7)

	editView := views.NewEditModel(entries, cfg)
	if initialView == views.ViewBrowse {
		_ = editView.SetMode(1) // Open directly in filepicker mode
		initialView = views.ViewEdit
	}

	return AppModel{
		currentView: initialView,
		home:        views.NewHomeModel(cfg.RepoPath),
		backup:      views.NewBackupModel(entries, cfg),
		setup:       views.NewSetupModel(entries, cfg),
		edit:        editView,
		cfg:         cfg,
		entries:     entries,
		spring:      spring,
		springPos:   0.0,
		springVel:   0.0,
	}
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
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
	case views.TickMsg:
		// Advance spring physics
		m.springPos, m.springVel = m.spring.Update(m.springPos, m.springVel, 1.0)

		// Loop animation tick
		cmds = append(cmds, tickCmd())

		// Forward tick to active child view to update header gradient and spinners
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
		// Universal quit with 'q' or Ctrl+C from ANY page/view
		if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
			m.quitting = true
			return m, tea.Quit
		}

	case views.NavigateMsg:
		var initCmd tea.Cmd
		if msg.View == views.ViewBrowse {
			initCmd = m.edit.SetMode(1) // switch to filepicker in edit view and trigger m.fp.Init()
			m.currentView = views.ViewEdit
		} else {
			m.currentView = msg.View
		}
		m.springPos = 0.0
		m.springVel = 0.0
		return m, initCmd

	case views.OpenEditorMsg:
		c := exec.Command(msg.Editor, msg.Path)
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			return editorFinishedMsg{err}
		})

	case editorFinishedMsg:
		m.entries = dotfile.LoadEntries(m.cfg)
		m.home = views.NewHomeModel(m.cfg.RepoPath)
		m.backup = views.NewBackupModel(m.entries, m.cfg)
		m.setup = views.NewSetupModel(m.entries, m.cfg)
		m.edit = views.NewEditModel(m.entries, m.cfg)

		sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
		m.backup, _ = m.backup.Update(sizeMsg)
		m.setup, _ = m.setup.Update(sizeMsg)
		m.edit, _ = m.edit.Update(sizeMsg)

		return m, m.edit.Init()
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
