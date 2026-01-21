package dsl

import (
	"regexp"
	"strings"
)

// BuiltinFunctions provides functions available in DSL conditions.
// These are registered with expr-lang for use in "when" expressions.
var BuiltinFunctions = map[string]any{
	// String functions
	"contains":   strings.Contains,
	"hasPrefix":  strings.HasPrefix,
	"hasSuffix":  strings.HasSuffix,
	"toLower":    strings.ToLower,
	"toUpper":    strings.ToUpper,
	"trim":       strings.TrimSpace,
	"startsWith": strings.HasPrefix,
	"endsWith":   strings.HasSuffix,

	// Regex functions
	"matches":    matchesRegex,
	"notMatches": notMatchesRegex,

	// SQL-specific functions
	"isSystemTable": isSystemTable,
	"isAggregate":   isAggregate,
	"isTempTable":   isTempTable,

	// List functions
	"len":          length,
	"any":          anyMatch,
	"all":          allMatch,
	"contains_any": containsAny,
}

// matchesRegex checks if a string matches a regex pattern.
// Used as the =~ operator.
func matchesRegex(s, pattern string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(s)
}

// notMatchesRegex checks if a string does NOT match a regex pattern.
// Used as the !~ operator.
func notMatchesRegex(s, pattern string) bool {
	return !matchesRegex(s, pattern)
}

// isSystemTable checks if a table name is a system/catalog table.
func isSystemTable(table string) bool {
	table = strings.ToLower(table)

	// PostgreSQL system schemas
	if strings.HasPrefix(table, "pg_") ||
		strings.HasPrefix(table, "information_schema.") ||
		table == "information_schema" {
		return true
	}

	// MySQL system schemas
	if strings.HasPrefix(table, "mysql.") ||
		strings.HasPrefix(table, "performance_schema.") ||
		strings.HasPrefix(table, "sys.") {
		return true
	}

	// SQL Server system schemas
	if strings.HasPrefix(table, "sys.") ||
		strings.HasPrefix(table, "INFORMATION_SCHEMA.") {
		return true
	}

	// SQLite system tables
	if strings.HasPrefix(table, "sqlite_") {
		return true
	}

	return false
}

// isAggregate checks if a query contains aggregate functions.
func isAggregate(query string) bool {
	query = strings.ToUpper(query)
	aggregates := []string{
		"COUNT(", "SUM(", "AVG(", "MIN(", "MAX(",
		"GROUP_CONCAT(", "STRING_AGG(", "ARRAY_AGG(",
		"LISTAGG(", "XMLAGG(",
	}
	for _, agg := range aggregates {
		if strings.Contains(query, agg) {
			return true
		}
	}
	return false
}

// isTempTable checks if a table name looks like a temporary table.
func isTempTable(table string) bool {
	table = strings.ToLower(table)
	return strings.HasPrefix(table, "temp_") ||
		strings.HasPrefix(table, "tmp_") ||
		strings.HasPrefix(table, "#") || // SQL Server temp tables
		strings.HasSuffix(table, "_temp") ||
		strings.HasSuffix(table, "_tmp")
}

// length returns the length of a string or slice.
func length(v any) int {
	switch val := v.(type) {
	case string:
		return len(val)
	case []string:
		return len(val)
	case []any:
		return len(val)
	default:
		return 0
	}
}

