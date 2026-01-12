package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestColorCodes(t *testing.T) {
	// Verify color codes are defined
	colors := []string{
		ColorReset, ColorRed, ColorGreen, ColorYellow,
		ColorBlue, ColorCyan, ColorGray, ColorBold,
	}
	for _, c := range colors {
		if c == "" {
			t.Error("Color code should not be empty")
		}
		if !strings.HasPrefix(c, "\033[") {
			t.Errorf("Color code %q should start with ANSI escape", c)
		}
	}
}

func TestNewOutput(t *testing.T) {
	out := NewOutput(true, 1, false)

	if out == nil {
		t.Fatal("NewOutput() returned nil")
	}
	if !out.useColors {
		t.Error("useColors should be true")
	}
	if out.verbose != 1 {
		t.Errorf("verbose = %d, want 1", out.verbose)
	}
	if out.quiet {
		t.Error("quiet should be false")
	}
}

func TestOutputColor(t *testing.T) {
	tests := []struct {
		name      string
		useColors bool
		colorCode string
		text      string
		wantColor bool
	}{
		{
			name:      "with colors",
			useColors: true,
			colorCode: ColorRed,
			text:      "test",
			wantColor: true,
		},
		{
			name:      "without colors",
			useColors: false,
			colorCode: ColorRed,
			text:      "test",
			wantColor: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := NewOutput(tt.useColors, 0, false)
			result := out.color(tt.colorCode, tt.text)

			if tt.wantColor {
				if !strings.Contains(result, ColorRed) {
					t.Error("Result should contain color code")
				}
				if !strings.Contains(result, ColorReset) {
					t.Error("Result should contain reset code")
				}
			} else {
				if result != tt.text {
					t.Errorf("Result = %q, want %q", result, tt.text)
				}
			}
		})
	}
}

func TestOutputError(t *testing.T) {
	var buf bytes.Buffer
	out := &Output{
		writer:      &buf,
		errorWriter: &buf,
		useColors:   false,
		verbose:     0,
		quiet:       false,
	}

	out.Error("test error: %s", "details")

	if !strings.Contains(buf.String(), "ERROR:") {
		t.Error("Error output should contain 'ERROR:'")
	}
	if !strings.Contains(buf.String(), "test error: details") {
		t.Error("Error output should contain message")
	}
}

func TestOutputWarning(t *testing.T) {
	t.Run("normal mode", func(t *testing.T) {
		var buf bytes.Buffer
		out := &Output{
			writer:    &buf,
			useColors: false,
			verbose:   0,
			quiet:     false,
		}

		out.Warning("test warning")

		if !strings.Contains(buf.String(), "WARNING:") {
			t.Error("Warning output should contain 'WARNING:'")
		}
	})

	t.Run("quiet mode", func(t *testing.T) {
		var buf bytes.Buffer
		out := &Output{
			writer:    &buf,
			useColors: false,
			verbose:   0,
			quiet:     true,
		}

		out.Warning("test warning")

		if buf.String() != "" {
			t.Error("Warning should not output in quiet mode")
		}
	})
}

func TestOutputInfo(t *testing.T) {
	t.Run("normal mode", func(t *testing.T) {
		var buf bytes.Buffer
		out := &Output{
			writer:    &buf,
			useColors: false,
			verbose:   0,
			quiet:     false,
		}

		out.Info("info message")

		if !strings.Contains(buf.String(), "info message") {
			t.Error("Info output should contain message")
		}
	})

	t.Run("quiet mode", func(t *testing.T) {
		var buf bytes.Buffer
		out := &Output{
			writer:    &buf,
			useColors: false,
			verbose:   0,
			quiet:     true,
		}

		out.Info("info message")

		if buf.String() != "" {
			t.Error("Info should not output in quiet mode")
		}
	})
}

func TestOutputSuccess(t *testing.T) {
	var buf bytes.Buffer
	out := &Output{
		writer:    &buf,
		useColors: false,
		verbose:   0,
		quiet:     false,
	}

	out.Success("success message")

	if !strings.Contains(buf.String(), "success message") {
		t.Error("Success output should contain message")
	}
}

