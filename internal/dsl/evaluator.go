package dsl

import (
	"fmt"
	"maps"
	"slices"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Evaluator executes compiled DSL rules against code/SQL.
type Evaluator struct {
	compiler *Compiler
	rules    []*CompiledRule
	config   *Config
}

// NewEvaluator creates a new rule evaluator.
func NewEvaluator(config *Config) (*Evaluator, error) {
	compiler := NewCompiler()

	rules, err := compiler.CompileConfig(config)
	if err != nil {
		return nil, err
	}

	return &Evaluator{
		compiler: compiler,
		rules:    rules,
		config:   config,
	}, nil
}

// EvaluateSQL evaluates all rules against a SQL query.
func (e *Evaluator) EvaluateSQL(ctx *EvalContext, query string) ([]*Match, error) {
	var matches []*Match

	// Update context with query info
	ctx.Query = query
	ctx.QueryType = ExtractQueryType(query)
	ctx.Tables = ExtractTableNames(query)
	if len(ctx.Tables) > 0 {
		ctx.Table = ctx.Tables[0]
	}
	ctx.HasWhere = HasWhereClause(query)
	ctx.HasJoin = HasJoinClause(query)

	// Check allow patterns first
	if e.isAllowed(query) {
		return nil, nil
	}

	// Evaluate each custom rule
	for _, rule := range e.rules {
		match, err := e.evaluateRule(rule, ctx, query)
		if err != nil {
			return nil, fmt.Errorf("rule %q: %w", rule.Rule.ID, err)
		}
		if match != nil {
			matches = append(matches, match)
		}
	}

	return matches, nil
}

// evaluateRule evaluates a single rule against the context.
func (e *Evaluator) evaluateRule(rule *CompiledRule, ctx *EvalContext, query string) (*Match, error) {
	// Try to match any pattern
	var matchedMetavars map[string]string
	var matched bool

	for _, pattern := range rule.Patterns {
		metavars, ok := pattern.Match(query)
		if ok {
			matched = true
			matchedMetavars = metavars
			break
		}
	}

	if !matched {
		return nil, nil
	}

	// Update context with captured metavariables
	ctx.Metavars = matchedMetavars

	// Evaluate condition if present
	if rule.ConditionProgram != nil {
		result, err := e.evaluateCondition(rule.ConditionProgram, ctx)
		if err != nil {
			return nil, err
		}
		if !result {
			return nil, nil
		}
	}

	// Check action
	switch rule.Rule.GetAction() {
	case ActionAllow, ActionIgnore:
		return nil, nil
	}

	// Create match
	return &Match{
		Rule:     rule.Rule,
		Message:  e.formatMessage(rule.Rule.GetMessage(), matchedMetavars),
		Severity: rule.Rule.GetSeverity(),
		Fix:      rule.Rule.Fix,
		Metavars: matchedMetavars,
	}, nil
}

// evaluateCondition evaluates a compiled condition expression.
func (e *Evaluator) evaluateCondition(program *vm.Program, ctx *EvalContext) (bool, error) {
	// Create environment with context and built-in functions
	env := map[string]any{
		// File context
		"file":     ctx.File,
		"package":  ctx.Package,
		"function": ctx.Function,

		// SQL context
		"query":      ctx.Query,
		"query_type": ctx.QueryType,
		"table":      ctx.Table,
		"tables":     ctx.Tables,
		"columns":    ctx.Columns,
		"has_join":   ctx.HasJoin,
		"has_where":  ctx.HasWhere,

		// Code context
		"in_loop":    ctx.InLoop,
		"loop_depth": ctx.LoopDepth,
		"builder":    ctx.Builder,

		// Metavariables
		"metavars": ctx.Metavars,
	}

	// Add built-in functions
	maps.Copy(env, BuiltinFunctions)

	result, err := expr.Run(program, env)
	if err != nil {
		return false, err
	}

	boolResult, ok := result.(bool)
	if !ok {
		return false, fmt.Errorf("condition must return bool, got %T", result)
	}

	return boolResult, nil
}

// isAllowed checks if a query matches any allow pattern.
func (e *Evaluator) isAllowed(query string) bool {
	for _, pattern := range e.config.Allow {
		cp, err := e.compiler.patternCompiler.Compile(pattern)
		if err != nil {
			continue
		}
		if _, ok := cp.Match(query); ok {
			return true
		}
	}
	return false
}

// formatMessage formats a message with metavariable substitution.
func (e *Evaluator) formatMessage(msg string, metavars map[string]string) string {
	result := msg
	for name, value := range metavars {
		result = replaceAll(result, "$"+name, value)
	}
	return result
}

// replaceAll is a simple string replacement (avoiding regexp for performance).
func replaceAll(s, old, new string) string {
	for {
		i := indexOf(s, old)
		if i < 0 {
			return s
		}
		s = s[:i] + new + s[i+len(old):]
	}
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// GetBuiltinRules returns the IDs of built-in rules.
func GetBuiltinRules() []string {
	return []string{
		"select-star",
		"n1-queries",
		"sql-injection",
	}
}

// IsBuiltinRule checks if a rule ID is a built-in rule.
func IsBuiltinRule(id string) bool {
	return slices.Contains(GetBuiltinRules(), id)
}

// GetRuleSeverity returns the severity for a rule from config.
func (e *Evaluator) GetRuleSeverity(ruleID string) Severity {
	if sev, ok := e.config.Rules[ruleID]; ok {
		return sev
	}
	return SeverityWarning
}

// ShouldIgnoreFile checks if a file should be ignored based on config.
func (e *Evaluator) ShouldIgnoreFile(file string) bool {
	for _, pattern := range e.config.Ignore {
		cp, err := e.compiler.patternCompiler.Compile(pattern)
		if err != nil {
			continue
		}
		if _, ok := cp.Match(file); ok {
			return true
		}
	}
	return false
}
