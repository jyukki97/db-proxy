package router

import (
	"fmt"

	pg_query "github.com/pganalyze/pg_query_go/v5"
)

// ParsedQuery holds a pre-parsed SQL AST to avoid redundant pg_query.Parse() calls
// across firewall, classify, cache key, and table extraction in a single request.
type ParsedQuery struct {
	SQL  string
	Tree *pg_query.ParseResult
}

// NewParsedQuery parses the SQL once and returns a ParsedQuery.
// Returns an error if parsing fails.
func NewParsedQuery(sql string) (*ParsedQuery, error) {
	tree, err := pg_query.Parse(sql)
	if err != nil {
		return nil, fmt.Errorf("parse SQL: %w", err)
	}
	return &ParsedQuery{SQL: sql, Tree: tree}, nil
}
