// SPDX-FileCopyrightText: 2025 The Karei Authors
// SPDX-License-Identifier: EUPL-1.2

// Package models implements installation progress tracking UI.
//
//nolint:funcorder // Methods grouped logically by functionality for better readability
package models

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/janderssonse/karei/internal/adapters/platform"
	"github.com/janderssonse/karei/internal/adapters/ubuntu"
	"github.com/janderssonse/karei/internal/apps"
	"github.com/janderssonse/karei/internal/domain"
	"github.com/janderssonse/karei/internal/tui/styles"
	"github.com/janderssonse/karei/internal/uninstall"
)

// Constants for progress messages.
const (
	msgUninstallationComplete = "Uninstallation complete"

	// UI layout constants.
	maxProgressWidth        = 100
	progressBarPadding      = 50
	defaultProgressBarWidth = 50
)

// Error variables for static error definitions.
var (
	ErrDownloadFailed     = errors.New("download failed")
	ErrInstallationFailed = errors.New("installation failed")
	ErrOperationFailed    = errors.New("operation failed")
)

// Note: Operation constants are defined in navigation.go

// ProgressData carries data needed to create progress screen.
type ProgressData struct {
	Operations []SelectedOperation
	Password   string
}

// InstallTask represents a single installation or uninstallation task.
type InstallTask struct {
	Name        string
	Description string
	Operation   string // "install" or "uninstall"
	Status      string // "pending", "downloading", "installing", "uninstalling", "completed", "failed"
	Progress    float64
	Size        string
	Speed       string
	ETA         string
	Duration    time.Duration
	Error       string
}

// ProgressMsg represents progress updates.
type ProgressMsg struct {
	TaskName string
	Progress float64
	Status   string
	Speed    string
	ETA      string
}

// CompletedMsg represents task completion.
type CompletedMsg struct {
	TaskName string
	Success  bool
	Duration time.Duration
	Error    string
}

// ProgressUpdateMsg carries progress updates for individual tasks.
type ProgressUpdateMsg struct {
	TaskIndex int
	Progress  float64
	Message   string
}

const (
	// TaskStatusPending represents a task that hasn't started yet.
	TaskStatusPending = "pending"
	// TaskStatusDownloading represents a task that is currently downloading.
	TaskStatusDownloading = "downloading"
	// TaskStatusInstalling represents a task that is currently installing.
	TaskStatusInstalling = "installing"
	// TaskStatusUninstalling represents a task that is currently uninstalling.
	TaskStatusUninstalling = "uninstalling"
	// TaskStatusCompleted represents a task that has completed successfully.
	TaskStatusCompleted = "completed"
	// TaskStatusFailed represents a task that has failed.
	TaskStatusFailed = "failed"
)

// Progress represents the installation progress screen model.
//
//nolint:containedctx // TUI models require context for proper cancellation propagation
type Progress struct {
	styles          *styles.Styles
	width           int
	height          int
	tasks           []InstallTask
	currentTask     int
	overallProgress float64
	spinner         spinner.Model
	progressBars    map[string]progress.Model
	logs            []string
	quitting        bool
	completed       bool
	startTime       time.Time
	paused          bool
	showingLogs     bool // For floating log viewer

	// Context for cancellation and timeout propagation
	ctx context.Context

	// Hexagonal architecture integration
	packageInstaller domain.PackageInstaller
	uninstaller      *uninstall.Uninstaller

	// Track operations for immediate status sync on navigation
	operations []SelectedOperation
}

// NewProgressWithOperations creates a new progress model with mixed install/uninstall operations.
func NewProgressWithOperations(ctx context.Context, styleConfig *styles.Styles, operations []SelectedOperation) *Progress {
	return NewProgressWithOperationsAndPassword(ctx, styleConfig, operations, "")
}

// NewProgressWithOperationsAndPassword creates a new progress model with password for sudo operations.
func NewProgressWithOperationsAndPassword(ctx context.Context, styleConfig *styles.Styles, operations []SelectedOperation, password string) *Progress {
	// Create tasks from operations
	tasks := make([]InstallTask, len(operations))
	progressBars := make(map[string]progress.Model)

	for index, operationItem := range operations {
		var (
			description string
			operation   string
		)

		switch operationItem.Operation {
		case StateInstall:
			description = fmt.Sprintf("Installing %s...", operationItem.AppName)
			operation = OperationInstall
		case StateUninstall:
			description = fmt.Sprintf("Uninstalling %s...", operationItem.AppName)
			operation = OperationUninstall
		default:
			description = fmt.Sprintf("Processing %s...", operationItem.AppName)
			operation = "unknown"
		}

		tasks[index] = InstallTask{
			Name:        operationItem.AppKey,
			Description: description,
			Operation:   operation,
			Status:      TaskStatusPending,
			Progress:    0.0,
			Size:        "Unknown",
			Speed:       "",
			ETA:         "",
		}

		// Create Bubble Tea progress bar with default gradient
		progressBar := progress.New(progress.WithDefaultGradient())
		progressBar.Width = 50 // Default width, will be updated dynamically
		// Initialize with task's starting progress (usually 0)
		progressBar.SetPercent(tasks[index].Progress)
		progressBars[operationItem.AppKey] = progressBar
	}

	model := createProgressModel(ctx, styleConfig, tasks, progressBars, password)
	model.operations = operations // Store operations for immediate sync on navigation

	return model
}

// NewProgress creates a new progress model (legacy compatibility).
func NewProgress(ctx context.Context, styleConfig *styles.Styles, taskNames []string) *Progress {
	// Create tasks from names - assume install operations
	tasks := make([]InstallTask, len(taskNames))
	progressBars := make(map[string]progress.Model)

	for i, name := range taskNames {
		tasks[i] = InstallTask{
			Name:        name,
			Description: fmt.Sprintf("Installing %s...", name),
			Operation:   OperationInstall,
			Status:      TaskStatusPending,
			Progress:    0.0,
			Size:        "Unknown",
			Speed:       "",
			ETA:         "",
		}

		// Create Bubble Tea progress bar with default gradient
		p := progress.New(progress.WithDefaultGradient())
		p.Width = 50 // Default width, will be updated dynamically
		progressBars[name] = p
	}

	return createProgressModel(ctx, styleConfig, tasks, progressBars, "")
}

