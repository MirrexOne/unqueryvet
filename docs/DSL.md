# Custom Rules DSL

Unqueryvet provides a powerful Domain-Specific Language (DSL) for defining custom SQL analysis rules. This allows you to create project-specific rules that match your team's conventions and requirements.

## Table of Contents

- [Quick Start](#quick-start)
- [Three Levels of Configuration](#three-levels-of-configuration)
- [Pattern Syntax](#pattern-syntax)
- [Condition Expressions](#condition-expressions)
- [Built-in Variables](#built-in-variables)
- [Built-in Functions](#built-in-functions)
- [Operators](#operators)
- [Complete Examples](#complete-examples)
- [JSON Schema for IDE Support](#json-schema-for-ide-support)

---

## Quick Start

Create a `.unqueryvet.yaml` file in your project root:

```yaml
# Simple rule configuration
rules:
  select-star: warning
  n1-queries: warning
  sql-injection: error

# Files to ignore
ignore:
  - "*_test.go"
  - "testdata/**"

# SQL patterns to whitelist
allow:
  - "COUNT(*)"
```

That's it! Run `unqueryvet ./...` and it will use your configuration.

---

## Three Levels of Configuration

The DSL follows a **progressive disclosure** principle - simple cases are simple, complex cases are possible.

### Level 1: Simple Configuration

Configure built-in rules and basic filtering:

```yaml
# Built-in rule severity
rules:
  select-star: error      # error | warning | info | ignore
  n1-queries: warning
  sql-injection: error

# File patterns to ignore (glob syntax)
ignore:
  - "*_test.go"
  - "testdata/**"
  - "vendor/**"
  - "mock_*.go"

# SQL patterns to allow (won't trigger warnings)
allow:
  - "COUNT(*)"
  - "information_schema.*"
  - "pg_catalog.*"
```

### Level 2: Pattern Matching

Define custom rules with SQL/code patterns:

```yaml
custom-rules:
  # Allow SELECT * for temporary tables
  - id: allow-temp-tables
    pattern: SELECT * FROM $TABLE
    when: isTempTable(table)
    action: allow

  # Detect dangerous DELETE without WHERE
  - id: dangerous-delete
    pattern: DELETE FROM $TABLE
    when: "!has_where"
    message: "DELETE without WHERE clause is dangerous"
    severity: error
    fix: "Add a WHERE clause to limit affected rows"
```

### Level 3: Advanced Conditions

Use full expression language for complex rules:

```yaml
custom-rules:
  # N+1 detection with exceptions
  - id: n1-in-loop
    pattern: $DB.Query($QUERY)
    when: |
      in_loop && 
      !contains(function, "batch") &&
      !contains(function, "bulk") &&
      !matches(file, "_test.go$")
    message: "Potential N+1 query detected in loop"
    severity: warning
    fix: "Consider using batch query, JOIN, or preloading"

  # Strict rules for production code only
  - id: strict-select-star
    pattern: SELECT * FROM $TABLE
    when: |
      !matches(file, "_test.go$") &&
      !matches(file, "testdata/") &&
      !isSystemTable(table) &&
      !isTempTable(table)
    message: "Avoid SELECT * in production code"
    severity: error
```

---

## Pattern Syntax

Patterns use a simple syntax with **metavariables** that capture parts of the matched text.

### Metavariables

| Metavariable | Matches | Example |
|--------------|---------|---------|
| `$TABLE` | Table name (with optional schema) | `users`, `public.orders` |
| `$VAR` | Identifier/variable name | `userID`, `db` |
| `$QUERY` | String literal (quoted) | `"SELECT * FROM users"` |
| `$COLS` | Column list or `*` | `id, name, email` |
| `$COLUMNS` | Alias for `$COLS` | Same as above |
| `$EXPR` | Any expression (non-greedy) | `user.ID + 1` |
| `$DB` | Database/connection object | `db`, `tx`, `conn` |

### Pattern Examples

```yaml
# Match SELECT * queries
pattern: SELECT * FROM $TABLE

# Match with schema prefix
pattern: SELECT * FROM $TABLE   # Matches: SELECT * FROM public.users

# Match database Query calls
pattern: $DB.Query($QUERY)

# Match GORM-style calls
pattern: $DB.Find($VAR)

# Match multiple patterns (any match triggers)
patterns:
  - DELETE FROM $TABLE
  - UPDATE $TABLE SET $COLS
```

### Negated Patterns

Prefix with `!` to match when pattern does NOT match:

```yaml
# Match queries that DON'T have WHERE
pattern: "!WHERE"
```

---

## Condition Expressions

The `when` field uses [expr-lang](https://github.com/expr-lang/expr) for powerful condition expressions.

### Basic Syntax

```yaml
# Simple boolean check
when: in_loop

# Negation
when: "!has_where"

# Combining conditions
when: in_loop && !isSystemTable(table)

# Complex conditions (use | for multiline)
when: |
  in_loop && 
  loop_depth > 1 &&
  !contains(function, "batch")
```

### String Matching

```yaml
# Contains check
when: contains(file, "internal/")

# Regex match with =~ operator
when: file =~ "_test.go$"

# Regex NOT match with !~ operator
when: file !~ "_test.go$"

# Using matches() function
when: matches(function, "^(Get|List|Find)")
```

---

## Built-in Variables

Variables provide context about the current code location and SQL query.

### File Context

| Variable | Type | Description |
|----------|------|-------------|
| `file` | string | Current file path |
| `package` | string | Current package name |
| `function` | string | Current function/method name |

### SQL Context

| Variable | Type | Description |
|----------|------|-------------|
| `query` | string | Full SQL query text |
| `query_type` | string | Query type: `SELECT`, `INSERT`, `UPDATE`, `DELETE` |
| `table` | string | Primary table name |
| `tables` | []string | List of all table names in query |
| `columns` | []string | List of selected columns |
| `has_join` | bool | True if query contains JOIN |
| `has_where` | bool | True if query contains WHERE |

### Code Context

| Variable | Type | Description |
|----------|------|-------------|
| `in_loop` | bool | True if code is inside a loop (for, range) |
| `loop_depth` | int | Nesting depth of loops (0 = not in loop) |
| `builder` | string | SQL builder type: `gorm`, `squirrel`, `sqlx`, etc. |

### Captured Metavariables

| Variable | Type | Description |
|----------|------|-------------|
| `metavars` | map[string]string | Captured values from pattern metavariables |

Access captured values: `metavars["TABLE"]`, `metavars["VAR"]`

---

## Built-in Functions

### String Functions

| Function | Description | Example |
|----------|-------------|---------|
| `contains(s, substr)` | Check if string contains substring | `contains(file, "test")` |
| `hasPrefix(s, prefix)` | Check if string starts with prefix | `hasPrefix(table, "temp_")` |
| `hasSuffix(s, suffix)` | Check if string ends with suffix | `hasSuffix(file, ".go")` |
| `startsWith(s, prefix)` | Alias for `hasPrefix` | Same as above |
| `endsWith(s, suffix)` | Alias for `hasSuffix` | Same as above |
| `toLower(s)` | Convert to lowercase | `toLower(table)` |
| `toUpper(s)` | Convert to uppercase | `toUpper(query_type)` |
| `trim(s)` | Trim whitespace | `trim(query)` |

### Regex Functions

| Function | Description | Example |
|----------|-------------|---------|
| `matches(s, pattern)` | Check if string matches regex | `matches(file, "_test\\.go$")` |
| `notMatches(s, pattern)` | Check if string does NOT match | `notMatches(file, "_test\\.go$")` |

### SQL-Specific Functions

| Function | Description | Example |
|----------|-------------|---------|
| `isSystemTable(table)` | Check if table is system/catalog table | `isSystemTable(table)` |
| `isTempTable(table)` | Check if table looks like temp table | `isTempTable(table)` |
| `isAggregate(query)` | Check if query has aggregate functions | `isAggregate(query)` |

**System tables include:**
- PostgreSQL: `pg_*`, `information_schema.*`
- MySQL: `mysql.*`, `performance_schema.*`, `sys.*`
- SQL Server: `sys.*`, `INFORMATION_SCHEMA.*`
- SQLite: `sqlite_*`

**Temp tables match:**
- `temp_*`, `tmp_*`, `#*` (SQL Server), `*_temp`, `*_tmp`

### List Functions

| Function | Description | Example |
|----------|-------------|---------|
| `len(list)` | Get length of list or string | `len(tables) > 1` |
| `any(list, substr)` | Check if any element contains substr | `any(columns, "password")` |
| `all(list, substr)` | Check if all elements contain substr | `all(tables, "audit_")` |

---

## Operators

### Comparison Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `==` | Equal | `query_type == "SELECT"` |
| `!=` | Not equal | `builder != "gorm"` |
| `>`, `<` | Greater/less than | `loop_depth > 1` |
| `>=`, `<=` | Greater/less or equal | `len(tables) >= 2` |

### Logical Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `&&` | Logical AND | `in_loop && !has_where` |
| `\|\|` | Logical OR | `isSystemTable(table) \|\| isTempTable(table)` |
| `!` | Logical NOT | `!in_loop` |

### Regex Operators

| Operator | Description | Example |
|----------|-------------|---------|
| `=~` | Regex match | `file =~ "_test\\.go$"` |
| `!~` | Regex NOT match | `file !~ "vendor/"` |

---

## Complete Examples

### Example 1: Allow SELECT * for Specific Tables

```yaml
custom-rules:
  - id: allow-audit-tables
    pattern: SELECT * FROM $TABLE
    when: |
      hasPrefix(table, "audit_") ||
      hasPrefix(table, "log_") ||
      isSystemTable(table)
    action: allow
```

### Example 2: Require WHERE for Destructive Operations

```yaml
custom-rules:
  - id: require-where-delete
    pattern: DELETE FROM $TABLE
    when: "!has_where"
    message: "DELETE without WHERE clause can delete all rows"
    severity: error
    fix: "Add WHERE clause: DELETE FROM $TABLE WHERE condition"

  - id: require-where-update
    pattern: UPDATE $TABLE SET $COLS
    when: "!has_where"
    message: "UPDATE without WHERE clause can modify all rows"
    severity: error
    fix: "Add WHERE clause: UPDATE $TABLE SET ... WHERE condition"
```

### Example 3: N+1 Query Detection with Exceptions

```yaml
custom-rules:
  - id: n1-query-detection
    pattern: $DB.Query($QUERY)
    when: |
      in_loop &&
      !contains(function, "batch") &&
      !contains(function, "bulk") &&
      !contains(function, "Preload") &&
      !matches(file, "_test\\.go$") &&
      !matches(file, "testdata/")
    message: "Potential N+1 query: database call inside loop"
    severity: warning
    fix: |
      Consider one of these solutions:
      1. Use JOIN to fetch related data in single query
      2. Use IN clause with collected IDs
      3. Use batch/bulk loading functions
      4. Use ORM preloading (e.g., GORM Preload())
```

### Example 4: Environment-Aware Rules

```yaml
custom-rules:
  # Strict in production code
  - id: strict-prod-select-star
    pattern: SELECT * FROM $TABLE
    when: |
      !matches(file, "_test\\.go$") &&
      !matches(file, "testdata/") &&
      !matches(file, "internal/debug/") &&
      !isSystemTable(table) &&
      !isTempTable(table)
    severity: error
    message: "SELECT * is not allowed in production code"

  # Warning in test code
  - id: test-select-star
    pattern: SELECT * FROM $TABLE
    when: matches(file, "_test\\.go$")
    severity: info
    message: "Consider using explicit columns even in tests"
```

### Example 5: SQL Builder Specific Rules

```yaml
custom-rules:
  - id: gorm-raw-query
    pattern: $DB.Raw($QUERY)
    when: |
      builder == "gorm" &&
      contains(query, "SELECT *")
    message: "Avoid SELECT * in GORM Raw() calls"
    severity: warning
    fix: "Use db.Select(columns).Find() instead of Raw()"

  - id: squirrel-columns-star
    patterns:
      - $VAR.Select("*")
      - $VAR.Columns("*")
    when: builder == "squirrel"
    message: "Avoid wildcard in Squirrel Select/Columns"
    severity: warning
```

---

## JSON Schema for IDE Support

unqueryvet configuration files (`.unqueryvet.yaml`, `.unqueryvet.yml`) have JSON Schema support registered in [SchemaStore.org](https://www.schemastore.org/).

### Automatic Support

Most modern editors automatically provide autocompletion and validation:

| Editor | Support |
|--------|---------|
| VS Code + YAML extension | Automatic |
| GoLand / IntelliJ | Automatic |
| Neovim + yaml-language-server | Automatic |
| Sublime Text + LSP-yaml | Automatic |

### Features

- **Autocompletion** - Press `Ctrl+Space` to see available options
- **Validation** - Real-time error highlighting for invalid values
- **Documentation** - Hover over properties for descriptions

---

## Rule Properties Reference

| Property | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `id` | string | Yes | - | Unique identifier for the rule |
| `pattern` | string | Yes* | - | SQL/code pattern to match |
| `patterns` | []string | Yes* | - | Multiple patterns (any match triggers) |
| `when` | string | No | - | Condition expression |
| `message` | string | No | Auto | Diagnostic message |
| `severity` | string | No | `warning` | `error`, `warning`, `info`, `ignore` |
| `action` | string | No | `report` | `report`, `allow`, `ignore` |
| `fix` | string | No | - | Suggested fix message |

*Either `pattern` or `patterns` is required.

---

## Troubleshooting

### Rule Not Triggering

1. Check pattern syntax - metavariables are case-sensitive (`$TABLE` not `$table`)
2. Verify `when` condition - use `unqueryvet -verbose` to see evaluation
3. Check if file is ignored in `ignore` list

### Condition Syntax Errors

```yaml
# Wrong - unquoted special characters
when: !has_where

# Correct - quote strings with special characters
when: "!has_where"
```

### Regex Escaping

```yaml
# Wrong - unescaped dot
when: matches(file, "_test.go$")

# Correct - escaped dot
when: matches(file, "_test\\.go$")
```

---

## See Also

- [CLI Features Guide](CLI_FEATURES.md)
- [Example Configuration](../.unqueryvet.example.yaml)
- [Main README](../README.md)
