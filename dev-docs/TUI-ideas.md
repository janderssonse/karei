# TUI Design Ideas - Bubble Tea Ecosystem

> **Approach**: Full TUI application using Bubble Tea + Huh + Lipgloss
> **Philosophy**: Transform karei into an interactive, stateful application with rich user interfaces

## Library Stack

- **Bubble Tea**: Core TUI framework for complex interactions
- **Huh**: High-level forms library for prompts and selections  
- **Lipgloss**: Styling and layout engine
- **Bubbles**: Pre-built components (lists, progress, spinners)
- **Glamour**: Markdown rendering for help/descriptions

## Migration Plan

### Phase 1: Foundation (Week 1-2)
- [x] Add Bubble Tea dependencies to go.mod
- [x] Create base TUI application structure
- [x] Implement simple welcome screen
- [x] Add basic navigation framework

### Phase 2: Core Screens (Week 3-4)
- [x] Application selection screen with Huh
- [x] Configuration screen with form components
- [x] Installation progress screen with real-time updates
- [x] Theme selection with live preview

### Phase 3: Advanced Features (Week 5-6)
- [x] Search and filtering
- [x] Help system with Glamour
- [x] Settings management
- [x] Error handling and recovery

### Phase 4: Polish (Week 7-8)
- [ ] Consistent styling with Lipgloss
- [ ] Keyboard shortcuts
- [ ] Accessibility improvements
- [ ] Performance optimization

## Design Examples

### 1. Main Menu Screen

```text
┌─────────────────────────────────────────────────────────────────────┐
│                                                                    │
│  ██╗  ██╗ █████╗ ██████╗ ███████╗██╗                               │
│  ██║ ██╔╝██╔══██╗██╔══██╗██╔════╝██║                               │
│  █████╔╝ ███████║██████╔╝█████╗  ██║                               │
│  ██╔═██╗ ██╔══██║██╔══██╗██╔══╝  ██║                               │
│  ██║  ██╗██║  ██║██║  ██║███████╗██║                               │
│  ╚═╝  ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚═╝                               │
│                                                                    │
│  Linux Development Environment Setup                               │
│                                                                    │
│  ┌─ What would you like to do? ─────────────────────────────────┐  │
│  │                                                              │  │
│  │  ❯ 🔧 Install Applications                                   │  │
│  │    🎨 Configure Themes                                       │  │
│  │    ⚙️  System Settings                                       │  │
│  │    📊 View Installation Status                               │  │
│  │    🔄 Update System                                          │  │
│  │    ❓ Help & Documentation                                   │  │
│  │                                                              │  │
│  └──────────────────────────────────────────────────────────────┘  │
│                                                                    │
│  [j/k] Navigate  [Enter] Select  [q] Quit  [?] Help                │
└────────────────────────────────────────────────────────────────────┘
```

### 2. Application Selection Screen

```text
┌─ Select Applications to Install ──────────────────────────────────────┐
│                                                                       │
│ Search: [git_____________] 🔍  Filter: [All ▼] Sort: [Name ▼]         │
│                                                                       │
│ ┌─ Development Tools ───────────────────────────── [12/18 selected] ┐ │
│ │ ✓ Git                     Fast distributed version control        │ │
│ │ ✓ Visual Studio Code      Extensible code editor                  │ │
│ │ ✓ Neovim                  Hyperextensible Vim-based text editor   │ │
│ │ ○ Docker Desktop          Containerization platform               │ │
│ │ ○ Postman                 API development environment             │ │
│ │ ✓ JetBrains Toolbox      IDE management tool                      │ │
│ │ ○ GitKraken               Git GUI client                          │ │
│ └───────────────────────────────────────────────────────────────────┘ │
│                                                                       │
│ ┌─ System Utilities ────────────────────────────── [5/8 selected] ──┐ │
│ │ ✓ htop                    Interactive process viewer              │ │
│ │ ✓ btop                    Resource monitor                        │ │
│ │ ○ Timeshift               System restore utility                  │ │
│ │ ✓ Flameshot              Screenshot tool                          │ │
│ └───────────────────────────────────────────────────────────────────┘ │
│                                                                       │
│ ┌─ Media & Graphics ────────────────────────────── [2/6 selected] ──┐ │
│ │ ○ GIMP                    Image manipulation program              │ │
│ │ ✓ VLC                     Media player                            │ │
│ │ ○ Spotify                 Music streaming                         │ │
│ └───────────────────────────────────────────────────────────────────┘ │
│                                                                       │
│ Selected: 19 apps • Download: 2.3 GB • Install time: ~15 min          │
│                                                                       │
│ [Space] Toggle  [Tab] Next Category  [/] Search  [Enter] Continue     │
└───────────────────────────────────────────────────────────────────────┘
```

