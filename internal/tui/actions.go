// Package tui provides interactive terminal UI for fixing SELECT * issues.
package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Action represents an action taken on an issue.
type Action struct {
	Type      ActionType `json:"type"`
	IssueIdx  int        `json:"issue_idx"`
	Timestamp time.Time  `json:"timestamp"`
	File      string     `json:"file"`
	Line      int        `json:"line"`
	Original  string     `json:"original,omitempty"`
	Modified  string     `json:"modified,omitempty"`
}

// ActionType represents the type of action.
type ActionType string

const (
	ActionApply ActionType = "apply"
	ActionSkip  ActionType = "skip"
	ActionUndo  ActionType = "undo"
	ActionReset ActionType = "reset"
)

// ActionHistory manages the history of actions for undo support.
type ActionHistory struct {
	actions []Action
	maxSize int
}

// NewActionHistory creates a new action history.
func NewActionHistory(maxSize int) *ActionHistory {
	return &ActionHistory{
		actions: make([]Action, 0),
		maxSize: maxSize,
	}
}

// Push adds an action to the history.
func (h *ActionHistory) Push(action Action) {
	h.actions = append(h.actions, action)
	if len(h.actions) > h.maxSize {
		h.actions = h.actions[1:]
	}
}

// Pop removes and returns the last action.
func (h *ActionHistory) Pop() (Action, bool) {
	if len(h.actions) == 0 {
		return Action{}, false
	}
	action := h.actions[len(h.actions)-1]
	h.actions = h.actions[:len(h.actions)-1]
	return action, true
}

// Peek returns the last action without removing it.
func (h *ActionHistory) Peek() (Action, bool) {
	if len(h.actions) == 0 {
		return Action{}, false
	}
	return h.actions[len(h.actions)-1], true
}

// CanUndo returns true if there's an action to undo.
func (h *ActionHistory) CanUndo() bool {
	return len(h.actions) > 0
}

// Clear clears all history.
func (h *ActionHistory) Clear() {
	h.actions = make([]Action, 0)
}

// All returns all actions.
func (h *ActionHistory) All() []Action {
	return h.actions
}

// Len returns the number of actions in history.
func (h *ActionHistory) Len() int {
	return len(h.actions)
}

// ExportResult represents the result of a TUI session for export.
type ExportResult struct {
	Timestamp    time.Time     `json:"timestamp"`
	TotalIssues  int           `json:"total_issues"`
	AppliedCount int           `json:"applied_count"`
	SkippedCount int           `json:"skipped_count"`
	Issues       []ExportIssue `json:"issues"`
	Actions      []Action      `json:"actions"`
	Duration     string        `json:"duration"`
}

// ExportIssue represents an issue in the export format.
type ExportIssue struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Column     int    `json:"column"`
	Message    string `json:"message"`
	Type       string `json:"type"`
	Status     string `json:"status"` // "applied", "skipped", "pending"
	Suggestion string `json:"suggestion,omitempty"`
}

// ExportToJSON exports the session results to a JSON file.
func ExportToJSON(filename string, result ExportResult) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// BatchApplyAll applies all remaining (non-applied, non-skipped) issues.
func BatchApplyAll(m *Model) []error {
	var errors []error

	for i := range m.issues {
		if !m.applied[i] && !m.skipped[i] {
			if err := applyFix(m.issues[i]); err != nil {
				errors = append(errors, fmt.Errorf("issue %d: %w", i, err))
			} else {
				m.applied[i] = true
				m.history.Push(Action{
					Type:      ActionApply,
					IssueIdx:  i,
					Timestamp: time.Now(),
					File:      m.issues[i].File,
					Line:      m.issues[i].Line,
					Original:  m.issues[i].CodeLine,
					Modified:  m.issues[i].Suggestion,
				})
			}
		}
	}

	return errors
}

// BatchSkipAll skips all remaining (non-applied, non-skipped) issues.
func BatchSkipAll(m *Model) {
	for i := range m.issues {
		if !m.applied[i] && !m.skipped[i] {
			m.skipped[i] = true
			m.history.Push(Action{
				Type:      ActionSkip,
				IssueIdx:  i,
				Timestamp: time.Now(),
				File:      m.issues[i].File,
				Line:      m.issues[i].Line,
			})
		}
	}
}