// anyMatch checks if any element in a slice matches a condition.
func anyMatch(slice []string, substr string) bool {
	for _, s := range slice {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// allMatch checks if all elements in a slice match a condition.
func allMatch(slice []string, substr string) bool {
	if len(slice) == 0 {
		return false
	}
	for _, s := range slice {
		if !strings.Contains(s, substr) {
			return false
		}
	}
	return true
}

// containsAny checks if a string contains any of the given substrings.
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}

// GetBuiltinFunctionNames returns the names of all built-in functions.
func GetBuiltinFunctionNames() []string {
	names := make([]string, 0, len(BuiltinFunctions))
	for name := range BuiltinFunctions {
		names = append(names, name)
	}
	return names
}

// SQLKeywords contains common SQL keywords for pattern matching.
var SQLKeywords = []string{
	"SELECT", "FROM", "WHERE", "JOIN", "LEFT", "RIGHT", "INNER", "OUTER",
	"ON", "AND", "OR", "NOT", "IN", "EXISTS", "BETWEEN", "LIKE", "IS",
	"NULL", "ORDER", "BY", "GROUP", "HAVING", "LIMIT", "OFFSET",
	"INSERT", "INTO", "VALUES", "UPDATE", "SET", "DELETE",
	"CREATE", "ALTER", "DROP", "TABLE", "INDEX", "VIEW",
	"UNION", "INTERSECT", "EXCEPT", "ALL", "DISTINCT",
	"AS", "ASC", "DESC", "CASE", "WHEN", "THEN", "ELSE", "END",
}

// ExtractQueryType extracts the type of SQL query (SELECT, INSERT, etc.).
func ExtractQueryType(query string) string {
	query = strings.TrimSpace(strings.ToUpper(query))

	types := []string{"SELECT", "INSERT", "UPDATE", "DELETE", "CREATE", "ALTER", "DROP", "TRUNCATE"}
	for _, t := range types {
		if strings.HasPrefix(query, t) {
			return t
		}
	}

	// Handle WITH (CTE) - look for the main query type
	if strings.HasPrefix(query, "WITH") {
		for _, t := range types {
			if strings.Contains(query, t) {
				return t
			}
		}
	}

	return "UNKNOWN"
}

// HasWhereClause checks if a query has a WHERE clause.
func HasWhereClause(query string) bool {
	// Simple check - could be improved with proper SQL parsing
	query = strings.ToUpper(query)
	return strings.Contains(query, " WHERE ")
}

// HasJoinClause checks if a query has a JOIN clause.
func HasJoinClause(query string) bool {
	query = strings.ToUpper(query)
	return strings.Contains(query, " JOIN ")
}

// ExtractTableNames attempts to extract table names from a query.
// This is a simple implementation - for complex queries, use a proper SQL parser.
func ExtractTableNames(query string) []string {
	query = strings.ToUpper(query)
	var tables []string

	// Find tables after FROM
	_, after, ok := strings.Cut(query, " FROM ")
	if ok {
		afterFrom := after
		// Take until WHERE, JOIN, ORDER, GROUP, etc.
		for _, keyword := range []string{" WHERE ", " JOIN ", " ORDER ", " GROUP ", " LIMIT ", " HAVING ", ";"} {
			if idx := strings.Index(afterFrom, keyword); idx >= 0 {
				afterFrom = afterFrom[:idx]
			}
		}
		// Split by comma and clean up
		parts := strings.SplitSeq(afterFrom, ",")
		for part := range parts {
			part = strings.TrimSpace(part)
			// Remove alias
			if spaceIdx := strings.Index(part, " "); spaceIdx >= 0 {
				part = part[:spaceIdx]
			}
			if part != "" {
				tables = append(tables, part)
			}
		}
	}

	// Find tables after JOIN
	joinTypes := []string{" JOIN ", " LEFT JOIN ", " RIGHT JOIN ", " INNER JOIN ", " OUTER JOIN "}
	for _, jt := range joinTypes {
		idx := 0
		for {
			joinIdx := strings.Index(query[idx:], jt)
			if joinIdx < 0 {
				break
			}
			afterJoin := query[idx+joinIdx+len(jt):]
			// Take until ON, WHERE, etc.
			for _, keyword := range []string{" ON ", " WHERE ", " JOIN ", ";"} {
				if kidx := strings.Index(afterJoin, keyword); kidx >= 0 {
					afterJoin = afterJoin[:kidx]
				}
			}
			tableName := strings.TrimSpace(afterJoin)
			// Remove alias
			if spaceIdx := strings.Index(tableName, " "); spaceIdx >= 0 {
				tableName = tableName[:spaceIdx]
			}
			if tableName != "" {
				tables = append(tables, tableName)
			}
			idx += joinIdx + len(jt)
		}
	}

	return tables
}
