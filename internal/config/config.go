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

	cfg := *DefaultConfig()
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if len(cfg.Dotfiles) == 0 {
		cfg.Dotfiles = DefaultDotfiles()
	}

	return &cfg, nil
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

	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
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

// LoadFromPath loads a Config from an explicit file path.
func LoadFromPath(configPath string) (*Config, error) {
	data, err := os.ReadFile(ExpandPath(configPath))
	if err != nil {
		return nil, err
	}
	cfg := *DefaultConfig()
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
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
