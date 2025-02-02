package main

import (
	"errors"
	"regexp"
	"strings"
)

// validateQuery は、与えられた SQL クエリが有効な SELECT 文であり、
// 禁止コマンドが含まれておらず、SELECT句で複数カラムが指定されていないかを検証します。
func validateQuery(query string) error {
	// 前後の空白を除去し、元のクエリ文字列も保持
	cleanQuery := strings.TrimSpace(query)
	// 小文字化した文字列は禁止語のチェックや FROM 句チェックに使用
	lowerQuery := strings.ToLower(cleanQuery)

	// SELECT 文であることのチェック
	if !strings.HasPrefix(lowerQuery, "select") {
		return errors.New("invalid query: only SELECT statements are allowed")
	}

	// FROM 句が存在するかチェック
	if !strings.Contains(lowerQuery, " from ") {
		return errors.New("invalid query: missing FROM clause")
	}

	// 禁止ワードチェック
	blacklist := []string{"insert", "update", "delete", "drop", "alter", "truncate", "create", "replace"}
	reBlack := regexp.MustCompile(`\b(` + strings.Join(blacklist, "|") + `)\b`)
	if reBlack.MatchString(lowerQuery) {
		return errors.New("invalid query: detected a forbidden SQL command")
	}

	// SELECT と FROM の間の部分（カラムリスト）を抽出
	reSelect := regexp.MustCompile(`(?i)^select\s+(.*?)\s+from\s+`)
	matches := reSelect.FindStringSubmatch(cleanQuery)
	if len(matches) < 2 {
		return errors.New("invalid query: unable to parse selected columns")
	}
	columns := matches[1]

	// トップレベル（括弧の外）でカンマがある場合、複数カラム指定と判断する
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
