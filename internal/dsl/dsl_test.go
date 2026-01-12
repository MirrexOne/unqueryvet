package dsl

import (
	"testing"
)

func TestPatternCompiler_Compile(t *testing.T) {
	pc := NewPatternCompiler()

	tests := []struct {
		name        string
		pattern     string
		input       string
		wantMatch   bool
		wantMetavar map[string]string
	}{
		{
			name:      "simple SELECT * pattern",
			pattern:   "SELECT * FROM $TABLE",
			input:     "SELECT * FROM users",
			wantMatch: true,
			wantMetavar: map[string]string{
				"TABLE": "users",
			},
		},
		{
			name:      "SELECT * with schema",
			pattern:   "SELECT * FROM $TABLE",
			input:     "SELECT * FROM public.users",
			wantMatch: true,
			wantMetavar: map[string]string{
				"TABLE": "public.users",
			},
		},
		{
			name:      "case insensitive",
			pattern:   "SELECT * FROM $TABLE",
			input:     "select * from users",
			wantMatch: true,
			wantMetavar: map[string]string{
				"TABLE": "users",
			},
		},
		{
			name:      "no match",
			pattern:   "SELECT * FROM $TABLE",
			input:     "SELECT id, name FROM users",
			wantMatch: false,
		},
		{
			name:      "DB.Query pattern",
			pattern:   "$DB.Query($QUERY)",
			input:     `db.Query("SELECT * FROM users")`,
			wantMatch: true,
			wantMetavar: map[string]string{
				"DB":    "db",
				"QUERY": `"SELECT * FROM users"`,
			},
		},
		{
			name:      "negated pattern - match means false",
			pattern:   "!SELECT * FROM $TABLE",
			input:     "SELECT * FROM users",
			wantMatch: false,
		},
		{
			name:      "negated pattern - no match means true",
			pattern:   "!SELECT * FROM $TABLE",
			input:     "SELECT id FROM users",
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp, err := pc.Compile(tt.pattern)
			if err != nil {
				t.Fatalf("Compile() error = %v", err)
			}

			metavars, matched := cp.Match(tt.input)
			if matched != tt.wantMatch {
				t.Errorf("Match() = %v, want %v", matched, tt.wantMatch)
			}

			if tt.wantMetavar != nil && matched {
				for k, v := range tt.wantMetavar {
					if got := metavars[k]; got != v {
						t.Errorf("metavar[%q] = %q, want %q", k, got, v)
					}
				}
			}
		})
	}
}

func TestParser_Parse(t *testing.T) {
	p := NewParser()

	yamlConfig := `
rules:
  select-star: error
  n1-queries: warning

ignore:
  - "*_test.go"
  - "testdata/**"

allow:
  - "COUNT(*)"

custom-rules:
  - id: temp-table-allowed
    pattern: SELECT * FROM $TABLE
    when: isTempTable(table)
    action: allow

  - id: n1-in-loop
    pattern: $DB.Query($QUERY)
    when: in_loop
    message: "N+1 query detected in loop"
    severity: error
`

	config, err := p.Parse([]byte(yamlConfig))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Check rules
	if config.Rules["select-star"] != SeverityError {
		t.Errorf("Rules[select-star] = %v, want error", config.Rules["select-star"])
	}
	if config.Rules["n1-queries"] != SeverityWarning {
		t.Errorf("Rules[n1-queries] = %v, want warning", config.Rules["n1-queries"])
	}

	// Check ignore
	if len(config.Ignore) != 2 {
		t.Errorf("len(Ignore) = %d, want 2", len(config.Ignore))
	}

	// Check allow
	if len(config.Allow) != 1 || config.Allow[0] != "COUNT(*)" {
		t.Errorf("Allow = %v, want [COUNT(*)]", config.Allow)
	}

	// Check custom rules
	if len(config.CustomRules) != 2 {
		t.Errorf("len(CustomRules) = %d, want 2", len(config.CustomRules))
	}

	rule0 := config.CustomRules[0]
	if rule0.ID != "temp-table-allowed" {
		t.Errorf("CustomRules[0].ID = %q, want %q", rule0.ID, "temp-table-allowed")
	}
	if rule0.Action != ActionAllow {
		t.Errorf("CustomRules[0].Action = %q, want %q", rule0.Action, ActionAllow)
	}

	rule1 := config.CustomRules[1]
	if rule1.ID != "n1-in-loop" {
		t.Errorf("CustomRules[1].ID = %q, want %q", rule1.ID, "n1-in-loop")
	}
	if rule1.Severity != SeverityError {
		t.Errorf("CustomRules[1].Severity = %q, want %q", rule1.Severity, SeverityError)
	}
}

