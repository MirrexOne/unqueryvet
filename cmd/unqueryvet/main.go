package main

import (
	"flag"
	"fmt"
	"os"

	internal "github.com/MirrexOne/unqueryvet/internal/analyzer"
	"github.com/MirrexOne/unqueryvet/internal/cli"
	"github.com/MirrexOne/unqueryvet/internal/runner"
	"github.com/MirrexOne/unqueryvet/internal/tui"
	"github.com/MirrexOne/unqueryvet/internal/version"
	"github.com/MirrexOne/unqueryvet/pkg/config"
)

var (
	versionFlag = flag.Bool("version", false, "print version information")
	verboseMode = flag.Bool("verbose", false, "enable verbose output with detailed explanations")
	quietFlag   = flag.Bool("quiet", false, "quiet mode (only errors)")
	statsFlag   = flag.Bool("stats", false, "show analysis statistics")
	noColorFlag = flag.Bool("no-color", false, "disable colored output")
	n1Flag      = flag.Bool("n1", false, "detect potential N+1 query problems")
	sqliFlag    = flag.Bool("sqli", false, "detect potential SQL injection vulnerabilities")
	txLeakFlag  = flag.Bool("tx-leak", false, "detect unclosed SQL transactions")
	fixFlag     = flag.Bool("fix", false, "interactive fix mode - step through issues and apply fixes")
)

func main() {
	// Parse flags
	flag.Parse()

	// Handle version flag
	if *versionFlag {
		info := version.GetInfo()
		fmt.Println(info.String())
		os.Exit(0)
	}

	// Determine if colors should be used
	useColors := cli.ShouldUseColors() && !*noColorFlag

	// Create output handler
	verboseLevel := 0
	if *verboseMode {
		verboseLevel = 1
	}
	out := cli.NewOutput(useColors, verboseLevel, *quietFlag)

	// Show version in verbose mode
	if *verboseMode {
		info := version.GetInfo()
		out.Debug("%s", info.Short())
	}

	// Interactive fix mode
	if *fixFlag {
		runFixMode()
		return
	}

	settings := config.DefaultSettings()

	settings.N1DetectionEnabled = *n1Flag
	settings.SQLInjectionDetectionEnabled = *sqliFlag
	settings.TxLeakDetectionEnabled = *txLeakFlag

	// Run the analyzer with our custom runner
	analyzer := internal.NewAnalyzerWithSettings(settings)
	exitCode := runner.Run(analyzer, out, *statsFlag)

	// Exit with proper code
	os.Exit(int(exitCode))
}

func runFixMode() {
	// Get paths to analyze
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}

	// Collect all Go files
	var files []string
	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			files = append(files, collectGoFiles(path)...)
		} else if isGoFile(path) {
			files = append(files, path)
		}
	}

	// Analyze files for SELECT * issues
	var issues []tui.Issue
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		content := string(data)
		lines := splitLines(content)

		for i, line := range lines {
			if containsSelectStar(line) && !isAllowedPattern(line) {
				issues = append(issues, tui.Issue{
					File:     file,
					Line:     i + 1,
					Column:   1,
					Message:  "avoid SELECT * - explicitly specify needed columns",
					CodeLine: line,
				})
			}
		}
	}

	// Run TUI
	if err := tui.Run(issues); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func containsSelectStar(line string) bool {
	// Simple check - in real implementation use regex
	return contains(line, "SELECT *") || contains(line, "SELECT\t*") ||
		contains(line, "select *") || contains(line, `Select("*")`)
}

func isAllowedPattern(line string) bool {
	return contains(line, "COUNT(*)") || contains(line, "count(*)") ||
		contains(line, "information_schema") || contains(line, "pg_catalog") ||
		(len(line) > 2 && line[0] == '/' && line[1] == '/')
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func collectGoFiles(dir string) []string {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return files
	}

	for _, entry := range entries {
		path := dir + "/" + entry.Name()
		if entry.IsDir() {
			// Skip common directories
			if entry.Name() == "vendor" || entry.Name() == "node_modules" ||
				entry.Name() == ".git" || entry.Name() == "testdata" {
				continue
			}
			files = append(files, collectGoFiles(path)...)
		} else if isGoFile(path) {
			files = append(files, path)
		}
	}
	return files
}

func isGoFile(path string) bool {
	return len(path) > 3 && path[len(path)-3:] == ".go"
}