### 3. Theme Selection with Live Preview

```text
┌─ Choose Theme ────────────────────────────────────────────────────────┐
│                                                                       │
│ Themes:                          Preview:                             │
│ ┌─────────────────────────┐     ┌─────────────────────────────────┐   │
│ │ ❯ 🌃 Tokyo Night (Dark) │     │ ┌─ Terminal ────────────────┐   │   │
│ │   🌃 Tokyo Night (Light)│     │ |                           │   │   │
  │   🐱 Catppuccin         │     │ │ user@karei:~$ ls -la      │   │   │
│ │   🧊 Nord               │     │ │ drwxr-xr-x  3 user group  │   │   │
│ │   🌲 Everforest         │     │ │ -rw-r--r--  1 user group  │   │   │
│ │   🟤 Gruvbox            │     │ │                           │   │   │
│ │   🌊 Kanagawa           │     │ │ user@karei:~$ _           │   │   │
│ │   🌹 Rose Pine          │     │ └───────────────────────────┘   │   │
│ └─────────────────────────┘     │                                 │   │
│                                 │ ┌─ Code ───────────────────── ┐ │   │
│                                 │ │ function hello() {          │ │   │
│                                 │ │   console.log("Hello!");    │ │   │
│                                 │ │ }                           │ │   │
│                                 │ │                             │ │   │
│                                 │ └─────────────────────────────┘ │   │
│                                 │                                 │   │   
│                                 │                                 │   │   
│                                 │                                 │   │   
│                                 │                                 │   │   
│                                 │                                 │   │   
│                                 └─────────────────────────────────┘   │
│                                                                       │
│ ┌─ Theme Details ────────────────────────────────────────────────────┐│
│ │  • Description: 
│ │    A dark theme inspired by Tokyo's neon-lit streets. Excellent
│ │    contrast and reduced eye strain.          
│ │ • Supports: Terminal, VS Code, Neovim, GNOME                       ││
│ │ • Variants: Dark (default), Light, Storm                           ││
│ │ • Wallpaper: Tokyo cityscape (4K available)                        ││
│ └────────────────────────────────────────────────────────────────────┘│
│───────────────────────────────────────────────────────────────────────│
│ [↑↓] Navigate  [Enter] Apply  [p] Preview All  [v] Change Variant     │
└───────────────────────────────────────────────────────────────────────┘
```

### 4. Installation Progress Screen

```text
┌─ Installing Applications ─────────────────────────────────────────────┐
│                                                                       │
│ Overall Progress: [████████████████▓▓▓▓] 80% (16/20 completed)       │
│                                                                       │
│ ✅ Git                          [████████████████████] 100%   2.1s    │
│ ✅ Visual Studio Code           [████████████████████] 100%  45.2s    │
│ ✅ Neovim                       [████████████████████] 100%   8.7s    │
│ ✅ htop                         [████████████████████] 100%   1.2s    │
│ ⚡ Docker Desktop               [████████████████▓▓▓▓]  85%  89.1s    │
│ ⏳ Postman                      [▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓]   0%    -      │
│ ⏳ GIMP                         [▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓]   0%    -      │
│                                                                       │
│ ┌─ Current Task ────────────────────────────────────────────────────┐ │
│ │ Installing Docker Desktop...                                      │ │
│ │ ⠋ Downloading container runtime (847.2 MB / 1.2 GB)             │ │
│ │ ⚡ Speed: 12.4 MB/s  ⏱️ ETA: 28s                                   │ │
│ └───────────────────────────────────────────────────────────────────┘ │
│                                                                       │
│ ┌─ Recent Activity ─────────────────────────────────────────────────┐ │
│ │ [14:23:15] ✅ Neovim installation completed                       │ │
│ │ [14:22:48] 🔧 Configuring Neovim plugins                         │ │
│ │ [14:22:12] ✅ VS Code installation completed                      │ │
│ │ [14:21:34] 🔧 Installing VS Code extensions                       │ │
│ └───────────────────────────────────────────────────────────────────┘ │
│                                                                       │
│ [Esc] Minimize  [l] View Logs  [p] Pause  [q] Quit (after current)   │
└───────────────────────────────────────────────────────────────────────┘
```

### 5. Configuration Screen

