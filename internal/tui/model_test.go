package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestIssueStruct(t *testing.T) {
	issue := Issue{
		File:       "main.go",
		Line:       10,
		Column:     5,
		Message:    "avoid SELECT *",
		CodeLine:   `query := "SELECT * FROM users"`,
		Suggestion: `query := "SELECT id, name FROM users"`,
		Type:       "select_star",
	}

	if issue.File != "main.go" {
		t.Errorf("File = %s, want 'main.go'", issue.File)
	}
	if issue.Line != 10 {
		t.Errorf("Line = %d, want 10", issue.Line)
	}
	if issue.Column != 5 {
		t.Errorf("Column = %d, want 5", issue.Column)
	}
}

func TestNewModel(t *testing.T) {
	issues := []Issue{
		{File: "a.go", Line: 1},
		{File: "b.go", Line: 2},
	}

	m := NewModel(issues)

	if len(m.issues) != 2 {
		t.Errorf("issues len = %d, want 2", len(m.issues))
	}
	if m.currentIndex != 0 {
		t.Errorf("currentIndex = %d, want 0", m.currentIndex)
	}
	if m.applied == nil {
		t.Error("applied should be initialized")
	}
	if m.skipped == nil {
		t.Error("skipped should be initialized")
	}
	if m.history == nil {
		t.Error("history should be initialized")
	}
	if !m.showPreview {
		t.Error("showPreview should default to true")
	}
	if m.startTime.IsZero() {
		t.Error("startTime should be set")
	}
}

func TestModelInit(t *testing.T) {
	m := NewModel([]Issue{})
	cmd := m.Init()

	if cmd != nil {
		t.Error("Init() should return nil")
	}
}

func TestModelUpdateNavigation(t *testing.T) {
	issues := []Issue{
		{File: "a.go", Line: 1},
		{File: "b.go", Line: 2},
		{File: "c.go", Line: 3},
	}

	m := NewModel(issues)

	// Test next
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.currentIndex != 1 {
		t.Errorf("After down, currentIndex = %d, want 1", m.currentIndex)
	}

	// Test prev
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.currentIndex != 0 {
		t.Errorf("After up, currentIndex = %d, want 0", m.currentIndex)
	}

	// Test boundary (can't go below 0)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)
	if m.currentIndex != 0 {
		t.Errorf("After up at 0, currentIndex = %d, want 0", m.currentIndex)
	}

	// Navigate to end
	m.currentIndex = 2
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.currentIndex != 2 {
		t.Errorf("After down at end, currentIndex = %d, want 2", m.currentIndex)
	}
}

func TestModelUpdateQuit(t *testing.T) {
	m := NewModel([]Issue{{File: "test.go", Line: 1}})

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)

	if !m.quitting {
		t.Error("quitting should be true after 'q'")
	}

	// Check that quit command is returned
	if cmd == nil {
		t.Error("Quit should return a command")
	}
}

func TestModelUpdateTogglePreview(t *testing.T) {
	m := NewModel([]Issue{{File: "test.go", Line: 1}})

	if !m.showPreview {
		t.Error("showPreview should be true initially")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(Model)

	if m.showPreview {
		t.Error("showPreview should be false after 'p'")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}})
	m = updated.(Model)

	if !m.showPreview {
		t.Error("showPreview should be true after second 'p'")
	}
}

