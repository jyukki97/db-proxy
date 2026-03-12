package router

import (
	"testing"

	pg_query "github.com/pganalyze/pg_query_go/v5"
)

func TestNewParsedQuery(t *testing.T) {
	pq, err := NewParsedQuery("SELECT * FROM users WHERE id = 1")
	if err != nil {
		t.Fatalf("NewParsedQuery failed: %v", err)
	}
	if pq.SQL != "SELECT * FROM users WHERE id = 1" {
		t.Errorf("SQL = %q, want %q", pq.SQL, "SELECT * FROM users WHERE id = 1")
	}
	if pq.Tree == nil {
		t.Fatal("Tree should not be nil")
	}
	if len(pq.Tree.GetStmts()) == 0 {
		t.Fatal("expected at least one statement")
	}
}

func TestNewParsedQuery_InvalidSQL(t *testing.T) {
	_, err := NewParsedQuery("SELECTT INVALID SQL;;;")
	if err == nil {
		t.Fatal("expected error for invalid SQL")
	}
}

func TestClassifyASTWithTree(t *testing.T) {
	tests := []struct {
		query string
		want  QueryType
	}{
		{"SELECT * FROM users", QueryRead},
		{"INSERT INTO users VALUES (1)", QueryWrite},
		{"UPDATE users SET name = 'a'", QueryWrite},
		{"DELETE FROM users WHERE id = 1", QueryWrite},
		{"CREATE TABLE foo (id int)", QueryWrite},
		{"TRUNCATE users", QueryWrite},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			pq, err := NewParsedQuery(tt.query)
			if err != nil {
				t.Fatalf("NewParsedQuery failed: %v", err)
			}
			got := ClassifyASTWithTree(tt.query, pq)
			if got != tt.want {
				t.Errorf("ClassifyASTWithTree(%q) = %d, want %d", tt.query, got, tt.want)
			}
		})
	}
}

func TestClassifyASTWithTree_HintComments(t *testing.T) {
	tests := []struct {
		query string
		want  QueryType
	}{
		{"/* route:writer */ SELECT * FROM users", QueryWrite},
		{"/* route:reader */ INSERT INTO users VALUES (1)", QueryRead},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			pq, err := NewParsedQuery(tt.query)
			if err != nil {
				t.Fatalf("NewParsedQuery failed: %v", err)
			}
			got := ClassifyASTWithTree(tt.query, pq)
			if got != tt.want {
				t.Errorf("ClassifyASTWithTree(%q) = %d, want %d", tt.query, got, tt.want)
			}
		})
	}
}

func TestClassifyASTWithTree_MatchesClassifyAST(t *testing.T) {
	queries := []string{
		"SELECT * FROM users",
		"INSERT INTO users VALUES (1)",
		"UPDATE users SET name = 'a'",
		"DELETE FROM users WHERE id = 1",
		"CREATE TABLE foo (id int)",
		"TRUNCATE users",
		"/* route:writer */ SELECT * FROM users",
		"/* route:reader */ INSERT INTO users VALUES (1)",
		"WITH x AS (UPDATE users SET score=0 RETURNING id) SELECT * FROM x",
		"WITH x AS (SELECT * FROM users) SELECT * FROM x",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			original := ClassifyAST(query)
			pq, err := NewParsedQuery(query)
			if err != nil {
				t.Fatalf("NewParsedQuery failed: %v", err)
			}
			withTree := ClassifyASTWithTree(query, pq)
			if original != withTree {
				t.Errorf("ClassifyAST=%d, ClassifyASTWithTree=%d for %q", original, withTree, query)
			}
		})
	}
}

func TestExtractTablesASTWithTree(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{"INSERT INTO users VALUES (1)", "users"},
		{"UPDATE orders SET status = 'done'", "orders"},
		{"DELETE FROM products WHERE id = 1", "products"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			pq, err := NewParsedQuery(tt.query)
			if err != nil {
				t.Fatalf("NewParsedQuery failed: %v", err)
			}
			tables := ExtractTablesASTWithTree(pq)
			if len(tables) == 0 {
				t.Fatalf("ExtractTablesASTWithTree(%q) returned empty", tt.query)
			}
			if tables[0] != tt.want {
				t.Errorf("ExtractTablesASTWithTree(%q) = %q, want %q", tt.query, tables[0], tt.want)
			}
		})
	}
}