func TestOutputDebug(t *testing.T) {
	tests := []struct {
		name    string
		verbose int
		want    bool
	}{
		{"verbose 0", 0, false},
		{"verbose 1", 1, true},
		{"verbose 2", 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			out := &Output{
				writer:    &buf,
				useColors: false,
				verbose:   tt.verbose,
				quiet:     false,
			}

			out.Debug("debug message")

			hasOutput := strings.Contains(buf.String(), "DEBUG:")
			if hasOutput != tt.want {
				t.Errorf("Debug output = %v, want %v", hasOutput, tt.want)
			}
		})
	}
}

func TestOutputTrace(t *testing.T) {
	tests := []struct {
		name    string
		verbose int
		want    bool
	}{
		{"verbose 0", 0, false},
		{"verbose 1", 1, false},
		{"verbose 2", 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			out := &Output{
				writer:    &buf,
				useColors: false,
				verbose:   tt.verbose,
				quiet:     false,
			}

			out.Trace("trace message")

			hasOutput := strings.Contains(buf.String(), "TRACE:")
			if hasOutput != tt.want {
				t.Errorf("Trace output = %v, want %v", hasOutput, tt.want)
			}
		})
	}
}

func TestOutputSection(t *testing.T) {
	t.Run("normal mode", func(t *testing.T) {
		var buf bytes.Buffer
		out := &Output{
			writer:    &buf,
			useColors: false,
			verbose:   0,
			quiet:     false,
		}

		out.Section("Test Section")

		output := buf.String()
		if !strings.Contains(output, "Test Section") {
			t.Error("Section should contain title")
		}
		if !strings.Contains(output, "─") {
			t.Error("Section should contain underline")
		}
	})

	t.Run("quiet mode", func(t *testing.T) {
		var buf bytes.Buffer
		out := &Output{
			writer:    &buf,
			useColors: false,
			verbose:   0,
			quiet:     true,
		}

		out.Section("Test Section")

		if buf.String() != "" {
			t.Error("Section should not output in quiet mode")
		}
	})
}

func TestNewStatistics(t *testing.T) {
	stats := NewStatistics()

	if stats == nil {
		t.Fatal("NewStatistics() returned nil")
	}
	if stats.ByType == nil {
		t.Error("ByType map should be initialized")
	}
	if stats.FilesWithIssues == nil {
		t.Error("FilesWithIssues slice should be initialized")
	}
	if stats.FilesAnalyzed != 0 {
		t.Error("FilesAnalyzed should be 0")
	}
	if stats.TotalIssues != 0 {
		t.Error("TotalIssues should be 0")
	}
}

func TestStatisticsAddIssue(t *testing.T) {
	tests := []struct {
		name       string
		issueType  string
		isError    bool
		wantErrors int
		wantWarns  int
		wantTotal  int
	}{
		{
			name:       "add error",
			issueType:  "select_star",
			isError:    true,
			wantErrors: 1,
			wantWarns:  0,
			wantTotal:  1,
		},
		{
			name:       "add warning",
			issueType:  "aliased_wildcard",
			isError:    false,
			wantErrors: 0,
			wantWarns:  1,
			wantTotal:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := NewStatistics()
			stats.AddIssue(tt.issueType, tt.isError)

			if stats.TotalIssues != tt.wantTotal {
				t.Errorf("TotalIssues = %d, want %d", stats.TotalIssues, tt.wantTotal)
			}
			if stats.Errors != tt.wantErrors {
				t.Errorf("Errors = %d, want %d", stats.Errors, tt.wantErrors)
			}
			if stats.Warnings != tt.wantWarns {
				t.Errorf("Warnings = %d, want %d", stats.Warnings, tt.wantWarns)
			}
			if stats.ByType[tt.issueType] != 1 {
				t.Errorf("ByType[%s] = %d, want 1", tt.issueType, stats.ByType[tt.issueType])
			}
		})
	}
}

func TestStatisticsMultipleIssues(t *testing.T) {
	stats := NewStatistics()

	stats.AddIssue("select_star", true)
	stats.AddIssue("select_star", false)
	stats.AddIssue("aliased_wildcard", true)

	if stats.TotalIssues != 3 {
		t.Errorf("TotalIssues = %d, want 3", stats.TotalIssues)
	}
	if stats.Errors != 2 {
		t.Errorf("Errors = %d, want 2", stats.Errors)
	}
	if stats.Warnings != 1 {
		t.Errorf("Warnings = %d, want 1", stats.Warnings)
	}
	if stats.ByType["select_star"] != 2 {
		t.Errorf("ByType[select_star] = %d, want 2", stats.ByType["select_star"])
	}
}

