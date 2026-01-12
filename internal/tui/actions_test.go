package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestActionTypes(t *testing.T) {
	tests := []struct {
		action ActionType
		want   string
	}{
		{ActionApply, "apply"},
		{ActionSkip, "skip"},
		{ActionUndo, "undo"},
		{ActionReset, "reset"},
	}

	for _, tt := range tests {
		if string(tt.action) != tt.want {
			t.Errorf("ActionType %v = %s, want %s", tt.action, tt.action, tt.want)
		}
	}
}

func TestActionStruct(t *testing.T) {
	now := time.Now()
	action := Action{
		Type:      ActionApply,
		IssueIdx:  5,
		Timestamp: now,
		File:      "test.go",
		Line:      10,
		Original:  "old code",
		Modified:  "new code",
	}

	if action.Type != ActionApply {
		t.Errorf("Type = %s, want 'apply'", action.Type)
	}
	if action.IssueIdx != 5 {
		t.Errorf("IssueIdx = %d, want 5", action.IssueIdx)
	}
	if action.File != "test.go" {
		t.Errorf("File = %s, want 'test.go'", action.File)
	}
	if action.Line != 10 {
		t.Errorf("Line = %d, want 10", action.Line)
	}
	if action.Original != "old code" {
		t.Errorf("Original = %s, want 'old code'", action.Original)
	}
	if action.Modified != "new code" {
		t.Errorf("Modified = %s, want 'new code'", action.Modified)
	}
}

func TestNewActionHistory(t *testing.T) {
	h := NewActionHistory(10)

	if h == nil {
		t.Fatal("NewActionHistory() returned nil")
	}
	if h.maxSize != 10 {
		t.Errorf("maxSize = %d, want 10", h.maxSize)
	}
	if len(h.actions) != 0 {
		t.Errorf("actions len = %d, want 0", len(h.actions))
	}
}

func TestActionHistoryPush(t *testing.T) {
	h := NewActionHistory(3)

	h.Push(Action{Type: ActionApply, IssueIdx: 0})
	h.Push(Action{Type: ActionSkip, IssueIdx: 1})

	if h.Len() != 2 {
		t.Errorf("Len() = %d, want 2", h.Len())
	}
}

func TestActionHistoryPushOverflow(t *testing.T) {
	h := NewActionHistory(2)

	h.Push(Action{Type: ActionApply, IssueIdx: 0})
	h.Push(Action{Type: ActionSkip, IssueIdx: 1})
	h.Push(Action{Type: ActionApply, IssueIdx: 2})

	if h.Len() != 2 {
		t.Errorf("Len() after overflow = %d, want 2", h.Len())
	}

	// First item should be removed
	actions := h.All()
	if actions[0].IssueIdx != 1 {
		t.Errorf("First action IssueIdx = %d, want 1", actions[0].IssueIdx)
	}
}

func TestActionHistoryPop(t *testing.T) {
	h := NewActionHistory(10)

	h.Push(Action{Type: ActionApply, IssueIdx: 0})
	h.Push(Action{Type: ActionSkip, IssueIdx: 1})

	action, ok := h.Pop()
	if !ok {
		t.Error("Pop() should return true")
	}
	if action.IssueIdx != 1 {
		t.Errorf("Popped action IssueIdx = %d, want 1", action.IssueIdx)
	}
	if h.Len() != 1 {
		t.Errorf("Len() after Pop = %d, want 1", h.Len())
	}
}

func TestActionHistoryPopEmpty(t *testing.T) {
	h := NewActionHistory(10)

	_, ok := h.Pop()
	if ok {
		t.Error("Pop() on empty history should return false")
	}
}

func TestActionHistoryPeek(t *testing.T) {
	h := NewActionHistory(10)

	h.Push(Action{Type: ActionApply, IssueIdx: 0})
	h.Push(Action{Type: ActionSkip, IssueIdx: 1})

	action, ok := h.Peek()
	if !ok {
		t.Error("Peek() should return true")
	}
	if action.IssueIdx != 1 {
		t.Errorf("Peeked action IssueIdx = %d, want 1", action.IssueIdx)
	}
	// Len should not change
	if h.Len() != 2 {
		t.Errorf("Len() after Peek = %d, want 2", h.Len())
	}
}

