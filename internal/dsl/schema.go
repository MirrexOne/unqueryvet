package dsl

// JSONSchema returns the JSON Schema for .unqueryvet.yaml configuration.
// This can be used by IDEs for validation and autocompletion.
func JSONSchema() string {
	return `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://github.com/MirrexOne/unqueryvet/schema.json",
  "title": "Unqueryvet Configuration",
  "description": "Configuration file for unqueryvet SQL static analyzer",
  "type": "object",
  "properties": {
    "rules": {
      "type": "object",
      "description": "Built-in rule severity configuration",
      "additionalProperties": {
        "type": "string",
        "enum": ["error", "warning", "info", "ignore"]
      },
      "examples": [
        {"select-star": "error", "n1-queries": "warning"}
      ]
    },
    "ignore": {
      "type": "array",
      "description": "File patterns to ignore",
      "items": {
        "type": "string"
      },
      "examples": [
        ["*_test.go", "testdata/**", "vendor/**"]
      ]
    },
    "allow": {
      "type": "array",
      "description": "SQL patterns to allow (whitelist)",
      "items": {
        "type": "string"
      },
      "examples": [
        ["COUNT(*)", "information_schema.*"]
      ]
    },
    "custom-rules": {
      "type": "array",
      "description": "User-defined custom rules",
      "items": {
        "$ref": "#/definitions/customRule"
      }
    },
    "check-sql-builders": {
      "type": "boolean",
      "description": "Enable SQL builder checking",
      "default": true
    },
    "allowed-patterns": {
      "type": "array",
      "description": "Legacy: regex patterns to allow",
      "items": {
        "type": "string"
      }
    },
    "ignored-functions": {
      "type": "array",
      "description": "Function name patterns to ignore",
      "items": {
        "type": "string"
      }
    },
    "ignored-files": {
      "type": "array",
      "description": "Legacy: file glob patterns to ignore",
      "items": {
        "type": "string"
      }
    },
    "severity": {
      "type": "string",
      "description": "Default severity level",
      "enum": ["error", "warning", "info"],
      "default": "warning"
    },
    "check-aliased-wildcard": {
      "type": "boolean",
      "description": "Check SELECT alias.* patterns",
      "default": true
    },
    "check-string-concat": {
      "type": "boolean",
      "description": "Check string concatenation",
      "default": true
    },
    "check-format-strings": {
      "type": "boolean",
      "description": "Check format functions like fmt.Sprintf",
      "default": true
    },
    "check-string-builder": {
      "type": "boolean",
      "description": "Check strings.Builder usage",
      "default": true
    },
    "check-subqueries": {
      "type": "boolean",
      "description": "Check SELECT * in subqueries",
      "default": true
    },
    "sql-builders": {
      "type": "object",
      "description": "SQL builder libraries to check",
      "properties": {
        "squirrel": {"type": "boolean", "default": true},
        "gorm": {"type": "boolean", "default": true},
        "sqlx": {"type": "boolean", "default": true},
        "ent": {"type": "boolean", "default": true},
        "pgx": {"type": "boolean", "default": true},
        "bun": {"type": "boolean", "default": true},
        "sqlboiler": {"type": "boolean", "default": true},
        "jet": {"type": "boolean", "default": true}
      }
    }
  },
  "definitions": {
    "customRule": {
      "type": "object",
      "required": ["id"],
      "properties": {
        "id": {
          "type": "string",
          "description": "Unique identifier for the rule",
          "pattern": "^[a-z][a-z0-9-]*$"
        },
        "pattern": {
          "type": "string",
          "description": "SQL or code pattern to match. Supports metavariables: $TABLE, $VAR, $QUERY, $COLS, $EXPR, $DB"
        },
        "patterns": {
          "type": "array",
          "description": "Multiple patterns for this rule",
          "items": {
            "type": "string"
          }
        },
        "when": {
          "type": "string",
          "description": "Condition expression (expr-lang). Available: file, package, function, query, query_type, table, tables, columns, has_join, has_where, in_loop, loop_depth, builder"
        },
        "message": {
          "type": "string",
          "description": "Diagnostic message to display"
        },
        "severity": {
          "type": "string",
          "description": "Severity level",
          "enum": ["error", "warning", "info", "ignore"],
          "default": "warning"
        },
        "action": {
          "type": "string",
          "description": "Action when pattern matches",
          "enum": ["report", "allow", "ignore"],
          "default": "report"
        },
        "fix": {
          "type": "string",
          "description": "Suggested fix message"
        }
      },
      "oneOf": [
        {"required": ["pattern"]},
        {"required": ["patterns"]}
      ]
    }
  },
  "additionalProperties": false
}`
}

