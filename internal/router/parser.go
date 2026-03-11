package router

import (
	"regexp"
	"strings"
)

type QueryType int

const (
	QueryRead  QueryType = iota
	QueryWrite
)

var writeKeywords = map[string]bool{
	"INSERT":   true,
	"UPDATE":   true,
	"DELETE":   true,
	"CREATE":   true,
	"ALTER":    true,
	"DROP":     true,
	"TRUNCATE": true,
	"GRANT":    true,
	"REVOKE":   true,
}

var hintRegex = regexp.MustCompile(`/\*\s*route:(writer|reader)\s*\*/`)

// Classify determines whether a query is a read or write operation.
func Classify(query string) QueryType {
	// 1. Check for routing hint
	if hint := extractHint(query); hint != "" {
		if hint == "writer" {
			return QueryWrite
		}
		return QueryRead
	}

	// 2. Classify by first keyword
	keyword := firstKeyword(query)
	if writeKeywords[keyword] {
		return QueryWrite
	}
	return QueryRead
}

func extractHint(query string) string {
	matches := hintRegex.FindStringSubmatch(query)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

func firstKeyword(query string) string {
	q := stripComments(query)
	q = strings.TrimSpace(q)
	fields := strings.Fields(q)
	if len(fields) == 0 {
		return ""
	}
	return strings.ToUpper(fields[0])
}

func stripComments(query string) string {
	// Remove /* ... */ comments
	re := regexp.MustCompile(`/\*.*?\*/`)
	q := re.ReplaceAllString(query, "")

	// Remove -- line comments
	re2 := regexp.MustCompile(`--[^\n]*`)
	q = re2.ReplaceAllString(q, "")

	return q
}

// ExtractTables extracts table names from write queries.
func ExtractTables(query string) []string {
	q := strings.TrimSpace(query)
	upper := strings.ToUpper(q)

	var tables []string

	switch {
	case strings.HasPrefix(upper, "INSERT INTO"):
		tables = append(tables, extractAfter(q, upper, "INSERT INTO"))
	case strings.HasPrefix(upper, "UPDATE"):
		tables = append(tables, extractAfter(q, upper, "UPDATE"))
	case strings.HasPrefix(upper, "DELETE FROM"):
		tables = append(tables, extractAfter(q, upper, "DELETE FROM"))
	case strings.HasPrefix(upper, "TRUNCATE"):
		tables = append(tables, extractAfter(q, upper, "TRUNCATE"))
	}

	return tables
}

func extractAfter(query, upper, keyword string) string {
	rest := strings.TrimSpace(query[len(keyword):])
	// Handle optional keywords like "TABLE"
	upperRest := strings.ToUpper(rest)
	if strings.HasPrefix(upperRest, "TABLE ") {
		rest = strings.TrimSpace(rest[6:])
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return ""
	}
	// Remove schema prefix and clean up
	name := strings.TrimRight(fields[0], "(;,")
	parts := strings.Split(name, ".")
	return strings.ToLower(parts[len(parts)-1])
}
