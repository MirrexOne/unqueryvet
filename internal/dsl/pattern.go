package dsl

import (
	"regexp"
	"strings"
)

// PatternCompiler compiles DSL patterns into regex patterns.
type PatternCompiler struct {
	metavarPattern *regexp.Regexp
}

// NewPatternCompiler creates a new pattern compiler.
func NewPatternCompiler() *PatternCompiler {
	return &PatternCompiler{
		// Matches metavariables like $TABLE, $VAR, $QUERY, etc.
		metavarPattern: regexp.MustCompile(`\$([A-Z][A-Z0-9_]*)`),
	}
}

// CompiledPattern holds a compiled pattern with metavariable extraction.
type CompiledPattern struct {
	Original    string
	Regex       *regexp.Regexp
	Metavars    []string // Names of metavariables in order of capture groups
	IsNegated   bool     // Pattern starts with !
	IsMultiline bool     // Pattern spans multiple lines
}

// Compile compiles a DSL pattern into a regex-based CompiledPattern.
// Patterns support:
// - $VAR: matches any identifier (alphanumeric + underscore)
// - $TABLE: matches a table name
// - $QUERY: matches a string literal
// - Literal text: matched exactly (with regex escaping)
// - !pattern: negated pattern
func (pc *PatternCompiler) Compile(pattern string) (*CompiledPattern, error) {
	original := pattern

	// Check for negation
	isNegated := false
	if strings.HasPrefix(pattern, "!") {
		isNegated = true
		pattern = pattern[1:]
	}

	// Check for multiline
	isMultiline := strings.Contains(pattern, "\n")

	// Extract metavariables
	var metavars []string
	matches := pc.metavarPattern.FindAllStringSubmatch(pattern, -1)
	for _, m := range matches {
		metavars = append(metavars, m[1])
	}

	// Convert pattern to regex
	regexPattern := pc.patternToRegex(pattern)

	// Compile the regex
	flags := "(?i)" // Case insensitive by default
	if isMultiline {
		flags = "(?ims)"
	}
	regex, err := regexp.Compile(flags + regexPattern)
	if err != nil {
		return nil, err
	}

	return &CompiledPattern{
		Original:    original,
		Regex:       regex,
		Metavars:    metavars,
		IsNegated:   isNegated,
		IsMultiline: isMultiline,
	}, nil
}

// patternToRegex converts a DSL pattern to a regex pattern.
func (pc *PatternCompiler) patternToRegex(pattern string) string {
	var result strings.Builder

	// Escape special regex characters except for metavariables
	i := 0
	for i < len(pattern) {
		// Check if this is a metavariable
		if pattern[i] == '$' && i+1 < len(pattern) {
			end := i + 1
			for end < len(pattern) && isMetavarChar(pattern[end]) {
				end++
			}
			if end > i+1 {
				// This is a metavariable
				metavar := pattern[i+1 : end]
				result.WriteString(pc.metavarToRegex(metavar))
				i = end
				continue
			}
		}

		// Escape regex special characters
		if isRegexSpecial(pattern[i]) {
			result.WriteByte('\\')
		}
		result.WriteByte(pattern[i])
		i++
	}

	return result.String()
}

// metavarToRegex converts a metavariable name to a regex capture group.
func (pc *PatternCompiler) metavarToRegex(name string) string {
	// Different metavariable types have different patterns
	switch {
	case strings.HasPrefix(name, "TABLE"):
		// Table names: identifier with optional schema prefix
		return `(?P<` + name + `>[a-zA-Z_][a-zA-Z0-9_.]*)`
	case strings.HasPrefix(name, "QUERY"):
		// String literals (quoted)
		return `(?P<` + name + `>"[^"]*"|'[^']*')`
	case strings.HasPrefix(name, "COLS") || strings.HasPrefix(name, "COLUMNS"):
		// Column list: comma-separated identifiers or *
		return `(?P<` + name + `>\*|[a-zA-Z_][a-zA-Z0-9_,\s.]*)`
	case strings.HasPrefix(name, "EXPR"):
		// Any expression (non-greedy)
		return `(?P<` + name + `>.+?)`
	case strings.HasPrefix(name, "VAR"):
		// Variable/identifier
		return `(?P<` + name + `>[a-zA-Z_][a-zA-Z0-9_]*)`
	case strings.HasPrefix(name, "DB"):
		// Database/connection object
		return `(?P<` + name + `>[a-zA-Z_][a-zA-Z0-9_.]*)`
	default:
		// Generic identifier
		return `(?P<` + name + `>[a-zA-Z_][a-zA-Z0-9_]*)`
	}
}

// Match attempts to match a compiled pattern against text.
// Returns the captured metavariables if successful.
func (cp *CompiledPattern) Match(text string) (map[string]string, bool) {
	matches := cp.Regex.FindStringSubmatch(text)
	if matches == nil {
		if cp.IsNegated {
			return nil, true // Negated pattern: no match means success
		}
		return nil, false
	}

	if cp.IsNegated {
		return nil, false // Negated pattern: match means failure
	}

	// Extract named groups
	result := make(map[string]string)
	for i, name := range cp.Regex.SubexpNames() {
		if i > 0 && name != "" && i < len(matches) {
			result[name] = matches[i]
		}
	}

	return result, true
}

// MatchAll finds all matches in the text.
func (cp *CompiledPattern) MatchAll(text string) []map[string]string {
	if cp.IsNegated {
		// Negated patterns don't return multiple matches
		if _, ok := cp.Match(text); ok {
			return []map[string]string{{}}
		}
		return nil
	}

	allMatches := cp.Regex.FindAllStringSubmatch(text, -1)
	if allMatches == nil {
		return nil
	}

	var results []map[string]string
	for _, matches := range allMatches {
		result := make(map[string]string)
		for i, name := range cp.Regex.SubexpNames() {
			if i > 0 && name != "" && i < len(matches) {
				result[name] = matches[i]
			}
		}
		results = append(results, result)
	}

	return results
}

// isMetavarChar returns true if c is valid in a metavariable name.
func isMetavarChar(c byte) bool {
	return (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

// isRegexSpecial returns true if c is a special regex character.
func isRegexSpecial(c byte) bool {
	switch c {
	case '.', '+', '*', '?', '^', '$', '(', ')', '[', ']', '{', '}', '|', '\\':
		return true
	}
	return false
}

// CompilePatterns compiles multiple patterns.
func (pc *PatternCompiler) CompilePatterns(patterns []string) ([]*CompiledPattern, error) {
	var compiled []*CompiledPattern
	for _, p := range patterns {
		cp, err := pc.Compile(p)
		if err != nil {
			return nil, err
		}
		compiled = append(compiled, cp)
	}
	return compiled, nil
}