// SchemaURL returns the URL for the JSON schema.
func SchemaURL() string {
	return "https://raw.githubusercontent.com/MirrexOne/unqueryvet/main/schema.json"
}

// BuiltinRuleDescriptions returns descriptions for built-in rules.
func BuiltinRuleDescriptions() map[string]string {
	return map[string]string{
		"select-star":   "Detects SELECT * usage which can cause performance issues and maintenance problems",
		"n1-queries":    "Detects N+1 query patterns in loops",
		"sql-injection": "Detects potential SQL injection vulnerabilities",
	}
}

// BuiltinVariableDescriptions returns descriptions for DSL variables.
func BuiltinVariableDescriptions() map[string]string {
	return map[string]string{
		"file":       "Current file path",
		"package":    "Current package name",
		"function":   "Current function name",
		"query":      "SQL query text",
		"query_type": "Query type: SELECT, INSERT, UPDATE, DELETE",
		"table":      "Primary table name in query",
		"tables":     "List of all table names in query",
		"columns":    "List of selected columns",
		"has_join":   "Whether query has JOIN clause",
		"has_where":  "Whether query has WHERE clause",
		"in_loop":    "Whether code is inside a loop",
		"loop_depth": "Nesting depth of loops",
		"builder":    "SQL builder type (gorm, squirrel, etc.)",
		"metavars":   "Captured metavariables from pattern matching",
	}
}

// BuiltinFunctionDescriptions returns descriptions for DSL functions.
func BuiltinFunctionDescriptions() map[string]string {
	return map[string]string{
		"contains":      "contains(str, substr) - Check if string contains substring",
		"hasPrefix":     "hasPrefix(str, prefix) - Check if string starts with prefix",
		"hasSuffix":     "hasSuffix(str, suffix) - Check if string ends with suffix",
		"startsWith":    "startsWith(str, prefix) - Alias for hasPrefix",
		"endsWith":      "endsWith(str, suffix) - Alias for hasSuffix",
		"toLower":       "toLower(str) - Convert string to lowercase",
		"toUpper":       "toUpper(str) - Convert string to uppercase",
		"trim":          "trim(str) - Trim whitespace from string",
		"matches":       "matches(str, regex) - Check if string matches regex",
		"notMatches":    "notMatches(str, regex) - Check if string does NOT match regex",
		"isSystemTable": "isSystemTable(table) - Check if table is a system/catalog table",
		"isAggregate":   "isAggregate(query) - Check if query contains aggregate functions",
		"isTempTable":   "isTempTable(table) - Check if table name looks like a temp table",
		"len":           "len(list) - Get length of string or list",
		"any":           "any(list, substr) - Check if any element contains substring",
		"all":           "all(list, substr) - Check if all elements contain substring",
		"contains_any":  "contains_any(str, ...substrs) - Check if string contains any of the substrings",
	}
}

// MetavariableDescriptions returns descriptions for pattern metavariables.
func MetavariableDescriptions() map[string]string {
	return map[string]string{
		"$TABLE":   "Matches a table name (identifier with optional schema prefix)",
		"$VAR":     "Matches a variable/identifier",
		"$QUERY":   "Matches a string literal (quoted)",
		"$COLS":    "Matches a column list (comma-separated or *)",
		"$COLUMNS": "Alias for $COLS",
		"$EXPR":    "Matches any expression (non-greedy)",
		"$DB":      "Matches a database/connection object",
	}
}