func TestModelUpdateToggleHelp(t *testing.T) {
	m := NewModel([]Issue{{File: "test.go", Line: 1}})

	if m.showHelp {
		t.Error("showHelp should be false initially")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	m = updated.(Model)

	if !m.showHelp {
		t.Error("showHelp should be true after '?'")
	}
}

func TestModelUpdateWindowSize(t *testing.T) {
	m := NewModel([]Issue{})

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	m = updated.(Model)

	if m.width != 100 {
		t.Errorf("width = %d, want 100", m.width)
	}
	if m.height != 50 {
		t.Errorf("height = %d, want 50", m.height)
	}
}

func TestModelViewEmpty(t *testing.T) {
	m := NewModel([]Issue{})
	view := m.View()

	if view == "" {
		t.Error("View should not be empty")
	}
	if !contains(view, "No SELECT * issues found") {
		t.Error("Empty view should show success message")
	}
}

func TestModelViewWithIssues(t *testing.T) {
	issues := []Issue{
		{
			File:       "main.go",
			Line:       10,
			Message:    "avoid SELECT *",
			CodeLine:   `query := "SELECT * FROM users"`,
			Suggestion: `query := "SELECT id, name FROM users"`,
		},
	}

	m := NewModel(issues)
	view := m.View()

	if !contains(view, "main.go") {
		t.Error("View should contain filename")
	}
	if !contains(view, "SELECT * Fixer") {
		t.Error("View should contain title")
	}
}

func TestModelViewQuitting(t *testing.T) {
	m := NewModel([]Issue{{File: "test.go", Line: 1}})
	m.quitting = true

	view := m.View()

	if !contains(view, "Summary") {
		t.Error("Quitting view should show summary")
	}
	if !contains(view, "Goodbye") {
		t.Error("Quitting view should show goodbye")
	}
}

func TestModelViewWithHelp(t *testing.T) {
	m := NewModel([]Issue{{File: "test.go", Line: 1}})
	m.showHelp = true

	view := m.View()

	if !contains(view, "help") || !contains(view, "close") {
		t.Error("Help view should have help content")
	}
}

func TestModelBuildExportResult(t *testing.T) {
	issues := []Issue{
		{File: "a.go", Line: 1, Type: "select_star"},
		{File: "b.go", Line: 2, Type: "aliased"},
	}

	m := NewModel(issues)
	m.applied[0] = true
	m.skipped[1] = true

	result := m.buildExportResult()

	if result.TotalIssues != 2 {
		t.Errorf("TotalIssues = %d, want 2", result.TotalIssues)
	}
	if result.AppliedCount != 1 {
		t.Errorf("AppliedCount = %d, want 1", result.AppliedCount)
	}
	if result.SkippedCount != 1 {
		t.Errorf("SkippedCount = %d, want 1", result.SkippedCount)
	}
	if len(result.Issues) != 2 {
		t.Errorf("Issues len = %d, want 2", len(result.Issues))
	}
	if result.Issues[0].Status != "applied" {
		t.Errorf("Issue[0].Status = %s, want 'applied'", result.Issues[0].Status)
	}
	if result.Issues[1].Status != "skipped" {
		t.Errorf("Issue[1].Status = %s, want 'skipped'", result.Issues[1].Status)
	}
}

func TestModelMoveToNextUnfixed(t *testing.T) {
	issues := []Issue{
		{File: "a.go", Line: 1},
		{File: "b.go", Line: 2},
		{File: "c.go", Line: 3},
	}

	m := NewModel(issues)
	m.applied[0] = true
	m.skipped[1] = true
	m.currentIndex = 0

	m.moveToNextUnfixed()

	if m.currentIndex != 2 {
		t.Errorf("currentIndex = %d, want 2 (first unfixed)", m.currentIndex)
	}
}

func TestModelMoveToNextUnfixedAllFixed(t *testing.T) {
	issues := []Issue{
		{File: "a.go", Line: 1},
		{File: "b.go", Line: 2},
	}

	m := NewModel(issues)
	m.applied[0] = true
	m.applied[1] = true
	m.currentIndex = 0

	m.moveToNextUnfixed()

	// Should wrap around and stay (all fixed)
	// The function will loop and not find any unfixed
}

func TestDefaultKeyMap(t *testing.T) {
	km := DefaultKeyMap()

	if km.Apply.Keys() == nil {
		t.Error("Apply keys should be set")
	}
	if km.Skip.Keys() == nil {
		t.Error("Skip keys should be set")
	}
	if km.Prev.Keys() == nil {
		t.Error("Prev keys should be set")
	}
	if km.Next.Keys() == nil {
		t.Error("Next keys should be set")
	}
	if km.Quit.Keys() == nil {
		t.Error("Quit keys should be set")
	}
	if km.Help.Keys() == nil {
		t.Error("Help keys should be set")
	}
}

func TestModelRenderIssue(t *testing.T) {
	m := NewModel([]Issue{})
	issue := Issue{
		File:       "test.go",
		Line:       10,
		Column:     5,
		Message:    "test message",
		CodeLine:   "some code",
		Suggestion: "fixed code",
	}

	rendered := m.renderIssue(issue, "")

	if !contains(rendered, "test.go") {
		t.Error("Should contain filename")
	}
	if !contains(rendered, "10") {
		t.Error("Should contain line number")
	}
	if !contains(rendered, "test message") {
		t.Error("Should contain message")
	}
}

func TestModelRenderSummary(t *testing.T) {
	issues := []Issue{
		{File: "a.go", Line: 1},
		{File: "b.go", Line: 2},
		{File: "c.go", Line: 3},
	}

	m := NewModel(issues)
	m.applied[0] = true
	m.skipped[1] = true

	summary := m.renderSummary()

	if !contains(summary, "Summary") {
		t.Error("Should contain Summary title")
	}
	if !contains(summary, "Applied") {
		t.Error("Should mention applied fixes")
	}
	if !contains(summary, "Skipped") {
		t.Error("Should mention skipped")
	}
}

func TestModelUpdateGoToFirst(t *testing.T) {
	issues := []Issue{
		{File: "a.go", Line: 1},
		{File: "b.go", Line: 2},
		{File: "c.go", Line: 3},
	}

	m := NewModel(issues)
	m.currentIndex = 2

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	m = updated.(Model)

	if m.currentIndex != 0 {
		t.Errorf("currentIndex = %d, want 0 after 'g'", m.currentIndex)
	}
}

func TestModelUpdateGoToLast(t *testing.T) {
	issues := []Issue{
		{File: "a.go", Line: 1},
		{File: "b.go", Line: 2},
		{File: "c.go", Line: 3},
	}

	m := NewModel(issues)
	m.currentIndex = 0

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	m = updated.(Model)

	if m.currentIndex != 2 {
		t.Errorf("currentIndex = %d, want 2 after 'G'", m.currentIndex)
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestStylesInitialized(t *testing.T) {
	// Just verify styles don't panic when used
	_ = titleStyle.Render("test")
	_ = issueHeaderStyle.Render("test")
	_ = fileStyle.Render("test")
	_ = lineNumberStyle.Render("test")
	_ = codeStyle.Render("test")
	_ = suggestionStyle.Render("test")
	_ = warningStyle.Render("test")
	_ = successStyle.Render("test")
	_ = helpStyle.Render("test")
	_ = statusBarStyle.Render("test")
	_ = appliedStyle.Render("test")
	_ = skippedStyle.Render("test")
	_ = currentStyle.Render("test")
}

func TestModelStartTimeSet(t *testing.T) {
	before := time.Now()
	m := NewModel([]Issue{})
	after := time.Now()

	if m.startTime.Before(before) || m.startTime.After(after) {
		t.Error("startTime should be between before and after")
	}
}