// UndoLastAction undoes the last action.
func UndoLastAction(m *Model) error {
	action, ok := m.history.Pop()
	if !ok {
		return fmt.Errorf("nothing to undo")
	}

	switch action.Type {
	case ActionApply:
		// Restore the original content
		if action.Original != "" {
			if err := restoreOriginal(action.File, action.Line, action.Original); err != nil {
				return fmt.Errorf("failed to restore: %w", err)
			}
		}
		m.applied[action.IssueIdx] = false
		m.currentIndex = action.IssueIdx

	case ActionSkip:
		m.skipped[action.IssueIdx] = false
		m.currentIndex = action.IssueIdx
	}

	return nil
}

// ResetAllActions resets all applied and skipped states.
func ResetAllActions(m *Model) []error {
	var errors []error

	// Restore all applied changes
	for i := range m.issues {
		if m.applied[i] {
			if err := restoreOriginal(m.issues[i].File, m.issues[i].Line, m.issues[i].CodeLine); err != nil {
				errors = append(errors, fmt.Errorf("issue %d: %w", i, err))
			}
		}
	}

	// Clear state
	m.applied = make(map[int]bool)
	m.skipped = make(map[int]bool)
	m.history.Clear()
	m.currentIndex = 0

	return errors
}

// restoreOriginal restores the original line content.
func restoreOriginal(filename string, lineNum int, original string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")
	if lineNum-1 >= len(lines) {
		return fmt.Errorf("line %d out of range", lineNum)
	}

	lines[lineNum-1] = original
	output := strings.Join(lines, "\n")

	return os.WriteFile(filename, []byte(output), 0644)
}

// GetIssueType determines the type of issue for smart fix suggestions.
func GetIssueType(issue Issue) string {
	msg := strings.ToLower(issue.Message)

	switch {
	case strings.Contains(msg, "select *") || strings.Contains(msg, "select_star"):
		return "select_star"
	case strings.Contains(msg, ".*") || strings.Contains(msg, "aliased"):
		return "aliased_wildcard"
	case strings.Contains(msg, "find") || strings.Contains(msg, "all()"):
		return "orm_find_all"
	case strings.Contains(msg, "builder") || strings.Contains(msg, "squirrel") ||
		strings.Contains(msg, "gorm") || strings.Contains(msg, "bun"):
		return "sql_builder_star"
	default:
		return "unknown"
	}
}

// GenerateSmartSuggestion generates a smart fix suggestion based on issue type.
func GenerateSmartSuggestion(issue Issue) string {
	issueType := GetIssueType(issue)

	switch issueType {
	case "select_star":
		// Try to extract table name and suggest explicit columns
		return suggestExplicitColumns(issue.CodeLine)
	case "aliased_wildcard":
		return suggestAliasedColumns(issue.CodeLine)
	case "orm_find_all":
		return suggestORMSelect(issue.CodeLine)
	case "sql_builder_star":
		return suggestBuilderColumns(issue.CodeLine)
	default:
		return issue.Suggestion
	}
}

func suggestExplicitColumns(codeLine string) string {
	// Simple heuristic: replace SELECT * with SELECT col1, col2, col3
	if strings.Contains(strings.ToUpper(codeLine), "SELECT *") {
		return strings.Replace(
			strings.Replace(codeLine, "SELECT *", "SELECT col1, col2, col3", 1),
			"select *", "SELECT col1, col2, col3", 1,
		)
	}
	return codeLine + " // TODO: specify columns"
}

func suggestAliasedColumns(codeLine string) string {
	// Replace t.* with t.col1, t.col2
	return codeLine + " // TODO: replace alias.* with explicit columns"
}

func suggestORMSelect(codeLine string) string {
	// Suggest adding .Select() before Find/All
	return codeLine + " // TODO: add .Select(\"col1\", \"col2\") before query"
}

func suggestBuilderColumns(codeLine string) string {
	// Suggest replacing "*" with column names
	return strings.Replace(codeLine, `"*"`, `"col1", "col2"`, 1)
}
