package protocol

import (
	"testing"
)

func TestParamToLiteral_NULL(t *testing.T) {
	got, err := ParamToLiteral(nil, OIDText, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "NULL" {
		t.Errorf("got %q, want NULL", got)
	}
}

func TestParamToLiteral_Boolean(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"t", "TRUE"},
		{"true", "TRUE"},
		{"1", "TRUE"},
		{"f", "FALSE"},
		{"false", "FALSE"},
		{"0", "FALSE"},
	}
	for _, tt := range tests {
		got, err := ParamToLiteral([]byte(tt.input), OIDBoolean, 0)
		if err != nil {
			t.Errorf("ParamToLiteral(%q, bool) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParamToLiteral(%q, bool) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParamToLiteral_Integers(t *testing.T) {
	tests := []struct {
		oid   uint32
		input string
		want  string
	}{
		{OIDInt2, "42", "42"},
		{OIDInt4, "-100", "-100"},
		{OIDInt8, "9999999999", "9999999999"},
		{OIDOid, "12345", "12345"},
	}
	for _, tt := range tests {
		got, err := ParamToLiteral([]byte(tt.input), tt.oid, 0)
		if err != nil {
			t.Errorf("ParamToLiteral(%q, %d) error: %v", tt.input, tt.oid, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParamToLiteral(%q, %d) = %q, want %q", tt.input, tt.oid, got, tt.want)
		}
	}
}

func TestParamToLiteral_IntegerInvalid(t *testing.T) {
	_, err := ParamToLiteral([]byte("not_a_number"), OIDInt4, 0)
	if err == nil {
		t.Error("expected error for invalid integer")
	}
}

func TestParamToLiteral_Float(t *testing.T) {
	tests := []struct {
		oid   uint32
		input string
		want  string
	}{
		{OIDFloat4, "3.14", "3.14"},
		{OIDFloat8, "-2.718", "-2.718"},
		{OIDFloat8, "NaN", "'NaN'::float8"},
		{OIDFloat8, "Infinity", "'Infinity'::float8"},
		{OIDFloat8, "-Infinity", "'-Infinity'::float8"},
	}
	for _, tt := range tests {
		got, err := ParamToLiteral([]byte(tt.input), tt.oid, 0)
		if err != nil {
			t.Errorf("ParamToLiteral(%q, %d) error: %v", tt.input, tt.oid, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParamToLiteral(%q, %d) = %q, want %q", tt.input, tt.oid, got, tt.want)
		}
	}
}

func TestParamToLiteral_Numeric(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"123.456", "123.456"},
		{"-0.001", "-0.001"},
		{"99999999999999999999.99", "99999999999999999999.99"},
		{"1e10", "1e10"},
		{"NaN", "NaN"},
	}
	for _, tt := range tests {
		got, err := ParamToLiteral([]byte(tt.input), OIDNumeric, 0)
		if err != nil {
			t.Errorf("ParamToLiteral(%q, numeric) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParamToLiteral(%q, numeric) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParamToLiteral_Text(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "'hello'"},
		{"it's", "'it''s'"},
		{"O'Brien", "'O''Brien'"},
		{"", "''"},
	}
	for _, tt := range tests {
		got, err := ParamToLiteral([]byte(tt.input), OIDText, 0)
		if err != nil {
			t.Errorf("ParamToLiteral(%q, text) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParamToLiteral(%q, text) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestParamToLiteral_UUID(t *testing.T) {
	uuid := "550e8400-e29b-41d4-a716-446655440000"
	got, err := ParamToLiteral([]byte(uuid), OIDUUID, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "'" + uuid + "'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParamToLiteral_UUIDInvalid(t *testing.T) {
	_, err := ParamToLiteral([]byte("not-a-uuid"), OIDUUID, 0)
	if err == nil {
		t.Error("expected error for invalid UUID")
	}
}

func TestParamToLiteral_Bytea(t *testing.T) {
	got, err := ParamToLiteral([]byte{0xDE, 0xAD, 0xBE, 0xEF}, OIDBytea, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// bytea in text format: the input IS the text representation
	want := "E'\\\\xdeadbeef'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParamToLiteral_BinaryInt4(t *testing.T) {
	// int4 binary: big-endian 4 bytes
	data := []byte{0, 0, 0, 42}
	got, err := ParamToLiteral(data, OIDInt4, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "42" {
		t.Errorf("got %q, want %q", got, "42")
	}
}

func TestParamToLiteral_BinaryBoolean(t *testing.T) {
	got, err := ParamToLiteral([]byte{1}, OIDBoolean, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "TRUE" {
		t.Errorf("got %q, want TRUE", got)
	}

	got, err = ParamToLiteral([]byte{0}, OIDBoolean, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "FALSE" {
		t.Errorf("got %q, want FALSE", got)
	}
}

func TestParamToLiteral_UnknownType(t *testing.T) {
	// Unknown OID should be treated as text with escaping
	got, err := ParamToLiteral([]byte("some value"), 99999, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "'some value'" {
		t.Errorf("got %q, want %q", got, "'some value'")
	}
}

// T19-6: SQL Injection Defense Test Matrix
func TestParamToLiteral_SQLInjection(t *testing.T) {
	tests := []struct {
		name  string
		input string
		oid   uint32
		want  string
	}{
		{
			"drop table",
			"'; DROP TABLE users; --",
			OIDText,
			"'''; DROP TABLE users; --'",
		},
		{
			"placeholder in string",
			"$1",
			OIDText,
			"'$1'",
		},
		{
			"unicode escape",
			"\\u0027",
			OIDText,
			"'\\u0027'",
		},
		{
			"backslash",
			"test\\",
			OIDText,
			"'test\\'",
		},
		{
			"nested quotes",
			"''''",
			OIDText,
			"''''''''''", // input has 4 quotes, each escaped to '' = 8 + outer quotes = 10
		},
		{
			"semicolon injection",
			"1; DELETE FROM users",
			OIDText,
			"'1; DELETE FROM users'",
		},
		{
			"comment injection",
			"1 /* admin */ --",
			OIDText,
			"'1 /* admin */ --'",
		},
		{
			"integer injection attempt",
			"1; DROP TABLE users;--",
			OIDInt4,
			"", // should error
		},
		{
			"long string (buffer overflow)",
			string(make([]byte, 10000)),
			OIDText,
			"", // should not crash
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParamToLiteral([]byte(tt.input), tt.oid, 0)
			if tt.want == "" {
				if tt.oid == OIDInt4 {
					if err == nil {
						t.Error("expected error for injection into integer type")
					}
					return
				}
				// long string: just verify it doesn't crash
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParamToLiteral_NullByte(t *testing.T) {
	// Null bytes should be stripped from SQL string literals
	got, err := ParamToLiteral([]byte("hello\x00world"), OIDText, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "'helloworld'"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEscapeStringLiteral(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "'hello'"},
		{"it's a test", "'it''s a test'"},
		{"double''quote", "'double''''quote'"},
		{"", "''"},
		{"null\x00byte", "'nullbyte'"},
	}
	for _, tt := range tests {
		got := escapeStringLiteral(tt.input)
		if got != tt.want {
			t.Errorf("escapeStringLiteral(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsValidNumeric(t *testing.T) {
	valid := []string{"0", "123", "-456", "3.14", "-0.001", "1e10", "1.5E-3", "+42", "NaN", "Infinity", "-Infinity"}
	for _, s := range valid {
		if !isValidNumeric(s) {
			t.Errorf("isValidNumeric(%q) = false, want true", s)
		}
	}
	invalid := []string{"", "abc", "1; DROP", "1..2", "--1", "++1"}
	for _, s := range invalid {
		if isValidNumeric(s) {
			t.Errorf("isValidNumeric(%q) = true, want false", s)
		}
	}
}

func TestIsValidUUID(t *testing.T) {
	if !isValidUUID("550e8400-e29b-41d4-a716-446655440000") {
		t.Error("valid UUID rejected")
	}
	if isValidUUID("not-a-uuid") {
		t.Error("invalid UUID accepted")
	}
	if isValidUUID("550e8400-e29b-41d4-a716-44665544000g") {
		t.Error("UUID with invalid char accepted")
	}
}