// createProgressModel creates the actual progress model (shared between constructors).
func createProgressModel(ctx context.Context, styleConfig *styles.Styles, tasks []InstallTask, progressBars map[string]progress.Model, password string) *Progress {
	// Create spinner
	sSpinner := spinner.New()
	sSpinner.Spinner = spinner.Dot
	sSpinner.Style = lipgloss.NewStyle().Foreground(styleConfig.Primary)

	// Create hexagonal package installer optimized for TUI
	commandRunner := platform.NewTUICommandRunner(false, false)                                 // verbose=false, dryRun=false, tuiMode=true
	fileManager := platform.NewFileManager(false)                                               // verbose=false
	packageInstaller := ubuntu.NewTUIPackageInstaller(commandRunner, fileManager, false, false) // verbose=false, dryRun=false, tuiMode=true
	// Note: Password handling will be managed by the command runner

	// Create uninstaller with password support
	uninstaller := uninstall.NewUninstaller(false) // verbose=false
	if password != "" {
		uninstaller.SetPassword(password)
	}

	return &Progress{
		styles:       styleConfig,
		tasks:        tasks,
		currentTask:  0,
		spinner:      sSpinner,
		progressBars: progressBars,
		logs:         make([]string, 0, 10), // Capacity of 10 since we keep only last 10 entries
		startTime:    time.Now(),
		ctx:          ctx, // Store context for proper propagation

		// Initialize hexagonal architecture systems
		packageInstaller: packageInstaller,
		uninstaller:      uninstaller,
	}
}

// Init initializes the progress model.
func (m *Progress) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.executeInstallations(), // Start actual installation
	)
}

// Update handles messages and returns updated model and commands.
//

// Update handles messages for the Progress model.
func (m *Progress) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ProgressMsg:
		return m.handleProgressMsg(msg)
	case ProgressUpdateMsg:
		return m.handleProgressUpdateMsg(msg)
	case CompletedMsg:
		return m.handleCompleted(msg)
	case UninstallStageMsg:
		return m.handleUninstallStage(msg)
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case tea.WindowSizeMsg:
		return m.handleWindowSizeMsg(msg)
	case spinner.TickMsg:
		var cmd tea.Cmd

		m.spinner, cmd = m.spinner.Update(msg)

		return m, cmd
	}

	return m, nil
}

func (m *Progress) handleProgressMsg(msg ProgressMsg) (tea.Model, tea.Cmd) {
	// Find the task by name and update its progress
	for i := range m.tasks {
		if m.tasks[i].Name == msg.TaskName {
			m.tasks[i].Progress = msg.Progress
			m.tasks[i].Status = msg.Status
			m.tasks[i].Speed = msg.Speed
			m.tasks[i].ETA = msg.ETA

			// Update the progress bar
			if progressBar, exists := m.progressBars[msg.TaskName]; exists {
				progressBar.SetPercent(msg.Progress)
				m.progressBars[msg.TaskName] = progressBar
			}

			// Update overall progress
			m.updateOverallProgress()

			break
		}
	}

	return m, nil
}

func (m *Progress) handleProgressUpdateMsg(msg ProgressUpdateMsg) (tea.Model, tea.Cmd) {
	// Update task progress by index
	if msg.TaskIndex >= 0 && msg.TaskIndex < len(m.tasks) {
		m.updateTaskProgress(msg.TaskIndex, msg.Progress, msg.Message)

		// Add log entry for progress stages
		if msg.Message != "" {
			m.logs = append(m.logs, msg.Message)

			// Keep only last 10 log entries
			if len(m.logs) > 10 {
				m.logs = m.logs[len(m.logs)-10:]
			}
		}

		// Update overall progress
		m.updateOverallProgress()
	}

	return m, nil
}

func (m *Progress) handleCompleted(msg CompletedMsg) (tea.Model, tea.Cmd) {
	if errorModel := m.handleCompletedTask(msg); errorModel != nil {
		return errorModel, nil
	}

	// Continue with next task if not completed
	if !m.completed {
		return m, m.executeNextTask()
	}

	return m, nil
}

func (m *Progress) handleUninstallStage(msg UninstallStageMsg) (tea.Model, tea.Cmd) {
	if !m.isValidTaskIndex(msg.TaskIndex) {
		return m, nil
	}

	progress, status := m.getUninstallStageProgress(msg.Stage)
	m.updateTaskProgress(msg.TaskIndex, progress, status)
	m.updateProgressBar(msg.TaskIndex, progress)
	m.updateOverallProgress()
	m.addStageLogEntry(msg.Stage, status, msg.AppName)

	if msg.Stage < 5 {
		return m, m.createNextStageCmd(msg)
	}

	return m, nil
}

// isValidTaskIndex checks if the task index is within valid bounds.
func (m *Progress) isValidTaskIndex(taskIndex int) bool {
	return taskIndex >= 0 && taskIndex < len(m.tasks)
}

// getUninstallStageProgress returns progress and status for an uninstall stage.
func (m *Progress) getUninstallStageProgress(stage int) (float64, string) {
	switch stage {
	case 1:
		return 0.1, "Preparing to remove"
	case 2:
		return 0.3, "Stopping services"
	case 3:
		return 0.6, "Removing package files"
	case 4:
		return 0.85, "Cleaning up configuration"
	case 5:
		return 0.95, "Finalizing removal"
	default:
		return 1.0, msgUninstallationComplete
	}
}

// updateTaskProgress updates the progress and status of a task.
func (m *Progress) updateTaskProgress(taskIndex int, progress float64, status string) {
	m.tasks[taskIndex].Progress = progress
	m.tasks[taskIndex].Status = status
}

// updateProgressBar updates the progress bar for a task if it exists.
func (m *Progress) updateProgressBar(taskIndex int, progress float64) {
	if progressBar, exists := m.progressBars[m.tasks[taskIndex].Name]; exists {
		progressBar.SetPercent(progress)
		m.progressBars[m.tasks[taskIndex].Name] = progressBar
	}
}

// addStageLogEntry adds a log entry for an uninstall stage.
func (m *Progress) addStageLogEntry(stage int, status, appName string) {
	logMsg := fmt.Sprintf("Stage %d: %s (%s)", stage, status, appName)

	m.logs = append(m.logs, logMsg)
	if len(m.logs) > 10 {
		m.logs = m.logs[len(m.logs)-10:]
	}
}

