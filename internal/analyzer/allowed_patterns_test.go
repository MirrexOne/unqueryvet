package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/MirrexOne/unqueryvet/pkg/config"
)

func TestAllowedPatternsAnalysis(t *testing.T) {
	const source = `package p
func query(string) {}
func queries() {
	query("SELECT * FROM ALLOWED")
	query("SELECT * FROM BLOCKED")
}
`
	tests := []struct {
		name     string
		patterns []string
		lines    []int
	}{
		{name: "none", lines: []int{4, 5}},
		{name: "match", patterns: []string{`^SELECT \* FROM ALLOWED$`}, lines: []int{5}},
		{name: "invalid then match", patterns: []string{`[`, `^SELECT \* FROM ALLOWED$`}, lines: []int{5}},
		{name: "match then invalid", patterns: []string{`^SELECT \* FROM ALLOWED$`, `[`}, lines: []int{5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config.UnqueryvetSettings{AllowedPatterns: tt.patterns}
			pass := allowedPatternsPass(t, source)
			var lines []int
			pass.Report = func(d analysis.Diagnostic) {
				lines = append(lines, pass.Fset.Position(d.Pos).Line)
			}

			_, err := RunWithConfig(pass, cfg)
			require.NoError(t, err)
			assert.Equal(t, tt.lines, lines)
		})
	}
}

func TestAllowedPatternsAnalysisPaths(t *testing.T) {
	const source = `package p
const allowed = "SELECT * FROM ALLOWED"
var blocked = "SELECT * FROM BLOCKED"
func query(string) {}
func Sprintf(format string, args ...any) string { return format }
func queries() {
	assigned := "SELECT * FROM BLOCKED"
	query("SELECT * FROM ALLOWED")
	query("SELECT * FROM BLOCKED")
	query("SELECT * " + "FROM ALLOWED")
	query("SELECT * " + "FROM BLOCKED")
	query(Sprintf("SELECT * FROM ALLOWED WHERE id = %d", 1))
	query(Sprintf("SELECT * FROM BLOCKED WHERE id = %d", 1))
	_ = assigned
}
`
	cfg := &config.UnqueryvetSettings{
		AllowedPatterns:    []string{`[`, `^SELECT \* FROM ALLOWED`},
		CheckStringConcat:  true,
		CheckFormatStrings: true,
	}
	pass := allowedPatternsPass(t, source)
	var lines []int
	pass.Report = func(d analysis.Diagnostic) {
		lines = append(lines, pass.Fset.Position(d.Pos).Line)
	}

	_, err := RunWithConfig(pass, cfg)
	require.NoError(t, err)
	assert.Equal(t, []int{3, 7, 9, 11, 13}, lines)
}

func TestAllowedPatternsChangesDuringAnalysis(t *testing.T) {
	const source = `package p
func query(string) {}
func queries() {
	query("SELECT * FROM ALLOWED")
	query("SELECT * FROM BLOCKED")
	query("SELECT * FROM ALLOWED")
	query("SELECT * FROM BLOCKED")
}
`
	cfg := &config.UnqueryvetSettings{AllowedPatterns: []string{`^SELECT \* FROM ALLOWED$`}}
	pass := allowedPatternsPass(t, source)
	var lines []int
	pass.Report = func(d analysis.Diagnostic) {
		lines = append(lines, pass.Fset.Position(d.Pos).Line)
		cfg.AllowedPatterns[0] = `^SELECT \* FROM BLOCKED$`
	}

	_, err := RunWithConfig(pass, cfg)
	require.NoError(t, err)
	assert.Equal(t, []int{5, 6}, lines)
}

