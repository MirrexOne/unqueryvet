// Package tui provides interactive terminal UI for fixing SELECT * issues.
package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Issue represents a SELECT * issue found in code.
type Issue struct {
	File       string
	Line       int
	Column     int
	Message    string
	CodeLine   string
	Suggestion string
	Type       string // Issue type for smart fix
}

// Model is the main TUI model.
type Model struct {
	issues       []Issue
	currentIndex int
	quitting     bool
	width        int
	height       int
	applied      map[int]bool   // Track which issues have been fixed
	skipped      map[int]bool   // Track which issues have been skipped
	actions      []string       // Action log for display
	history      *ActionHistory // Action history for undo
	showPreview  bool           // Toggle diff preview
	showHelp     bool           // Toggle full help
	startTime    time.Time      // Session start time
}

// KeyMap defines key bindings.
type KeyMap struct {
	Apply key.Binding
	Skip  key.Binding
	Prev  key.Binding
	Next  key.Binding
	Quit  key.Binding
	Help  key.Binding
}

// DefaultKeyMap returns default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Apply: key.NewBinding(
			key.WithKeys("enter", "a"),
			key.WithHelp("enter/a", "apply fix"),
		),
		Skip: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "skip"),
		),
		Prev: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "previous"),
		),
		Next: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "next"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q/esc", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

var keys = DefaultKeyMap()

// Styles for TUI rendering.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	issueHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("39"))

	fileStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))

	lineNumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	codeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1)

	suggestionStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("22")).
			Foreground(lipgloss.Color("46")).
			Padding(0, 1)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1)

	appliedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("46")).
			Bold(true)

	skippedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	currentStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Padding(1, 2).
			MarginBottom(1)
)

