package tools

import "testing"

func TestParseAuditdLineBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name string
		line string
		want AuditLogEntry
	}{
		{
			name: "prefixed",
			line: "node=host type=AVC msg=audit(1700000000.123:456): avc: denied { read }",
			want: AuditLogEntry{
				Timestamp: "1700000000.123:456",
				Type:      "AVC",
				Message:   "avc: denied { read }",
			},
		},
		{
			name: "leading_space",
			line: " type=AVC msg=audit(1:2): denied",
			want: AuditLogEntry{Timestamp: "1:2", Type: "AVC", Message: "denied"},
		},
		{
			name: "missing_type_separator",
			line: "type=AVC): denied",
			want: AuditLogEntry{Type: "unknown", Message: "type=AVC): denied"},
		},
		{
			name: "empty_type",
			line: "type= msg=audit(1:2): denied",
			want: AuditLogEntry{Type: "unknown", Message: "type= msg=audit(1:2): denied"},
		},
		{
			name: "missing_type",
			line: "msg=audit(1:2): denied",
			want: AuditLogEntry{Type: "unknown", Message: "msg=audit(1:2): denied"},
		},
		{
			name: "empty",
			want: AuditLogEntry{Type: "unknown"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseAuditdLine(tc.line); got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}
