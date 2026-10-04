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
	browse      views.BrowseModel
	addConfig   views.AddConfigModel
	wizard      views.WizardModel
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

	return AppModel{
		currentView: initialView,
		home:        views.NewHomeModel(cfg.RepoPath),
		backup:      views.NewBackupModel(entries, cfg),
		setup:       views.NewSetupModel(entries, cfg),
		edit:        views.NewEditModel(entries, cfg),
		browse:      views.NewBrowseModel(cfg),
		addConfig:   views.NewAddConfigModel(cfg),
		wizard:      views.NewWizardModel(cfg),
		cfg:         cfg,
		entries:     entries,
		spring:      spring,
		springPos:   0.0,
		springVel:   0.0,
	}
}

func NewAppWithPath(cfg *config.Config, initialView views.ViewType, path string) AppModel {
	app := NewApp(cfg, initialView)
	if initialView == views.ViewAddConfig && path != "" {
		app.addConfig.PreFill(path)
	}
	return app
}

func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		m.home.Init(),
		m.backup.Init(),
		m.setup.Init(),
		m.edit.Init(),
		m.browse.Init(),
		m.addConfig.Init(),
		m.wizard.Init(),
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
		case views.ViewBrowse:
			m.browse, cmd = m.browse.Update(msg)
			cmds = append(cmds, cmd)
		case views.ViewAddConfig:
			m.addConfig, cmd = m.addConfig.Update(msg)
			cmds = append(cmds, cmd)
		case views.ViewWizard:
			m.wizard, cmd = m.wizard.Update(msg)
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
		m.browse, childCmd = m.browse.Update(msg)
		cmds = append(cmds, childCmd)
		m.addConfig, childCmd = m.addConfig.Update(msg)
		cmds = append(cmds, childCmd)
		m.wizard, childCmd = m.wizard.Update(msg)
		cmds = append(cmds, childCmd)

	case tea.KeyMsg:
		// Universal quit with 'q' or Ctrl+C from ANY page/view
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			return m, tea.Quit
		}
		if msg.String() == "q" {
			isTyping := false
			if m.currentView == views.ViewAddConfig {
				isTyping = m.addConfig.IsTyping()
			} else if m.currentView == views.ViewWizard {
				isTyping = m.wizard.IsTyping()
			}
			if !isTyping {
				m.quitting = true
				return m, tea.Quit
			}
		}

	case views.NavigateMsg:
		var initCmd tea.Cmd
		m.currentView = msg.View
		if msg.View == views.ViewBrowse {
			if msg.Path != "" {
				initCmd = m.browse.SetDirectory(msg.Path)
			} else {
				initCmd = m.browse.Refresh()
			}
		} else if msg.View == views.ViewBackup {
			m.backup.Reset(m.entries)
		} else if msg.View == views.ViewSetup {
			m.setup.Reset(m.entries)
		} else if msg.View == views.ViewAddConfig {
			m.addConfig = views.NewAddConfigModel(m.cfg)
			if msg.Path != "" {
				m.addConfig.PreFill(msg.Path)
			}
			if m.width > 0 && m.height > 0 {
				sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
				m.addConfig, _ = m.addConfig.Update(sizeMsg)
			}
		} else if msg.View == views.ViewWizard {
			m.wizard = views.NewWizardModel(m.cfg)
			if m.width > 0 && m.height > 0 {
				sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
				m.wizard, _ = m.wizard.Update(sizeMsg)
			}
			initCmd = m.wizard.Init()
		} else if msg.View == views.ViewHome {
			// Reload entries in case new configs were added
			m.entries = dotfile.LoadEntries(m.cfg)
			m.home = views.NewHomeModel(m.cfg.RepoPath)
			m.backup = views.NewBackupModel(m.entries, m.cfg)
			m.setup = views.NewSetupModel(m.entries, m.cfg)
			m.edit = views.NewEditModel(m.entries, m.cfg)

			sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
			m.home, _ = m.home.Update(sizeMsg)
			m.backup, _ = m.backup.Update(sizeMsg)
			m.setup, _ = m.setup.Update(sizeMsg)
			m.edit, _ = m.edit.Update(sizeMsg)
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
		// Note: m.browse is preserved so directory traversal position is retained!

		sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
		m.home, _ = m.home.Update(sizeMsg)
		m.backup, _ = m.backup.Update(sizeMsg)
		m.setup, _ = m.setup.Update(sizeMsg)
		m.edit, _ = m.edit.Update(sizeMsg)
		m.browse, _ = m.browse.Update(sizeMsg)

		if m.currentView == views.ViewBrowse {
			return m, m.browse.Refresh()
		}
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
	case views.ViewBrowse:
		m.browse, cmd = m.browse.Update(msg)
		cmds = append(cmds, cmd)
	case views.ViewAddConfig:
		m.addConfig, cmd = m.addConfig.Update(msg)
		cmds = append(cmds, cmd)
	case views.ViewWizard:
		m.wizard, cmd = m.wizard.Update(msg)
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
	case views.ViewBrowse:
		return m.browse.View()
	case views.ViewAddConfig:
		return m.addConfig.View()
	case views.ViewWizard:
		return m.wizard.View()
	default:
		return m.home.View()
	}
}
