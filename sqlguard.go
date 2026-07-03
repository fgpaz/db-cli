package main

import (
	"fmt"
	"strings"
	"unicode"
)

type SQLMode string

const (
	SQLModeQuery   SQLMode = "query"
	SQLModeWrite   SQLMode = "write"
	SQLModeMigrate SQLMode = "migrate"
)

func ValidateSQL(sql string, mode SQLMode) error {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return fmt.Errorf("sql is empty")
	}
	if mode == SQLModeMigrate {
		return nil
	}
	if hasMultipleStatements(sql) {
		return fmt.Errorf("multi-statement SQL is not allowed in v1")
	}
	if hasForbiddenKeyword(sql) {
		return fmt.Errorf("SQL contains a forbidden keyword for v1")
	}
	if mode == SQLModeQuery && hasSelectInto(sql) {
		return fmt.Errorf("query mode does not allow SELECT INTO")
	}
	if mode == SQLModeWrite && !isSimpleWrite(sql) {
		return fmt.Errorf("write only allows simple INSERT, UPDATE, or DELETE statements in v1")
	}
	return nil
}

func hasMultipleStatements(sql string) bool {
	inSingle := false
	inDouble := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(sql); i++ {
		ch := sql[i]
		next := byte(0)
		if i+1 < len(sql) {
			next = sql[i+1]
		}

		if inLineComment {
			if ch == '\n' || ch == '\r' {
				inLineComment = false
			}
			continue
		}
		if inBlockComment {
			if ch == '*' && next == '/' {
				inBlockComment = false
				i++
			}
			continue
		}
		if !inSingle && !inDouble {
			if ch == '-' && next == '-' {
				inLineComment = true
				i++
				continue
			}
			if ch == '/' && next == '*' {
				inBlockComment = true
				i++
				continue
			}
		}
		if ch == '\'' && !inDouble {
			if inSingle && next == '\'' {
				i++
				continue
			}
			inSingle = !inSingle
			continue
		}
		if ch == '"' && !inSingle {
			inDouble = !inDouble
			continue
		}
		if ch == ';' && !inSingle && !inDouble {
			if strings.TrimSpace(sql[i+1:]) != "" {
				return true
			}
		}
	}
	return false
}

func hasForbiddenKeyword(sql string) bool {
	upper := " " + strings.ToUpper(sql) + " "
	for _, bad := range []string{
		" COPY ", " ALTER ", " CREATE ", " DROP ", " TRUNCATE ",
		" GRANT ", " REVOKE ", " VACUUM ", " LISTEN ", " NOTIFY ",
		" CLUSTER ", " REFRESH ", " MERGE ", " EXECUTE ", " CALL ",
	} {
		if strings.Contains(upper, bad) {
			return true
		}
	}
	return false
}

func hasSelectInto(sql string) bool {
	upper := " " + strings.ToUpper(sql) + " "
	return strings.Contains(upper, " SELECT ") && strings.Contains(upper, " INTO ")
}

func isSimpleWrite(sql string) bool {
	return switchFirstKeyword(sql, map[string]bool{
		"INSERT": true,
		"UPDATE": true,
		"DELETE": true,
	})
}

func switchFirstKeyword(sql string, allowed map[string]bool) bool {
	kw := firstKeyword(sql)
	return allowed[kw]
}

func firstKeyword(sql string) string {
	sql = skipLeadingSQLNoise(sql)
	i := 0
	for i < len(sql) && unicode.IsLetter(rune(sql[i])) {
		i++
	}
	return strings.ToUpper(sql[:i])
}

func skipLeadingSQLNoise(sql string) string {
	for {
		sql = strings.TrimLeftFunc(sql, unicode.IsSpace)
		if strings.HasPrefix(sql, "--") {
			if idx := strings.IndexByte(sql, '\n'); idx >= 0 {
				sql = sql[idx+1:]
				continue
			}
			return ""
		}
		if strings.HasPrefix(sql, "/*") {
			if idx := strings.Index(sql, "*/"); idx >= 0 {
				sql = sql[idx+2:]
				continue
			}
			return ""
		}
		return sql
	}
}
