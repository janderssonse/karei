# TUI Architecture Documentation

<!-- SPDX-FileCopyrightText: 2025 The Karei Authors -->
<!-- SPDX-License-Identifier: CC0-1.0 -->

Guidelines for developing the Karei Terminal User Interface using Bubble Tea framework.

## Table of Contents

1. [Architecture Principles](#architecture-principles)
2. [Layout Management](#layout-management)
3. [Navigation System](#navigation-system)
4. [Viewport Implementation](#viewport-implementation)
5. [State Management](#state-management)
6. [Testing Strategy](#testing-strategy)
7. [Development Workflow](#development-workflow)
8. [Performance Guidelines](#performance-guidelines)
9. [Troubleshooting](#troubleshooting)

## Architecture Principles

### Core Design

Follows **Tree-of-Models** pattern ([reference](https://leg100.github.io/en/posts/building-bubbletea-programs/#6-build-a-tree-of-models)):

- **Main App**: Root model managing layout and navigation
- **Screen Models**: Leaf models handling specific functionality
- **Message Flow**: Clean communication via message passing

### Key Rules

1. **No manual layout arithmetic** - Use Lipgloss `Height()` method
2. **No pagination hacks** - Use pure viewport scrolling
3. **Vim-like navigation** - `hjkl` for items, `HJKL` for screens
4. **Test viewport behavior** - Cover edge cases
5. **Use functional patterns** - Immutable state, pure functions

## Layout Management

### Problem: Manual Calculations

```go
// Wrong - Breaks with dynamic content
contentHeight := terminalHeight - headerHeight - footerHeight - 2
```

### Solution: Lipgloss Height()

```go
// Correct - Reliable and maintainable
func (a *App) getContentHeight() int {
    if a.height <= 0 {
        return 0
    }

    reservedHeight := 0

    // Use Lipgloss Height() for header
    if a.shouldShowHeader() {
        header := a.renderHeader()
        if header != "" {
            reservedHeight += lipgloss.Height(header)
        }
    }

    // Use Lipgloss Height() for footer
    if a.shouldShowFooter() {
        footer := a.renderFooter()
        if footer != "" {
            reservedHeight += lipgloss.Height(footer)
        }
    }

    return a.height - reservedHeight
}
```

### Layout Composition

```go
func (a *App) View() string {
    components := []string{}

    if header := a.renderHeader(); header != "" {
        components = append(components, header)
    }

    components = append(components, a.renderContent())

    if footer := a.renderFooter(); footer != "" {
        components = append(components, footer)
    }

    return lipgloss.JoinVertical(lipgloss.Left, components...)
}
```

## Navigation System

### Key Mappings

| Key | Action | Context |
|-----|--------|---------|
| `h/←` | Previous item/tab | Within screen |
| `l/→` | Next item/tab | Within screen |
| `j/↓` | Down/scroll down | Lists/viewports |
| `k/↑` | Up/scroll up | Lists/viewports |
| `H` | Previous screen | Global |
| `L` | Next screen | Global |
| `tab` | Next field | Forms |
| `shift+tab` | Previous field | Forms |
| `enter` | Select/confirm | Any |
| `esc` | Cancel/back | Any |
| `q` | Quit | Global |
| `/` | Search | Lists |
| `g` | Go to top | Viewports |
| `G` | Go to bottom | Viewports |
| `ctrl+d` | Half-page down | Viewports |
| `ctrl+u` | Half-page up | Viewports |

### Implementation

```go
func (m *Model) handleNavigation(msg tea.KeyMsg) tea.Cmd {
    switch msg.String() {
    case "h", "left":
        return m.navigateLeft()
    case "l", "right":
        return m.navigateRight()
    case "j", "down":
        m.moveDown()
        return m.ensureSelectedVisible()
    case "k", "up":
        m.moveUp()
        return m.ensureSelectedVisible()
    case "H":
        return func() tea.Msg { return NavigateScreenMsg{Direction: "prev"} }
    case "L":
        return func() tea.Msg { return NavigateScreenMsg{Direction: "next"} }
    case "g":
        m.viewport.GotoTop()
    case "G":
        m.viewport.GotoBottom()
    default:
        var cmd tea.Cmd
        m.viewport, cmd = m.viewport.Update(msg)
        return cmd
    }
    return nil
}
```

## Viewport Implementation

### Initialization

```go
func (m *Model) initViewport(width, height int) {
    m.viewport = viewport.New(width, height)
    m.viewport.MouseWheelEnabled = true
    m.viewport.KeyMap = viewport.KeyMap{
        PageDown: key.NewBinding(key.WithKeys("pgdown", "ctrl+d")),
        PageUp:   key.NewBinding(key.WithKeys("pgup", "ctrl+u")),
        Down:     key.NewBinding(key.WithKeys("j", "down")),
        Up:       key.NewBinding(key.WithKeys("k", "up")),
    }
}
```

### Content Updates

```go
func (m *Model) updateViewportContent() {
    content := m.renderVisibleContent()
    m.viewport.SetContent(content)
}
```

### Auto-scrolling

```go
func (m *Model) ensureSelectedVisible() tea.Cmd {
    selectedLine := m.getSelectedLineNumber()
    viewportTop := m.viewport.YOffset
    viewportBottom := viewportTop + m.viewport.Height - 1

    if selectedLine < viewportTop {
        m.viewport.LineUp(viewportTop - selectedLine)
    } else if selectedLine > viewportBottom {
        m.viewport.LineDown(selectedLine - viewportBottom)
    }

    return nil
}
```

## State Management

### Model Structure

```go
type Model struct {
    // Dimensions
    width    int
    height   int
    ready    bool

    // Navigation
    selected     int
    searchQuery  string
    searchActive bool

    // Display
    viewport viewport.Model
    styles   Styles

    // Data
    items []Item

    // State
    loading bool
    error   error
}
```

### Update Pattern

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        m.initViewport(m.width, m.getContentHeight())
        m.ready = true

    case tea.KeyMsg:
        if cmd := m.handleKeyPress(msg); cmd != nil {
            cmds = append(cmds, cmd)
        }

    case CustomMsg:
        m = m.handleCustomMessage(msg)
        m.updateViewportContent()
    }

    // Update viewport last
    m.viewport, cmd = m.viewport.Update(msg)
    cmds = append(cmds, cmd)

    return m, tea.Batch(cmds...)
}
```

## Testing Strategy

### Required Test Coverage

1. **Viewport Behavior**

- Content scrolling with large datasets
- Auto-scroll to selection
- Boundary conditions (top/bottom)
- Window resize handling

2. **Navigation**

- All key bindings work correctly
- Screen transitions preserve state
- Search filtering updates viewport

3. **State Management**

- Model updates are immutable
- State transitions are predictable
- Error states handled gracefully

### Test Example

```go
func TestViewportAutoScroll(t *testing.T) {
    m := NewModel()
    m.items = generateTestItems(100)
    m.width = 80
    m.height = 24
    m.initViewport(m.width, m.getContentHeight())

    // Select item below viewport
    m.selected = 50
    m.ensureSelectedVisible()

    // Verify viewport scrolled
    assert.True(t, m.viewport.YOffset > 0)
    assert.True(t, m.isSelectedVisible())
}
```

## Development Workflow

### Setup

```bash
# Install tools
go install github.com/charmbracelet/bubbletea@latest
go install github.com/charmbracelet/bubbles@latest
go install github.com/charmbracelet/lipgloss@latest

# Run TUI
just dev-tui
```

### Testing

```bash
# Unit tests
go test ./internal/tui/...

# Manual testing
KAREI_TUI_DEBUG=1 ./karei  # Enable debug output
```

### Debugging

```go
// Add debug output
func (m Model) View() string {
    if os.Getenv("KAREI_TUI_DEBUG") != "" {
        return fmt.Sprintf("Debug: selected=%d viewport=%d/%d\n%s",
            m.selected, m.viewport.YOffset, m.viewport.Height,
            m.normalView())
    }
    return m.normalView()
}
```

## Performance Guidelines

### Optimization Rules

1. **Minimize re-renders** - Only update changed content
2. **Lazy loading** - Load data as needed
3. **Efficient string building** - Use `strings.Builder`
4. **Cache rendered content** - Avoid redundant computation

### Example

```go
type Model struct {
    // Cache rendered content
    contentCache   string
    contentDirty   bool
}

func (m *Model) getContent() string {
    if !m.contentDirty && m.contentCache != "" {
        return m.contentCache
    }

    m.contentCache = m.renderContent()
    m.contentDirty = false
    return m.contentCache
}

func (m *Model) markDirty() {
    m.contentDirty = true
}
```

## Troubleshooting

### Common Issues

## Viewport not scrolling

- Check `YOffset` updates
- Verify content height > viewport height
- Ensure `SetContent()` called after updates

## Selection not visible

- Implement `ensureSelectedVisible()`
- Calculate line numbers correctly
- Account for multi-line items

## Layout broken after resize

- Handle `tea.WindowSizeMsg`
- Recalculate heights using Lipgloss
- Re-initialize viewport with new dimensions

## Performance issues

- Profile with `pprof`
- Check for unnecessary re-renders
- Optimize string operations
- Use viewport for large lists

### Debug Commands

```bash
# Profile CPU
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Profile memory
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Trace execution
go test -trace=trace.out
go tool trace trace.out
```

## Summary

The Karei TUI architecture provides a solid foundation by:
- Following established Bubble Tea patterns
- Using Lipgloss for reliable layouts
- Implementing proper viewport scrolling
- Maintaining clean state management
- Ensuring testability and performance

Keep code simple, test edge cases, and always use framework methods over manual calculations.