func TestExtractTablesASTWithTree_MatchesExtractTablesAST(t *testing.T) {
	queries := []string{
		"INSERT INTO users VALUES (1)",
		"UPDATE orders SET status = 'done'",
		"DELETE FROM products WHERE id = 1",
		"TRUNCATE TABLE logs",
		"WITH x AS (UPDATE users SET score=0) UPDATE ranking SET total=0",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			original := ExtractTablesAST(query)
			pq, err := NewParsedQuery(query)
			if err != nil {
				t.Fatalf("NewParsedQuery failed: %v", err)
			}
			withTree := ExtractTablesASTWithTree(pq)
			if len(original) != len(withTree) {
				t.Errorf("table count mismatch: %d vs %d", len(original), len(withTree))
				return
			}
			for i := range original {
				if original[i] != withTree[i] {
					t.Errorf("table[%d]: %q vs %q", i, original[i], withTree[i])
				}
			}
		})
	}
}

func TestCheckFirewallWithTree(t *testing.T) {
	cfg := FirewallConfig{
		Enabled:                 true,
		BlockDeleteWithoutWhere: true,
		BlockUpdateWithoutWhere: true,
		BlockDropTable:          true,
		BlockTruncate:           true,
	}

	tests := []struct {
		query   string
		blocked bool
		rule    FirewallRule
	}{
		{"DELETE FROM users", true, RuleDeleteNoWhere},
		{"DELETE FROM users WHERE id = 1", false, ""},
		{"UPDATE users SET x=1", true, RuleUpdateNoWhere},
		{"UPDATE users SET x=1 WHERE id=1", false, ""},
		{"DROP TABLE users", true, RuleDropTable},
		{"TRUNCATE users", true, RuleTruncate},
		{"SELECT * FROM users", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			pq, err := NewParsedQuery(tt.query)
			if err != nil {
				t.Fatalf("NewParsedQuery failed: %v", err)
			}

			result := CheckFirewallWithTree(pq, cfg)
			if result.Blocked != tt.blocked {
				t.Errorf("blocked = %v, want %v", result.Blocked, tt.blocked)
			}
			if tt.blocked && result.Rule != tt.rule {
				t.Errorf("rule = %q, want %q", result.Rule, tt.rule)
			}
		})
	}
}

func TestCheckFirewallWithTree_MatchesCheckFirewall(t *testing.T) {
	cfg := FirewallConfig{
		Enabled:                 true,
		BlockDeleteWithoutWhere: true,
		BlockUpdateWithoutWhere: true,
		BlockDropTable:          true,
		BlockTruncate:           true,
	}

	queries := []string{
		"DELETE FROM users",
		"DELETE FROM users WHERE id = 1",
		"UPDATE users SET x=1",
		"DROP TABLE users",
		"TRUNCATE users",
		"SELECT * FROM users",
		"INSERT INTO users VALUES (1)",
	}

	for _, query := range queries {
		t.Run(query, func(t *testing.T) {
			original := CheckFirewall(query, cfg)
			pq, err := NewParsedQuery(query)
			if err != nil {
				t.Fatalf("NewParsedQuery failed: %v", err)
			}
			withTree := CheckFirewallWithTree(pq, cfg)
			if original.Blocked != withTree.Blocked {
				t.Errorf("blocked mismatch: %v vs %v", original.Blocked, withTree.Blocked)
			}
			if original.Rule != withTree.Rule {
				t.Errorf("rule mismatch: %q vs %q", original.Rule, withTree.Rule)
			}
		})
	}
}

// BenchmarkQueryPipeline_WithoutParsedQuery benchmarks the old approach where
// each function parses the SQL independently (5 parse calls per query).
func BenchmarkQueryPipeline_WithoutParsedQuery(b *testing.B) {
	query := "DELETE FROM users WHERE id = 1"
	cfg := FirewallConfig{
		Enabled:                 true,
		BlockDeleteWithoutWhere: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CheckFirewall(query, cfg)
		_ = ClassifyAST(query)
		_ = ExtractTablesAST(query)
		// SemanticCacheKey also parses independently
		_, _ = pg_query.Parse(query)
	}
}

// BenchmarkQueryPipeline_WithParsedQuery benchmarks the new approach where
// the SQL is parsed once and the tree is reused across all functions.
func BenchmarkQueryPipeline_WithParsedQuery(b *testing.B) {
	query := "DELETE FROM users WHERE id = 1"
	cfg := FirewallConfig{
		Enabled:                 true,
		BlockDeleteWithoutWhere: true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pq, err := NewParsedQuery(query)
		if err != nil {
			b.Fatalf("parse failed: %v", err)
		}
		_ = CheckFirewallWithTree(pq, cfg)
		_ = ClassifyASTWithTree(query, pq)
		_ = ExtractTablesASTWithTree(pq)
		// Tree is already available for cache key generation
		_ = pq.Tree
	}
}

// BenchmarkParseSQLAlone measures the raw cost of a single pg_query.Parse call
// to show how much each redundant call costs.
func BenchmarkParseSQLAlone(b *testing.B) {
	query := "SELECT * FROM users WHERE id = 1 AND name = 'alice' ORDER BY created_at"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = pg_query.Parse(query)
	}
}
