# TUI Graphics Corruption Debugging Guide

## Problem Description
Graphics corruption/glitches in the TUI theme selection screen showing misaligned text in multiple rows. The issue appears in the split view with theme list (left) and preview (right).

## Root Causes Identified
1. **Unicode character width miscalculation** - Characters like ✓, ⚠, ✗ can cause alignment issues
2. **Double-bordered rendering** - Nested borders in viewports wrapped in more borders
3. **JoinHorizontal + viewport issues** - Known problematic combination in BubbleTea
4. **Renderer line skipping bug** - Fixed in BubbleTea issue #1232

## Solutions Applied

### 1. Fixed Viewport Update Logic (CRITICAL)
- **Both viewports must be updated on every Update() call** when both are visible
- Previous code only updated one viewport at a time, causing render artifacts
- Viewports must process ALL messages (not just keyboard events)

### 2. Proper Message Handling Order
- Process WindowSizeMsg first (affects viewport dimensions)
- Update viewports BEFORE handling keyboard input
- Batch all commands properly using tea.Batch()

### 3. Simplified View Rendering
- Use lipgloss.Place() for consistent sizing
- Removed double borders (viewport content already has borders)
- Let Lipgloss handle layout instead of manual line-by-line processing

### 4. Replaced Unicode Characters
- Changed ✓ to [OK]
- Changed ⚠ to [!]  
- Changed ✗ to [X]
- ASCII characters have predictable width

### 5. Adjusted Width Calculations
- Account for space separator between columns (Width-36 instead of Width-37)
- Removed Width() constraints on bordered boxes (let content flow naturally)

## Testing the Fix

```bash
# Build and run
just build-host
./bin/karei tui

# Navigate to themes screen and test:
# 1. Press arrow keys to navigate
# 2. Press 'p' to toggle preview
# 3. Resize terminal window
# 4. Check for misaligned text
```

## If Issues Persist

### Debug Approach 1: Test Without Viewports
Replace viewport rendering with direct content rendering:

```go
// In View() method
leftColumn := m.renderThemeList()
rightColumn := m.renderPreview()
return lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)
```

### Debug Approach 2: Log ANSI Output
```bash
# Capture raw ANSI output
./bin/karei tui 2>&1 | tee tui-output.log

# Analyze ANSI codes
cat tui-output.log | sed 's/\x1b/\\x1b/g' > tui-ansi.txt
```

### Debug Approach 3: Test Different Terminals
- Try in different terminal emulators:
  - gnome-terminal
  - alacritty  
  - kitty
  - xterm
- Different terminals handle ANSI differently

### Debug Approach 4: Simplify Content
Test with minimal content to isolate the issue:

```go
lines := []string{
    "Line 1",
    "Line 2", 
    "Line 3",
}
```

## Related Issues
- https://github.com/charmbracelet/bubbletea/issues/1232 - Line skipping bug
- https://github.com/charmbracelet/bubbletea/issues/573 - Altscreen artifacts
- https://github.com/charmbracelet/lipgloss/issues/286 - JoinHorizontal alignment
- https://github.com/charmbracelet/lipgloss/issues/562 - Unicode width calculation

## Terminal-Specific Fixes

### For iTerm2/Terminal.app
These terminals add extra whitespace padding. Consider:
- Adding empty lines to fill height
- Using altscreen mode differently
- Testing with `TERM=xterm-256color`

### For tmux/screen
Multiplexers can interfere with rendering:
- Test outside tmux first
- Check `$TERM` environment variable
- Try with `tmux -2` for 256 color support