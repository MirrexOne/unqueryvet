package runner

import (
	"go/token"
	"testing"
	"time"

	"golang.org/x/tools/go/analysis"
)

func TestExitCodes(t *testing.T) {
	tests := []struct {
		name string
		code ExitCode
		want int
	}{
		{"success", ExitSuccess, 0},
		{"warnings", ExitWarnings, 1},
		{"errors", ExitErrors, 2},
		{"failure", ExitFailure, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if int(tt.code) != tt.want {
				t.Errorf("ExitCode %s = %d, want %d", tt.name, tt.code, tt.want)
			}
		})
	}
}

func TestNewStatistics(t *testing.T) {
	stats := NewStatistics()

	if stats == nil {
		t.Fatal("NewStatistics() returned nil")
	}
	if stats.ByType == nil {
		t.Error("ByType should be initialized")
	}
	if stats.FilesWithIssues == nil {
		t.Error("FilesWithIssues should be initialized")
	}
	if stats.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
	if stats.FilesAnalyzed != 0 {
		t.Errorf("FilesAnalyzed = %d, want 0", stats.FilesAnalyzed)
	}
}

func TestStatisticsAddDiagnostic(t *testing.T) {
	tests := []struct {
		name       string
		severity   string
		wantErrors int
		wantWarns  int
	}{
		{"error", "error", 1, 0},
		{"warning", "warning", 0, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := NewStatistics()
			diag := analysis.Diagnostic{
				Pos:     token.NoPos,
				Message: "avoid SELECT * in SQL builder",
			}

			stats.AddDiagnostic(diag, nil, tt.severity)

			if stats.TotalIssues != 1 {
				t.Errorf("TotalIssues = %d, want 1", stats.TotalIssues)
			}
			if stats.Errors != tt.wantErrors {
				t.Errorf("Errors = %d, want %d", stats.Errors, tt.wantErrors)
			}
			if stats.Warnings != tt.wantWarns {
				t.Errorf("Warnings = %d, want %d", stats.Warnings, tt.wantWarns)
			}
		})
	}
}

func TestStatisticsFinalize(t *testing.T) {
	stats := NewStatistics()

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	stats.Finalize()

	if stats.Duration == 0 {
		t.Error("Duration should be > 0 after Finalize()")
	}
}

func TestExtractMessageType(t *testing.T) {
	tests := []struct {
		message string
		want    string
	}{
		{"avoid SELECT * in SQL builder", "sql_builder"},
		{"avoid SELECT alias.* usage", "aliased_wildcard"},
		{"avoid SELECT * in subquery", "subquery"},
		{"avoid SELECT * in concatenated string", "concatenation"},
		{"avoid SELECT * in format string", "format_string"},
		{"empty Select() without columns", "empty_select"},
		{"avoid SELECT * usage", "select_star"},
		{"short", "other"},
		{"", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			got := extractMessageType(tt.message)
			if got != tt.want {
				t.Errorf("extractMessageType(%q) = %q, want %q", tt.message, got, tt.want)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello", "hello", true},
		{"hello", "world", false},
		{"", "hello", false},
		{"hi", "hello", false},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := contains(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestFindSubstring(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"hello world", "world", true},
		{"hello world", "lo wo", true},
		{"hello", "hello", true},
		{"hello", "world", false},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.substr, func(t *testing.T) {
			got := findSubstring(tt.s, tt.substr)
			if got != tt.want {
				t.Errorf("findSubstring(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
			}
		})
	}
}

func TestStatisticsAddDiagnosticWithFileSet(t *testing.T) {
	stats := NewStatistics()
	fset := token.NewFileSet()

	// Create a file in the fileset
	file := fset.AddFile("test.go", fset.Base(), 100)
	pos := file.Pos(10)

	diag := analysis.Diagnostic{
		Pos:     pos,
		Message: "test message for select_star",
	}

	stats.AddDiagnostic(diag, fset, "warning")

	if stats.TotalIssues != 1 {
		t.Errorf("TotalIssues = %d, want 1", stats.TotalIssues)
	}

	// File should be tracked
	if len(stats.FilesWithIssues) != 1 {
		t.Errorf("FilesWithIssues len = %d, want 1", len(stats.FilesWithIssues))
	}

	if stats.FilesWithIssues["test.go"] != 1 {
		t.Errorf("FilesWithIssues[test.go] = %d, want 1", stats.FilesWithIssues["test.go"])
	}
}

func TestStatisticsMultipleDiagnostics(t *testing.T) {
	stats := NewStatistics()

	messages := []string{
		"avoid SELECT * in SQL builder",
		"avoid SELECT * in SQL builder",
		"avoid SELECT alias.* usage",
	}

	for _, msg := range messages {
		diag := analysis.Diagnostic{
			Pos:     token.NoPos,
			Message: msg,
		}
		stats.AddDiagnostic(diag, nil, "warning")
	}

	if stats.TotalIssues != 3 {
		t.Errorf("TotalIssues = %d, want 3", stats.TotalIssues)
	}

	if stats.ByType["sql_builder"] != 2 {
		t.Errorf("ByType[sql_builder] = %d, want 2", stats.ByType["sql_builder"])
	}

	if stats.ByType["aliased_wildcard"] != 1 {
		t.Errorf("ByType[aliased_wildcard] = %d, want 1", stats.ByType["aliased_wildcard"])
	}
}

func TestStatisticsStruct(t *testing.T) {
	now := time.Now()
	stats := Statistics{
		FilesAnalyzed:   10,
		PackagesLoaded:  5,
		TotalIssues:     20,
		Errors:          5,
		Warnings:        15,
		ByType:          map[string]int{"select_star": 10},
		Duration:        100 * time.Millisecond,
		FilesWithIssues: map[string]int{"main.go": 3},
		StartTime:       now,
	}

	if stats.FilesAnalyzed != 10 {
		t.Errorf("FilesAnalyzed = %d, want 10", stats.FilesAnalyzed)
	}
	if stats.PackagesLoaded != 5 {
		t.Errorf("PackagesLoaded = %d, want 5", stats.PackagesLoaded)
	}
	if stats.TotalIssues != 20 {
		t.Errorf("TotalIssues = %d, want 20", stats.TotalIssues)
	}
	if stats.Errors != 5 {
		t.Errorf("Errors = %d, want 5", stats.Errors)
	}
	if stats.Warnings != 15 {
		t.Errorf("Warnings = %d, want 15", stats.Warnings)
	}
}
