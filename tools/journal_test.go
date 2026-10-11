package tools

import "testing"

func TestJournalMessage(t *testing.T) {
	tests := []struct {
		name string
		raw  map[string]any
		want string
	}{
		{"string", map[string]any{"MESSAGE": "hello"}, "hello"},
		{"empty string", map[string]any{"MESSAGE": ""}, ""},
		{"missing", map[string]any{}, ""},
		{
			"byte array",
			map[string]any{"MESSAGE": []any{
				float64(
					27,
				),
				float64(91),
				float64(49),
				float64(109),
				float64(65),
			}},
			"\x1b[1mA",
		},
		{"empty array", map[string]any{"MESSAGE": []any{}}, ""},
		{
			"out of range byte",
			map[string]any{"MESSAGE": []any{float64(300)}},
			"",
		},
		{"negative byte", map[string]any{"MESSAGE": []any{float64(-1)}}, ""},
		{"non-number element", map[string]any{"MESSAGE": []any{"x"}}, ""},
		{"non-string scalar", map[string]any{"MESSAGE": float64(5)}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := journalMessage(tt.raw); got != tt.want {
				t.Errorf("journalMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}
