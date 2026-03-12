package cache

import (
	"hash/fnv"
	"log/slog"

	pg_query "github.com/pganalyze/pg_query_go/v5"
)

// SemanticCacheKey generates a cache key from a normalized AST representation.
// Semantically equivalent queries (different whitespace, casing) produce the same key
// while preserving literal values to prevent cross-query cache collisions.
// Falls back to CacheKey on parse failure.
func SemanticCacheKey(query string) uint64 {
	// Parse → Deparse normalizes whitespace and casing while preserving literal values.
	// Unlike FingerprintToUInt64 which strips all constants, Deparse keeps them intact.
	tree, err := pg_query.Parse(query)
	if err != nil {
		slog.Debug("semantic cache key: parse failed, fallback", "error", err)
		return CacheKey(query)
	}
	return semanticCacheKeyFromTree(tree, query)
}

// SemanticCacheKeyWithTree generates a cache key using a pre-parsed AST tree,
// avoiding a redundant pg_query.Parse() call.
func SemanticCacheKeyWithTree(tree *pg_query.ParseResult, query string) uint64 {
	return semanticCacheKeyFromTree(tree, query)
}

// semanticCacheKeyFromTree generates a cache key from a pre-parsed tree.
func semanticCacheKeyFromTree(tree *pg_query.ParseResult, query string) uint64 {
	deparsed, err := pg_query.Deparse(tree)
	if err != nil {
		slog.Debug("semantic cache key: deparse failed, fallback", "error", err)
		return CacheKey(query)
	}
	h := fnv.New64a()
	h.Write([]byte(deparsed))
	return h.Sum64()
}

// NormalizeQuery returns a canonical string representation of the query
// with constants replaced by $N placeholders. Useful for logging and debugging.
func NormalizeQuery(query string) string {
	normalized, err := pg_query.Normalize(query)
	if err != nil {
		return query
	}
	return normalized
}

// SemanticCacheKeyWithParams generates a semantic cache key that also considers
// the actual parameter values. This allows caching different results for
// the same query structure with different parameters.
func SemanticCacheKeyWithParams(query string, params ...any) uint64 {
	normalized, err := pg_query.Normalize(query)
	if err != nil {
		return CacheKey(query, params...)
	}

	h := fnv.New64a()
	h.Write([]byte(normalized))
	for _, p := range params {
		if s, ok := p.(string); ok {
			h.Write([]byte(s))
		}
	}
	return h.Sum64()
}