func TestPrintStatistics(t *testing.T) {
	t.Run("no issues", func(t *testing.T) {
		var buf bytes.Buffer
		out := &Output{
			writer:    &buf,
			useColors: false,
			verbose:   0,
			quiet:     false,
		}

		stats := NewStatistics()
		stats.FilesAnalyzed = 10
		stats.Duration = 100 * time.Millisecond

		out.PrintStatistics(stats)

		output := buf.String()
		if !strings.Contains(output, "No issues found") {
			t.Error("Should indicate no issues found")
		}
		if !strings.Contains(output, "10") {
			t.Error("Should show files analyzed count")
		}
	})

	t.Run("with issues", func(t *testing.T) {
		var buf bytes.Buffer
		out := &Output{
			writer:    &buf,
			useColors: false,
			verbose:   0,
			quiet:     false,
		}

		stats := NewStatistics()
		stats.FilesAnalyzed = 5
		stats.Duration = 50 * time.Millisecond
		stats.AddIssue("select_star", true)
		stats.AddIssue("aliased", false)

		out.PrintStatistics(stats)

		output := buf.String()
		if !strings.Contains(output, "Issues found") {
			t.Error("Should show issues found")
		}
		if !strings.Contains(output, "2") {
			t.Error("Should show total issues count")
		}
	})

	t.Run("verbose with breakdown", func(t *testing.T) {
		var buf bytes.Buffer
		out := &Output{
			writer:    &buf,
			useColors: false,
			verbose:   1,
			quiet:     false,
		}

		stats := NewStatistics()
		stats.FilesAnalyzed = 5
		stats.Duration = 50 * time.Millisecond
		stats.AddIssue("select_star", true)
		stats.FilesWithIssues = []string{"main.go"}

		out.PrintStatistics(stats)

		output := buf.String()
		if !strings.Contains(output, "select_star") {
			t.Error("Verbose mode should show breakdown by type")
		}
		if !strings.Contains(output, "Files with issues") {
			t.Error("Verbose mode should show files with issues")
		}
	})

	t.Run("quiet with no issues", func(t *testing.T) {
		var buf bytes.Buffer
		out := &Output{
			writer:    &buf,
			useColors: false,
			verbose:   0,
			quiet:     true,
		}

		stats := NewStatistics()

		out.PrintStatistics(stats)

		if buf.String() != "" {
			t.Error("Quiet mode with no issues should not output")
		}
	})
}

func TestPrintDiagnostic(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		quiet    bool
		wantOut  bool
	}{
		{"error normal", "error", false, true},
		{"warning normal", "warning", false, true},
		{"error quiet", "error", true, true},
		{"warning quiet", "warning", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			out := &Output{
				writer:    &buf,
				useColors: false,
				verbose:   0,
				quiet:     tt.quiet,
			}

			out.PrintDiagnostic("main.go", 10, 5, tt.severity, "test message")

			hasOutput := buf.Len() > 0
			if hasOutput != tt.wantOut {
				t.Errorf("PrintDiagnostic output = %v, want %v", hasOutput, tt.wantOut)
			}

			if tt.wantOut {
				output := buf.String()
				if !strings.Contains(output, "main.go:10:5:") {
					t.Error("Should contain file location")
				}
				if !strings.Contains(output, "test message") {
					t.Error("Should contain message")
				}
			}
		})
	}
}

func TestPrintDiagnosticWithColors(t *testing.T) {
	var buf bytes.Buffer
	out := &Output{
		writer:    &buf,
		useColors: true,
		verbose:   0,
		quiet:     false,
	}

	out.PrintDiagnostic("main.go", 10, 5, "error", "test message")

	output := buf.String()
	if !strings.Contains(output, ColorRed) {
		t.Error("Error severity should use red color")
	}
}

func TestShouldUseColors(t *testing.T) {
	// This function depends on environment, just test it doesn't panic
	_ = ShouldUseColors()
}
