package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/BitwiseSang/dots/internal/config"
	"github.com/BitwiseSang/dots/internal/dotfile"
	"github.com/BitwiseSang/dots/internal/editor"
	"github.com/BitwiseSang/dots/internal/ui"
	"github.com/BitwiseSang/dots/internal/ui/theme"
	"github.com/BitwiseSang/dots/internal/ui/views"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	version = "0.1.0"

	editorFlag string
	repoFlag   string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "dots [command]",
		Short:   "dots — A beautiful TUI for managing your dotfiles",
		Long:    theme.Logo() + "\nA modern, selective dotfiles manager with backup, setup, editing, and diff previews.",
		Version: version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(views.ViewHome)
		},
	}

	rootCmd.PersistentFlags().StringVarP(&editorFlag, "editor", "e", "", "Editor command to use (overrides config and $EDITOR)")
	rootCmd.PersistentFlags().StringVarP(&repoFlag, "repo", "r", "", "Path to dotfiles repository (overrides config)")

	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Selectively back up system configs into the repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(views.ViewBackup)
		},
	}

	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Selectively link repository configs to your system",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(views.ViewSetup)
		},
	}

	editCmd := &cobra.Command{
		Use:   "edit [name]",
		Short: "Open dotfile configs in your editor",
		Long: `Open dotfile configs in your editor.

When [name] is provided, opens the specified config directly in your editor without launching the TUI.
When [name] is omitted, launches the interactive TUI edit selection menu.`,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) != 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			cfg, err := loadAppConfig()
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			entries := dotfile.LoadEntries(cfg)
			var names []string
			for _, e := range entries {
				names = append(names, e.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runTUI(views.ViewEdit)
			}
			return runDirectEdit(args[0])
		},
	}

	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(editCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func loadAppConfig() (*config.Config, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	if editorFlag != "" {
		cfg.Editor = editorFlag
	}
	if repoFlag != "" {
		cfg.RepoPath = repoFlag
	}
	return cfg, nil
}

func runTUI(initialView views.ViewType) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	appModel := ui.NewApp(cfg, initialView)
	p := tea.NewProgram(appModel, tea.WithAltScreen())

	_, err = p.Run()
	return err
}

func runDirectEdit(name string) error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	entries := dotfile.LoadEntries(cfg)
	var targetEntry *dotfile.Entry
	var available []string

	for i := range entries {
		available = append(available, entries[i].Name)
		if strings.EqualFold(entries[i].Name, name) {
			targetEntry = &entries[i]
		}
	}

	if targetEntry == nil {
		header := theme.TitleStyle.Render("Unknown dotfile: ") + theme.ErrorStyle.Render(name)
		msg := fmt.Sprintf("%s\n\nAvailable dotfiles:\n", header)
		for _, a := range available {
			msg += fmt.Sprintf("  • %s\n", theme.SelectedStyle.Render(a))
		}
		msg += fmt.Sprintf("\nUsage: %s\n", theme.SubtitleStyle.Render("dots edit <name>"))
		return fmt.Errorf("%s", msg)
	}

	ed := editor.Resolve(cfg.Editor)
	targetPath := targetEntry.EditPath()

	fmt.Printf("%s Opening %s with %s (%s)...\n",
		theme.SelectedStyle.Render("✏️ "),
		lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(targetEntry.Name),
		theme.SubtitleStyle.Render(ed),
		theme.MutedStyle.Render(targetPath),
	)

	return editor.OpenAndWait(ed, targetPath)
}