// NewModel creates a new TUI model with issues.
func NewModel(issues []Issue) Model {
	// Assign types to issues if not set
	for i := range issues {
		if issues[i].Type == "" {
			issues[i].Type = GetIssueType(issues[i])
		}
	}

	return Model{
		issues:       issues,
		currentIndex: 0,
		applied:      make(map[int]bool),
		skipped:      make(map[int]bool),
		actions:      []string{},
		history:      NewActionHistory(100),
		showPreview:  true,
		showHelp:     false,
		startTime:    time.Now(),
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		keyStr := msg.String()

		switch {
		case key.Matches(msg, keys.Quit):
			m.quitting = true
			return m, tea.Quit

		case key.Matches(msg, keys.Apply):
			if len(m.issues) > 0 && !m.applied[m.currentIndex] && !m.skipped[m.currentIndex] {
				issue := m.issues[m.currentIndex]
				originalCode := issue.CodeLine

				// Apply the fix
				if err := applyFix(issue); err != nil {
					m.actions = append(m.actions, fmt.Sprintf("Error: %v", err))
				} else {
					m.applied[m.currentIndex] = true
					m.actions = append(m.actions, fmt.Sprintf("Applied fix for %s:%d", issue.File, issue.Line))

					// Record in history for undo
					m.history.Push(Action{
						Type:      ActionApply,
						IssueIdx:  m.currentIndex,
						Timestamp: time.Now(),
						File:      issue.File,
						Line:      issue.Line,
						Original:  originalCode,
						Modified:  issue.Suggestion,
					})
				}
				// Move to next unfixed issue
				m.moveToNextUnfixed()
			}

		case key.Matches(msg, keys.Skip):
			if len(m.issues) > 0 && !m.applied[m.currentIndex] && !m.skipped[m.currentIndex] {
				issue := m.issues[m.currentIndex]
				m.skipped[m.currentIndex] = true
				m.actions = append(m.actions, fmt.Sprintf("Skipped %s:%d", issue.File, issue.Line))

				// Record in history for undo
				m.history.Push(Action{
					Type:      ActionSkip,
					IssueIdx:  m.currentIndex,
					Timestamp: time.Now(),
					File:      issue.File,
					Line:      issue.Line,
				})
				m.moveToNextUnfixed()
			}

		case key.Matches(msg, keys.Prev):
			if m.currentIndex > 0 {
				m.currentIndex--
			}

		case key.Matches(msg, keys.Next):
			if m.currentIndex < len(m.issues)-1 {
				m.currentIndex++
			}

		// New keybindings
		case keyStr == "u": // Undo
			if m.history.CanUndo() {
				if err := UndoLastAction(&m); err != nil {
					m.actions = append(m.actions, fmt.Sprintf("Undo error: %v", err))
				} else {
					m.actions = append(m.actions, "Undid last action")
				}
			}

		case keyStr == "p": // Toggle preview
			m.showPreview = !m.showPreview

		case keyStr == "?": // Toggle help
			m.showHelp = !m.showHelp

		case keyStr == "A": // Apply all
			errors := BatchApplyAll(&m)
			m.actions = append(m.actions, fmt.Sprintf("Applied all remaining (%d errors)", len(errors)))

		case keyStr == "S": // Skip all
			BatchSkipAll(&m)
			m.actions = append(m.actions, "Skipped all remaining")

		case keyStr == "R": // Reset all
			errors := ResetAllActions(&m)
			m.actions = append(m.actions, fmt.Sprintf("Reset all actions (%d errors)", len(errors)))

		case keyStr == "e": // Export to JSON
			result := m.buildExportResult()
			filename := fmt.Sprintf("unqueryvet-results-%s.json", time.Now().Format("20060102-150405"))
			if err := ExportToJSON(filename, result); err != nil {
				m.actions = append(m.actions, fmt.Sprintf("Export error: %v", err))
			} else {
				m.actions = append(m.actions, fmt.Sprintf("Exported to %s", filename))
			}

		case keyStr == "g": // Go to first
			m.currentIndex = 0

		case keyStr == "G": // Go to last
			m.currentIndex = len(m.issues) - 1
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// buildExportResult creates an export result from current state.
func (m Model) buildExportResult() ExportResult {
	issues := make([]ExportIssue, len(m.issues))
	for i, issue := range m.issues {
		status := "pending"
		if m.applied[i] {
			status = "applied"
		} else if m.skipped[i] {
			status = "skipped"
		}

		issues[i] = ExportIssue{
			File:       issue.File,
			Line:       issue.Line,
			Column:     issue.Column,
			Message:    issue.Message,
			Type:       issue.Type,
			Status:     status,
			Suggestion: issue.Suggestion,
		}
	}

	return ExportResult{
		Timestamp:    time.Now(),
		TotalIssues:  len(m.issues),
		AppliedCount: len(m.applied),
		SkippedCount: len(m.skipped),
		Issues:       issues,
		Actions:      m.history.All(),
		Duration:     time.Since(m.startTime).String(),
	}
}

// moveToNextUnfixed moves to the next unfixed issue.
func (m *Model) moveToNextUnfixed() {
	start := m.currentIndex
	for i := 0; i < len(m.issues); i++ {
		idx := (start + i + 1) % len(m.issues)
		if !m.applied[idx] && !m.skipped[idx] {
			m.currentIndex = idx
			return
		}
	}
}

// View renders the model.
func (m Model) View() string {
	if m.quitting {
		return m.renderSummary()
	}

	if len(m.issues) == 0 {
		return successStyle.Render("✓ No SELECT * issues found!")
	}

	// Show full help if toggled
	if m.showHelp {
		var b strings.Builder
		b.WriteString(titleStyle.Render("🔍 SELECT * Fixer"))
		b.WriteString("\n\n")
		b.WriteString(RenderHelpFull())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Press ? to close help"))
		return b.String()
	}

	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("🔍 SELECT * Fixer"))
	b.WriteString("\n\n")

	// Progress with batch info
	b.WriteString(RenderBatchInfo(len(m.issues), len(m.applied), len(m.skipped), m.history.CanUndo()))
	b.WriteString("\n\n")

	// Current issue
	issue := m.issues[m.currentIndex]

	// Status indicator
	var statusIndicator string
	if m.applied[m.currentIndex] {
		statusIndicator = appliedStyle.Render(" [APPLIED]")
	} else if m.skipped[m.currentIndex] {
		statusIndicator = skippedStyle.Render(" [SKIPPED]")
	}

	issueContent := m.renderIssue(issue, statusIndicator)
	b.WriteString(currentStyle.Render(issueContent))

	// Show diff preview if enabled and there's a suggestion
	if m.showPreview && issue.Suggestion != "" && !m.applied[m.currentIndex] {
		b.WriteString("\n")
		b.WriteString(RenderDiffInline(issue.CodeLine, issue.Suggestion))
		b.WriteString("\n")
	}

	// Smart fix suggestion
	if !m.applied[m.currentIndex] && !m.skipped[m.currentIndex] {
		b.WriteString("\n")
		b.WriteString(RenderSmartFixSuggestion(issue.Type, issue.Suggestion))
	}

	// Help
	b.WriteString("\n")
	helpText := "↑/↓ navigate • a apply • s skip • u undo • A apply all • S skip all • p preview • e export • ? help • q quit"
	b.WriteString(helpStyle.Render(helpText))

	// Recent actions
	if len(m.actions) > 0 {
		b.WriteString("\n\n")
		b.WriteString(lineNumberStyle.Render("Recent actions:"))
		b.WriteString("\n")
		start := 0
		if len(m.actions) > 3 {
			start = len(m.actions) - 3
		}
		for _, action := range m.actions[start:] {
			b.WriteString("  " + action + "\n")
		}
	}

	return b.String()
}

// renderIssue renders a single issue.
func (m Model) renderIssue(issue Issue, status string) string {
	var b strings.Builder

	// File and line
	b.WriteString(issueHeaderStyle.Render("📍 Location") + status + "\n")
	b.WriteString(fileStyle.Render(issue.File))
	b.WriteString(lineNumberStyle.Render(fmt.Sprintf(":%d:%d", issue.Line, issue.Column)))
	b.WriteString("\n\n")

	// Message
	b.WriteString(warningStyle.Render("⚠ " + issue.Message))
	b.WriteString("\n\n")

	// Code
	b.WriteString(issueHeaderStyle.Render("📝 Current code"))
	b.WriteString("\n")
	b.WriteString(codeStyle.Render(issue.CodeLine))
	b.WriteString("\n\n")

	// Suggestion
	if issue.Suggestion != "" {
		b.WriteString(issueHeaderStyle.Render("💡 Suggestion"))
		b.WriteString("\n")
		b.WriteString(suggestionStyle.Render(issue.Suggestion))
	} else {
		b.WriteString(issueHeaderStyle.Render("💡 Suggestion"))
		b.WriteString("\n")
		b.WriteString(suggestionStyle.Render("Replace SELECT * with explicit column names"))
	}

	return b.String()
}

// renderSummary renders the exit summary.
func (m Model) renderSummary() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("📊 Summary"))
	b.WriteString("\n\n")

	applied := len(m.applied)
	skipped := len(m.skipped)
	total := len(m.issues)

	if applied > 0 {
		b.WriteString(successStyle.Render(fmt.Sprintf("✓ Applied fixes: %d", applied)))
		b.WriteString("\n")
	}
	if skipped > 0 {
		b.WriteString(skippedStyle.Render(fmt.Sprintf("→ Skipped: %d", skipped)))
		b.WriteString("\n")
	}
	if total-applied-skipped > 0 {
		b.WriteString(warningStyle.Render(fmt.Sprintf("○ Remaining: %d", total-applied-skipped)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString("Goodbye! 👋\n")

	return b.String()
}

// applyFix applies a fix to the issue.
func applyFix(issue Issue) error {
	// Read file
	data, err := os.ReadFile(issue.File)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	if issue.Line-1 >= len(lines) {
		return fmt.Errorf("line %d out of range", issue.Line)
	}

	// Apply suggestion if available
	if issue.Suggestion != "" {
		lines[issue.Line-1] = issue.Suggestion
	} else {
		// Default: add TODO comment
		lines[issue.Line-1] = lines[issue.Line-1] + " // TODO: replace SELECT * with explicit columns"
	}

	// Write back
	output := strings.Join(lines, "\n")
	if err := os.WriteFile(issue.File, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Run starts the interactive TUI.
func Run(issues []Issue) error {
	if len(issues) == 0 {
		fmt.Println(successStyle.Render("✓ No SELECT * issues found!"))
		return nil
	}

	p := tea.NewProgram(NewModel(issues), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
