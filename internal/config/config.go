package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Editor   string        `toml:"editor"`
	RepoPath string        `toml:"repo_path"`
	Git      GitConfig     `toml:"git"`
	Dotfiles []DotfileSpec `toml:"dotfiles"`
}

type GitConfig struct {
	AutoCommit   bool   `toml:"auto_commit"`
	AutoPush     bool   `toml:"auto_push"`
	CommitPrefix string `toml:"commit_prefix"`
}

type DotfileSpec struct {
	Name       string   `toml:"name"`
	RepoPath   string   `toml:"repo_path"`
	SystemPath string   `toml:"system_path"`
	AltPaths   []string `toml:"alt_paths"`
	Method     string   `toml:"method"`
	IsDir      bool     `toml:"is_dir"`
}

func DefaultConfig() *Config {
	return &Config{
		Editor:   "nvim",
		RepoPath: "~/Documents/dotfiles",
		Git: GitConfig{
			AutoCommit: false,
			AutoPush:   false,
		},
		Dotfiles: DefaultDotfiles(),
	}
}

func EnsureConfigDir() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configDir := filepath.Join(home, ".config", "dots")
	return os.MkdirAll(configDir, 0755)
}

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	} else if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	}
	return path
}

// CollapsePath replaces the user's home directory prefix with "~".
func CollapsePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	cleaned := filepath.Clean(path)
	if cleaned == home {
		return "~"
	}
	if strings.HasPrefix(cleaned, home+string(filepath.Separator)) {
		return "~" + cleaned[len(home):]
	}
	return path
}

// IsRemoteRepoInput returns true if input looks like a git URL or GitHub shorthand.
func IsRemoteRepoInput(input string) bool {
	t := strings.TrimSpace(input)
	if t == "" {
		return false
	}
	if strings.HasPrefix(t, "https://") || strings.HasPrefix(t, "http://") ||
		strings.HasPrefix(t, "git@") || strings.HasPrefix(t, "ssh://") {
		return true
	}
	if !strings.HasPrefix(t, "~") && !strings.HasPrefix(t, "/") && !strings.HasPrefix(t, ".") {
		parts := strings.Split(t, "/")
		if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
			return true
		}
	}
	return false
}

// NormalizeRepoPath ensures repo path is a valid local path (e.g. ~/dotfiles),
// resolving remote URLs or shorthand if accidentally saved.
func NormalizeRepoPath(path string) string {
	t := strings.TrimSpace(path)
	if t == "" {
		return "~/dotfiles"
	}
	if IsRemoteRepoInput(t) {
		clean := strings.TrimSuffix(t, ".git")
		parts := strings.Split(clean, "/")
		base := parts[len(parts)-1]
		if base == "" {
			base = "dotfiles"
		}
		return filepath.Join("~", base)
	}
	return CollapsePath(t)
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	configPath := filepath.Join(home, ".config", "dots", "config.toml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultConfig(), nil
		}
		return nil, err
	}

	var raw map[string]any
	_ = toml.Unmarshal(data, &raw)
	_, hasDotfiles := raw["dotfiles"]

	cfg := Config{
		Editor:   "nvim",
		RepoPath: "~/Documents/dotfiles",
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if !hasDotfiles {
		cfg.Dotfiles = DefaultDotfiles()
	}
	cfg.RepoPath = NormalizeRepoPath(cfg.RepoPath)

	return &cfg, nil
}

// SaveToPath serializes and writes the configuration to a specific file path.
func SaveToPath(cfg *Config, targetPath string) error {
	cfg.RepoPath = NormalizeRepoPath(cfg.RepoPath)
	expanded := ExpandPath(targetPath)
	if err := os.MkdirAll(filepath.Dir(expanded), 0755); err != nil {
		return err
	}
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(expanded, data, 0644)
}

func Save(cfg *Config) error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, ".config", "dots", "config.toml")
	return SaveToPath(cfg, configPath)
}

// ConfigExists reports whether ~/.config/dots/config.toml already exists.
func ConfigExists() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	configPath := filepath.Join(home, ".config", "dots", "config.toml")
	_, err = os.Stat(configPath)
	return err == nil
}

// AddDotfile adds a spec to cfg.Dotfiles if not already present by name, or updates it.
func (c *Config) AddDotfile(spec DotfileSpec) bool {
	for i, d := range c.Dotfiles {
		if strings.EqualFold(d.Name, spec.Name) {
			c.Dotfiles[i] = spec
			return false // replaced existing
		}
	}
	c.Dotfiles = append(c.Dotfiles, spec)
	return true // added new
}

// RemoveDotfile removes a spec from cfg.Dotfiles by name (case-insensitive).
// Returns true if an entry was found and removed, false otherwise.
func (c *Config) RemoveDotfile(name string) bool {
	for i, d := range c.Dotfiles {
		if strings.EqualFold(d.Name, name) {
			c.Dotfiles = append(c.Dotfiles[:i], c.Dotfiles[i+1:]...)
			return true
		}
	}
	return false
}

// LoadFromPath loads a Config from an explicit file path.
func LoadFromPath(configPath string) (*Config, error) {
	data, err := os.ReadFile(ExpandPath(configPath))
	if err != nil {
		return nil, err
	}
	var raw map[string]any
	_ = toml.Unmarshal(data, &raw)
	_, hasDotfiles := raw["dotfiles"]

	cfg := Config{
		Editor:   "nvim",
		RepoPath: "~/Documents/dotfiles",
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if !hasDotfiles {
		cfg.Dotfiles = DefaultDotfiles()
	}
	cfg.RepoPath = NormalizeRepoPath(cfg.RepoPath)
	return &cfg, nil
}

// FindRepoConfigFile checks if a dotfiles repository contains a dots config file.
// Common locations checked:
// 1. dots/config.toml
// 2. .config/dots/config.toml
// 3. config.toml
// 4. .dots.toml
func FindRepoConfigFile(repoPath string) string {
	abs := ExpandPath(repoPath)
	candidates := []string{
		filepath.Join(abs, "dots", "config.toml"),
		filepath.Join(abs, ".config", "dots", "config.toml"),
		filepath.Join(abs, "config.toml"),
		filepath.Join(abs, ".dots.toml"),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c
		}
	}
	return ""
}