// createNextStageCmd creates a command for the next uninstall stage.
func (m *Progress) createNextStageCmd(msg UninstallStageMsg) tea.Cmd {
	return func() tea.Msg {
		return UninstallStageMsg{
			TaskIndex: msg.TaskIndex,
			AppKey:    msg.AppKey,
			AppName:   msg.AppName,
			Stage:     msg.Stage + 1,
		}
	}
}

func (m *Progress) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m.handleQuit()
	case "p":
		return m.handlePauseToggle()
	case "l":
		return m.handleLogToggle()
	case KeyEsc:
		return m.handleEscape()
	}

	return m, nil
}

func (m *Progress) handleQuit() (tea.Model, tea.Cmd) {
	m.quitting = true

	return m, tea.Quit
}

func (m *Progress) handlePauseToggle() (tea.Model, tea.Cmd) {
	m.paused = !m.paused

	return m, nil
}

func (m *Progress) handleLogToggle() (tea.Model, tea.Cmd) {
	m.showingLogs = !m.showingLogs

	return m, nil
}

func (m *Progress) handleEscape() (tea.Model, tea.Cmd) {
	if m.completed {
		return m, func() tea.Msg {
			return NavigateMsg{Screen: AppsScreen, Data: CompletedOperationsMsg{Operations: m.operations}}
		}
	}

	return m, func() tea.Msg {
		return NavigateMsg{Screen: AppsScreen, Data: RefreshStatusData}
	}
}

func (m *Progress) handleWindowSizeMsg(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	return m, nil
}

// Legacy handleKeyInput - kept for compatibility.
//

// handleCompletedTask processes completed task message and returns error model if needed.
//

func (m *Progress) handleCompletedTask(msg CompletedMsg) tea.Model {
	// Mark task as completed
	for taskIndex := range m.tasks {
		if m.tasks[taskIndex].Name == msg.TaskName {
			// Handle task completion based on success/failure
			if errorModel := m.handleTaskCompletion(taskIndex, msg); errorModel != nil {
				return errorModel
			}

			m.tasks[taskIndex].Duration = msg.Duration
			m.addToLogs(taskIndex, msg)

			break
		}
	}

	// Check if all tasks are completed
	m.checkCompletion()
	m.updateOverallProgress()

	return nil
}

// addToLogs adds a log entry for the completed task.
//
//nolint:funcorder // Helper method grouped with related functionality
func (m *Progress) addToLogs(taskIndex int, msg CompletedMsg) {
	// Clean, user-friendly format without timestamps
	var logEntry string

	if msg.Success {
		// "Chrome installation completed"
		logEntry = fmt.Sprintf("%s%s",
			m.tasks[taskIndex].Name,
			m.getSuccessMessage(m.tasks[taskIndex].Operation))
	} else {
		// "Chrome installation failed: error details"
		logEntry = fmt.Sprintf("%s%s",
			m.tasks[taskIndex].Name,
			m.getFailureMessage(m.tasks[taskIndex].Operation, msg.Error))
	}

	m.logs = append(m.logs, logEntry)

	// Keep only last 10 log entries
	if len(m.logs) > 10 {
		m.logs = m.logs[len(m.logs)-10:]
	}
}

// View renders the progress screen.
func (m *Progress) View() string {
	if m.quitting && !m.completed {
		return "Installation cancelled.\n"
	}

	// Render components
	header := m.renderHeader()
	progress := m.renderProgress() // Combined progress box with everything
	logs := m.renderLogs()
	footer := m.renderFooter()

	// Build sections for vertical composition
	sections := []string{header, progress}

	// Add logs and footer
	sections = append(sections, logs, footer)

	baseView := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// If showing logs, overlay floating log viewer
	if m.showingLogs {
		floatingLogs := m.renderFloatingLogViewer()
		// Place floating logs over the base view (simplified overlay)
		return lipgloss.JoinVertical(lipgloss.Left, baseView, "", floatingLogs)
	}

	// Use lipgloss to compose layout with consistent spacing
	return baseView
}

// handleProgressUpdateMsg handles ProgressUpdateMsg updates for real-time progress.

// getSuccessMessage returns the appropriate success message for an operation.
func (m *Progress) getSuccessMessage(operation string) string {
	if operation == OperationUninstall {
		return " uninstalled"
	}

	return " installation completed"
}

// getFailureMessage returns the appropriate failure message for an operation.
func (m *Progress) getFailureMessage(operation, errorMsg string) string {
	if operation == OperationUninstall {
		return " uninstallation failed: " + errorMsg
	}

	return " installation failed: " + errorMsg
}

// renderHeader creates the header.
func (m *Progress) renderHeader() string {
	installCount, uninstallCount := m.getOperationCounts()

	title := m.getProgressTitle(installCount, uninstallCount)
	subtitle := m.getProgressSubtitle(installCount, uninstallCount)

	titleStyled := m.styles.Title.Render(title)
	subtitleStyled := m.styles.Subtitle.Render(subtitle)

	return lipgloss.JoinVertical(lipgloss.Left, titleStyled, subtitleStyled)
}

// getProgressTitle returns the appropriate title based on progress state.
func (m *Progress) getProgressTitle(installCount, uninstallCount int) string {
	switch {
	case m.completed:
		return m.getCompletedTitle(installCount, uninstallCount)
	case m.paused:
		return "⏸ Operations Paused"
	default:
		return m.getActiveTitle(installCount, uninstallCount)
	}
}

// getCompletedTitle returns the title for completed operations.
func (m *Progress) getCompletedTitle(installCount, uninstallCount int) string {
	switch {
	case installCount > 0 && uninstallCount > 0:
		return "✓ Operations Complete"
	case uninstallCount > 0:
		return "✓ Uninstallation Complete"
	default:
		return "✓ Installation Complete"
	}
}

// getActiveTitle returns the title for active operations.
func (m *Progress) getActiveTitle(installCount, uninstallCount int) string {
	switch {
	case installCount > 0 && uninstallCount > 0:
		return "⚬ Processing Applications"
	case uninstallCount > 0:
		return "⚬ Uninstalling Applications"
	default:
		return "⚬ Installing Applications"
	}
}