func TestActionHistoryPeekEmpty(t *testing.T) {
	h := NewActionHistory(10)

	_, ok := h.Peek()
	if ok {
		t.Error("Peek() on empty history should return false")
	}
}

func TestActionHistoryCanUndo(t *testing.T) {
	h := NewActionHistory(10)

	if h.CanUndo() {
		t.Error("CanUndo() should return false for empty history")
	}

	h.Push(Action{Type: ActionApply})

	if !h.CanUndo() {
		t.Error("CanUndo() should return true after push")
	}
}

func TestActionHistoryClear(t *testing.T) {
	h := NewActionHistory(10)

	h.Push(Action{Type: ActionApply})
	h.Push(Action{Type: ActionSkip})
	h.Clear()

	if h.Len() != 0 {
		t.Errorf("Len() after Clear = %d, want 0", h.Len())
	}
	if h.CanUndo() {
		t.Error("CanUndo() should return false after Clear")
	}
}

func TestActionHistoryAll(t *testing.T) {
	h := NewActionHistory(10)

	h.Push(Action{Type: ActionApply, IssueIdx: 0})
	h.Push(Action{Type: ActionSkip, IssueIdx: 1})

	all := h.All()

	if len(all) != 2 {
		t.Errorf("All() len = %d, want 2", len(all))
	}
	if all[0].IssueIdx != 0 {
		t.Errorf("All()[0].IssueIdx = %d, want 0", all[0].IssueIdx)
	}
	if all[1].IssueIdx != 1 {
		t.Errorf("All()[1].IssueIdx = %d, want 1", all[1].IssueIdx)
	}
}

func TestExportResultStruct(t *testing.T) {
	now := time.Now()
	result := ExportResult{
		Timestamp:    now,
		TotalIssues:  10,
		AppliedCount: 5,
		SkippedCount: 3,
		Issues:       []ExportIssue{},
		Actions:      []Action{},
		Duration:     "1m30s",
	}

	if result.TotalIssues != 10 {
		t.Errorf("TotalIssues = %d, want 10", result.TotalIssues)
	}
	if result.AppliedCount != 5 {
		t.Errorf("AppliedCount = %d, want 5", result.AppliedCount)
	}
	if result.SkippedCount != 3 {
		t.Errorf("SkippedCount = %d, want 3", result.SkippedCount)
	}
}

func TestExportIssueStruct(t *testing.T) {
	issue := ExportIssue{
		File:       "main.go",
		Line:       10,
		Column:     5,
		Message:    "test message",
		Type:       "select_star",
		Status:     "applied",
		Suggestion: "fixed code",
	}

	if issue.File != "main.go" {
		t.Errorf("File = %s, want 'main.go'", issue.File)
	}
	if issue.Status != "applied" {
		t.Errorf("Status = %s, want 'applied'", issue.Status)
	}
}

func TestExportToJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "export.json")

	result := ExportResult{
		Timestamp:    time.Now(),
		TotalIssues:  2,
		AppliedCount: 1,
		SkippedCount: 1,
		Issues: []ExportIssue{
			{File: "a.go", Line: 1, Status: "applied"},
			{File: "b.go", Line: 2, Status: "skipped"},
		},
		Actions:  []Action{},
		Duration: "10s",
	}

	err := ExportToJSON(filename, result)
	if err != nil {
		t.Fatalf("ExportToJSON() error = %v", err)
	}

	// Verify file exists and has content
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	if len(data) == 0 {
		t.Error("Export file should not be empty")
	}

	// Check it contains expected fields
	content := string(data)
	if !containsStr(content, "total_issues") {
		t.Error("Export should contain 'total_issues'")
	}
	if !containsStr(content, "applied_count") {
		t.Error("Export should contain 'applied_count'")
	}
}

