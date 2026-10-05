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
			if !config.ConfigExists() {
				return runTUI(views.ViewWizard)
			}
			return runTUI(views.ViewHome)
		},
	}

	rootCmd.PersistentFlags().StringVarP(&editorFlag, "editor", "e", "", "Editor command to use (overrides config and $EDITOR)")
	rootCmd.PersistentFlags().StringVarP(&repoFlag, "repo", "r", "", "Path to dotfiles repository (overrides config)")

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive setup wizard to initialize dots and your repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(views.ViewWizard)
		},
	}

	addCmd := &cobra.Command{
		Use:   "add [path]",
		Short: "Add and track a new configuration interactively",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := ""
			if len(args) > 0 {
				path = args[0]
			}
			return runTUIWithPath(views.ViewAddConfig, path)
		},
	}

	backupCmd := &cobra.Command{
		Use:   "backup",
		Short: "Selectively back up system configs into the repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(views.ViewBackup)
		},
	}

	setupCmd := &cobra.Command{
		Use:   "setup [name]",
		Short: "Selectively link repository configs to your system",
		Long: `Selectively link repository configs to your system.

When [name] is provided, links the specified config directly without launching the TUI.
When [name] is omitted, launches the interactive TUI setup menu.`,
		ValidArgsFunction: configArgsFunction,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runTUI(views.ViewSetup)
			}
			return runDirectSetup(args[0])
		},
	}

	editCmd := &cobra.Command{
		Use:   "edit [name]",
		Short: "Open dotfile configs in your editor",
		Long: `Open dotfile configs in your editor.

When [name] is provided, opens the specified config directly in your editor without launching the TUI.
When [name] is omitted, launches the interactive TUI edit selection menu.`,
		ValidArgsFunction: configArgsFunction,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runTUI(views.ViewEdit)
			}
			return runDirectEdit(args[0])
		},
	}

	var (
		removeModeFlag    string
		removeSymlinkFlag bool
		removeAllFlag     bool
		removeForceFlag   bool
		removeCommitFlag  bool
		removeMessageFlag string
		removePushFlag    bool
	)

	removeCmd := &cobra.Command{
		Use:     "remove [name]",
		Aliases: []string{"rm", "delete", "unlink"},
		Short:   "Remove configurations (unlinks symlink or deletes from repo)",
		Long: `Remove configurations from your system and/or dotfiles repository.

When [name] is provided without flags, launches an interactive terminal prompt to choose
removal mode and git commit options without opening the full TUI.
When [name] is omitted, launches the interactive full TUI remove menu.

Modes:
  • symlink: Removes only system symlinks. Preserves repository files and keeps config tracked in dots.
  • all:     Removes system symlinks, deletes files from repository, and untracks from dots.`,
		ValidArgsFunction: configArgsFunction,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runTUI(views.ViewRemove)
			}
			return runDirectRemove(args[0], removeFlags{
				mode:     removeModeFlag,
				symlinks: removeSymlinkFlag,
				all:      removeAllFlag,
				force:    removeForceFlag,
				commit:   removeCommitFlag,
				message:  removeMessageFlag,
				push:     removePushFlag,
			})
		},
	}

	removeCmd.Flags().StringVarP(&removeModeFlag, "mode", "m", "", "Removal mode: 'symlink' (unlink only) or 'all' (repo & symlink)")
	removeCmd.Flags().BoolVarP(&removeSymlinkFlag, "symlinks-only", "s", false, "Remove only system symlinks (shortcut for --mode=symlink)")
	removeCmd.Flags().BoolVarP(&removeAllFlag, "all", "a", false, "Remove repository files and system symlinks (shortcut for --mode=all)")
	removeCmd.Flags().BoolVarP(&removeForceFlag, "force", "f", false, "Skip confirmation prompts")
	removeCmd.Flags().BoolVarP(&removeCommitFlag, "commit", "c", false, "Commit removal to git repository")
	removeCmd.Flags().StringVarP(&removeMessageFlag, "message", "M", "", "Git commit message")
	removeCmd.Flags().BoolVarP(&removePushFlag, "push", "p", false, "Push git commit to remote")

	refreshCmd := &cobra.Command{
		Use:   "refresh",
		Short: "Scan repository and synchronize new dotfile configurations",
		Long: `Scan repository and synchronize new dotfile configurations into the dots database.

Automatically detects newly added files and directories in your dotfiles repository
(as well as configs defined in repo-level config.toml) without needing to re-run 'dots init'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDirectRefresh()
		},
	}

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(refreshCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func configArgsFunction(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
}

func loadAppConfig() (*config.Config, error) {
	if !config.ConfigExists() {
		return nil, fmt.Errorf("dots is not initialized. Please run 'dots init' to set up your dotfiles repository first")
	}
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
	if cfg.RepoPath == "" {
		return nil, fmt.Errorf("no repository path configured in dots. Please run 'dots init' to set up your repository")
	}
	if config.ConfigExists() && cfg.RepoPath != "" {
		_, _ = dotfile.RefreshDatabase(cfg)
	}
	return cfg, nil
}

func runTUI(initialView views.ViewType) error {
	return runTUIWithPath(initialView, "")
}

func runTUIWithPath(initialView views.ViewType, path string) error {
	var cfg *config.Config
	var err error
	if initialView == views.ViewWizard && !config.ConfigExists() {
		cfg = config.DefaultConfig()
		if editorFlag != "" {
			cfg.Editor = editorFlag
		}
		if repoFlag != "" {
			cfg.RepoPath = repoFlag
		}
	} else {
		cfg, err = loadAppConfig()
		if err != nil {
			return err
		}
	}

	appModel := ui.NewAppWithPath(cfg, initialView, path)
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

	icon, iconColor := theme.FileIconStyled(targetEntry.Name, targetEntry.IsDir)
	coloredIcon := lipgloss.NewStyle().Foreground(iconColor).Bold(true).Render(icon + " ")

	fmt.Printf("%sOpening %s with %s (%s)...\n",
		coloredIcon,
		lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(targetEntry.Name),
		theme.SubtitleStyle.Render(ed),
		theme.MutedStyle.Render(targetPath),
	)

	return editor.OpenAndWait(ed, targetPath)
}

func runDirectSetup(name string) error {
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
		msg += fmt.Sprintf("\nUsage: %s\n", theme.SubtitleStyle.Render("dots setup <name>"))
		return fmt.Errorf("%s", msg)
	}

	backedUp, backupPath, err := dotfile.Setup(*targetEntry, "")
	if err != nil {
		return fmt.Errorf("failed to setup %s: %w", targetEntry.Name, err)
	}

	icon, iconColor := theme.FileIconStyled(targetEntry.Name, targetEntry.IsDir)
	coloredIcon := lipgloss.NewStyle().Foreground(iconColor).Bold(true).Render(icon + " ")

	fmt.Printf("%s%s %s\n",
		coloredIcon,
		lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(targetEntry.Name),
		theme.SuccessStyle.Render("✓ Linked"),
	)
	fmt.Printf("  • System: %s\n", theme.MutedStyle.Render(targetEntry.ResolveSystemPath()))
	fmt.Printf("  • Repo:   %s\n", theme.MutedStyle.Render(targetEntry.AbsRepoPath()))
	if backedUp {
		fmt.Printf("  • Backup: %s\n", theme.WarningStyle.Render(backupPath))
	}
	return nil
}

type removeFlags struct {
	mode     string
	symlinks bool
	all      bool
	force    bool
	commit   bool
	message  string
	push     bool
}

func runDirectRemove(name string, flags removeFlags) error {
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
		msg += fmt.Sprintf("\nUsage: %s\n", theme.SubtitleStyle.Render("dots remove <name>"))
		return fmt.Errorf("%s", msg)
	}

	// Determine mode if specified via flags
	var mode *dotfile.RemoveMode
	if flags.symlinks || strings.EqualFold(flags.mode, "symlink") || strings.EqualFold(flags.mode, "unlink") {
		m := dotfile.RemoveModeSymlink
		mode = &m
	} else if flags.all || strings.EqualFold(flags.mode, "all") || strings.EqualFold(flags.mode, "repo") || strings.EqualFold(flags.mode, "purge") {
		m := dotfile.RemoveModeAll
		mode = &m
	}

	if mode != nil {
		if *mode == dotfile.RemoveModeAll && !flags.force {
			fmt.Printf("%s %s\n",
				theme.ErrorStyle.Render("⚠️  WARNING:"),
				theme.WarningStyle.Render(fmt.Sprintf("Permanently delete %s from repository and unlink %s?", targetEntry.AbsRepoPath(), targetEntry.ResolveSystemPath())),
			)
			fmt.Print("Proceed? [y/N]: ")
			var answer string
			_, _ = fmt.Scanln(&answer)
			if !strings.EqualFold(answer, "y") && !strings.EqualFold(answer, "yes") {
				fmt.Println(theme.MutedStyle.Render("Removal cancelled."))
				return nil
			}
		}

		res, err := dotfile.Remove(*targetEntry, *mode, cfg, flags.commit, flags.message, flags.push)
		if err != nil {
			return err
		}

		fmt.Println(theme.TitleStyle.Foreground(theme.Success).Render(fmt.Sprintf("✓ Removed %s successfully", targetEntry.Name)))
		if res.SymlinkRemoved {
			fmt.Println(theme.SuccessStyle.Render("  ✓ ") + theme.MutedStyle.Render(fmt.Sprintf("Unlinked system symlink: %s", targetEntry.ResolveSystemPath())))
		} else if res.SymlinkSkipped {
			fmt.Println(theme.WarningStyle.Render("  · ") + theme.MutedStyle.Render(res.Message))
		}
		if res.RepoFilesRemoved {
			fmt.Println(theme.SuccessStyle.Render("  ✓ ") + theme.MutedStyle.Render(fmt.Sprintf("Deleted repository files: %s", targetEntry.AbsRepoPath())))
		}
		if res.ConfigRemoved {
			fmt.Println(theme.SuccessStyle.Render("  ✓ ") + theme.MutedStyle.Render("Untracked from dots configuration"))
		}
		if res.GitCommitted {
			fmt.Println(theme.SuccessStyle.Render("  ✓ ") + theme.MutedStyle.Render("Committed deletion to git"))
		}
		if res.GitPushed {
			fmt.Println(theme.SuccessStyle.Render("  ✓ ") + theme.MutedStyle.Render("Pushed changes to remote repository"))
		}
		return nil
	}

	// Interactive CLI Bubble Tea prompt
	cliModel := views.NewCLIRemoveModel(*targetEntry, cfg)
	p := tea.NewProgram(cliModel)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	if fm, ok := finalModel.(views.CLIRemoveModel); ok && fm.Cancelled {
		return nil
	}
	return nil
}

func runDirectRefresh() error {
	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}

	fmt.Printf("%s Scanning repository: %s...\n\n",
		theme.SubtitleStyle.Render("⚡"),
		theme.MutedStyle.Render(cfg.RepoPath),
	)

	newlyAdded, updated := dotfile.RefreshDatabase(cfg)
	if !updated || len(newlyAdded) == 0 {
		fmt.Printf("%s Dots database is up to date (%d configurations tracked).\n",
			theme.SuccessStyle.Render("✓"),
			len(cfg.Dotfiles),
		)
		return nil
	}

	fmt.Printf("%s Discovered and added %d new configuration(s):\n",
		theme.SuccessStyle.Render("✓"),
		len(newlyAdded),
	)
	for _, spec := range newlyAdded {
		icon, iconColor := theme.FileIconStyled(spec.Name, spec.IsDir)
		coloredIcon := lipgloss.NewStyle().Foreground(iconColor).Bold(true).Render(icon + " ")
		fmt.Printf("  • %s%s (%s)\n",
			coloredIcon,
			lipgloss.NewStyle().Bold(true).Foreground(theme.Primary).Render(spec.Name),
			theme.MutedStyle.Render(spec.SystemPath),
		)
	}
	fmt.Printf("\n%s Database updated and sorted alphabetically.\n", theme.MutedStyle.Render("→"))
	return nil
}

