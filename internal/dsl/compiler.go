package dsl

import (
	"fmt"
	"maps"
	"sync"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
)

// Compiler compiles DSL rules for efficient evaluation.
type Compiler struct {
	patternCompiler *PatternCompiler
	programCache    sync.Map // map[string]*vm.Program
}

// CompiledRule is a rule ready for evaluation.
type CompiledRule struct {
	Rule             *Rule
	Patterns         []*CompiledPattern
	ConditionProgram *vm.Program
}

// NewCompiler creates a new rule compiler.
func NewCompiler() *Compiler {
	return &Compiler{
		patternCompiler: NewPatternCompiler(),
	}
}

// Compile compiles a single rule.
func (c *Compiler) Compile(rule *Rule) (*CompiledRule, error) {
	// Compile patterns
	patterns, err := c.compilePatterns(rule)
	if err != nil {
		return nil, fmt.Errorf("rule %q: %w", rule.ID, err)
	}

	// Compile condition if present
	var condProgram *vm.Program
	if rule.When != "" {
		condProgram, err = c.compileCondition(rule.When)
		if err != nil {
			return nil, fmt.Errorf("rule %q condition: %w", rule.ID, err)
		}
	}

	return &CompiledRule{
		Rule:             rule,
		Patterns:         patterns,
		ConditionProgram: condProgram,
	}, nil
}

// CompileConfig compiles all rules in a configuration.
func (c *Compiler) CompileConfig(config *Config) ([]*CompiledRule, error) {
	var compiled []*CompiledRule

	for i := range config.CustomRules {
		rule := &config.CustomRules[i]
		cr, err := c.Compile(rule)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, cr)
	}

	return compiled, nil
}

// compilePatterns compiles all patterns for a rule.
func (c *Compiler) compilePatterns(rule *Rule) ([]*CompiledPattern, error) {
	patterns := rule.GetPatterns()
	if len(patterns) == 0 {
		return nil, fmt.Errorf("no patterns defined")
	}

	return c.patternCompiler.CompilePatterns(patterns)
}

// compileCondition compiles a condition expression.
func (c *Compiler) compileCondition(condition string) (*vm.Program, error) {
	// Check cache first
	if cached, ok := c.programCache.Load(condition); ok {
		return cached.(*vm.Program), nil
	}

	// Create environment with built-in functions for operator overloading
	env := map[string]any{
		// Context fields (will be filled at runtime)
		"file":       "",
		"package":    "",
		"function":   "",
		"query":      "",
		"query_type": "",
		"table":      "",
		"tables":     []string{},
		"columns":    []string{},
		"has_join":   false,
		"has_where":  false,
		"in_loop":    false,
		"loop_depth": 0,
		"builder":    "",
		"metavars":   map[string]string{},

		// Functions needed for operators
		"matches":    matchesRegex,
		"notMatches": notMatchesRegex,
	}

	// Add all built-in functions
	maps.Copy(env, BuiltinFunctions)

	// Compile with the environment
	program, err := expr.Compile(condition,
		expr.Env(env),
		expr.AsBool(),
		// Add custom operators
		expr.Operator("=~", "matches"),    // regex match
		expr.Operator("!~", "notMatches"), // regex not match
	)
	if err != nil {
		return nil, err
	}

	// Cache the compiled program
	c.programCache.Store(condition, program)

	return program, nil
}

// ClearCache clears the compiled program cache.
func (c *Compiler) ClearCache() {
	c.programCache = sync.Map{}
}
