package main

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// validateDBURL checks if the provided database connection URL is in a valid format.
// It verifies:
// - If the URL can be parsed successfully
// - If the scheme is "postgres" or "postgresql"
// - If the host part is not empty
// - If the path part (database name) is specified (not just "/" or empty string)
// If these conditions are not met, it returns an error.
func validateDBURL(dbURL string) error {
	u, err := url.Parse(dbURL)
	if err != nil {
		return fmt.Errorf("invalid database URL: %w", err)
	}

	// Check scheme - case insensitive comparison
	scheme := strings.ToLower(u.Scheme)
	if scheme != "postgres" && scheme != "postgresql" {
		return errors.New("invalid database URL: scheme must be 'postgres' or 'postgresql'")
	}

	// Check host
	if u.Host == "" {
		return errors.New("invalid database URL: host is empty")
	}

	// Check database name (path part)
	// u.Path has a leading "/", so if it's just "/" or empty, it's invalid
	if u.Path == "" || u.Path == "/" {
		return errors.New("invalid database URL: database name is missing")
	}

	return nil
}

// validateQuery verifies that the given SQL query is a valid SELECT statement,
// doesn't contain forbidden commands, and doesn't specify multiple columns in the SELECT clause.
func validateQuery(query string) error {
	// Remove leading and trailing whitespace, and preserve the original query string
	cleanQuery := strings.TrimSpace(query)
	// Lowercase string is used for checking forbidden words and FROM clause
	lowerQuery := strings.ToLower(cleanQuery)

	// Check if it's a SELECT statement
	if !strings.HasPrefix(lowerQuery, "select") {
		return errors.New("invalid query: only SELECT statements are allowed")
	}

	// Check if FROM clause exists
	if !strings.Contains(lowerQuery, " from ") {
		return errors.New("invalid query: missing FROM clause")
	}

	// Check for forbidden words
	blacklist := []string{"insert", "update", "delete", "drop", "alter", "truncate", "create", "replace"}
	reBlack := regexp.MustCompile(`\b(` + strings.Join(blacklist, "|") + `)\b`)
	if reBlack.MatchString(lowerQuery) {
		return errors.New("invalid query: detected a forbidden SQL command")
	}

	// Extract the column list (between SELECT and FROM)
	reSelect := regexp.MustCompile(`(?i)^select\s+(.*?)\s+from\s+`)
	matches := reSelect.FindStringSubmatch(cleanQuery)
	if len(matches) < 2 {
		return errors.New("invalid query: unable to parse selected columns")
	}
	columns := matches[1]

	// If there's a comma at the top level (outside of parentheses), consider it as multiple column specification
	depth := 0
	for _, r := range columns {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				return errors.New("invalid query: multiple columns are not allowed")
			}
		}
	}

	return nil
}
