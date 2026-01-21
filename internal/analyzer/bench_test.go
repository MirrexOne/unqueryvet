package analyzer

import (
	"testing"

	"github.com/MirrexOne/unqueryvet/pkg/config"
)

// BenchmarkNormalizeSQLQuery tests the performance of SQL query normalization
func BenchmarkNormalizeSQLQuery(b *testing.B) {
	testQueries := []string{
		`"SELECT * FROM users"`,
		`"SELECT *\n\tFROM \"users\"\n\tWHERE id = 1"`,
		"`SELECT * FROM users WHERE active = 1`",
		`"SELECT * FROM users -- comment\nWHERE active = 1"`,
		`"SELECT   *   FROM   users   WHERE   id   =   1"`,
	}

	for _, query := range testQueries {
		b.Run("query_len_"+string(rune(len(query))), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = normalizeSQLQuery(query)
			}
		})
	}
}

// BenchmarkIsSelectStarQuery tests the performance of SELECT * detection
func BenchmarkIsSelectStarQuery(b *testing.B) {
	cfg := &config.UnqueryvetSettings{
		AllowedPatterns: []string{
			`SELECT \* FROM information_schema\..*`,
			`SELECT \* FROM pg_catalog\..*`,
			`(?i)COUNT\(\s*\*\s*\)`,
		},
	}

	testQueries := []string{
		"SELECT * FROM users",
		"SELECT * FROM users WHERE active = 1",
		"SELECT COUNT(*) FROM users",
		"SELECT * FROM information_schema.tables",
		"SELECT id, name, email FROM users",
		"INSERT INTO users VALUES (1, 'test')",
		"UPDATE users SET name = 'test' WHERE id = 1",
	}

	for _, query := range testQueries {
		b.Run("is_select_star", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = isSelectStarQuery(query, cfg)
			}
		})
	}
}

// BenchmarkIsFileInDirectory tests directory filtering performance
func BenchmarkIsFileInDirectory(b *testing.B) {
	testCases := []struct {
		path string
		dir  string
	}{
		{"/project/vendor/github.com/pkg/module.go", "vendor"},
		{"/project/src/main.go", "vendor"},
		{"/very/long/path/to/deeply/nested/file.go", "vendor"},
		{"/project/.git/config", ".git"},
		{"/project/testdata/input.go", "testdata"},
	}

	for _, tc := range testCases {
		b.Run("path_check", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				// Skip directory checking - ignore functionality removed
				_ = tc.path // Use path to avoid unused variable
			}
		})
	}
}

// BenchmarkFullAnalysisWorkflow benchmarks the complete analysis process
func BenchmarkFullAnalysisWorkflow(b *testing.B) {
	// This would benchmark the complete analysis workflow
	// For now, we'll benchmark the key components
	cfg := &config.UnqueryvetSettings{
		CheckSQLBuilders: true,
		AllowedPatterns: []string{
			`SELECT \* FROM information_schema\..*`,
			`(?i)COUNT\(\s*\*\s*\)`,
		},
	}

	queries := []string{
		`"SELECT * FROM users WHERE active = 1"`,
		`"SELECT COUNT(*) FROM orders"`,
		`"SELECT id, name FROM products"`,
		`"INSERT INTO logs (message) VALUES ('test')"`,
	}

	for b.Loop() {
		for _, query := range queries {
			normalized := normalizeSQLQuery(query)
			_ = isSelectStarQuery(normalized, cfg)
		}
	}
}

// BenchmarkRegexPatterns tests the performance of regex pattern matching
func BenchmarkRegexPatterns(b *testing.B) {
	cfg := &config.UnqueryvetSettings{
		AllowedPatterns: []string{
			`SELECT \* FROM information_schema\..*`,
			`SELECT \* FROM pg_catalog\..*`,
			`SELECT \* FROM sys\..*`,
			`(?i)COUNT\(\s*\*\s*\)`,
			`(?i)MAX\(\s*\*\s*\)`,
			`(?i)MIN\(\s*\*\s*\)`,
			`(?i)EXISTS\s*\(`,
		},
	}

	queries := []string{
		"SELECT * FROM information_schema.tables",
		"SELECT * FROM pg_catalog.pg_tables",
		"SELECT COUNT(*) FROM users",
		"SELECT * FROM users WHERE active = 1",
		"SELECT MAX(*) FROM scores",
	}

	for b.Loop() {
		for _, query := range queries {
			_ = isSelectStarQuery(query, cfg)
		}
	}
}