```text
┌─ System Configuration ────────────────────────────────────────────────┐
│                                                                       │
│ ┌─ Shell Environment ──────────────────────────────────────────────┐  │
│ │ Default Shell:     ❯ Fish 🐟  ○ Zsh  ○ Bash                    │  │
│ │ Terminal:          ❯ Ghostty  ○ Alacritty  ○ GNOME Terminal     │  │
│ │ Font:              ❯ JetBrains Mono  ○ Fira Code  ○ Cascadia    │  │
│ │ Font Size:         [14___________] 14pt                          │  │
│ └──────────────────────────────────────────────────────────────────┘  │
│                                                                       │
│ ┌─ Development Tools ──────────────────────────────────────────────┐  │
│ │ Primary Editor:    ❯ VS Code  ○ Neovim  ○ Emacs                 │  │
│ │ Git Username:      [John Doe________________]                    │  │
│ │ Git Email:         [john@example.com________]                    │  │
│ │ SSH Key:           ✅ Generated (4096-bit RSA)                   │  │
│ └──────────────────────────────────────────────────────────────────┘  │
│                                                                       │
│ ┌─ System Preferences ─────────────────────────────────────────────┐  │
│ │ Theme:             ❯ Tokyo Night (Dark)  [Preview]              │  │
│ │ Wallpaper:         ❯ Tokyo Cityscape  [Browse...]               │  │
│ │ Icon Theme:        ❯ Papirus-Dark  ○ Adwaita  ○ Breeze         │  │
│ │ Auto Updates:      ✅ Security  ✅ Applications  ○ System        │  │
│ └──────────────────────────────────────────────────────────────────┘  │
│                                                                       │
│ ┌─ Privacy & Security ─────────────────────────────────────────────┐  │
│ │ Telemetry:         ○ Full  ○ Minimal  ❯ Disabled               │  │
│ │ Firewall:          ✅ Enabled (UFW)                             │  │
│ │ Automatic Backup:  ✅ Daily snapshots (Timeshift)              │  │
│ └──────────────────────────────────────────────────────────────────┘  │
│                                                                       │
│ [Tab] Next Section  [Shift+Tab] Previous  [Enter] Apply  [r] Reset   │
└───────────────────────────────────────────────────────────────────────┘
```

## Architecture Overview

### Application Structure

```text
cmd/
├── main.go                 # CLI entry point
├── tui.go                  # TUI entry point
└── commands/               # CLI commands (backwards compatible)

internal/
├── tui/
│   ├── app.go             # Main TUI application
│   ├── models/            # Bubble Tea models
│   │   ├── menu.go
│   │   ├── apps.go
│   │   ├── progress.go
│   │   └── config.go
│   ├── components/        # Reusable components
│   ├── styles/            # Lipgloss styles
│   └── utils/             # TUI utilities
└── shared/                # Shared business logic
```

### State Management

```go
type AppState struct {
    CurrentScreen   Screen
    SelectedApps    []Application
    Configuration   Config
    InstallProgress map[string]Progress
    Theme          Theme
    SearchQuery    string
    FilterOptions  FilterState
}

type Screen int
const (
    MenuScreen Screen = iota
    AppsScreen
    ConfigScreen
    ProgressScreen
    ThemeScreen
)
```

### Component Design

```go
// Reusable components with consistent styling
type Components struct {
    Header      lipgloss.Style
    Card        lipgloss.Style
    Button      lipgloss.Style
    ProgressBar lipgloss.Style
    ErrorBox    lipgloss.Style
}
```

## Benefits of This Approach

### ✅ Pros
- **Rich Interactivity**: Full keyboard/mouse support, complex state management
- **Beautiful UI**: Sophisticated styling with Lipgloss
- **User Friendly**: Guided workflows, real-time feedback
- **Extensible**: Easy to add new screens and features
- **Professional**: Desktop-app-like experience in terminal

### ❌ Cons
- **Complexity**: Significant development time and learning curve
- **Dependencies**: Multiple new dependencies
- **Maintenance**: More code to maintain and debug
- **Performance**: Higher memory usage for TUI state

## Implementation Priority

1. **High Impact, Low Effort**: Application selection with Huh
2. **High Impact, Medium Effort**: Progress visualization with Bubbles
3. **Medium Impact, High Effort**: Theme preview system
4. **Polish Phase**: Advanced styling and animations

## Compatibility Strategy

```bash
# Maintain CLI compatibility
karei install git code docker    # Original CLI behavior

# Add TUI modes
karei                            # Launch interactive TUI
karei --tui                      # Explicitly request TUI
karei wizard                     # Guided setup mode
```

This approach transforms karei into a modern, interactive application while maintaining its CLI roots for power users and automation.
