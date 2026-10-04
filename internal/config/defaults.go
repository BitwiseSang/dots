package config

func DefaultDotfiles() []DotfileSpec {
	return []DotfileSpec{
		{Name: "ghostty", RepoPath: "ghostty/config.ghostty", SystemPath: "~/.config/ghostty/config", AltPaths: []string{"~/.config/ghostty/config.ghostty"}, Method: "copy", IsDir: false},
		{Name: "kitty", RepoPath: "kitty/kitty.conf", SystemPath: "~/.config/kitty/kitty.conf", Method: "copy", IsDir: false},
		{Name: "tmux", RepoPath: "tmux/tmux.conf", SystemPath: "~/.tmux.conf", Method: "copy", IsDir: false},
		{Name: "clang-format", RepoPath: "clang-format/.clang-format", SystemPath: "~/.clang-format", Method: "copy", IsDir: false},
		{Name: "aria2", RepoPath: "aria2/aria2.conf", SystemPath: "~/.config/aria2/aria2.conf", AltPaths: []string{"~/aria2.conf"}, Method: "copy", IsDir: false},
		{Name: "nvim", RepoPath: "nvim", SystemPath: "~/.config/nvim", Method: "rsync", IsDir: true},
		{Name: "doom", RepoPath: "doom", SystemPath: "~/.config/doom", Method: "rsync", IsDir: true},
		{Name: "fish", RepoPath: "fish", SystemPath: "~/.config/fish", Method: "rsync", IsDir: true},
	}
}
