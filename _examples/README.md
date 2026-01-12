# Unqueryvet Examples

This directory contains configuration examples and usage demonstrations for unqueryvet.

## Configuration Files

### golangci.yml

Standard golangci-lint configuration with unqueryvet enabled.

```bash
# Copy to your project
cp golangci.yml ~/.golangci.yml
golangci-lint run ./...
```

### golangci-with-unqueryvet.yml

Extended golangci-lint configuration showing all unqueryvet options.

### strict-config.yml

**Use case:** New projects or strict SQL policies

Features:
- Catches almost all SELECT * usage
- Only allows essential system queries (information_schema)
- COUNT(*) is always allowed
- Recommended for greenfield projects

```bash
# Run with strict config
golangci-lint run -c strict-config.yml ./...
```

### permissive-config.yml

**Use case:** Legacy projects or gradual adoption

Features:
- Generous allowed patterns
- Permits SELECT * for:
  - System/catalog tables (pg_catalog, information_schema, mysql.*, sys.*)
  - Temporary tables (temp_*, tmp_*, #*)
  - Debug/test tables
  - Backup/archive tables
  - Aggregate functions (COUNT, MAX, MIN, SUM, AVG)
  - Queries with LIMIT 1
  - Debug comments

```bash
# Run with permissive config
golangci-lint run -c permissive-config.yml ./...
```

## Code Examples

See `testdata/example.go` for code examples demonstrating:

- Patterns that unqueryvet will warn about
- Recommended patterns that pass validation
- How to suppress warnings with `//nolint:unqueryvet`
- SQL builder usage patterns for all 12 supported builders

## Running the Examples

### Analyze Example File

```bash
# From project root
go run ./cmd/unqueryvet ./_examples/testdata/example.go

# With verbose output
go run ./cmd/unqueryvet -verbose ./_examples/testdata/example.go

# With N+1 and SQL injection detection
go run ./cmd/unqueryvet -n1 -sqli ./_examples/testdata/example.go
```

### Using Configuration Files

```bash
# Copy configuration to project root
cp _examples/strict-config.yml .golangci.yml

# Run golangci-lint
golangci-lint run ./...
```

## Choosing a Configuration

| Scenario | Recommended Config |
|----------|-------------------|
| New project | `strict-config.yml` |
| Existing project (first adoption) | `permissive-config.yml` |
| Gradual migration | Start permissive, tighten over time |
| CI/CD pipeline | `strict-config.yml` with `fail-on-issues: false` initially |

## Creating Custom Configuration

Start with the example that best matches your needs and customize:

```yaml
# .unqueryvet.yaml in your project root
severity: warning

# Core checks
check-sql-builders: true
check-aliased-wildcard: true
check-string-concat: true
check-format-strings: true
check-subqueries: true

# Advanced checks
check-n1-queries: true
check-sql-injection: true

# SQL builders (enable only what you use)
sql-builders:
  gorm: true
  sqlx: true
  # ... others disabled

# Project-specific patterns
ignored-files:
  - "*_test.go"
  - "testdata/**"
  - "vendor/**"
  - "migrations/**"

# Allow SELECT * for specific tables
allowed-patterns:
  - "SELECT \\* FROM audit_log"
  - "SELECT \\* FROM temp_.*"
```

## See Also

- [CLI Features Guide](../docs/CLI_FEATURES.md)
- [Custom Rules DSL](../docs/DSL.md)
- [Full Configuration Reference](../.unqueryvet.example.yaml)