func TestGetIssueType(t *testing.T) {
	tests := []struct {
		message string
		want    string
	}{
		{"avoid SELECT * usage", "select_star"},
		{"avoid select_star usage", "select_star"},
		{"avoid t.* aliased wildcard", "aliased_wildcard"},
		{"gorm Find all records", "orm_find_all"},
		{"All() returns all", "orm_find_all"},
		{"squirrel SQL builder", "sql_builder_star"},
		{"gorm builder pattern", "sql_builder_star"},
		{"bun ORM builder", "sql_builder_star"},
		{"unknown issue type", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			issue := Issue{Message: tt.message}
			got := GetIssueType(issue)
			if got != tt.want {
				t.Errorf("GetIssueType(%q) = %q, want %q", tt.message, got, tt.want)
			}
		})
	}
}

func TestGenerateSmartSuggestion(t *testing.T) {
	tests := []struct {
		name     string
		issue    Issue
		contains string
	}{
		{
			name:     "select star",
			issue:    Issue{Message: "SELECT *", CodeLine: `query := "SELECT * FROM users"`},
			contains: "col1",
		},
		{
			name:     "aliased",
			issue:    Issue{Message: "t.* aliased", CodeLine: "SELECT t.* FROM"},
			contains: "TODO",
		},
		{
			name:     "orm find",
			issue:    Issue{Message: "Find all", CodeLine: "db.Find(&users)"},
			contains: "Select",
		},
		{
			name:     "sql builder",
			issue:    Issue{Message: "squirrel builder", CodeLine: `Select("*")`},
			contains: "col1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateSmartSuggestion(tt.issue)
			if !containsStr(result, tt.contains) {
				t.Errorf("GenerateSmartSuggestion() = %q, should contain %q", result, tt.contains)
			}
		})
	}
}

func TestSuggestExplicitColumns(t *testing.T) {
	tests := []struct {
		codeLine string
		contains string
	}{
		{`query := "SELECT * FROM users"`, "col1"},
		{`query := "select * from users"`, "col1"},
		{`query := "SELECT id FROM users"`, "TODO"},
	}

	for _, tt := range tests {
		t.Run(tt.codeLine, func(t *testing.T) {
			result := suggestExplicitColumns(tt.codeLine)
			if !containsStr(result, tt.contains) {
				t.Errorf("suggestExplicitColumns(%q) = %q, should contain %q", tt.codeLine, result, tt.contains)
			}
		})
	}
}

func TestSuggestBuilderColumns(t *testing.T) {
	codeLine := `sq.Select("*").From("users")`
	result := suggestBuilderColumns(codeLine)

	if containsStr(result, `"*"`) {
		t.Error("suggestBuilderColumns should replace \"*\"")
	}
	if !containsStr(result, "col1") {
		t.Error("suggestBuilderColumns should add column names")
	}
}

func TestBatchSkipAll(t *testing.T) {
	issues := []Issue{
		{File: "a.go", Line: 1},
		{File: "b.go", Line: 2},
		{File: "c.go", Line: 3},
	}

	m := NewModel(issues)
	m.applied[0] = true // Already applied

	BatchSkipAll(&m)

	// First should remain applied, others should be skipped
	if !m.applied[0] {
		t.Error("Issue 0 should still be applied")
	}
	if m.skipped[0] {
		t.Error("Issue 0 should not be skipped")
	}
	if !m.skipped[1] {
		t.Error("Issue 1 should be skipped")
	}
	if !m.skipped[2] {
		t.Error("Issue 2 should be skipped")
	}
}

func TestUndoLastActionEmpty(t *testing.T) {
	m := NewModel([]Issue{})

	err := UndoLastAction(&m)
	if err == nil {
		t.Error("UndoLastAction() should error on empty history")
	}
}

func TestUndoLastActionSkip(t *testing.T) {
	issues := []Issue{
		{File: "a.go", Line: 1},
	}

	m := NewModel(issues)
	m.skipped[0] = true
	m.history.Push(Action{
		Type:     ActionSkip,
		IssueIdx: 0,
	})

	err := UndoLastAction(&m)
	if err != nil {
		t.Errorf("UndoLastAction() error = %v", err)
	}

	if m.skipped[0] {
		t.Error("Issue should no longer be skipped after undo")
	}
	if m.currentIndex != 0 {
		t.Errorf("currentIndex = %d, want 0", m.currentIndex)
	}
}

// Helper function
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
