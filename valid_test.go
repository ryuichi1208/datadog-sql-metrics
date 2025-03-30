package main

import (
	"strings"
	"testing"
)

func TestValidateDBURL(t *testing.T) {
	tests := []struct {
		name    string
		dbURL   string
		wantErr bool
		errMsg  string // Expected keyword in error message (optional)
	}{
		{
			name:    "Valid URL with postgres scheme",
			dbURL:   "postgres://user:pass@localhost:5432/dbname?sslmode=disable",
			wantErr: false,
		},
		{
			name:    "Valid URL with postgresql scheme",
			dbURL:   "postgresql://user:pass@localhost:5432/dbname",
			wantErr: false,
		},
		{
			name:    "Invalid scheme",
			dbURL:   "mysql://user:pass@localhost:3306/dbname",
			wantErr: true,
			errMsg:  "scheme must be 'postgres' or 'postgresql'",
		},
		{
			name:    "Missing host",
			dbURL:   "postgres://user:pass@/dbname",
			wantErr: true,
			errMsg:  "host is empty",
		},
		{
			name:    "Missing database name",
			dbURL:   "postgres://user:pass@localhost:5432",
			wantErr: true,
			errMsg:  "database name is missing",
		},
		{
			name:    "Malformed URL",
			dbURL:   "postgres:invalid-url-format",
			wantErr: true,
			errMsg:  "invalid database URL",
		},
		{
			name:    "URL with slash instead of database name",
			dbURL:   "postgres://user:pass@localhost:5432/",
			wantErr: true,
			errMsg:  "database name is missing",
		},
		{
			name:    "Valid URL with additional parameters",
			dbURL:   "postgres://user:pass@localhost:5432/dbname?connect_timeout=10&application_name=myapp",
			wantErr: false,
		},
		{
			name:    "URL without credentials",
			dbURL:   "postgres://localhost:5432/dbname",
			wantErr: false,
		},
		{
			name:    "URL with IPv6 address",
			dbURL:   "postgres://user:pass@[::1]:5432/dbname",
			wantErr: false,
		},
		{
			name:    "URL with mixed case scheme",
			dbURL:   "PostgreSQL://user:pass@localhost:5432/dbname",
			wantErr: false,
		},
		{
			name:    "URL without port",
			dbURL:   "postgres://user:pass@localhost/dbname",
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc // range variable capture
		t.Run(tc.name, func(t *testing.T) {
			err := validateDBURL(tc.dbURL)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Expected error but got nil for URL: %q", tc.dbURL)
				}
				if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tc.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, but got %v for URL: %q", err, tc.dbURL)
				}
			}
		})
	}
}

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
		errMsg  string // Expected string in error message (optional)
	}{
		{
			name:    "Valid single column query",
			query:   "SELECT age FROM users LIMIT 1;",
			wantErr: false,
		},
		{
			name:    "Valid single column with extra whitespace",
			query:   "   SELECT name FROM users   ",
			wantErr: false,
		},
		{
			name:    "Missing FROM clause",
			query:   "SELECT age",
			wantErr: true,
			errMsg:  "missing FROM clause",
		},
		{
			name:    "Not a SELECT statement",
			query:   "UPDATE users SET age = 30",
			wantErr: true,
			errMsg:  "only SELECT statements are allowed",
		},
		{
			name:    "Contains forbidden command (DROP)",
			query:   "SELECT age FROM users; DROP TABLE users;",
			wantErr: true,
			errMsg:  "detected a forbidden SQL command",
		},
		{
			name:    "Multiple columns specified",
			query:   "SELECT age, name FROM users",
			wantErr: true,
			errMsg:  "multiple columns are not allowed",
		},
		{
			name:    "Comma inside function call is allowed",
			query:   "SELECT func(age, name) FROM users",
			wantErr: false,
		},
		{
			name:    "Subquery without top-level comma is allowed",
			query:   "SELECT (SELECT count(*) FROM orders) FROM users",
			wantErr: false,
		},
		{
			name:    "Case insensitive SELECT keyword",
			query:   "select age from users",
			wantErr: false,
		},
		{
			name:    "Complex function with multiple arguments",
			query:   "SELECT COALESCE(age, 0, default_age) FROM users",
			wantErr: false,
		},
		{
			name:    "Query with WHERE clause",
			query:   "SELECT age FROM users WHERE active = true",
			wantErr: false,
		},
		{
			name:    "Query with GROUP BY",
			query:   "SELECT MAX(age) FROM users GROUP BY department_id",
			wantErr: false,
		},
		{
			name:    "Query with multiple forbidden words",
			query:   "SELECT age FROM users; CREATE TABLE new_users; DROP TABLE old_users;",
			wantErr: true,
			errMsg:  "detected a forbidden SQL command",
		},
		{
			name:    "Query with nested subqueries",
			query:   "SELECT (SELECT MAX(age) FROM (SELECT age FROM older_users) AS t) FROM users",
			wantErr: false,
		},
		{
			name:    "Query with alias in FROM clause",
			query:   "SELECT age FROM users AS u",
			wantErr: false,
		},
		{
			name:    "Query with CASE statement",
			query:   "SELECT CASE WHEN age > 18 THEN 'adult' ELSE 'minor' END FROM users",
			wantErr: false,
		},
		{
			name:    "Query with JOIN clause",
			query:   "SELECT u.age FROM users u JOIN orders o ON u.id = o.user_id",
			wantErr: false,
		},
		{
			name:    "Query with multiple JOINs",
			query:   "SELECT u.age FROM users u JOIN orders o ON u.id = o.user_id JOIN products p ON o.product_id = p.id",
			wantErr: false,
		},
		{
			name:    "Query with HAVING clause",
			query:   "SELECT MAX(age) FROM users GROUP BY department_id HAVING MAX(age) > 40",
			wantErr: false,
		},
		{
			name:    "Query with ORDER BY clause",
			query:   "SELECT age FROM users ORDER BY age DESC",
			wantErr: false,
		},
		{
			name:    "Query with LIMIT and OFFSET",
			query:   "SELECT age FROM users LIMIT 10 OFFSET 20",
			wantErr: false,
		},
		{
			name:    "Query with inline comment",
			query:   "SELECT age FROM users -- This is a comment",
			wantErr: false,
		},
		{
			name:    "Query with block comment",
			query:   "SELECT age FROM users /* This is a block comment */",
			wantErr: false,
		},
		{
			name:    "Query with quoted identifiers",
			query:   "SELECT \"user\".\"age\" FROM \"users\" AS \"user\"",
			wantErr: false,
		},
		{
			name:    "Function with blacklisted word as substring",
			query:   "SELECT COUNT(*) FROM users",
			wantErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			err := validateQuery(tc.query)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Expected error but got nil for query: %q", tc.query)
				}
				if tc.errMsg != "" && !strings.Contains(err.Error(), tc.errMsg) {
					t.Errorf("Expected error message to contain %q, got %q", tc.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Expected no error, but got %v for query: %q", err, tc.query)
				}
			}
		})
	}
}
