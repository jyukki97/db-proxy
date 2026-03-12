package proxy

import (
	"testing"
)

func TestSynthesizer_RegisterAndSynthesize(t *testing.T) {
	s := NewSynthesizer()
	s.RegisterStatement("s1", "SELECT * FROM users WHERE id = $1", []uint32{23}) // int4

	got, err := s.Synthesize("s1", [][]byte{[]byte("42")}, []int16{0})
	if err != nil {
		t.Fatalf("Synthesize error: %v", err)
	}
	want := "SELECT * FROM users WHERE id = 42"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSynthesizer_MultipleParams(t *testing.T) {
	s := NewSynthesizer()
	s.RegisterStatement("s2",
		"SELECT * FROM users WHERE name = $1 AND age > $2",
		[]uint32{25, 23}, // text, int4
	)

	got, err := s.Synthesize("s2",
		[][]byte{[]byte("Alice"), []byte("30")},
		[]int16{0, 0},
	)
	if err != nil {
		t.Fatalf("Synthesize error: %v", err)
	}
	want := "SELECT * FROM users WHERE name = 'Alice' AND age > 30"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSynthesizer_NullParam(t *testing.T) {
	s := NewSynthesizer()
	s.RegisterStatement("s3",
		"INSERT INTO users (name, email) VALUES ($1, $2)",
		[]uint32{25, 25},
	)

	got, err := s.Synthesize("s3",
		[][]byte{[]byte("Bob"), nil},
		[]int16{0, 0},
	)
	if err != nil {
		t.Fatalf("Synthesize error: %v", err)
	}
	want := "INSERT INTO users (name, email) VALUES ('Bob', NULL)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSynthesizer_UnnamedStatement(t *testing.T) {
	s := NewSynthesizer()
	s.RegisterStatement("", "SELECT $1::int", []uint32{23})

	got, err := s.Synthesize("", [][]byte{[]byte("99")}, []int16{0})
	if err != nil {
		t.Fatalf("Synthesize error: %v", err)
	}
	want := "SELECT 99::int"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSynthesizer_UnknownStatement(t *testing.T) {
	s := NewSynthesizer()
	_, err := s.Synthesize("nonexistent", nil, nil)
	if err == nil {
		t.Error("expected error for unknown statement")
	}
}

func TestSynthesizer_CloseStatement(t *testing.T) {
	s := NewSynthesizer()
	s.RegisterStatement("s1", "SELECT 1", nil)
	s.CloseStatement("s1")
	_, err := s.Synthesize("s1", nil, nil)
	if err == nil {
		t.Error("expected error after CloseStatement")
	}
}

func TestSynthesizer_PlaceholderInStringLiteral(t *testing.T) {
	s := NewSynthesizer()
	s.RegisterStatement("s4",
		"SELECT * FROM users WHERE name = '$1 is not a param' AND id = $1",
		[]uint32{23},
	)

	got, err := s.Synthesize("s4", [][]byte{[]byte("42")}, []int16{0})
	if err != nil {
		t.Fatalf("Synthesize error: %v", err)
	}
	// $1 inside quotes should NOT be replaced
	want := "SELECT * FROM users WHERE name = '$1 is not a param' AND id = 42"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSynthesizer_SQLInjectionViaParams(t *testing.T) {
	s := NewSynthesizer()
	s.RegisterStatement("inject",
		"SELECT * FROM users WHERE name = $1",
		[]uint32{25}, // text
	)

	// Attempt SQL injection via parameter value
	got, err := s.Synthesize("inject",
		[][]byte{[]byte("'; DROP TABLE users; --")},
		[]int16{0},
	)
	if err != nil {
		t.Fatalf("Synthesize error: %v", err)
	}
	// The injected value should be safely escaped
	want := "SELECT * FROM users WHERE name = '''; DROP TABLE users; --'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestSynthesizer_SingleFormatCode(t *testing.T) {
	s := NewSynthesizer()
	s.RegisterStatement("s5",
		"SELECT * FROM t WHERE a = $1 AND b = $2",
		[]uint32{23, 25},
	)

	// Single format code applies to all params
	got, err := s.Synthesize("s5",
		[][]byte{[]byte("1"), []byte("hello")},
		[]int16{0}, // single code = all text
	)
	if err != nil {
		t.Fatalf("Synthesize error: %v", err)
	}
	want := "SELECT * FROM t WHERE a = 1 AND b = 'hello'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestReplacePlaceholders(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		literals []string
		want     string
	}{
		{
			"simple",
			"SELECT $1",
			[]string{"42"},
			"SELECT 42",
		},
		{
			"multiple",
			"SELECT $1, $2, $3",
			[]string{"1", "'hello'", "NULL"},
			"SELECT 1, 'hello', NULL",
		},
		{
			"repeated placeholder",
			"SELECT $1, $1",
			[]string{"42"},
			"SELECT 42, 42",
		},
		{
			"no placeholders",
			"SELECT 1",
			nil,
			"SELECT 1",
		},
		{
			"placeholder in single quotes",
			"SELECT '$1' || $1",
			[]string{"42"},
			"SELECT '$1' || 42",
		},
		{
			"escaped quote in string",
			"SELECT '''$1''' || $1",
			[]string{"42"},
			"SELECT '''$1''' || 42",
		},
		{
			"dollar quoting",
			"SELECT $$ $1 $$ || $1",
			[]string{"42"},
			"SELECT $$ $1 $$ || 42",
		},
		{
			"double digit placeholder",
			"SELECT $10",
			[]string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10"},
			"SELECT 10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := replacePlaceholders(tt.query, tt.literals)
			if err != nil {
				t.Fatalf("replacePlaceholders error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
