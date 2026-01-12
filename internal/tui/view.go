// Package tui provides interactive terminal UI for fixing SELECT * issues.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DiffLine represents a line in a diff view.
type DiffLine struct {
	Type    DiffType
	Content string
}

// DiffType represents the type of diff line.
type DiffType int

const (
	DiffContext DiffType = iota
	DiffRemoved
	DiffAdded
)

// Diff view styles.
var (
	diffHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1)

	diffContextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("246"))

	diffRemovedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("52")).
				Foreground(lipgloss.Color("196"))

	diffAddedStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("22")).
			Foreground(lipgloss.Color("46"))

	lineNumStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Width(5).
			Align(lipgloss.Right)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(0, 1)

	previewTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("214")).
				MarginBottom(1)

	batchInfoStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("24")).
			Foreground(lipgloss.Color("255")).
			Padding(0, 1).
			MarginTop(1)

	undoAvailableStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("214"))
)

// RenderDiffPreview renders a diff preview showing before/after changes.
func RenderDiffPreview(original, modified string, lineNum int, contextLines int) string {
	var b strings.Builder

	b.WriteString(previewTitleStyle.Render("📋 Preview Changes"))
	b.WriteString("\n\n")

	// Split into lines
	origLines := strings.Split(original, "\n")
	modLines := strings.Split(modified, "\n")

	// Calculate context range
	startLine := lineNum - contextLines - 1
	if startLine < 0 {
		startLine = 0
	}
	endLine := lineNum + contextLines
	if endLine > len(origLines) {
		endLine = len(origLines)
	}

	// Render diff
	b.WriteString(diffHeaderStyle.Render("--- Original"))
	b.WriteString("\n")
	b.WriteString(diffHeaderStyle.Render("+++ Modified"))
	b.WriteString("\n\n")

	for i := startLine; i < endLine; i++ {
		lineNumStr := lineNumStyle.Render(fmt.Sprintf("%d", i+1))

		if i == lineNum-1 {
			// This is the changed line
			if i < len(origLines) {
				b.WriteString(lineNumStr)
				b.WriteString(" ")
				b.WriteString(diffRemovedStyle.Render("- " + origLines[i]))
				b.WriteString("\n")
			}
			if i < len(modLines) {
				b.WriteString(lineNumStr)
				b.WriteString(" ")
				b.WriteString(diffAddedStyle.Render("+ " + modLines[i]))
				b.WriteString("\n")
			}
		} else {
			// Context line
			if i < len(origLines) {
				b.WriteString(lineNumStr)
				b.WriteString(" ")
				b.WriteString(diffContextStyle.Render("  " + origLines[i]))
				b.WriteString("\n")
			}
		}
	}

	return boxStyle.Render(b.String())
}

// RenderDiffInline renders an inline diff for compact view.
func RenderDiffInline(original, modified string) string {
	var b strings.Builder

	b.WriteString(diffRemovedStyle.Render("- " + original))
	b.WriteString("\n")
	b.WriteString(diffAddedStyle.Render("+ " + modified))

	return b.String()
}

// RenderBatchInfo renders information about batch operations.
func RenderBatchInfo(totalIssues, appliedCount, skippedCount int, undoAvailable bool) string {
	var b strings.Builder

	remaining := totalIssues - appliedCount - skippedCount

	b.WriteString(fmt.Sprintf("Total: %d | ", totalIssues))
	b.WriteString(successStyle.Render(fmt.Sprintf("Applied: %d", appliedCount)))
	b.WriteString(" | ")
	b.WriteString(skippedStyle.Render(fmt.Sprintf("Skipped: %d", skippedCount)))
	b.WriteString(" | ")
	b.WriteString(warningStyle.Render(fmt.Sprintf("Remaining: %d", remaining)))

	if undoAvailable {
		b.WriteString(" | ")
		b.WriteString(undoAvailableStyle.Render("u: undo"))
	}

	return batchInfoStyle.Render(b.String())
}

// RenderIssueTypeStats renders statistics by issue type.
func RenderIssueTypeStats(typeCounts map[string]int) string {
	var b strings.Builder

	b.WriteString(issueHeaderStyle.Render("📊 Issues by Type"))
	b.WriteString("\n")

	for issueType, count := range typeCounts {
		b.WriteString(fmt.Sprintf("  • %s: %d\n", issueType, count))
	}

	return b.String()
}

// RenderSmartFixSuggestion renders smart fix based on issue type.
func RenderSmartFixSuggestion(issueType string, suggestion string) string {
	var b strings.Builder

	b.WriteString(issueHeaderStyle.Render("🧠 Smart Fix"))
	b.WriteString("\n")

	switch issueType {
	case "select_star":
		b.WriteString("Detected: SELECT * query\n")
		b.WriteString("Suggestion: Replace with explicit column names\n")
	case "aliased_wildcard":
		b.WriteString("Detected: Aliased wildcard (t.*)\n")
		b.WriteString("Suggestion: Replace with alias.column_name\n")
	case "orm_find_all":
		b.WriteString("Detected: ORM Find/All without Select\n")
		b.WriteString("Suggestion: Add .Select(\"col1\", \"col2\") before query\n")
	case "sql_builder_star":
		b.WriteString("Detected: SQL Builder with *\n")
		b.WriteString("Suggestion: Use explicit column list in builder\n")
	default:
		b.WriteString("Type: " + issueType + "\n")
	}

	if suggestion != "" {
		b.WriteString("\n")
		b.WriteString(suggestionStyle.Render(suggestion))
	}

	return boxStyle.Render(b.String())
}

// RenderHelpFull renders full help with all keybindings.
func RenderHelpFull() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("⌨️  Keyboard Shortcuts"))
	b.WriteString("\n\n")

	sections := []struct {
		title string
		keys  []struct {
			key  string
			desc string
		}
	}{
		{
			title: "Navigation",
			keys: []struct {
				key  string
				desc string
			}{
				{"↑/k", "Previous issue"},
				{"↓/j", "Next issue"},
				{"g", "Go to first issue"},
				{"G", "Go to last issue"},
			},
		},
		{
			title: "Actions",
			keys: []struct {
				key  string
				desc string
			}{
				{"Enter/a", "Apply fix"},
				{"s", "Skip issue"},
				{"u", "Undo last action"},
				{"p", "Toggle preview"},
			},
		},
		{
			title: "Batch Operations",
			keys: []struct {
				key  string
				desc string
			}{
				{"A", "Apply all remaining"},
				{"S", "Skip all remaining"},
				{"R", "Reset all actions"},
			},
		},
		{
			title: "Other",
			keys: []struct {
				key  string
				desc string
			}{
				{"e", "Export results to JSON"},
				{"?", "Toggle help"},
				{"q/Esc", "Quit"},
			},
		},
	}

	for _, section := range sections {
		b.WriteString(issueHeaderStyle.Render(section.title))
		b.WriteString("\n")
		for _, k := range section.keys {
			b.WriteString(fmt.Sprintf("  %s\t%s\n", helpStyle.Render(k.key), k.desc))
		}
		b.WriteString("\n")
	}

	return boxStyle.Render(b.String())
}