func TestRule_Validate(t *testing.T) {
	tests := []struct {
		name    string
		rule    Rule
		wantErr bool
	}{
		{
			name: "valid rule",
			rule: Rule{
				ID:       "test-rule",
				Pattern:  "SELECT * FROM $TABLE",
				Severity: SeverityWarning,
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			rule: Rule{
				Pattern: "SELECT * FROM $TABLE",
			},
			wantErr: true,
		},
		{
			name: "missing pattern",
			rule: Rule{
				ID: "test-rule",
			},
			wantErr: true,
		},
		{
			name: "invalid severity",
			rule: Rule{
				ID:       "test-rule",
				Pattern:  "SELECT * FROM $TABLE",
				Severity: "critical",
			},
			wantErr: true,
		},
		{
			name: "invalid action",
			rule: Rule{
				ID:      "test-rule",
				Pattern: "SELECT * FROM $TABLE",
				Action:  "block",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.rule.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuiltinFunctions(t *testing.T) {
	t.Run("isSystemTable", func(t *testing.T) {
		tests := []struct {
			table string
			want  bool
		}{
			{"pg_catalog", true},
			{"pg_tables", true},
			{"information_schema.tables", true},
			{"users", false},
			{"public.orders", false},
		}

		for _, tt := range tests {
			if got := isSystemTable(tt.table); got != tt.want {
				t.Errorf("isSystemTable(%q) = %v, want %v", tt.table, got, tt.want)
			}
		}
	})

	t.Run("isTempTable", func(t *testing.T) {
		tests := []struct {
			table string
			want  bool
		}{
			{"temp_users", true},
			{"tmp_orders", true},
			{"users_temp", true},
			{"#temp", true},
			{"users", false},
		}

		for _, tt := range tests {
			if got := isTempTable(tt.table); got != tt.want {
				t.Errorf("isTempTable(%q) = %v, want %v", tt.table, got, tt.want)
			}
		}
	})

	t.Run("isAggregate", func(t *testing.T) {
		tests := []struct {
			query string
			want  bool
		}{
			{"SELECT COUNT(*) FROM users", true},
			{"SELECT SUM(amount) FROM orders", true},
			{"SELECT AVG(price) FROM products", true},
			{"SELECT * FROM users", false},
			{"SELECT id, name FROM users", false},
		}

		for _, tt := range tests {
			if got := isAggregate(tt.query); got != tt.want {
				t.Errorf("isAggregate(%q) = %v, want %v", tt.query, got, tt.want)
			}
		}
	})

	t.Run("matchesRegex", func(t *testing.T) {
		tests := []struct {
			s       string
			pattern string
			want    bool
		}{
			{"test_file.go", "_test.go$", false},
			{"file_test.go", "_test.go$", true},
			{"main.go", "_test.go$", false},
		}

		for _, tt := range tests {
			if got := matchesRegex(tt.s, tt.pattern); got != tt.want {
				t.Errorf("matchesRegex(%q, %q) = %v, want %v", tt.s, tt.pattern, got, tt.want)
			}
		}
	})
}

func TestExtractQueryType(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{"SELECT * FROM users", "SELECT"},
		{"select id from users", "SELECT"},
		{"INSERT INTO users VALUES (1)", "INSERT"},
		{"UPDATE users SET name = 'test'", "UPDATE"},
		{"DELETE FROM users WHERE id = 1", "DELETE"},
		{"CREATE TABLE users (id int)", "CREATE"},
		{"WITH cte AS (SELECT * FROM t) SELECT * FROM cte", "SELECT"},
		{"some random text", "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			if got := ExtractQueryType(tt.query); got != tt.want {
				t.Errorf("ExtractQueryType(%q) = %q, want %q", tt.query, got, tt.want)
			}
		})
	}
}

func TestHasWhereClause(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{"SELECT * FROM users WHERE id = 1", true},
		{"SELECT * FROM users", false},
		{"DELETE FROM users WHERE status = 'deleted'", true},
		{"UPDATE users SET active = false", false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			if got := HasWhereClause(tt.query); got != tt.want {
				t.Errorf("HasWhereClause(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestHasJoinClause(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{"SELECT * FROM users JOIN orders ON users.id = orders.user_id", true},
		{"SELECT * FROM users LEFT JOIN orders ON users.id = orders.user_id", true},
		{"SELECT * FROM users", false},
		{"SELECT * FROM users, orders", false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			if got := HasJoinClause(tt.query); got != tt.want {
				t.Errorf("HasJoinClause(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestCompiler_Compile(t *testing.T) {
	compiler := NewCompiler()

	rule := &Rule{
		ID:       "test-rule",
		Pattern:  "SELECT * FROM $TABLE",
		When:     "!isSystemTable(table)",
		Message:  "Avoid SELECT * from $TABLE",
		Severity: SeverityWarning,
	}

	compiled, err := compiler.Compile(rule)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}

	if compiled.Rule != rule {
		t.Error("compiled.Rule != rule")
	}

	if len(compiled.Patterns) != 1 {
		t.Errorf("len(compiled.Patterns) = %d, want 1", len(compiled.Patterns))
	}

	if compiled.ConditionProgram == nil {
		t.Error("compiled.ConditionProgram is nil")
	}
}

func TestEvaluator_EvaluateSQL(t *testing.T) {
	config := &Config{
		Allow: []string{"COUNT(*)"},
		CustomRules: []Rule{
			{
				ID:       "select-star",
				Pattern:  "SELECT * FROM $TABLE",
				When:     `!isSystemTable(table)`,
				Message:  "Avoid SELECT * from $TABLE",
				Severity: SeverityWarning,
			},
		},
	}

	evaluator, err := NewEvaluator(config)
	if err != nil {
		t.Fatalf("NewEvaluator() error = %v", err)
	}

	tests := []struct {
		name      string
		query     string
		wantMatch bool
		wantAllow bool
	}{
		{
			name:      "SELECT * triggers rule",
			query:     "SELECT * FROM users",
			wantMatch: true,
		},
		{
			name:      "COUNT(*) is allowed",
			query:     "SELECT COUNT(*) FROM users",
			wantMatch: false,
			wantAllow: true,
		},
		{
			name:      "explicit columns - no match",
			query:     "SELECT id, name FROM users",
			wantMatch: false,
		},
		{
			name:      "system table - condition fails",
			query:     "SELECT * FROM pg_catalog.pg_tables",
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &EvalContext{
				File:     "main.go",
				Package:  "main",
				Function: "GetUsers",
			}

			matches, err := evaluator.EvaluateSQL(ctx, tt.query)
			if err != nil {
				t.Fatalf("EvaluateSQL() error = %v", err)
			}

			gotMatch := len(matches) > 0
			if gotMatch != tt.wantMatch {
				t.Errorf("EvaluateSQL() match = %v, want %v", gotMatch, tt.wantMatch)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Rules: map[string]Severity{
					"select-star": SeverityError,
				},
				CustomRules: []Rule{
					{
						ID:      "test",
						Pattern: "SELECT * FROM $TABLE",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid custom rule",
			config: Config{
				CustomRules: []Rule{
					{
						ID: "", // missing ID
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParser_DefaultConfig(t *testing.T) {
	p := NewParser()
	config := p.DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	if len(config.Rules) == 0 {
		t.Error("DefaultConfig().Rules is empty")
	}

	if len(config.Ignore) == 0 {
		t.Error("DefaultConfig().Ignore is empty")
	}

	if len(config.Allow) == 0 {
		t.Error("DefaultConfig().Allow is empty")
	}
}

func TestParser_MergeConfigs(t *testing.T) {
	p := NewParser()

	config1 := &Config{
		Rules: map[string]Severity{
			"select-star": SeverityWarning,
		},
		Ignore: []string{"*_test.go"},
	}

	config2 := &Config{
		Rules: map[string]Severity{
			"select-star": SeverityError, // override
			"n1-queries":  SeverityWarning,
		},
		Ignore: []string{"vendor/**"},
	}

	merged := p.MergeConfigs(config1, config2)

	// Later config takes precedence for rules
	if merged.Rules["select-star"] != SeverityError {
		t.Errorf("merged.Rules[select-star] = %v, want error", merged.Rules["select-star"])
	}

	// Both rules should be present
	if merged.Rules["n1-queries"] != SeverityWarning {
		t.Errorf("merged.Rules[n1-queries] = %v, want warning", merged.Rules["n1-queries"])
	}

	// Ignore patterns should be appended
	if len(merged.Ignore) != 2 {
		t.Errorf("len(merged.Ignore) = %d, want 2", len(merged.Ignore))
	}
}
