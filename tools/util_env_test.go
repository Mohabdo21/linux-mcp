package tools

import (
	"reflect"
	"testing"
)

func TestRedactSensitiveEnv(t *testing.T) {
	for _, tt := range []struct {
		name string
		env  []string
		want []string
	}{
		{"nil", nil, []string{}},
		{"empty", []string{}, []string{}},
		{
			"all values",
			[]string{
				"REDIS_URL=redis://user:password@host/0",
				"ARBITRARY=plain-secret",
				"PATH=/private/bin",
				"TOKEN=known-secret",
				"EMPTY=",
				"EQUALS=first=second\nthird",
				"=unnamed-secret",
				"malformed-secret",
			},
			[]string{
				"REDIS_URL=***", "ARBITRARY=***", "PATH=***",
				"TOKEN=***", "EMPTY=***", "EQUALS=***", "=***", "***",
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactSensitiveEnv(
				tt.env,
			); !reflect.DeepEqual(
				got,
				tt.want,
			) {
				t.Errorf("redactSensitiveEnv() = %q, want %q", got, tt.want)
			}
		})
	}
}
