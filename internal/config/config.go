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