// getProgressSubtitle returns the subtitle showing operation counts.
func (m *Progress) getProgressSubtitle(installCount, uninstallCount int) string {
	switch {
	case installCount > 0 && uninstallCount > 0:
		return fmt.Sprintf("Installing %d, uninstalling %d applications", installCount, uninstallCount)
	case uninstallCount > 0:
		return fmt.Sprintf("Uninstalling %d applications", uninstallCount)
	default:
		return fmt.Sprintf("Installing %d applications", installCount)
	}
}

// renderProgress creates the combined progress display with border title.
func (m *Progress) renderProgress() string {
	availableWidth := m.getAvailableWidth()
	content := m.buildProgressContent(availableWidth)
	titleText := m.getProgressBorderTitle()

	return m.renderProgressWithBorder(content, titleText, availableWidth)
}

// getAvailableWidth calculates the available width for progress display.
func (m *Progress) getAvailableWidth() int {
	availableWidth := m.width - 4 // Account for padding
	if availableWidth < 40 {
		availableWidth = 40 // Minimum width
	}

	return availableWidth
}

// buildProgressContent creates the progress content sections.
func (m *Progress) buildProgressContent(availableWidth int) string {
	contentParts := make([]string, 0, 10) // Pre-allocate with estimated capacity

	// Add overall progress section
	overallSection := m.buildOverallProgressSection(availableWidth)
	contentParts = append(contentParts, overallSection...)

	// Add individual task progress bars
	taskSection := m.buildTaskProgressSection(availableWidth)
	contentParts = append(contentParts, taskSection...)

	// Add time information
	timeSection := m.buildTimeSection()
	contentParts = append(contentParts, timeSection...)

	return strings.Join(contentParts, "\n")
}

// buildOverallProgressSection creates the overall progress display.
func (m *Progress) buildOverallProgressSection(availableWidth int) []string {
	completed := m.getCompletedTaskCount()
	hasErrors := m.hasFailedTasks()

	overallText := fmt.Sprintf("Overall: %d/%d (%.0f%%)", completed, len(m.tasks), m.overallProgress*100)
	overallBar := m.styles.ContextualProgressBar(completed, len(m.tasks), availableWidth-15, hasErrors, m.completed)

	return []string{overallText, overallBar, ""} // Empty line for spacing
}

// buildTaskProgressSection creates individual task progress displays.
func (m *Progress) buildTaskProgressSection(availableWidth int) []string {
	progressBarWidth := availableWidth - 25 // Account for task name and status
	if progressBarWidth < 15 {
		progressBarWidth = 15
	}

	// Update progress bar widths
	for name, progressBar := range m.progressBars {
		progressBar.Width = progressBarWidth
		m.progressBars[name] = progressBar
	}

	taskLines := make([]string, 0, len(m.tasks)) // Pre-allocate with exact capacity

	for taskIndex, task := range m.tasks {
		taskLine := m.renderSingleTask(taskIndex, task)
		taskLines = append(taskLines, taskLine)
	}

	return taskLines
}

// buildTimeSection creates the time information display.
func (m *Progress) buildTimeSection() []string {
	elapsed := time.Since(m.startTime)
	timeInfo := fmt.Sprintf("Elapsed: %s", elapsed.Round(time.Second))

	if !m.completed && m.overallProgress > 0 && m.overallProgress < 1.0 {
		eta := time.Duration(float64(elapsed) / m.overallProgress * (1 - m.overallProgress))
		timeInfo += fmt.Sprintf(" • ETA: %s", eta.Round(time.Second))
	}

	timeStyled := lipgloss.NewStyle().Foreground(m.styles.Muted).Render(timeInfo)

	return []string{"", timeStyled} // Empty line for spacing
}

// getCompletedTaskCount returns the number of completed tasks.
func (m *Progress) getCompletedTaskCount() int {
	var completed int

	for _, task := range m.tasks {
		if task.Status == TaskStatusCompleted {
			completed++
		}
	}

	return completed
}

// hasFailedTasks checks if any tasks have failed.
func (m *Progress) hasFailedTasks() bool {
	for _, task := range m.tasks {
		if task.Status == TaskStatusFailed {
			return true
		}
	}

	return false
}

// getProgressBorderTitle returns the title for the progress border.
func (m *Progress) getProgressBorderTitle() string {
	installCount, uninstallCount := m.getOperationCounts()

	switch {
	case installCount > 0 && uninstallCount > 0:
		return "Installation/Deinstallation Progress"
	case uninstallCount > 0:
		return "Deinstallation Progress"
	default:
		return "Installation Progress"
	}
}

// renderProgressWithBorder wraps content in a styled border.
func (m *Progress) renderProgressWithBorder(content, title string, availableWidth int) string {
	styledContent := lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.Title.Render(title),
		"",
		content,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Primary).
		Padding(1, 2).
		Width(availableWidth).
		Render(styledContent)
}

// renderSingleTask renders a single task line with status, progress bar, and timing.
func (m *Progress) renderSingleTask(taskIndex int, task InstallTask) string {
	// Task description and status - first line
	statusIcon := m.getStatusIcon(task.Status)
	nameStyle := m.getTaskNameStyle(taskIndex, task.Status)
	taskLine := fmt.Sprintf("%s %s", statusIcon, task.Description)

	// Status text - right aligned on first line
	statusText := m.getTaskStatusText(task)

	// Use lipgloss to compose first line with proper spacing
	firstLine := lipgloss.JoinHorizontal(
		lipgloss.Left,
		nameStyle.Render(taskLine),
		lipgloss.NewStyle().Width(5).Render(" "), // Spacer
		lipgloss.NewStyle().Foreground(m.styles.Muted).Render(statusText),
	)

	// Progress bar - second line (always show for all tasks)
	// IDIOMATIC BUBBLE TEA: Use pre-created progress bars, no on-demand creation during rendering
	progressBar, exists := m.progressBars[task.Name]
	if !exists {
		// This should not happen if properly initialized - log error and skip
		return "ERROR: Progress bar not found for task " + task.Name
	}

	// Update progress bar width dynamically
	progressBarWidth := defaultProgressBarWidth
	if m.width > maxProgressWidth {
		progressBarWidth = m.width - progressBarPadding
	}

	progressBar.Width = progressBarWidth

	// Ensure progress is set correctly for completed tasks
	if task.Status == TaskStatusCompleted && task.Progress < 1.0 {
		task.Progress = 1.0
	}

	// PRAGMATIC SOLUTION: Use ViewAs() for immediate visual feedback
	// While we still trigger animations via SetPercent() for smooth transitions,
	// ViewAs() ensures users see the current progress immediately
	progressLine := "  " + progressBar.ViewAs(task.Progress)

	// Update the progress bar in the map
	m.progressBars[task.Name] = progressBar

	// Use pure Lipgloss vertical composition
	return lipgloss.JoinVertical(
		lipgloss.Left,
		firstLine,
		lipgloss.NewStyle().Foreground(m.styles.Muted).Render(progressLine),
	)
}

