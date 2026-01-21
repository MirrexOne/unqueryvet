package dsl

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// DSLAnalyzer wraps the DSL evaluator for use with go/analysis.
type DSLAnalyzer struct {
	evaluator *Evaluator
	config    *Config
}

// NewDSLAnalyzer creates a new DSL-based analyzer.
func NewDSLAnalyzer(config *Config) (*DSLAnalyzer, error) {
	evaluator, err := NewEvaluator(config)
	if err != nil {
		return nil, err
	}

	return &DSLAnalyzer{
		evaluator: evaluator,
		config:    config,
	}, nil
}

// CreateAnalyzer creates a go/analysis.Analyzer using DSL rules.
func (d *DSLAnalyzer) CreateAnalyzer() *analysis.Analyzer {
	return &analysis.Analyzer{
		Name:     "unqueryvet-dsl",
		Doc:      "custom rules defined via DSL configuration",
		Run:      d.run,
		Requires: []*analysis.Analyzer{inspect.Analyzer},
	}
}

// run is the main analysis function.
func (d *DSLAnalyzer) run(pass *analysis.Pass) (any, error) {
	// Check if file should be ignored
	if len(pass.Files) > 0 {
		fileName := pass.Fset.File(pass.Files[0].Pos()).Name()
		if d.evaluator.ShouldIgnoreFile(filepath.Base(fileName)) {
			return nil, nil
		}
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Track current function for context
	var currentFunc string
	var inLoop bool
	var loopDepth int

	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
		(*ast.FuncLit)(nil),
		(*ast.ForStmt)(nil),
		(*ast.RangeStmt)(nil),
		(*ast.CallExpr)(nil),
		(*ast.BasicLit)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.GenDecl)(nil),
	}

	insp.Nodes(nodeFilter, func(n ast.Node, push bool) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if push {
				currentFunc = node.Name.Name
			} else {
				currentFunc = ""
			}

		case *ast.FuncLit:
			if push {
				if currentFunc == "" {
					currentFunc = "<anonymous>"
				}
			}

		case *ast.ForStmt, *ast.RangeStmt:
			if push {
				loopDepth++
				inLoop = true
			} else {
				loopDepth--
				if loopDepth == 0 {
					inLoop = false
				}
			}

		case *ast.CallExpr:
			if push {
				d.checkCallExpr(pass, node, currentFunc, inLoop, loopDepth)
			}

		case *ast.BasicLit:
			if push && node.Kind == token.STRING {
				d.checkStringLiteral(pass, node, currentFunc, inLoop, loopDepth)
			}

		case *ast.AssignStmt:
			if push {
				for _, expr := range node.Rhs {
					if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
						d.checkStringLiteral(pass, lit, currentFunc, inLoop, loopDepth)
					}
				}
			}

		case *ast.GenDecl:
			if push && (node.Tok == token.CONST || node.Tok == token.VAR) {
				for _, spec := range node.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok {
						for _, value := range vs.Values {
							if lit, ok := value.(*ast.BasicLit); ok && lit.Kind == token.STRING {
								d.checkStringLiteral(pass, lit, currentFunc, inLoop, loopDepth)
							}
						}
					}
				}
			}
		}

		return true
	})

	return nil, nil
}

// checkCallExpr checks function calls for SQL queries.
func (d *DSLAnalyzer) checkCallExpr(pass *analysis.Pass, call *ast.CallExpr, funcName string, inLoop bool, loopDepth int) {
	// Determine builder type
	builder := d.detectBuilder(call)

	for _, arg := range call.Args {
		if lit, ok := arg.(*ast.BasicLit); ok && lit.Kind == token.STRING {
			d.evaluateSQL(pass, lit, funcName, inLoop, loopDepth, builder)
		}
	}
}

// checkStringLiteral checks a string literal for SQL queries.
func (d *DSLAnalyzer) checkStringLiteral(pass *analysis.Pass, lit *ast.BasicLit, funcName string, inLoop bool, loopDepth int) {
	d.evaluateSQL(pass, lit, funcName, inLoop, loopDepth, "")
}

// evaluateSQL evaluates DSL rules against a SQL string.
func (d *DSLAnalyzer) evaluateSQL(pass *analysis.Pass, lit *ast.BasicLit, funcName string, inLoop bool, loopDepth int, builder string) {
	query := strings.Trim(lit.Value, "`\"'")

	// Skip non-SQL strings (simple heuristic)
	if !looksLikeSQL(query) {
		return
	}

	// Build evaluation context
	fileName := pass.Fset.File(lit.Pos()).Name()
	ctx := &EvalContext{
		File:      fileName,
		Package:   pass.Pkg.Name(),
		Function:  funcName,
		InLoop:    inLoop,
		LoopDepth: loopDepth,
		Builder:   builder,
	}

	// Evaluate rules
	matches, err := d.evaluator.EvaluateSQL(ctx, query)
	if err != nil {
		// Log error but don't fail analysis
		return
	}

	// Report matches
	for _, match := range matches {
		pass.Report(analysis.Diagnostic{
			Pos:      lit.Pos(),
			End:      lit.End(),
			Message:  match.Message,
			Category: string(match.Severity),
		})
	}
}

// detectBuilder tries to detect which SQL builder is being used.
func (d *DSLAnalyzer) detectBuilder(call *ast.CallExpr) string {
	var funcName string

	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		funcName = fn.Sel.Name
		// Check receiver type
		if ident, ok := fn.X.(*ast.Ident); ok {
			switch strings.ToLower(ident.Name) {
			case "db", "tx":
				return "database/sql"
			case "gorm":
				return "gorm"
			case "sq", "squirrel":
				return "squirrel"
			case "bun":
				return "bun"
			}
		}
	case *ast.Ident:
		funcName = fn.Name
	}

	// Check function names
	switch strings.ToLower(funcName) {
	case "query", "queryrow", "exec", "prepare":
		return "database/sql"
	case "raw", "find", "where":
		return "gorm"
	case "select", "from":
		return "squirrel"
	}

	return ""
}

// looksLikeSQL checks if a string looks like an SQL query.
func looksLikeSQL(s string) bool {
	upper := strings.ToUpper(strings.TrimSpace(s))

	sqlKeywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE",
		"FROM", "WHERE", "JOIN", "CREATE", "ALTER", "DROP",
	}

	for _, kw := range sqlKeywords {
		if strings.Contains(upper, kw) {
			return true
		}
	}

	return false
}

// LoadAndCreateAnalyzer loads config from file and creates an analyzer.
func LoadAndCreateAnalyzer(configPath string) (*analysis.Analyzer, error) {
	parser := NewParser()

	var config *Config
	var err error

	if configPath != "" {
		config, err = parser.ParseFile(configPath)
	} else {
		config, err = parser.LoadConfig(".")
	}
	if err != nil {
		return nil, err
	}

	dslAnalyzer, err := NewDSLAnalyzer(config)
	if err != nil {
		return nil, err
	}

	return dslAnalyzer.CreateAnalyzer(), nil
}
