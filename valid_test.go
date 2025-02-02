package main

import (
	"strings"
	"testing"
)

func TestValidateQuery(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
		errMsg  string // エラーメッセージに含まれるべき文字列（任意）
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
