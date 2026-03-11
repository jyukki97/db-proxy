package router

import (
	"reflect"
	"testing"
)

func TestStripStringLiterals(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no quotes", "SELECT 1", "SELECT 1"},
		{"single quotes", "SELECT 'hello world'", "SELECT ''"},
		{"double quotes", `SELECT "column name"`, `SELECT ""`},
		{"escaped single quote", "SELECT 'it''s fine'", "SELECT ''''"},
		{"mixed quotes", `SELECT 'a' FROM "b"`, `SELECT '' FROM ""`},
		{"keyword inside single", "WHERE x = 'INSERT INTO foo'", "WHERE x = ''"},
		{"hint inside single", "WHERE x = '/* route:writer */'", "WHERE x = ''"},
		{"no content change outside", "INSERT INTO users VALUES (1)", "INSERT INTO users VALUES (1)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripStringLiterals(tt.input)
			if got != tt.want {
				t.Errorf("stripStringLiterals(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEdgeCases_Classify(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  QueryType
	}{
		{
			name:  "quoted write keyword in SELECT",
			query: "SELECT * FROM users WHERE name = 'UPDATE admin'",
			want:  QueryRead,
		},
		{
			name:  "quoted write keyword in CTE SELECT",
			query: "WITH CTE AS (SELECT 1) SELECT * FROM foo WHERE x = 'UPDATE bar'",
			want:  QueryRead,
		},
		{
			name:  "hint injection via string literal",
			query: "SELECT * FROM users WHERE note = '/* route:writer */ trick'",
			want:  QueryRead,
		},
		{
			name:  "real hint outside string literal",
			query: "/* route:writer */ SELECT * FROM users",
			want:  QueryWrite,
		},
		{
			name:  "real hint mid-query",
			query: "SELECT * FROM /* route:writer */ users",
			want:  QueryWrite,
		},
		{
			name:  "line comment containing write keyword",
			query: "SELECT 1; -- UPDATE users",
			want:  QueryRead,
		},
		{
			name:  "INSERT keyword inside single-quoted value",
			query: "SELECT * FROM logs WHERE action = 'INSERT INTO admin_table'",
			want:  QueryRead,
		},
		{
			name:  "DELETE keyword inside CTE string",
			query: "WITH x AS (SELECT * FROM a WHERE b = 'DELETE FROM oops') SELECT 1",
			want:  QueryRead,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.query); got != tt.want {
				t.Errorf("Classify(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestEdgeCases_ExtractTables(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  []string
	}{
		{
			name:  "quoted write keyword in UPDATE",
			query: "UPDATE users SET note = 'DELETE FROM admin';",
			want:  []string{"users"},
		},
		{
			name:  "CTE WITH quoted write keyword",
			query: "WITH x AS (SELECT * FROM a WHERE b = 'INSERT INTO oops') SELECT 1;",
			want:  nil,
		},
		{
			name:  "line comment with write keyword",
			query: "UPDATE users -- DELETE FROM posts\nSET name = 'a'",
			want:  []string{"users"},
		},
		{
			name:  "INSERT keyword inside value does not extract",
			query: "SELECT * FROM logs WHERE action = 'INSERT INTO admin_table'",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractTables(tt.query)
			if !reflect.DeepEqual(got, tt.want) {
				if len(got) == 0 && len(tt.want) == 0 {
					return
				}
				t.Errorf("ExtractTables(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}
