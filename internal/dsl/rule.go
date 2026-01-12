// Package dsl provides a domain-specific language for defining custom SQL analysis rules.
package dsl

import (
	"fmt"
	"regexp"
)

// Severity represents the severity level of a rule violation.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
	SeverityIgnore  Severity = "ignore"
)

// Action represents what to do when a pattern matches.
type Action string

const (
	ActionReport Action = "report" // Report as violation (default)
	ActionAllow  Action = "allow"  // Allow/whitelist this pattern
	ActionIgnore Action = "ignore" // Silently ignore
)

// Rule represents a custom analysis rule.
type Rule struct {
	// ID is a unique identifier for the rule.
	ID string `yaml:"id"`

	// Pattern is the SQL or code pattern to match.
	// Supports metavariables like $TABLE, $VAR, etc.
	Pattern string `yaml:"pattern"`

	// Patterns allows multiple patterns for a single rule.
	Patterns []string `yaml:"patterns,omitempty"`

	// When is an optional condition expression (evaluated with expr-lang).
	// Available variables: file, package, function, query, table, in_loop, etc.
	When string `yaml:"when,omitempty"`

	// Message is the diagnostic message shown when the rule triggers.
	Message string `yaml:"message,omitempty"`

	// Severity is the severity level (error, warning, info, ignore).
	Severity Severity `yaml:"severity,omitempty"`

	// Action determines what to do when the pattern matches.
	Action Action `yaml:"action,omitempty"`

	// Fix is an optional suggested fix message.
	Fix string `yaml:"fix,omitempty"`

	// compiledPatterns holds compiled regex patterns.
	compiledPatterns []*regexp.Regexp

	// compiledCondition holds the compiled expr program.
	compiledCondition interface{}
}

// Config represents the complete DSL configuration.
type Config struct {
	// Rules is a map of built-in rule IDs to their severity.
	// Example: {"select-star": "error", "n1-queries": "warning"}
	Rules map[string]Severity `yaml:"rules,omitempty"`

	// Ignore is a list of file patterns to ignore.
	Ignore []string `yaml:"ignore,omitempty"`

	// Allow is a list of SQL patterns to allow (whitelist).
	Allow []string `yaml:"allow,omitempty"`

	// CustomRules is a list of user-defined rules.
	CustomRules []Rule `yaml:"custom-rules,omitempty"`

	// LegacyConfig holds backward-compatible options.
	LegacyConfig `yaml:",inline"`
}

// LegacyConfig holds options from the original .unqueryvet.yaml format.
type LegacyConfig struct {
	CheckSQLBuilders bool     `yaml:"check-sql-builders,omitempty"`
	AllowedPatterns  []string `yaml:"allowed-patterns,omitempty"`
	IgnoredFiles     []string `yaml:"ignored-files,omitempty"`
}

// EvalContext provides context for evaluating rule conditions.
type EvalContext struct {
	// File context
	File     string `expr:"file"`
	Package  string `expr:"package"`
	Function string `expr:"function"`

	// SQL context
	Query     string   `expr:"query"`
	QueryType string   `expr:"query_type"` // SELECT, INSERT, UPDATE, DELETE
	Table     string   `expr:"table"`
	Tables    []string `expr:"tables"`
	Columns   []string `expr:"columns"`
	HasJoin   bool     `expr:"has_join"`
	HasWhere  bool     `expr:"has_where"`

	// Code context
	InLoop    bool   `expr:"in_loop"`
	LoopDepth int    `expr:"loop_depth"`
	Builder   string `expr:"builder"` // gorm, squirrel, sqlx, etc.

	// Metavariables captured from pattern matching
	Metavars map[string]string `expr:"metavars"`
}

// Match represents a single match of a rule against code.
type Match struct {
	Rule     *Rule
	File     string
	Line     int
	Column   int
	Message  string
	Severity Severity
	Fix      string
	Metavars map[string]string
}

// Validate checks if the rule configuration is valid.
func (r *Rule) Validate() error {
	if r.ID == "" {
		return fmt.Errorf("rule must have an id")
	}

	if r.Pattern == "" && len(r.Patterns) == 0 {
		return fmt.Errorf("rule %q must have a pattern or patterns", r.ID)
	}

	if r.Severity != "" && !isValidSeverity(r.Severity) {
		return fmt.Errorf("rule %q has invalid severity %q", r.ID, r.Severity)
	}

	if r.Action != "" && !isValidAction(r.Action) {
		return fmt.Errorf("rule %q has invalid action %q", r.ID, r.Action)
	}

	return nil
}

// Validate checks if the config is valid.
func (c *Config) Validate() error {
	for _, rule := range c.CustomRules {
		if err := rule.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// GetPatterns returns all patterns for the rule.
func (r *Rule) GetPatterns() []string {
	if r.Pattern != "" {
		return append([]string{r.Pattern}, r.Patterns...)
	}
	return r.Patterns
}

// GetSeverity returns the severity, defaulting to warning if not set.
func (r *Rule) GetSeverity() Severity {
	if r.Severity == "" {
		return SeverityWarning
	}
	return r.Severity
}

// GetAction returns the action, defaulting to report if not set.
func (r *Rule) GetAction() Action {
	if r.Action == "" {
		return ActionReport
	}
	return r.Action
}

// GetMessage returns the message, or a default if not set.
func (r *Rule) GetMessage() string {
	if r.Message == "" {
		return fmt.Sprintf("Rule %q triggered", r.ID)
	}
	return r.Message
}

func isValidSeverity(s Severity) bool {
	switch s {
	case SeverityError, SeverityWarning, SeverityInfo, SeverityIgnore:
		return true
	}
	return false
}

func isValidAction(a Action) bool {
	switch a {
	case ActionReport, ActionAllow, ActionIgnore:
		return true
	}
	return false
}