// getTaskNameStyle returns the appropriate style for a task name based on its status.
func (m *Progress) getTaskNameStyle(taskIndex int, status string) lipgloss.Style {
	switch {
	case status == TaskStatusCompleted:
		return lipgloss.NewStyle().Foreground(m.styles.Success)
	case status == TaskStatusFailed:
		return lipgloss.NewStyle().Foreground(m.styles.Error)
	case taskIndex == m.currentTask:
		return lipgloss.NewStyle().Foreground(m.styles.Primary).Bold(true)
	default:
		return m.styles.Unselected
	}
}

// getTaskStatusText returns the appropriate status text for a task.
func (m *Progress) getTaskStatusText(task InstallTask) string {
	switch {
	case task.Status == TaskStatusCompleted:
		return fmt.Sprintf("100%% (%s)", task.Duration.Round(time.Second))
	case task.Status == TaskStatusFailed:
		return "Failed"
	case task.Progress > 0:
		statusText := fmt.Sprintf("%.0f%%", task.Progress*100)
		if task.ETA != "" {
			statusText += " • " + task.ETA
		}

		return statusText
	default:
		return "Pending"
	}
}

// renderLogs creates the recent activity log display with scrolling capability.
func (m *Progress) renderLogs() string {
	if len(m.logs) == 0 {
		return ""
	}

	// Calculate available dimensions
	availableWidth := m.width - 4 // Account for padding
	if availableWidth < 40 {
		availableWidth = 40 // Minimum width
	}

	// Make the box 3 lines taller as requested
	availableHeight := 6 // 3 lines + 3 more = 6 lines for log content

	// Handle scrolling when logs overflow
	displayLogs := m.getDisplayLogs(availableHeight)

	// Build log content with proper line wrapping
	maxContentWidth := availableWidth - 6 // Account for border and padding
	wrappedLines := m.wrapLogLines(displayLogs, maxContentWidth, availableHeight)

	content := strings.Join(wrappedLines, "\n")

	// Add scroll indicator if there are more logs
	scrollIndicator := ""

	if len(m.logs) > availableHeight {
		hiddenCount := len(m.logs) - availableHeight
		scrollIndicator = fmt.Sprintf(" (+%d more)", hiddenCount)
	}

	// Use Lipgloss border with title and scroll indicator
	title := "Recent Activity" + scrollIndicator
	styledContent := lipgloss.JoinVertical(
		lipgloss.Left,
		m.styles.Title.Render(title),
		"",
		content,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(m.styles.Primary).
		Padding(1, 2).
		Width(availableWidth).
		Height(availableHeight + 4). // +4 for title, border, and padding
		Render(styledContent)
}

// getDisplayLogs returns the logs that should be displayed based on available height.
func (m *Progress) getDisplayLogs(availableHeight int) []string {
	if len(m.logs) <= availableHeight {
		// All logs fit, show them all
		return m.logs
	}
	// More logs than fit, show the most recent ones
	startIdx := len(m.logs) - availableHeight

	return m.logs[startIdx:]
}

// wrapLogLines wraps log lines and ensures proper height.
func (m *Progress) wrapLogLines(displayLogs []string, maxContentWidth, availableHeight int) []string {
	var wrappedLines []string

	for _, logLine := range displayLogs {
		if len(logLine) <= maxContentWidth {
			wrappedLines = append(wrappedLines, logLine)
		} else {
			// Wrap long lines
			for len(logLine) > maxContentWidth {
				wrappedLines = append(wrappedLines, logLine[:maxContentWidth])
				logLine = logLine[maxContentWidth:]
			}

			if len(logLine) > 0 {
				wrappedLines = append(wrappedLines, logLine)
			}
		}
	}

	// Fill with empty lines if needed to maintain consistent height
	for len(wrappedLines) < availableHeight {
		wrappedLines = append(wrappedLines, "")
	}

	// Trim if we have too many lines after wrapping
	if len(wrappedLines) > availableHeight {
		wrappedLines = wrappedLines[len(wrappedLines)-availableHeight:]
	}

	return wrappedLines
}

// renderFooter creates the footer with keybindings.
func (m *Progress) renderFooter() string {
	var keybindings []string

	if !m.completed {
		if m.paused {
			keybindings = append(keybindings, m.styles.Keybinding("p", "resume"))
		} else {
			keybindings = append(keybindings, m.styles.Keybinding("p", "pause"))
		}

		keybindings = append(keybindings, m.styles.Keybinding("l", "logs"))
		keybindings = append(keybindings, m.styles.Keybinding("q", "cancel"))
	} else {
		keybindings = append(keybindings, m.styles.Keybinding("esc", "back"))
		keybindings = append(keybindings, m.styles.Keybinding("q", "quit"))
	}

	footer := strings.Join(keybindings, "  ")

	return m.styles.Footer.Render(footer)
}

// getStatusIcon returns the appropriate icon for a task status.
func (m *Progress) getStatusIcon(status string) string {
	return m.styles.StatusIcon(status)
}

// updateOverallProgress calculates the overall installation progress.
func (m *Progress) updateOverallProgress() {
	if len(m.tasks) == 0 {
		m.overallProgress = 1.0

		return
	}

	total := 0.0
	for _, task := range m.tasks {
		total += task.Progress
	}

	m.overallProgress = total / float64(len(m.tasks))
}

// checkCompletion checks if all tasks are completed.
func (m *Progress) checkCompletion() {
	allDone := true

	for _, task := range m.tasks {
		if task.Status != TaskStatusCompleted && task.Status != TaskStatusFailed {
			allDone = false

			break
		}
	}

	m.completed = allDone
}

// executeInstallations starts the actual installation process.
func (m *Progress) executeInstallations() tea.Cmd {
	if m.paused || m.completed || len(m.tasks) == 0 {
		return nil
	}

	// Start the first task
	return m.executeNextTask()
}

// executeNextTask executes the next pending task.
func (m *Progress) executeNextTask() tea.Cmd {
	// Find next pending task
	for taskIndex, task := range m.tasks {
		if task.Status == TaskStatusPending {
			m.currentTask = taskIndex

			// Start the task based on operation type
			switch task.Operation {
			case OperationInstall:
				return m.executeInstallTask(task.Name, taskIndex)
			case OperationUninstall:
				return m.executeUninstallTask(task.Name, taskIndex)
			}
		}
	}

	// No more tasks - mark as completed
	m.completed = true

	return nil
}

// executeInstallTask executes an installation task with granular progress tracking.
func (m *Progress) executeInstallTask(appKey string, taskIndex int) tea.Cmd {
	// Start installation process - update status (log entries come from progress stages)
	m.tasks[taskIndex].Status = TaskStatusInstalling

	// Start installation with staged progress updates using Bubble Tea commands
	return m.startStagedInstallation(appKey, taskIndex)
}

// startStagedInstallation starts an installation with progressive updates.
func (m *Progress) startStagedInstallation(appKey string, taskIndex int) tea.Cmd {
	// Look up app in catalog first
	app, exists := apps.Apps[appKey]
	if !exists {
		return func() tea.Msg {
			return CompletedMsg{
				TaskName: appKey,
				Success:  false,
				Duration: time.Second,
				Error:    fmt.Sprintf("App %s not found in catalog", appKey),
			}
		}
	}

	// Start with immediate progress update (Stage 1: Preparing)
	return tea.Batch(
		func() tea.Msg {
			return ProgressUpdateMsg{
				TaskIndex: taskIndex,
				Progress:  0.1,
				Message:   app.Name + ": Preparing installation...",
			}
		},
		tea.Tick(time.Millisecond*500, func(_ time.Time) tea.Msg {
			return m.continueStage2(appKey, taskIndex, app)
		}),
	)
}

// continueStage2 continues with stage 2 of installation.
//

func (m *Progress) continueStage2(appKey string, taskIndex int, app apps.App) tea.Msg {
	return tea.Batch(
		func() tea.Msg {
			return ProgressUpdateMsg{
				TaskIndex: taskIndex,
				Progress:  0.3,
				Message:   app.Name + ": Downloading packages...",
			}
		},
		tea.Tick(time.Millisecond*800, func(_ time.Time) tea.Msg {
			return m.continueStage3(appKey, taskIndex, app)
		}),
	)()
}

// continueStage3 continues with stage 3 of installation.
//

func (m *Progress) continueStage3(appKey string, taskIndex int, app apps.App) tea.Msg {
	return tea.Batch(
		func() tea.Msg {
			return ProgressUpdateMsg{
				TaskIndex: taskIndex,
				Progress:  0.6,
				Message:   app.Name + ": Installing application...",
			}
		},
		tea.Tick(time.Millisecond*1200, func(_ time.Time) tea.Msg {
			return m.continueStage4(appKey, taskIndex, app)
		}),
	)()
}

// continueStage4 continues with stage 4 of installation.
//

func (m *Progress) continueStage4(appKey string, taskIndex int, app apps.App) tea.Msg {
	return tea.Batch(
		func() tea.Msg {
			return ProgressUpdateMsg{
				TaskIndex: taskIndex,
				Progress:  0.8,
				Message:   app.Name + ": Configuring application...",
			}
		},
		tea.Tick(time.Millisecond*600, func(_ time.Time) tea.Msg {
			return m.executeActualInstallation(appKey, taskIndex, app)
		}),
	)()
}

// executeActualInstallation performs the actual installation with real progress parsing.
//

func (m *Progress) executeActualInstallation(appKey string, taskIndex int, app apps.App) tea.Msg {
	startTime := time.Now()

	// Use the stored context for proper timeout and cancellation propagation
	ctx := m.ctx

	// Convert to domain package
	pkg := &domain.Package{
		Name:        appKey,
		Group:       app.Group,
		Description: app.Description,
		Method:      app.Method,
		Source:      app.Source,
	}

	// For now, simulate real progress - in future this should parse actual installer output
	// Send progress updates during installation
	progress, message, hasProgress := parseDpkgProgress("Setting up "+app.Name, app.Name)
	if hasProgress {
		// Send an intermediate progress update
		return tea.Batch(
			func() tea.Msg {
				return ProgressUpdateMsg{
					TaskIndex: taskIndex,
					Progress:  progress,
					Message:   message,
				}
			},
			func() tea.Msg {
				// Actually execute installation
				result, err := m.packageInstaller.Install(ctx, pkg)
				_ = result // Result contains additional metadata if needed

				if err != nil {
					return CompletedMsg{
						TaskName: appKey,
						Success:  false,
						Duration: time.Since(startTime),
						Error:    err.Error(),
					}
				}

				return CompletedMsg{
					TaskName: appKey,
					Success:  true,
					Duration: time.Since(startTime),
					Error:    "",
				}
			},
		)()
	}

	// Fallback to direct installation
	result, err := m.packageInstaller.Install(ctx, pkg)
	_ = result // Result contains additional metadata if needed

	if err != nil {
		return CompletedMsg{
			TaskName: appKey,
			Success:  false,
			Duration: time.Since(startTime),
			Error:    err.Error(),
		}
	}

	return CompletedMsg{
		TaskName: appKey,
		Success:  true,
		Duration: time.Since(startTime),
		Error:    "",
	}
}

// UninstallStageMsg represents a stage in the uninstallation process.
type UninstallStageMsg struct {
	TaskIndex int
	AppKey    string
	AppName   string
	Stage     int
}

// Private methods that are actually used (keeping these at the bottom)

// getOperationCounts returns the counts of install and uninstall operations.
func (m *Progress) getOperationCounts() (int, int) {
	var installCount, uninstallCount int

	for _, task := range m.tasks {
		switch task.Operation {
		case OperationInstall:
			installCount++
		case OperationUninstall:
			uninstallCount++
		}
	}

	return installCount, uninstallCount
}

// executeUninstallTask executes an uninstallation task with staged progression.
func (m *Progress) executeUninstallTask(appKey string, taskIndex int) tea.Cmd {
	// Start uninstallation process
	m.tasks[taskIndex].Status = TaskStatusUninstalling

	// Start staged uninstallation similar to installation
	return m.startStagedUninstallation(appKey, taskIndex)
}

// startStagedUninstallation starts an uninstallation with progressive updates.
func (m *Progress) startStagedUninstallation(appKey string, taskIndex int) tea.Cmd {
	// Look up app in catalog first
	app, exists := apps.Apps[appKey]
	if !exists {
		return func() tea.Msg {
			return CompletedMsg{
				TaskName: appKey,
				Success:  false,
				Duration: time.Second,
				Error:    fmt.Sprintf("App %s not found in catalog", appKey),
			}
		}
	}

	// Start with immediate progress update (Stage 1: Preparing uninstallation)
	return tea.Batch(
		func() tea.Msg {
			return ProgressUpdateMsg{
				TaskIndex: taskIndex,
				Progress:  0.2,
				Message:   app.Name + ": Preparing uninstallation...",
			}
		},
		tea.Tick(time.Millisecond*400, func(_ time.Time) tea.Msg {
			return m.continueUninstallStage2(appKey, taskIndex, app)
		}),
	)
}

// continueUninstallStage2 continues with stage 2 of uninstallation.
//

func (m *Progress) continueUninstallStage2(appKey string, taskIndex int, app apps.App) tea.Msg {
	return tea.Batch(
		func() tea.Msg {
			return ProgressUpdateMsg{
				TaskIndex: taskIndex,
				Progress:  0.4,
				Message:   app.Name + ": Checking dependencies...",
			}
		},
		tea.Tick(time.Millisecond*600, func(_ time.Time) tea.Msg {
			return m.continueUninstallStage3(appKey, taskIndex, app)
		}),
	)()
}

// continueUninstallStage3 continues with stage 3 of uninstallation.
//

func (m *Progress) continueUninstallStage3(appKey string, taskIndex int, app apps.App) tea.Msg {
	return tea.Batch(
		func() tea.Msg {
			return ProgressUpdateMsg{
				TaskIndex: taskIndex,
				Progress:  0.6,
				Message:   app.Name + ": Removing package...",
			}
		},
		tea.Tick(time.Millisecond*800, func(_ time.Time) tea.Msg {
			return m.continueUninstallStage4(appKey, taskIndex, app)
		}),
	)()
}

// continueUninstallStage4 continues with stage 4 of uninstallation.
//

func (m *Progress) continueUninstallStage4(appKey string, taskIndex int, app apps.App) tea.Msg {
	return tea.Batch(
		func() tea.Msg {
			return ProgressUpdateMsg{
				TaskIndex: taskIndex,
				Progress:  0.8,
				Message:   app.Name + ": Cleaning up configuration...",
			}
		},
		tea.Tick(time.Millisecond*400, func(_ time.Time) tea.Msg {
			return m.executeActualUninstallation(appKey, taskIndex, app)
		}),
	)()
}

// executeActualUninstallation performs the actual uninstallation after showing progress.
//

func (m *Progress) executeActualUninstallation(appKey string, taskIndex int, app apps.App) tea.Msg {
	startTime := time.Now()

	// Use the stored context for proper timeout and cancellation propagation
	ctx := m.ctx

	// Check if we can parse dpkg output for more detailed progress
	progress, message, hasProgress := parseDpkgUninstallProgress("Removing "+app.Name, app.Name)
	if hasProgress {
		// Send an intermediate progress update based on dpkg output
		return tea.Batch(
			func() tea.Msg {
				return ProgressUpdateMsg{
					TaskIndex: taskIndex,
					Progress:  progress,
					Message:   message,
				}
			},
			func() tea.Msg {
				// Actually execute uninstallation
				err := m.uninstaller.UninstallApp(ctx, appKey)
				if err != nil {
					return CompletedMsg{
						TaskName: appKey,
						Success:  false,
						Duration: time.Since(startTime),
						Error:    err.Error(),
					}
				}

				return CompletedMsg{
					TaskName: appKey,
					Success:  true,
					Duration: time.Since(startTime),
					Error:    "",
				}
			},
		)()
	}

	// Fallback to direct uninstallation
	err := m.uninstaller.UninstallApp(ctx, appKey)
	if err != nil {
		return CompletedMsg{
			TaskName: appKey,
			Success:  false,
			Duration: time.Since(startTime),
			Error:    err.Error(),
		}
	}

	return CompletedMsg{
		TaskName: appKey,
		Success:  true,
		Duration: time.Since(startTime),
		Error:    "",
	}
}

// handleTaskCompletion handles task completion and returns error model if needed.
//

func (m *Progress) handleTaskCompletion(taskIndex int, msg CompletedMsg) tea.Model {
	if msg.Success {
		m.tasks[taskIndex].Status = TaskStatusCompleted
		m.tasks[taskIndex].Progress = 1.0
	} else {
		m.tasks[taskIndex].Status = TaskStatusFailed
		m.tasks[taskIndex].Error = msg.Error

		// Only return error screen for critical system errors, not installation failures
		// Regular installation failures should just show in the progress screen and logs
		if strings.Contains(msg.Error, "CRITICAL") || strings.Contains(msg.Error, "SYSTEM") {
			return &ErrorModel{
				ErrorMessage: msg.Error,
				TaskName:     msg.TaskName,
			}
		}
	}

	return nil
}

// ErrorModel represents a simple error screen for failed installations.
type ErrorModel struct {
	ErrorMessage string
	TaskName     string
}

// Init implements tea.Model interface.
func (e *ErrorModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model interface.
//

// Update handles messages for the ErrorModel.
func (e *ErrorModel) Update(_ tea.Msg) (tea.Model, tea.Cmd) {
	return e, nil
}

// View implements tea.Model interface.
func (e *ErrorModel) View() string {
	return fmt.Sprintf("Error installing %s: %s", e.TaskName, e.ErrorMessage)
}

// renderFloatingLogViewer renders a floating log viewer overlay.
func (m *Progress) renderFloatingLogViewer() string {
	// Simple implementation for floating logs
	content := strings.Join(m.logs, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(m.styles.Primary).
		Padding(1).
		Render("Floating Logs:\n\n" + content)
}

// GetTasksForTesting returns tasks for testing purposes.
func (m *Progress) GetTasksForTesting() []InstallTask {
	return m.tasks
}

// parseDpkgProgress parses dpkg output and returns progress information.
func parseDpkgProgress(output, appName string) (float64, string, bool) {
	// Trim whitespace from output
	output = strings.TrimSpace(output)

	if output == "" {
		return 0, "", false
	}

	// Check for dpkg installation patterns with specific progress values
	if progress, message, found := checkInstallBasicStages(output, appName); found {
		return progress, message, true
	}

	if progress, message, found := checkInstallSetupStage(output, appName); found {
		return progress, message, true
	}

	if progress, message, found := checkInstallTriggerStage(output, appName); found {
		return progress, message, true
	}

	// No match found
	return 0, "", false
}

func checkInstallBasicStages(output, appName string) (float64, string, bool) {
	// Stage 1: Selecting package (62%)
	if strings.Contains(output, "Selecting previously unselected package") {
		return 0.62, "Selecting " + appName + " package", true
	}

	// Stage 2: Reading database (65%)
	if strings.Contains(output, "Reading database") {
		return 0.65, "Reading package database", true
	}

	// Stage 3: Preparing to unpack (68%)
	if strings.Contains(output, "Preparing to unpack") {
		return 0.68, "Preparing to unpack " + appName, true
	}

	// Stage 4: Unpacking (72%)
	if strings.Contains(output, "Unpacking") && !strings.Contains(output, "Preparing") {
		return 0.72, "Unpacking " + appName + " package", true
	}

	return 0, "", false
}

func checkInstallSetupStage(output, appName string) (float64, string, bool) {
	// Stage 5: Setting up (75%)
	if strings.Contains(output, "Setting up") {
		message := "Setting up " + appName
		// Handle empty app name case
		if appName == "" {
			message = "Setting up "
		}

		return 0.75, message, true
	}

	// Stage 6: Update alternatives (92%)
	if strings.Contains(output, "update-alternatives:") {
		return 0.92, "Configuring " + appName + " alternatives", true
	}

	return 0, "", false
}

func checkInstallTriggerStage(output, _ string) (float64, string, bool) {
	// Stage 7: Processing triggers (96-100%)
	if strings.Contains(output, "Processing triggers") {
		return checkSpecificTriggerType(output)
	}

	return 0, "", false
}

// checkSpecificTriggerType checks for specific trigger types and returns appropriate progress.
func checkSpecificTriggerType(output string) (float64, string, bool) {
	if strings.Contains(output, "mailcap") {
		return 0.96, "Processing MIME type triggers", true
	}

	if strings.Contains(output, "gnome-menus") {
		return 0.97, "Processing GNOME menu triggers", true
	}

	if strings.Contains(output, "desktop-file-utils") {
		return 0.98, "Processing desktop file triggers", true
	}

	if strings.Contains(output, "man-db") {
		return 0.99, "Processing manual page triggers", true
	}

	if strings.Contains(output, "menu") && !strings.Contains(output, "gnome-menus") {
		return 1.0, "Processing menu triggers", true
	}

	// Generic trigger processing
	return 0.96, "Processing system triggers", true
}

// parseDpkgUninstallProgress parses dpkg uninstall output and returns progress information.
func parseDpkgUninstallProgress(output, appName string) (float64, string, bool) {
	// Trim whitespace from output
	output = strings.TrimSpace(output)

	if output == "" {
		return 0, "", false
	}

	// Check for dpkg uninstall patterns with specific progress values
	if progress, message, found := checkUninstallBasicStages(output, appName); found {
		return progress, message, true
	}

	if progress, message, found := checkUninstallRemovalStage(output, appName); found {
		return progress, message, true
	}

	if progress, message, found := checkUninstallTriggerStage(output); found {
		return progress, message, true
	}

	if progress, message, found := checkUninstallFinalStages(output); found {
		return progress, message, true
	}

	// No match found
	return 0, "", false
}

func checkUninstallBasicStages(output string, appName string) (float64, string, bool) {
	// Stage 1: Reading package lists (25%)
	if strings.Contains(output, "Reading package lists") {
		return 0.25, "Reading package lists", true
	}

	// Stage 2: Building dependency tree (35%)
	if strings.Contains(output, "Building dependency tree") {
		return 0.35, "Building dependency tree", true
	}

	// Stage 3: Reading state information (45%)
	if strings.Contains(output, "Reading state information") {
		return 0.45, "Reading state information", true
	}

	// Stage 4: Preparing to remove (55%)
	if strings.Contains(output, "Preparing to remove") {
		message := "Preparing to remove " + appName
		if appName == "" {
			message = "Preparing to remove"
		}

		return 0.55, message, true
	}

	return 0, "", false
}

func checkUninstallRemovalStage(output, appName string) (float64, string, bool) {
	// Stage 5: Removing package (65%)
	if strings.Contains(output, "Removing") && (strings.Contains(output, appName) || appName == "") {
		message := "Removing " + appName
		if appName == "" {
			message = "Removing package"
		}

		return 0.65, message, true
	}

	return 0, "", false
}

func checkUninstallTriggerStage(output string) (float64, string, bool) {
	// Stage 6: Processing triggers (75-85%)
	if strings.Contains(output, "Processing triggers") {
		if strings.Contains(output, "man-db") {
			return 0.75, "Processing manual page triggers", true
		}

		if strings.Contains(output, "desktop-file-utils") {
			return 0.78, "Processing desktop file triggers", true
		}

		if strings.Contains(output, "gnome-menus") {
			return 0.80, "Processing GNOME menu triggers", true
		}

		if strings.Contains(output, "mailcap") {
			return 0.82, "Processing MIME type triggers", true
		}
		// Generic trigger processing
		return 0.75, "Processing system triggers", true
	}

	return 0, "", false
}

func checkUninstallFinalStages(output string) (float64, string, bool) {
	// Stage 7: Purging configuration (90%)
	if strings.Contains(output, "Purging configuration files") {
		return 0.90, "Purging configuration files", true
	}

	// Stage 8: dpkg warnings about dependencies (95%)
	if strings.Contains(output, "dpkg: warning") && strings.Contains(output, "removing") {
		return 0.95, "Checking dependencies", true
	}

	// Stage 9: Completion indicators (100%)
	if strings.Contains(output, "removed") {
		return 1.0, msgUninstallationComplete, true
	}

	return 0, "", false
}
