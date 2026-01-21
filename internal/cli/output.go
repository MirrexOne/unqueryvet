// Package cli provides command-line interface utilities.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Color codes for terminal output.
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
	ColorGray   = "\033[90m"
	ColorBold   = "\033[1m"
)

// Output handles formatted output with colors and verbosity levels.
type Output struct {
	writer      io.Writer
	errorWriter io.Writer
	useColors   bool
	verbose     int // 0 = normal, 1 = verbose, 2 = very verbose
	quiet       bool
}

// NewOutput creates a new Output instance.
func NewOutput(useColors bool, verbose int, quiet bool) *Output {
	return &Output{
		writer:      os.Stdout,
		errorWriter: os.Stderr,
		useColors:   useColors,
		verbose:     verbose,
		quiet:       quiet,
	}
}

// color applies color code if colors are enabled.
func (o *Output) color(colorCode, text string) string {
	if !o.useColors {
		return text
	}
	return colorCode + text + ColorReset
}

// Error prints an error message in red.
func (o *Output) Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(o.errorWriter, "%s\n", o.color(ColorRed, "ERROR: "+msg))
}

// Warning prints a warning message in yellow.
func (o *Output) Warning(format string, args ...any) {
	if o.quiet {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(o.writer, "%s\n", o.color(ColorYellow, "WARNING: "+msg))
}

// Info prints an info message.
func (o *Output) Info(format string, args ...any) {
	if o.quiet {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(o.writer, "%s\n", msg)
}

// Success prints a success message in green.
func (o *Output) Success(format string, args ...any) {
	if o.quiet {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(o.writer, "%s\n", o.color(ColorGreen, msg))
}

// Debug prints a debug message (only in verbose mode).
func (o *Output) Debug(format string, args ...any) {
	if o.verbose < 1 {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(o.writer, "%s\n", o.color(ColorGray, "DEBUG: "+msg))
}

// Trace prints a trace message (only in very verbose mode).
func (o *Output) Trace(format string, args ...any) {
	if o.verbose < 2 {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(o.writer, "%s\n", o.color(ColorGray, "TRACE: "+msg))
}

// Section prints a section header.
func (o *Output) Section(title string) {
	if o.quiet {
		return
	}
	fmt.Fprintf(o.writer, "\n%s\n", o.color(ColorBold+ColorCyan, title))
	fmt.Fprintf(o.writer, "%s\n", strings.Repeat("─", len(title)))
}

// Statistics holds analysis statistics.
type Statistics struct {
	FilesAnalyzed   int
	TotalIssues     int
	Errors          int
	Warnings        int
	ByType          map[string]int
	Duration        time.Duration
	FilesWithIssues []string
}

// NewStatistics creates a new Statistics instance.
func NewStatistics() *Statistics {
	return &Statistics{
		ByType:          make(map[string]int),
		FilesWithIssues: make([]string, 0),
	}
}

// AddIssue adds an issue to statistics.
func (s *Statistics) AddIssue(issueType string, isError bool) {
	s.TotalIssues++
	if isError {
		s.Errors++
	} else {
		s.Warnings++
	}
	s.ByType[issueType]++
}

// PrintStatistics prints formatted statistics.
func (o *Output) PrintStatistics(stats *Statistics) {
	if o.quiet && stats.TotalIssues == 0 {
		return
	}

	o.Section("Analysis Summary")

	// Files analyzed
	fmt.Fprintf(o.writer, "Files analyzed:  %s\n",
		o.color(ColorBold, fmt.Sprintf("%d", stats.FilesAnalyzed)))

	// Duration
	fmt.Fprintf(o.writer, "Duration:        %s\n",
		o.color(ColorGray, stats.Duration.String()))

	// Issues found
	if stats.TotalIssues == 0 {
		fmt.Fprintf(o.writer, "\n%s\n",
			o.color(ColorGreen+ColorBold, "✓ No issues found!"))
		return
	}

	fmt.Fprintf(o.writer, "\nIssues found:    %s\n",
		o.color(ColorBold, fmt.Sprintf("%d", stats.TotalIssues)))

	if stats.Errors > 0 {
		fmt.Fprintf(o.writer, "  Errors:        %s\n",
			o.color(ColorRed+ColorBold, fmt.Sprintf("%d", stats.Errors)))
	}

	if stats.Warnings > 0 {
		fmt.Fprintf(o.writer, "  Warnings:      %s\n",
			o.color(ColorYellow, fmt.Sprintf("%d", stats.Warnings)))
	}

	// Breakdown by type
	if len(stats.ByType) > 0 && o.verbose > 0 {
		fmt.Fprintf(o.writer, "\nBreakdown by type:\n")
		for issueType, count := range stats.ByType {
			fmt.Fprintf(o.writer, "  - %s: %d\n", issueType, count)
		}
	}

	// Files with issues
	if len(stats.FilesWithIssues) > 0 && o.verbose > 0 {
		fmt.Fprintf(o.writer, "\nFiles with issues: %d\n", len(stats.FilesWithIssues))
		if o.verbose > 1 {
			for _, file := range stats.FilesWithIssues {
				fmt.Fprintf(o.writer, "  - %s\n", file)
			}
		}
	}
}

// PrintDiagnostic prints a single diagnostic with context.
func (o *Output) PrintDiagnostic(file string, line, col int, severity, message string) {
	if o.quiet && severity == "warning" {
		return
	}

	// Format: file:line:col: severity: message
	location := fmt.Sprintf("%s:%d:%d:", file, line, col)

	var severityColored string
	switch severity {
	case "error":
		severityColored = o.color(ColorRed+ColorBold, "error")
	case "warning":
		severityColored = o.color(ColorYellow, "warning")
	default:
		severityColored = severity
	}

	fmt.Fprintf(o.writer, "%s %s %s\n",
		o.color(ColorBold, location),
		severityColored+":",
		message)
}

// IsTerminal checks if output is a terminal (for color support detection).
func IsTerminal(f *os.File) bool {
	fileInfo, err := f.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// ShouldUseColors determines if colors should be used based on environment.
func ShouldUseColors() bool {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check TERM environment variable
	term := os.Getenv("TERM")
	if term == "dumb" {
		return false
	}

	// Check if stdout is a terminal
	return IsTerminal(os.Stdout)
}
