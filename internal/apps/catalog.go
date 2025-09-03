// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

package apps

import (
	"os/exec"

	"github.com/janderssonse/karei/internal/domain"
)

// App represents an application that can be installed.
type App struct {
	Name        string
	Group       string
	Description string
	Method      domain.InstallMethod
	Source      string
	PostInstall func() error
}

// Apps contains the catalog of available applications.
var Apps = map[string]App{ //nolint:gochecknoglobals
	// Development Tools
	"vscode": {
		Name:        "Visual Studio Code",
		Group:       "development",
		Description: "Code editor",
		Method:      domain.MethodDEB,
		Source:      "https://code.visualstudio.com/sha/download?build=stable&os=linux-deb-x64",
	},
	"cursor": {
		Name:        "Cursor",
		Group:       "development",
		Description: "AI-powered code editor",
		Method:      domain.MethodDEB,
		Source:      "https://download.cursor.sh/linux/appImage/x64",
	},
	"zed": {
		Name:        "Zed",
		Group:       "development",
		Description: "Lightning-fast editor",
		Method:      domain.MethodFlatpak,
		Source:      "dev.zed.Zed",
	},
	"windsurf": {
		Name:        "Windsurf",
		Group:       "development",
		Description: "AI development environment",
		Method:      domain.MethodDEB,
		Source:      "https://windsurf-stable.codeiumdata.com/wVxQEIWkwPUEAGf3/windsurf-linux-x64-1.0.6.deb",
	},
	"rubymine": {
		Name:        "RubyMine",
		Group:       "development",
		Description: "Ruby IDE",
		Method:      domain.MethodFlatpak,
		Source:      "com.jetbrains.RubyMine",
	},
	"mise": {
		Name:        "mise",
		Group:       "development",
		Description: "Fast polyglot tool version manager (asdf replacement)",
		Method:      domain.MethodGitHubBinary,
		Source:      "https://github.com/jdx/mise/releases/latest/download/mise-v2025.8.7-linux-x64",
	},
	// Rust Development Tools
	"rust": {
		Name:        "rust",
		Group:       "rustlang",
		Description: "Rust programming language",
		Method:      domain.MethodMise,
		Source:      "rust",
	},
	"cargo-audit": {
		Name:        "cargo-audit",
		Group:       "rustlang",
		Description: "Security vulnerability scanner",
		Method:      domain.MethodMise,
		Source:      "cargo-audit",
	},
	"cargo-watch": {
		Name:        "cargo-watch",
		Group:       "rustlang",
		Description: "Auto-rebuild on file changes",
		Method:      domain.MethodMise,
		Source:      "cargo-watch",
	},
	"cargo-edit": {
		Name:        "cargo-edit",
		Group:       "rustlang",
		Description: "Add/remove dependencies from CLI",
		Method:      domain.MethodMise,
		Source:      "cargo-edit",
	},
	"cargo-expand": {
		Name:        "cargo-expand",
		Group:       "rustlang",
		Description: "Show macro expansions",
		Method:      domain.MethodMise,
		Source:      "cargo-expand",
	},
	"cargo-tarpaulin": {
		Name:        "cargo-tarpaulin",
		Group:       "rustlang",
		Description: "Code coverage tool",
		Method:      domain.MethodMise,
		Source:      "cargo-tarpaulin",
	},
	"cargo-nextest": {
		Name:        "cargo-nextest",
		Group:       "rustlang",
		Description: "Next-generation test runner",
		Method:      domain.MethodMise,
		Source:      "cargo-nextest",
	},
	"cargo-deny": {
		Name:        "cargo-deny",
		Group:       "rustlang",
		Description: "Dependency policy checker",
		Method:      domain.MethodMise,
		Source:      "cargo-deny",
	},
	"cargo-bloat": {
		Name:        "cargo-bloat",
		Group:       "rustlang",
		Description: "Binary size analyzer",
		Method:      domain.MethodMise,
		Source:      "cargo-bloat",
	},
	"cargo-outdated": {
		Name:        "cargo-outdated",
		Group:       "rustlang",
		Description: "Check for outdated dependencies",
		Method:      domain.MethodMise,
		Source:      "cargo-outdated",
	},
	"cargo-cross": {
		Name:        "cargo-cross",
		Group:       "rustlang",
		Description: "Zero-setup cross compilation",
		Method:      domain.MethodMise,
		Source:      "cargo-cross",
	},
	"cargo-flamegraph": {
		Name:        "cargo-flamegraph",
		Group:       "rustlang",
		Description: "Profiling flame graphs",
		Method:      domain.MethodMise,
		Source:      "cargo-flamegraph",
	},
	"cargo-geiger": {
		Name:        "cargo-geiger",
		Group:       "rustlang",
		Description: "Detect unsafe code usage",
		Method:      domain.MethodMise,
		Source:      "cargo-geiger",
	},
	// Python Development Tools
	"python": {
		Name:        "python",
		Group:       "pythonlang",
		Description: "Python programming language",
		Method:      domain.MethodMise,
		Source:      "python",
	},
	"pipx": {
		Name:        "pipx",
		Group:       "pythonlang",
		Description: "Install Python applications in isolated environments",
		Method:      domain.MethodMise,
		Source:      "pipx",
	},
	"poetry": {
		Name:        "poetry",
		Group:       "pythonlang",
		Description: "Modern dependency management and packaging",
		Method:      domain.MethodMise,
		Source:      "poetry",
	},
	"black": {
		Name:        "black",
		Group:       "pythonlang",
		Description: "Uncompromising code formatter",
		Method:      domain.MethodMise,
		Source:      "black",
	},
	"flake8": {
		Name:        "flake8",
		Group:       "pythonlang",
		Description: "Style guide enforcement and linting",
		Method:      domain.MethodMise,
		Source:      "flake8",
	},
	"mypy": {
		Name:        "mypy",
		Group:       "pythonlang",
		Description: "Static type checker for Python",
		Method:      domain.MethodMise,
		Source:      "mypy",
	},
	"pytest": {
		Name:        "pytest",
		Group:       "pythonlang",
		Description: "Modern testing framework",
		Method:      domain.MethodMise,
		Source:      "pytest",
	},
	"isort": {
		Name:        "isort",
		Group:       "pythonlang",
		Description: "Import statement sorter",
		Method:      domain.MethodMise,
		Source:      "isort",
	},
	"bandit": {
		Name:        "bandit",
		Group:       "pythonlang",
		Description: "Security linter for common security issues",
		Method:      domain.MethodMise,
		Source:      "bandit",
	},
	"ruff": {
		Name:        "ruff",
		Group:       "pythonlang",
		Description: "Extremely fast Python linter and formatter",
		Method:      domain.MethodMise,
		Source:      "ruff",
	},
	"pre-commit": {
		Name:        "pre-commit",
		Group:       "pythonlang",
		Description: "Git hooks for code quality",
		Method:      domain.MethodMise,
		Source:      "pre-commit",
	},
	"pyenv": {
		Name:        "pyenv",
		Group:       "pythonlang",
		Description: "Python version management",
		Method:      domain.MethodMise,
		Source:      "pyenv",
	},
	"pip-tools": {
		Name:        "pip-tools",
		Group:       "pythonlang",
		Description: "Requirements management with version pinning",
		Method:      domain.MethodMise,
		Source:      "pip-tools",
	},
	"coverage": {
		Name:        "coverage",
		Group:       "pythonlang",
		Description: "Code coverage measurement",
		Method:      domain.MethodMise,
		Source:      "coverage",
	},
	"ipython": {
		Name:        "ipython",
		Group:       "pythonlang",
		Description: "Enhanced interactive Python shell",
		Method:      domain.MethodMise,
		Source:      "ipython",
	},
	"jupyter": {
		Name:        "jupyter",
		Group:       "pythonlang",
		Description: "Interactive notebooks for data science",
		Method:      domain.MethodMise,
		Source:      "jupyter",
	},
	"sphinx": {
		Name:        "sphinx",
		Group:       "pythonlang",
		Description: "Documentation generator",
		Method:      domain.MethodMise,
		Source:      "sphinx",
	},

	// Browsers
	"chrome": {
		Name:        "Google Chrome",
		Group:       "browsers",
		Description: "Web browser",
		Method:      domain.MethodDEB,
		Source:      "https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb",
		PostInstall: func() error {
			return exec.Command("xdg-settings", "set", "default-web-browser", "google-chrome.desktop").Run()
		},
	},
	"brave": {
		Name:        "Brave Browser",
		Group:       "browsers",
		Description: "Privacy-focused browser",
		Method:      domain.MethodFlatpak,
		Source:      "com.brave.Browser",
	},
	"firefox": {
		Name:        "Firefox",
		Group:       "browsers",
		Description: "Open source web browser",
		Method:      domain.MethodFlatpak,
		Source:      "org.mozilla.firefox",
	},

	// Communication
	"signal": {
		Name:        "Signal",
		Group:       "communication",
		Description: "Secure messaging",
		Method:      domain.MethodFlatpak,
		Source:      "org.signal.Signal",
	},
	"discord": {
		Name:        "Discord",
		Group:       "communication",
		Description: "Chat platform",
		Method:      domain.MethodFlatpak,
		Source:      "com.discordapp.Discord",
	},
	"zoom": {
		Name:        "Zoom",
		Group:       "communication",
		Description: "Video conferencing",
		Method:      domain.MethodFlatpak,
		Source:      "us.zoom.Zoom",
	},

	// Media
	"vlc": {
		Name:        "VLC Media Player",
		Group:       "media",
		Description: "Media player",
		Method:      domain.MethodAPT,
		Source:      "vlc",
	},
	"spotify": {
		Name:        "Spotify",
		Group:       "media",
		Description: "Music streaming",
		Method:      domain.MethodFlatpak,
		Source:      "com.spotify.Client",
	},
	"obs": {
		Name:        "OBS Studio",
		Group:       "media",
		Description: "Video recording/streaming",
		Method:      domain.MethodFlatpak,
		Source:      "com.obsproject.Studio",
	},
	"audacity": {
		Name:        "Audacity",
		Group:       "media",
		Description: "Audio editor",
		Method:      domain.MethodFlatpak,
		Source:      "org.audacityteam.Audacity",
	},

	// Productivity
	"obsidian": {
		Name:        "Obsidian",
		Group:       "productivity",
		Description: "Note taking",
		Method:      domain.MethodFlatpak,
		Source:      "md.obsidian.Obsidian",
	},
	"libreoffice": {
		Name:        "LibreOffice",
		Group:       "productivity",
		Description: "Office suite",
		Method:      domain.MethodAPT,
		Source:      "libreoffice",
	},
	"dropbox": {
		Name:        "Dropbox",
		Group:       "productivity",
		Description: "Cloud storage",
		Method:      domain.MethodFlatpak,
		Source:      "com.dropbox.Client",
	},
	"1password": {
		Name:        "1Password",
		Group:       "productivity",
		Description: "Password manager",
		Method:      domain.MethodFlatpak,
		Source:      "com.1password.1Password",
	},

	// Graphics
	"gimp": {
		Name:        "GIMP",
		Group:       "graphics",
		Description: "Image editor",
		Method:      domain.MethodAPT,
		Source:      "gimp",
	},
	"pinta": {
		Name:        "Pinta",
		Group:       "graphics",
		Description: "Simple image editor",
		Method:      domain.MethodAPT,
		Source:      "pinta",
	},

	// Utilities
	"flameshot": {
		Name:        "Flameshot",
		Group:       "utilities",
		Description: "Screenshot tool",
		Method:      domain.MethodAPT,
		Source:      "flameshot",
	},
	"virtualbox": {
		Name:        "VirtualBox",
		Group:       "utilities",
		Description: "Virtual machines",
		Method:      domain.MethodAPT,
		Source:      "virtualbox",
	},

	// Gaming
	"steam": {
		Name:        "Steam",
		Group:       "gaming",
		Description: "Gaming platform",
		Method:      domain.MethodFlatpak,
		Source:      "com.valvesoftware.Steam",
	},
	"heroic": {
		Name:        "Heroic Games Launcher",
		Group:       "gaming",
		Description: "Open source Epic Games and GOG launcher",
		Method:      domain.MethodFlatpak,
		Source:      "com.heroicgameslauncher.hgl",
	},
	"minecraft": {
		Name:        "Minecraft",
		Group:       "gaming",
		Description: "Block building game",
		Method:      domain.MethodFlatpak,
		Source:      "com.mojang.Minecraft",
	},
	"retroarch": {
		Name:        "RetroArch",
		Group:       "gaming",
		Description: "Retro gaming emulator",
		Method:      domain.MethodFlatpak,
		Source:      "org.libretro.RetroArch",
	},

	// Terminal Tools
	"gh": {
		Name:        "GitHub CLI",
		Group:       "terminal",
		Description: "GitHub command line",
		Method:      domain.MethodMise,
		Source:      "gh",
	},
	"lazygit": {
		Name:        "Lazygit",
		Group:       "terminal",
		Description: "Git TUI",
		Method:      domain.MethodMise,
		Source:      "lazygit",
	},
	"lazydocker": {
		Name:        "Lazydocker",
		Group:       "terminal",
		Description: "Docker TUI",
		Method:      domain.MethodMise,
		Source:      "lazydocker",
	},
	"btop": {
		Name:        "btop",
		Group:       "terminal",
		Description: "System monitor",
		Method:      domain.MethodMise,
		Source:      "btop",
	},
	"neovim": {
		Name:        "Neovim",
		Group:       "terminal",
		Description: "Text editor",
		Method:      domain.MethodMise,
		Source:      "neovim",
	},
	"zellij": {
		Name:        "Zellij",
		Group:       "terminal",
		Description: "Terminal multiplexer",
		Method:      domain.MethodMise,
		Source:      "zellij",
	},
	"starship": {
		Name:        "Starship",
		Group:       "terminal",
		Description: "Shell prompt",
		Method:      domain.MethodMise,
		Source:      "starship",
	},
	"fish": {
		Name:        "Fish Shell",
		Group:       "terminal",
		Description: "Interactive shell",
		Method:      domain.MethodAPT,
		Source:      "fish",
	},
	"fzf": {
		Name:        "fzf",
		Group:       "terminal",
		Description: "Fuzzy finder",
		Method:      domain.MethodMise,
		Source:      "fzf",
	},
	"ripgrep": {
		Name:        "ripgrep",
		Group:       "terminal",
		Description: "Fast grep",
		Method:      domain.MethodMise,
		Source:      "ripgrep",
	},
	"bat": {
		Name:        "bat",
		Group:       "terminal",
		Description: "Better cat",
		Method:      domain.MethodMise,
		Source:      "bat",
	},
	"eza": {
		Name:        "eza",
		Group:       "terminal",
		Description: "Better ls",
		Method:      domain.MethodMise,
		Source:      "eza",
	},
	"zoxide": {
		Name:        "zoxide",
		Group:       "terminal",
		Description: "Smart cd",
		Method:      domain.MethodAPT,
		Source:      "zoxide",
	},
	"delta": {
		Name:        "delta",
		Group:       "terminal",
		Description: "Git diff pager",
		Method:      domain.MethodMise,
		Source:      "delta",
	},
	"fd": {
		Name:        "fd",
		Group:       "terminal",
		Description: "Fast find alternative",
		Method:      domain.MethodMise,
		Source:      "fd",
	},
	"hyperfine": {
		Name:        "hyperfine",
		Group:       "terminal",
		Description: "Command-line benchmarking",
		Method:      domain.MethodMise,
		Source:      "hyperfine",
	},
	"bottom": {
		Name:        "bottom",
		Group:       "terminal",
		Description: "System monitor (btm)",
		Method:      domain.MethodMise,
		Source:      "bottom",
	},

	// CLI Version Managers
	"aqua": {
		Name:        "Aqua",
		Group:       "development",
		Description: "Declarative CLI Version Manager",
		Method:      domain.MethodMise,
		Source:      "aqua",
	},

	// Linters and Security Tools
	"hadolint": {
		Name:        "Hadolint",
		Group:       "linters",
		Description: "Dockerfile linter",
		Method:      domain.MethodMise,
		Source:      "hadolint",
	},
	"trivy": {
		Name:        "Trivy",
		Group:       "linters",
		Description: "Vulnerability scanner",
		Method:      domain.MethodMise,
		Source:      "trivy",
	},
	"gitleaks": {
		Name:        "Gitleaks",
		Group:       "linters",
		Description: "Git secrets scanner",
		Method:      domain.MethodMise,
		Source:      "gitleaks",
	},
	"yamlfmt": {
		Name:        "yamlfmt",
		Group:       "linters",
		Description: "YAML formatter",
		Method:      domain.MethodMise,
		Source:      "yamlfmt",
	},
	"taplo": {
		Name:        "Taplo",
		Group:       "linters",
		Description: "TOML formatter and linter",
		Method:      domain.MethodMise,
		Source:      "taplo",
	},
	"cosign": {
		Name:        "Cosign",
		Group:       "linters",
		Description: "Container signing tool",
		Method:      domain.MethodMise,
		Source:      "cosign",
	},
	"scorecard": {
		Name:        "Scorecard",
		Group:       "linters",
		Description: "Security scorecard for projects",
		Method:      domain.MethodMise,
		Source:      "scorecard",
	},
	"syft": {
		Name:        "Syft",
		Group:       "linters",
		Description: "SBOM generation tool",
		Method:      domain.MethodMise,
		Source:      "syft",
	},
	"actionlint": {
		Name:        "Actionlint",
		Group:       "linters",
		Description: "GitHub Actions workflow linter",
		Method:      domain.MethodMise,
		Source:      "actionlint",
	},
	"shellcheck": {
		Name:        "Shellcheck",
		Group:       "linters",
		Description: "Shell script linter",
		Method:      domain.MethodMise,
		Source:      "shellcheck",
	},
	"shfmt": {
		Name:        "shfmt",
		Group:       "linters",
		Description: "Shell script formatter",
		Method:      domain.MethodMise,
		Source:      "shfmt",
	},
	"dockle": {
		Name:        "Dockle",
		Group:       "linters",
		Description: "Container security scanner",
		Method:      domain.MethodMise,
		Source:      "dockle",
	},

	// Go Development Tools
	"go": {
		Name:        "Go",
		Group:       "golang",
		Description: "Go programming language",
		Method:      domain.MethodMise,
		Source:      "go",
	},
	"golangci-lint": {
		Name:        "golangci-lint",
		Group:       "golang",
		Description: "Go linter aggregator",
		Method:      domain.MethodMise,
		Source:      "golangci-lint",
	},
	"goreleaser": {
		Name:        "GoReleaser",
		Group:       "golang",
		Description: "Release automation for Go projects",
		Method:      domain.MethodMise,
		Source:      "goreleaser",
	},

	// Java Development Tools
	"java": {
		Name:        "Java",
		Group:       "javalang",
		Description: "Java programming language",
		Method:      domain.MethodMise,
		Source:      "java",
	},
	"maven": {
		Name:        "Maven",
		Group:       "javalang",
		Description: "Java build automation tool",
		Method:      domain.MethodMise,
		Source:      "maven",
	},
	"gradle": {
		Name:        "Gradle",
		Group:       "javalang",
		Description: "Java build automation tool",
		Method:      domain.MethodMise,
		Source:      "gradle",
	},
	"checkstyle": {
		Name:        "Checkstyle",
		Group:       "javalang",
		Description: "Java code style checker",
		Method:      domain.MethodMise,
		Source:      "checkstyle",
	},
	"pmd": {
		Name:        "PMD",
		Group:       "javalang",
		Description: "Java source code analyzer",
		Method:      domain.MethodGitHubJava,
		Source:      "pmd/pmd",
	},
	"spotbugs": {
		Name:        "SpotBugs",
		Group:       "javalang",
		Description: "Java static analysis tool",
		Method:      domain.MethodMise,
		Source:      "spotbugs",
	},
	"jmeter": {
		Name:        "JMeter",
		Group:       "javalang",
		Description: "Java performance testing tool",
		Method:      domain.MethodMise,
		Source:      "jmeter",
	},
	"visualvm": {
		Name:        "VisualVM",
		Group:       "javalang",
		Description: "Java profiling and monitoring tool",
		Method:      domain.MethodMise,
		Source:      "visualvm",
	},
	"kse": {
		Name:        "KeyStore Explorer",
		Group:       "javalang",
		Description: "Java keystore management tool",
		Method:      domain.MethodMise,
		Source:      "kse",
	},
	"jreleaser": {
		Name:        "jreleaser",
		Group:       "javalang",
		Description: "Release automation for Java projects",
		Method:      domain.MethodMise,
		Source:      "jreleaser",
	},

	// Missing apps from master branch
	"fastfetch": {
		Name:        "Fastfetch",
		Group:       "utilities",
		Description: "System information display",
		Method:      domain.MethodDEB,
		Source:      "https://github.com/fastfetch-cli/fastfetch/releases/latest/download/fastfetch-linux-amd64.deb",
	},
	"gnome-sushi": {
		Name:        "GNOME Sushi",
		Group:       "utilities",
		Description: "File preview with spacebar",
		Method:      domain.MethodAPT,
		Source:      "gnome-sushi",
	},
	"gnome-tweaks": {
		Name:        "GNOME Tweaks",
		Group:       "utilities",
		Description: "Advanced GNOME configuration",
		Method:      domain.MethodAPT,
		Source:      "gnome-tweaks",
	},
	"localsend": {
		Name:        "LocalSend",
		Group:       "utilities",
		Description: "Share files across devices",
		Method:      domain.MethodFlatpak,
		Source:      "org.localsend.localsend_app",
	},
	"wl-clipboard": {
		Name:        "wl-clipboard",
		Group:       "utilities",
		Description: "Wayland clipboard utilities",
		Method:      domain.MethodAPT,
		Source:      "wl-clipboard",
	},
	"xournalpp": {
		Name:        "Xournal++",
		Group:       "productivity",
		Description: "PDF annotation and note-taking",
		Method:      domain.MethodAPT,
		Source:      "xournalpp",
	},
	"zettlr": {
		Name:        "Zettlr",
		Group:       "productivity",
		Description: "Modern markdown editor for writers and researchers",
		Method:      domain.MethodFlatpak,
		Source:      "com.zettlr.Zettlr",
	},
}

