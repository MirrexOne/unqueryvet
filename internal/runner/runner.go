// Package runner provides a custom analyzer runner with statistics and exit codes.
package runner

import (
	"flag"
	"fmt"
	"go/token"
	"go/types"
	"time"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/packages"

	"github.com/MirrexOne/unqueryvet/internal/cli"
)

// ExitCode represents the exit code for the analyzer.
type ExitCode int

const (
	// ExitSuccess indicates no issues found
	ExitSuccess ExitCode = 0
	// ExitWarnings indicates warnings were found
	ExitWarnings ExitCode = 1
	// ExitErrors indicates errors were found
	ExitErrors ExitCode = 2
	// ExitFailure indicates analysis failed
	ExitFailure ExitCode = 3
)

// Statistics holds analysis statistics.
type Statistics struct {
	FilesAnalyzed   int
	PackagesLoaded  int
	TotalIssues     int
	Errors          int
	Warnings        int
	ByType          map[string]int
	Duration        time.Duration
	FilesWithIssues map[string]int
	StartTime       time.Time
}

// NewStatistics creates a new Statistics instance.
func NewStatistics() *Statistics {
	return &Statistics{
		ByType:          make(map[string]int),
		FilesWithIssues: make(map[string]int),
		StartTime:       time.Now(),
	}
}

// AddDiagnostic adds a diagnostic to statistics.
func (s *Statistics) AddDiagnostic(d analysis.Diagnostic, fset *token.FileSet, severity string) {
	s.TotalIssues++

	if severity == "error" {
		s.Errors++
	} else {
		s.Warnings++
	}

	// Track by type (extract from message)
	msgType := extractMessageType(d.Message)
	s.ByType[msgType]++

	// Track file
	if fset != nil && d.Pos.IsValid() {
		file := fset.File(d.Pos)
		if file != nil {
			s.FilesWithIssues[file.Name()]++
		}
	}
}

// Finalize completes the statistics.
func (s *Statistics) Finalize() {
	s.Duration = time.Since(s.StartTime)
}

// extractMessageType extracts issue type from message.
func extractMessageType(message string) string {
	// Simple extraction based on message prefix
	if len(message) < 20 {
		return "other"
	}

	switch {
	case contains(message, "SQL builder"):
		return "sql_builder"
	case contains(message, "alias.*"):
		return "aliased_wildcard"
	case contains(message, "subquery"):
		return "subquery"
	case contains(message, "concatenat"):
		return "concatenation"
	case contains(message, "format string"):
		return "format_string"
	case contains(message, "empty Select"):
		return "empty_select"
	default:
		return "select_star"
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s[:len(substr)] == substr ||
			findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Run executes the analyzer with statistics collection and proper exit codes.
func Run(analyzer *analysis.Analyzer, output *cli.Output, showStats bool) ExitCode {
	// Get packages to analyze from command line
	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"."}
	}

	output.Debug("Loading packages: %v", patterns)

	// Load packages
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedTypesSizes |
			packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		output.Error("Failed to load packages: %v", err)
		return ExitFailure
	}

	if packages.PrintErrors(pkgs) > 0 {
		output.Debug("Some packages had errors during loading")
	}

	stats := NewStatistics()
	stats.PackagesLoaded = len(pkgs)
	hasErrors := false
	hasWarnings := false

	// Analyze each package
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			output.Debug("Package %s has errors, skipping", pkg.PkgPath)
			continue
		}

		output.Trace("Analyzing package: %s", pkg.PkgPath)
		stats.FilesAnalyzed += len(pkg.Syntax)

		// Create pass
		pass := &analysis.Pass{
			Analyzer:     analyzer,
			Fset:         pkg.Fset,
			Files:        pkg.Syntax,
			OtherFiles:   pkg.OtherFiles,
			IgnoredFiles: nil,
			Pkg:          pkg.Types,
			TypesInfo:    pkg.TypesInfo,
			TypesSizes:   pkg.TypesSizes,
			ResultOf:     make(map[*analysis.Analyzer]interface{}),
			Report: func(d analysis.Diagnostic) {
				// Determine severity (for now, treat all as warnings)
				severity := "warning"

				// Add to statistics
				stats.AddDiagnostic(d, pkg.Fset, severity)

				// Print diagnostic
				if d.Pos.IsValid() {
					position := pkg.Fset.Position(d.Pos)
					output.PrintDiagnostic(
						position.Filename,
						position.Line,
						position.Column,
						severity,
						d.Message,
					)
				} else {
					output.Warning("%s", d.Message)
				}

				if severity == "error" {
					hasErrors = true
				} else {
					hasWarnings = true
				}
			},
			AllObjectFacts:  nil,
			AllPackageFacts: nil,
			ImportObjectFact: func(obj types.Object, ptr analysis.Fact) bool {
				return false
			},
			ImportPackageFact: func(pkg *types.Package, ptr analysis.Fact) bool {
				return false
			},
			ExportObjectFact: func(obj types.Object, fact analysis.Fact) {
			},
			ExportPackageFact: func(fact analysis.Fact) {
			},
		}

		// Run requires first
		for _, req := range analyzer.Requires {
			output.Trace("Running required analyzer: %s", req.Name)
			result, err := req.Run(pass)
			if err != nil {
				output.Error("Required analyzer %s failed: %v", req.Name, err)
				return ExitFailure
			}
			pass.ResultOf[req] = result
		}

		// Run main analyzer
		_, err := analyzer.Run(pass)
		if err != nil {
			output.Error("Analysis failed for package %s: %v", pkg.PkgPath, err)
			return ExitFailure
		}
	}

	// Finalize statistics
	stats.Finalize()

	// Show statistics if requested
	if showStats {
		printStatistics(output, stats)
	}

	// Determine exit code
	if hasErrors {
		return ExitErrors
	}
	if hasWarnings {
		return ExitWarnings
	}
	return ExitSuccess
}

// printStatistics prints formatted statistics.
func printStatistics(output *cli.Output, stats *Statistics) {
	output.Section("Analysis Statistics")

	fmt.Printf("Packages loaded:  %d\n", stats.PackagesLoaded)
	fmt.Printf("Files analyzed:   %d\n", stats.FilesAnalyzed)
	fmt.Printf("Duration:         %s\n", stats.Duration.Round(time.Millisecond))

	if stats.TotalIssues == 0 {
		output.Success("\n✓ No issues found!")
		return
	}

	fmt.Printf("\nIssues found:     %d\n", stats.TotalIssues)

	if stats.Errors > 0 {
		fmt.Printf("  Errors:         %d\n", stats.Errors)
	}
	if stats.Warnings > 0 {
		fmt.Printf("  Warnings:       %d\n", stats.Warnings)
	}

	// Show breakdown by type
	if len(stats.ByType) > 0 {
		fmt.Printf("\nBreakdown by type:\n")
		for issueType, count := range stats.ByType {
			fmt.Printf("  %-20s %d\n", issueType+":", count)
		}
	}

	// Show files with issues
	if len(stats.FilesWithIssues) > 0 {
		fmt.Printf("\nFiles with issues: %d\n", len(stats.FilesWithIssues))
	}
}
