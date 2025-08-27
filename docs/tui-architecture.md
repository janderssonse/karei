# Title

<!-- SPDX-FileCopyrightText: 2025 The Karei Authors -->
<!-- SPDX-License-Identifier: CC0-1.0 -->

## TUI Architecture Documentation

This document provides comprehensive guidelines for developing and maintaining the Karei Terminal User Interface (TUI) using the Bubble Tea framework.

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

### Core Design Philosophy

Karei TUI follows the **Tree-of-Models** pattern as described in [Building Bubble Tea Programs](https://leg100.github.io/en/posts/building-bubbletea-programs/#6-build-a-tree-of-models):

- **Main App**: Root model managing layout and navigation
- **Screen Models**: Leaf models handling specific functionality
- **Message Flow**: Clean communication between models via message passing

### STRICT Guidelines

1. **❌ NEVER use manual layout arithmetic** - Always use Lipgloss `Height()` method
2. **❌ NEVER implement pagination hacks** - Use pure viewport scrolling
3. **✅ ALWAYS follow vim-like navigation** - `hjkl` for items, `HJKL` for screens
4. **✅ ALWAYS test viewport behavior** - Comprehensive edge case coverage
5. **✅ ALWAYS use functional patterns** - Immutable state, pure functions

## Layout Management

### The Problem: Layout Arithmetic Errors

Manual layout calculations are error-prone and fragile:

```go
// ❌ DON'T DO THIS - Breaks with dynamic content
contentHeight := terminalHeight - headerHeight - footerHeight - 2
```

### The Solution: Lipgloss Height() Method

Use Lipgloss built-in methods for reliable layout:

```go
// ✅ CORRECT - Reliable and maintainable
func (a *App) getContentHeight() int {
    if a.height <= 0 {
        return 0
    }

    reservedHeight := 0

    // Use Lipgloss Height() method for header
    if a.shouldShowHeader() {
        header := a.renderHeader()
        if header != "" {
            reservedHeight += lipgloss.Height(header)
        }
    }

    // Use Lipgloss Height() method for footer
    if a.shouldShowFooter() {
        footer := a.renderFooter()
        if footer != "" {
            reservedHeight += lipgloss.Height(footer)
        }
    }

    return a.height - reservedHeight
}
```

### Layout Composition Pattern

Always use Lipgloss for layout composition:

```go
func (a *App) View() string {
    // Build layout components
    components := []string{}

    if header := a.renderHeader(); header != "" {
        components = append(components, header)
    }

    components = append(components, a.renderContent())

    if footer := a.renderFooter(); footer != "" {
        components = append(components, footer)
    }

    // Pure Lipgloss composition - no arithmetic needed
    return lipgloss.JoinVertical(lipgloss.Top, components...)
}
```

## Navigation System

### Navigation Hierarchy

Karei implements a **layered navigation system** that respects vim conventions:

#### Layer 1: Global Navigation (Main App)
- `H` / `Shift+H`: Previous screen
- `L` / `Shift+L`: Next screen
- `q` / `Ctrl+C`: Quit application
- `/`: Toggle search

#### Layer 2: Content Navigation (Screen Models)
- `h`: Move left (categories, tabs)
- `l`: Move right (categories, tabs)
- `j` / `Down`: Move down (items)
- `k` / `Up`: Move up (items)

#### Layer 3: Page Navigation (Screen Models)
- `J` / `Shift+J`: Page down / Next category page
- `K` / `Shift+K`: Page up / Previous category page

#### Layer 4: Actions
- `Space`: Toggle installation (None ↔ Install)
- `d`: Mark for uninstallation (vim-style delete)
- `Enter`: Apply all selected operations (install + uninstall)
- `Esc`: Cancel/Back

### Navigation Implementation Pattern

```go
func (a *App) handleNavigationKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "shift+h", "H":
        return a.navigateToPreviousScreen()
    case "shift+l", "L":
        return a.navigateToNextScreen()
    case "shift+j", "J", "shift+k", "K":
        return a.handleVerticalNavigation(msg)
    default:
        // CRITICAL: Delegate ALL other keys to content model
        // This includes hjkl which should be handled by screen models
        var cmd tea.Cmd
        a.contentModel, cmd = a.contentModel.Update(msg)
        return a, cmd
    }
}
```

### Key Delegation Rules

1. **Main App handles**: Global navigation only (`H/L`, `/`, `q`)
2. **Screen Models handle**: All content navigation (`hjkl`, `Space`, `d`, `Enter`)
3. **Never overlap**: Each key should have one clear handler

## Viewport Implementation

### The Problem: Pagination Hacks

Many TUI implementations use complex pagination logic:

```go
// ❌ DON'T DO THIS - Complex, error-prone, hard to maintain
func (m *Model) renderPage() string {
    startIdx := m.currentPage * m.itemsPerPage
    endIdx := min(startIdx + m.itemsPerPage, len(m.items))

    if startIdx >= len(m.items) {
        startIdx = len(m.items) - m.itemsPerPage
        if startIdx < 0 { startIdx = 0 }
    }

    visibleItems := m.items[startIdx:endIdx]
    // ... complex pagination rendering logic
}
```

### The Solution: Pure Viewport Scrolling

Render everything, let viewport handle visibility:

```go
// ✅ CORRECT - Simple, reliable, maintainable
func (m *AppsModel) renderAllCategories() string {
    if len(m.categories) == 0 {
        return "No categories available"
    }

    // Render ALL categories - viewport handles what's visible
    categoryViews := make([]string, 0, len(m.categories))
    for i, cat := range m.categories {
        isCurrent := i == m.currentCat
        categoryViews = append(categoryViews, m.renderCategory(cat, isCurrent))
    }

    // Pure Lipgloss composition - no pagination calculations
    return lipgloss.JoinVertical(lipgloss.Top, categoryViews...)
}

func (m *AppsModel) View() string {
    if !m.ready {
        return "Loading..."
    }

    // Update viewport content with current state
    m.viewport.SetContent(m.renderAllCategories())

    // Return viewport view - handles scrolling automatically
    return m.viewport.View()
}
```

### Smart Auto-scrolling

Implement intelligent scrolling that follows user selection:

```go
func (m *AppsModel) ensureSelectionVisible() {
    if !m.ready || len(m.categories) == 0 {
        return
    }

    // Calculate EXACT line position of current selection
    selectionLine := m.calculateActualSelectionLine()

    // Get current viewport window
    viewportTop := m.viewport.YOffset
    viewportBottom := viewportTop + m.viewport.Height - 1

    // Buffer zones for comfortable viewing
    topBuffer := 6    // Aggressive upward scrolling
    bottomBuffer := 3 // Conservative downward scrolling

    // Scroll viewport to keep selection visible with buffers
    if selectionLine <= viewportTop+topBuffer {
        // Selection too close to top - scroll up
        newOffset := selectionLine - topBuffer
        if newOffset < 0 { newOffset = 0 }
        m.viewport.SetYOffset(newOffset)
    } else if selectionLine >= viewportBottom-bottomBuffer {
        // Selection too close to bottom - scroll down
        newOffset := selectionLine - m.viewport.Height + bottomBuffer + 1
        maxOffset := m.viewport.TotalLineCount() - m.viewport.Height
        if maxOffset < 0 { maxOffset = 0 }
        if newOffset > maxOffset { newOffset = maxOffset }
        m.viewport.SetYOffset(newOffset)
    }
    // Selection is comfortably visible - no scroll needed
}
```

### Exact Line Calculation

Accurate scrolling requires precise line position calculation:

```go
func (m *AppsModel) calculateActualSelectionLine() int {
    line := 0

    // Count lines for each category before current one
    for catIdx := 0; catIdx < m.currentCat && catIdx < len(m.categories); catIdx++ {
        cat := m.categories[catIdx]

        // Render category to count exact lines (including borders, padding)
        categoryContent := m.renderCategory(cat, false)
        categoryLines := lipgloss.Height(categoryContent)
        line += categoryLines

        // Add spacing between categories
        if catIdx < len(m.categories)-1 {
            line++ // Empty line between categories
        }
    }

    // Add lines within current category up to current app
    if m.currentCat < len(m.categories) {
        currentCat := m.categories[m.currentCat]

        // Category header (1 line)
        line++

        // Border top and padding (2 lines)
        line += 2

        // Apps before current selection
        line += currentCat.currentApp
    }

    return line
}
```

## State Management

### Model Caching Strategy

Karei caches screen models to preserve user state:

```go
type App struct {
    models map[Screen]tea.Model // Model cache
    // ...
}

func (a *App) navigateToScreen(targetScreen Screen, data interface{}) (tea.Model, tea.Cmd) {
    // Check if model is already cached
    if cachedModel, exists := a.models[targetScreen]; exists {
        return a.useCachedModel(targetScreen, cachedModel)
    }

    // Create new model if not cached
    newModel := a.createModelForScreen(targetScreen, data)
    return a.setupNewModel(newModel, targetScreen)
}
```

### Caching Rules

- **Always Cache**: Menu, Apps, Theme, Config, Status, Help screens
- **Never Cache**: Progress screen (always fresh for new operations)
- **State Preservation**: Maintain selection state, scroll position, search queries

### Message Flow Architecture

Use structured messages for clean communication:

```go
// NavigateMsg enables communication between models
type NavigateMsg struct {
    Screen int
    Data   interface{} // Optional data to pass to new screen
}

// Usage example: Navigate from Apps to Progress with selected apps
return func() tea.Msg {
    return NavigateMsg{Screen: ProgressScreen, Data: selectedApps}
}
```

## Testing Strategy

### Viewport Testing Requirements

**CRITICAL**: All viewport behavior must be thoroughly tested:

```go
func TestAppsModel_ViewportScrolling(t *testing.T) {
    tests := []struct {
        name           string
        terminalHeight int
    }{
        {"Very small terminal", 15},
        {"Small terminal", 25},
        {"Medium terminal", 40},
        {"Large terminal", 60},
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            model := NewAppsWithSize(styles.New(), 80, tc.terminalHeight)

            // Initialize viewport
            model.Update(tea.WindowSizeMsg{Width: 80, Height: tc.terminalHeight})

            // Verify viewport initialization
            if !model.ready {
                t.Error("Viewport should be ready after window size message")
            }

            // Verify viewport dimensions
            if model.viewport.Height != tc.terminalHeight {
                t.Errorf("Expected viewport height %d, got %d",
                    tc.terminalHeight, model.viewport.Height)
            }
        })
    }
}
```

### Navigation Testing

Test navigation behavior across all scenarios:

```go
func TestAppsModel_ViewportAutoScroll_ExactLineCalculation(t *testing.T) {
    model := setupLargeModel(t) // Model with many categories/items

    // Navigate through all items and verify viewport follows
    for i := 0; i < 50; i++ {
        model.navigateDown()

        // Verify selection is always visible
        selectionLine := model.calculateActualSelectionLine()
        viewportTop := model.viewport.YOffset
        viewportBottom := viewportTop + model.viewport.Height - 1

        if selectionLine < viewportTop || selectionLine > viewportBottom {
            t.Errorf("Selection line %d not visible in viewport [%d, %d]",
                selectionLine, viewportTop, viewportBottom)
        }
    }
}
```

### Layout Testing

Verify header/footer behavior:

```go
func TestAppsModel_Header_AlwaysVisible(t *testing.T) {
    app := NewApp()
    app.navigateToScreen(AppsScreen, nil)

    // Header should be visible
    if !app.ShouldShowHeader() {
        t.Error("Header should be visible on apps screen")
    }

    // Navigate content and verify header remains visible
    appsModel := app.contentModel.(*AppsModel)
    for i := 0; i < 20; i++ {
        appsModel.navigateDown()
    }

    if !app.ShouldShowHeader() {
        t.Error("Header should remain visible after content scrolling")
    }
}
```

## Development Workflow

### File Organization

```text
internal/tui/
├── app.go                 # Main application (tree root)
├── app_test.go           # Main application tests
├── models/               # Screen models (tree leaves)
│   ├── navigation.go     # Navigation message types
│   ├── menu.go          # Main menu screen
│   ├── apps.go          # Application selection
│   ├── apps_test.go     # Apps model tests (CRITICAL)
│   ├── themes.go        # Theme selection
│   ├── config.go        # Configuration screen
│   ├── status.go        # Status display
│   ├── help.go          # Help system
│   └── progress.go      # Installation progress
└── styles/
    └── styles.go        # Consistent styling
```

### Development Process

1. **Design Screen Model**: Plan model state and behavior
2. **Implement Core Logic**: Focus on state management and update logic
3. **Add Viewport Integration**: Implement proper scrolling behavior
4. **Write Comprehensive Tests**: Cover viewport, navigation, edge cases
5. **Test Edge Cases**: Very small terminals, large datasets, rapid navigation
6. **Integration Testing**: Test with main app navigation

### Code Quality Checklist

Before committing TUI code, verify:

- [ ] ✅ No manual layout arithmetic - only Lipgloss `Height()`
- [ ] ✅ Pure viewport scrolling - no pagination hacks
- [ ] ✅ Vim-like navigation - `hjkl` for items, `HJKL` for screens
- [ ] ✅ Comprehensive tests - viewport behavior tested
- [ ] ✅ Error handling - graceful degradation for edge cases
- [ ] ✅ Performance - efficient rendering and scrolling
- [ ] ✅ Documentation - comments explain complex logic

## Performance Guidelines

### Model Optimization

- **Cache Models**: Preserve state between screen switches
- **Lazy Initialization**: Only create models when needed
- **Efficient Updates**: Batch related state changes
- **Memory Management**: Clean up unused resources

### Rendering Optimization

- **Minimize Re-renders**: Only update when state changes
- **Efficient String Building**: Use `strings.Builder` for complex content
- **Viewport Efficiency**: Let viewport handle visibility, don't pre-filter
- **Style Caching**: Reuse lipgloss styles where possible

### Scrolling Performance

```go
// ✅ Efficient - Let viewport handle everything
func (m *AppsModel) View() string {
    m.viewport.SetContent(m.renderAllCategories())
    return m.viewport.View()
}

// ❌ Inefficient - Manual viewport management
func (m *AppsModel) View() string {
    // Complex logic to determine what to render
    visibleContent := m.calculateVisibleContent()
    return visibleContent
}
```

## Troubleshooting

### Common Issues

#### Issue: Layout Breaks on Terminal Resize

**Cause**: Manual layout arithmetic
**Solution**: Use Lipgloss `Height()` method exclusively

#### Issue: Scrolling Doesn't Follow Selection

**Cause**: Incorrect line position calculation
**Solution**: Account for all rendered elements (borders, padding, spacing)

#### Issue: Navigation Keys Don't Work

**Cause**: Key handling overlap between main app and screen models
**Solution**: Clear separation - global keys in app, content keys in models

#### Issue: Viewport Content Flickers

**Cause**: Frequent viewport content updates
**Solution**: Only update viewport content when necessary

#### Issue: Tests Fail on Different Terminal Sizes

**Cause**: Hardcoded assumptions about viewport dimensions
**Solution**: Test across multiple terminal sizes, use relative calculations

### Debugging Techniques

#### Viewport State Inspection

```go
// Add temporary debugging in tests
t.Logf("Viewport: YOffset=%d, Height=%d, TotalLines=%d",
    m.viewport.YOffset, m.viewport.Height, m.viewport.TotalLineCount())
t.Logf("Selection: Line=%d, Category=%d, App=%d",
    selectionLine, m.currentCat, currentCat.currentApp)
```

#### Layout Debugging

```go
// Verify layout component sizes
header := a.renderHeader()
footer := a.renderFooter()
content := a.renderContent()

t.Logf("Layout: Header=%d, Footer=%d, Content=%d, Total=%d",
    lipgloss.Height(header), lipgloss.Height(footer),
    lipgloss.Height(content), a.height)
```

### Performance Profiling

Use Go's built-in profiling tools:

```bash
# CPU profiling during TUI operation
go test -cpuprofile=cpu.prof -bench=BenchmarkTUINavigation

# Memory profiling
go test -memprofile=mem.prof -bench=BenchmarkTUIRendering

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

## Uninstall Mode Implementation

### Design Philosophy

Karei TUI implements a **dual-mode selection system** that allows users to mark applications for both installation and uninstallation in a single workflow, following vim-style conventions.

### Selection State System

#### Enhanced State Model

```go
// SelectionState represents the intended operation for an application
type SelectionState int

const (
    StateNone SelectionState = iota  // No operation selected
    StateInstall                     // Mark for installation
    StateUninstall                   // Mark for uninstallation
)

// Visual indicators for each state
const (
    StatusNotInstalled = "○"  // Not selected for any operation
    StatusInstalled    = "●"  // Already installed, no operation
    StatusSelected     = "✓"  // Selected for installation
    StatusUninstall    = "✗"  // Selected for uninstallation
)
```

#### State Transitions

```text
Current State      → Key   → New State    → Visual → Description
───────────────────────────────────────────────────────────────
Not selected       → Space → Install      → ✓     → Mark for install
Not selected       → d     → Uninstall    → ✗     → Mark for uninstall
Selected install   → Space → None         → ○/●   → Toggle off install
Selected install   → d     → Uninstall    → ✗     → Switch to uninstall
Selected uninstall → Space → Install      → ✓     → Switch to install
Selected uninstall → d     → Uninstall    → ✗     → Keep uninstall (no-op)
Already installed  → Space → Install      → ✓     → Reinstall/update
Already installed  → d     → Uninstall    → ✗     → Mark for removal
```

### Implementation Architecture

#### Model Structure

```go
type AppsModel struct {
    // Enhanced selection tracking
    selected map[string]SelectionState  // App key -> operation type

    // ... existing fields
}

// Operation tracking for progress screen
type SelectedOperation struct {
    AppKey    string
    Operation SelectionState
    AppName   string
}
```

#### Key Handling Implementation

```go
func (m *AppsModel) handleSelectionKeys(msg tea.KeyMsg) {
    switch msg.String() {
    case " ":  // Space key
        m.toggleInstallSelection()
    case "d":  // Vim-style delete
        m.markForUninstall()
    }
}

func (m *AppsModel) toggleInstallSelection() {
    if m.currentCat >= len(m.categories) || 
       m.categories[m.currentCat].currentApp >= len(m.categories[m.currentCat].apps) {
        return
    }

    app := m.categories[m.currentCat].apps[m.categories[m.currentCat].currentApp]

    // Toggle behavior: None -> Install -> None
    currentState := m.selected[app.Key]
    switch currentState {
    case StateNone:
        m.selected[app.Key] = StateInstall
    case StateInstall:
        delete(m.selected, app.Key) // Return to None state
    case StateUninstall:
        m.selected[app.Key] = StateInstall // Switch from uninstall to install
    }
}

func (m *AppsModel) markForUninstall() {
    if m.currentCat >= len(m.categories) || 
       m.categories[m.currentCat].currentApp >= len(m.categories[m.currentCat].apps) {
        return
    }

    app := m.categories[m.currentCat].apps[m.categories[m.currentCat].currentApp]
    m.selected[app.Key] = StateUninstall
}

```

#### Visual Indicator Logic

```go
func (m *AppsModel) getAppIndicator(app app, selectionState SelectionState) string {
    switch selectionState {
    case StateInstall:
        return StatusSelected    // ✓
    case StateUninstall:
        return StatusUninstall   // ✗
    default:
        // No selection - show install status
        if app.Installed {
            return StatusInstalled    // ●
        }
        return StatusNotInstalled     // ○
    }
}
```

### Progress Screen Integration

#### Mixed Operations Support

```go
// Enhanced navigation message for mixed operations
type NavigateMsg struct {
    Screen     int
    Operations []SelectedOperation  // Both installs and uninstalls
}

// Progress screen handles both operation types
func (m *ProgressModel) processOperations(operations []SelectedOperation) {
    for _, op := range operations {
        switch op.Operation {
        case StateInstall:
            m.installApp(op.AppKey, op.AppName)
        case StateUninstall:
            m.uninstallApp(op.AppKey, op.AppName)
        }
    }
}
```

#### Underlying System Integration

The TUI delegates to existing installation/uninstallation systems:

```go
func (m *ProgressModel) installApp(appKey, appName string) {
    // Delegates to internal/installer package
    // Handles apt, snap, flatpak, mise, etc.
}

func (m *ProgressModel) uninstallApp(appKey, appName string) {
    // Delegates to internal/uninstall package
    // Uses same package managers for removal
}
```

### User Experience Enhancements

#### Enhanced Footer Display

```text
[hjkl] Navigate  [Space] Toggle Install  [d] Uninstall  [Enter] Apply
```

#### Category Statistics

```go
// Enhanced category statistics
func (m *AppsModel) renderCategoryHeader(cat category) string {
    installCount := 0
    uninstallCount := 0

    for _, app := range cat.apps {
        switch m.selected[app.Key] {
        case StateInstall:
            installCount++
        case StateUninstall:
            uninstallCount++
        }
    }

    stats := fmt.Sprintf("[+%d/-%d of %d]", 
        installCount, uninstallCount, len(cat.apps))

    return fmt.Sprintf("─ %s %s", cat.name, stats)
}
```

### Testing Strategy

#### Comprehensive State Testing

```go
func TestUninstallMode_StateTransitions(t *testing.T) {
    model := NewAppsWithSize(styles.New(), 80, 40)

    // Test all state transitions
    transitions := []struct {
        initial  SelectionState
        key      string
        expected SelectionState
        visual   string
    }{
        {StateNone, " ", StateInstall, "✓"},
        {StateNone, "d", StateUninstall, "✗"},
        {StateInstall, "d", StateUninstall, "✗"},
        {StateInstall, "x", StateNone, "○"},
        {StateUninstall, " ", StateInstall, "✓"},
        {StateUninstall, "x", StateNone, "○"},
    }

    for _, tt := range transitions {
        // Test each transition
        model.setAppState(testApp, tt.initial)
        model.handleKey(tt.key)

        actual := model.getAppState(testApp)
        if actual != tt.expected {
            t.Errorf("Expected state %v, got %v", tt.expected, actual)
        }

        visual := model.getAppIndicator(testApp, actual)
        if visual != tt.visual {
            t.Errorf("Expected visual %s, got %s", tt.visual, visual)
        }
    }
}
```

#### Mixed Operations Testing

```go
func TestUninstallMode_MixedOperations(t *testing.T) {
    model := NewAppsWithSize(styles.New(), 80, 40)

    // Mark some apps for install, others for uninstall
    model.markAppForInstall("git")
    model.markAppForInstall("vscode")
    model.markAppForUninstall("firefox")
    model.markAppForUninstall("chrome")

    operations := model.getSelectedOperations()

    installOps := 0
    uninstallOps := 0

    for _, op := range operations {
        switch op.Operation {
        case StateInstall:
            installOps++
        case StateUninstall:
            uninstallOps++
        }
    }

    assert.Equal(t, 2, installOps, "Should have 2 install operations")
    assert.Equal(t, 2, uninstallOps, "Should have 2 uninstall operations")
}
```

### Security Considerations

#### Uninstall Safety

- **Confirmation Required**: Uninstall operations require explicit user confirmation
- **Dependency Checking**: Warn about package dependencies before removal
- **System Package Protection**: Prevent removal of critical system packages
- **Backup Recommendations**: Suggest configuration backups before removal

#### Implementation Safeguards

```go
func (m *ProgressModel) uninstallApp(appKey, appName string) error {
    // Check if app is safe to remove
    if isCriticalSystemPackage(appKey) {
        return fmt.Errorf("cannot remove critical system package: %s", appName)
    }

    // Delegate to uninstall system with safety checks
    uninstaller := uninstall.NewUninstaller(m.verbose)
    return uninstaller.UninstallApp(context.Background(), appKey)
}
```

### Performance Optimizations

#### Efficient State Management

- **Sparse Storage**: Only store non-default states in selection map
- **Batch Operations**: Group install/uninstall operations for efficiency
- **Lazy Validation**: Validate operations only when executing, not during selection

#### Memory Management

```go
// Efficient state cleanup
func (m *AppsModel) cleanupSelections() {
    for appKey, state := range m.selected {
        if state == StateNone {
            delete(m.selected, appKey)  // Remove default states
        }
    }
}
```

## Conclusion

The Karei TUI architecture provides a robust foundation for terminal user interfaces by:

1. **Following Proven Patterns**: Tree-of-models, pure viewport scrolling
2. **Avoiding Common Pitfalls**: No layout arithmetic, no pagination hacks
3. **Comprehensive Testing**: Thorough coverage of edge cases and viewport behavior
4. **Performance Focus**: Efficient rendering and navigation
5. **Maintainable Code**: Clear separation of concerns, functional patterns
6. **Dual-Mode Operations**: Unified install/uninstall workflow with vim-style navigation

The enhanced selection system allows users to efficiently manage their system packages through a single, intuitive interface while maintaining the safety and reliability expected from a system management tool.

By strictly adhering to these guidelines, the TUI remains reliable, performant, and maintainable as new features are added.

---

**References:**

- [Building Bubble Tea Programs - Layout Arithmetic](https://leg100.github.io/en/posts/building-bubbletea-programs/#7-layout-arithmetic-is-error-prone)
- [Building Bubble Tea Programs - Tree of Models](https://leg100.github.io/en/posts/building-bubbletea-programs/#6-build-a-tree-of-models)
- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Lipgloss Documentation](https://github.com/charmbracelet/lipgloss)
