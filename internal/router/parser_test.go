package router

import "testing"

func TestClassify(t *testing.T) {
	tests := []struct {
		query string
		want  QueryType
	}{
		{"SELECT * FROM users", QueryRead},
		{"select * from users", QueryRead},
		{"  SELECT 1", QueryRead},
		{"SHOW tables", QueryRead},
		{"EXPLAIN SELECT 1", QueryRead},
		{"INSERT INTO users VALUES (1)", QueryWrite},
		{"insert into users values (1)", QueryWrite},
		{"UPDATE users SET name = 'a'", QueryWrite},
		{"DELETE FROM users WHERE id = 1", QueryWrite},
		{"CREATE TABLE foo (id int)", QueryWrite},
		{"ALTER TABLE foo ADD col int", QueryWrite},
		{"DROP TABLE foo", QueryWrite},
		{"TRUNCATE users", QueryWrite},
		// Hint comments
		{"/* route:writer */ SELECT * FROM users", QueryWrite},
		{"/* route:reader */ INSERT INTO users VALUES (1)", QueryRead},
		{"/*route:writer*/ SELECT 1", QueryWrite},
		{"/* route:reader */ SELECT 1", QueryRead},
		// Regular comments should be stripped
		{"-- comment\nSELECT 1", QueryRead},
		{"/* normal comment */ SELECT 1", QueryRead},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got := Classify(tt.query)
			if got != tt.want {
				t.Errorf("Classify(%q) = %d, want %d", tt.query, got, tt.want)
			}
		})
	}
}

func TestExtractTables(t *testing.T) {
	tests := []struct {
		query string
		want  string
	}{
		{"INSERT INTO users VALUES (1)", "users"},
		{"insert into users values (1)", "users"},
		{"UPDATE orders SET status = 'done'", "orders"},
		{"DELETE FROM products WHERE id = 1", "products"},
		{"TRUNCATE TABLE logs", "logs"},
		{"INSERT INTO public.users VALUES (1)", "users"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			tables := ExtractTables(tt.query)
			if len(tables) == 0 {
				t.Fatalf("ExtractTables(%q) returned empty", tt.query)
			}
			if tables[0] != tt.want {
				t.Errorf("ExtractTables(%q) = %q, want %q", tt.query, tables[0], tt.want)
			}
		})
	}
}