// Groups defines application groups for bulk installation.
var Groups = map[string][]string{ //nolint:gochecknoglobals
	"development":   {"vscode", "cursor", "zed", "windsurf", "rubymine", "mise", "gh", "aqua"},
	"browsers":      {"chrome", "brave", "firefox"},
	"communication": {"signal", "discord", "zoom"},
	"media":         {"vlc", "spotify", "obs", "audacity"},
	"productivity":  {"obsidian", "libreoffice", "dropbox", "1password", "xournalpp", "zettlr"},
	"graphics":      {"gimp", "pinta"},
	"utilities":     {"flameshot", "virtualbox", "fastfetch", "gnome-sushi", "gnome-tweaks", "localsend", "wl-clipboard"},
	"gaming":        {"steam", "heroic", "minecraft", "retroarch"},
	"golang":        {"go", "golangci-lint", "goreleaser"},
	"javalang":      {"java", "maven", "gradle", "checkstyle", "pmd", "spotbugs", "jmeter", "visualvm", "kse", "jreleaser"},
	"rustlang":      {"rust", "cargo-audit", "cargo-watch", "cargo-edit", "cargo-expand", "cargo-tarpaulin", "cargo-nextest", "cargo-deny", "cargo-bloat", "cargo-outdated", "cargo-cross", "cargo-flamegraph", "cargo-geiger"},
	"pythonlang":    {"python", "pipx", "poetry", "black", "flake8", "mypy", "pytest", "isort", "bandit", "ruff", "pre-commit", "pyenv", "pip-tools", "coverage", "ipython", "jupyter", "sphinx"},
	"linters":       {"hadolint", "trivy", "gitleaks", "yamlfmt", "taplo", "cosign", "scorecard", "syft", "actionlint", "shellcheck", "shfmt", "dockle"},
	"terminal":      {"gh", "lazygit", "lazydocker", "btop", "neovim", "zellij", "starship", "fish", "fzf", "ripgrep", "bat", "eza", "zoxide", "delta", "fd", "hyperfine", "bottom"},
}

// Languages contains supported programming languages.
var Languages = map[string]string{ //nolint:gochecknoglobals
	"nodejs": "nodejs",
	"ruby":   "ruby",
	"elixir": "elixir",
}

// ListApps returns apps for a group, or all apps if group is empty.
func ListApps(group string) []App {
	var apps []App

	if group == "" {
		for _, app := range Apps {
			apps = append(apps, app)
		}

		return apps
	}

	appNames, exists := Groups[group]
	if !exists {
		return apps
	}

	for _, name := range appNames {
		if app, exists := Apps[name]; exists {
			apps = append(apps, app)
		}
	}

	return apps
}