func TestInvalidAllowedPatternDisablesFilters(t *testing.T) {
	const source = `package p
func query(string) {}
func queries() {
	query("SELECT * FROM ALLOWED")
	query("SELECT * FROM BLOCKED")
}
`
	cfg := &config.UnqueryvetSettings{
		AllowedPatterns:  []string{`[`, `^SELECT \* FROM ALLOWED$`},
		IgnoredFunctions: []string{`query`},
		IgnoredFiles:     []string{`*.go`},
	}
	pass := allowedPatternsPass(t, source)
	var lines []int
	pass.Report = func(d analysis.Diagnostic) {
		lines = append(lines, pass.Fset.Position(d.Pos).Line)
	}

	_, err := RunWithConfig(pass, cfg)
	require.NoError(t, err)
	assert.Equal(t, []int{5}, lines)
}

func TestStandaloneAllowedPatternsChanges(t *testing.T) {
	cfg := &config.UnqueryvetSettings{AllowedPatterns: []string{`^SELECT \* FROM ALLOWED$`}}
	concat := NewStringConcatAnalyzer(nil, cfg)
	format := NewFormatStringAnalyzer(nil, cfg)
	binary, err := parser.ParseExpr(`"SELECT * " + "FROM ALLOWED"`)
	require.NoError(t, err)
	call, err := parser.ParseExpr(`Sprintf("SELECT * FROM ALLOWED")`)
	require.NoError(t, err)

	assert.False(t, IsSelectStarQuery("SELECT * FROM ALLOWED", cfg))
	assert.False(t, concat.AnalyzeBinaryExpr(binary.(*ast.BinaryExpr)))
	assert.False(t, format.AnalyzeFormatCall(call.(*ast.CallExpr)))

	cfg.AllowedPatterns[0] = `^SELECT \* FROM BLOCKED$`

	assert.True(t, IsSelectStarQuery("SELECT * FROM ALLOWED", cfg))
	assert.True(t, concat.AnalyzeBinaryExpr(binary.(*ast.BinaryExpr)))
	assert.True(t, format.AnalyzeFormatCall(call.(*ast.CallExpr)))
}

func TestAllowedPatternCache(t *testing.T) {
	patterns := make(allowedPatternCache)
	const pattern = `^SELECT \* FROM ALLOWED$`

	assert.True(t, patterns.matchString(pattern, "SELECT * FROM ALLOWED"))
	compiled := patterns[pattern]
	require.NotNil(t, compiled)
	assert.False(t, patterns.matchString(pattern, "SELECT * FROM BLOCKED"))
	assert.Same(t, compiled, patterns[pattern])

	assert.False(t, patterns.matchString("[", "SELECT * FROM ALLOWED"))
	assert.Contains(t, patterns, "[")
	assert.Nil(t, patterns["["])
	assert.False(t, patterns.matchString("[", "SELECT * FROM BLOCKED"))
	assert.Len(t, patterns, 2)
}

func TestAllowedPatternsConcurrentAnalysis(t *testing.T) {
	const source = `package p
func query(string) {}
func queries() {
	query("SELECT * FROM ALLOWED")
	query("SELECT * FROM BLOCKED")
}
`
	cfg := config.UnqueryvetSettings{AllowedPatterns: []string{`^SELECT \* FROM ALLOWED$`}}
	analyzer := NewAnalyzerWithSettings(cfg)
	for i := range 4 {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()

			pass := allowedPatternsPass(t, source)
			var lines []int
			pass.Report = func(d analysis.Diagnostic) {
				lines = append(lines, pass.Fset.Position(d.Pos).Line)
			}

			_, err := analyzer.Run(pass)
			require.NoError(t, err)
			assert.Equal(t, []int{5}, lines)
		})
	}
}

func allowedPatternsPass(tb testing.TB, source string) *analysis.Pass {
	tb.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "queries.go", source, parser.ParseComments)
	require.NoError(tb, err)
	files := []*ast.File{file}
	return &analysis.Pass{
		Fset:      fset,
		Files:     files,
		Pkg:       types.NewPackage("p", "p"),
		TypesInfo: &types.Info{},
		ResultOf: map[*analysis.Analyzer]any{
			inspect.Analyzer: inspector.New(files),
		},
		Report: func(analysis.Diagnostic) {},
	}
}
