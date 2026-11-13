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

func TestIsSensitiveEnv(t *testing.T) {
	for _, tt := range []struct {
		name, value string
		want        bool
	}{
		{"GITHUB_TOKEN", "x", true},
		{"db_password", "x", true},
		{"AWS_ACCESS_KEY_ID", "x", true},
		{"AWS_SECRET_ACCESS_KEY", "x", true},
		{"MYSQL_PWD", "x", true},
		{"PGPASSFILE", "x", false},
		{"LDAP_PASSWD", "x", true},
		{"GPG_PASSPHRASE", "x", true},
		{"GH_PAT", "x", true},
		{"SENTRY_DSN", "x", true},
		{"OPENAI_APIKEY", "x", true},
		{"SSH_KEY", "x", true},
		{"REDIS_URL", "redis://user:pass@host:6379/0", true},
		{"HTTPS_PROXY", "http://user:pass@proxy:3128", true},
		{"GIT_REMOTE", "https://ghp_abc123@github.com/o/r.git", true},
		{"HTTPS_PROXY", "http://proxy:3128", false},
		{"REDIS_URL", "redis://host:6379/0", false},
		{"PWD", "/home/user", false},
		{"OLDPWD", "/home/user", false},
		{"KEYMAP", "us", false},
		{"XDG_SESSION_PATH", "/org/freedesktop", false},
		{"PATH", "/usr/bin:/bin", false},
		{"MAIL", "user@example.com", false},
		{"LANG", "en_US.UTF-8", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSensitiveEnv(tt.name, tt.value); got != tt.want {
				t.Errorf("isSensitiveEnv(%q, %q) = %v, want %v",
					tt.name, tt.value, got, tt.want)
			}
		})
	}
}

func TestGatherEnvironmentVariablesRedaction(t *testing.T) {
	t.Setenv("LINUXMCP_TEST_ACCESS_KEY_ID", "AKIAEXAMPLE")
	t.Setenv("LINUXMCP_TEST_URL", "redis://user:hunter2@host/0")
	t.Setenv("LINUXMCP_TEST_PLAIN", "visible")

	out, err := GatherEnvironmentVariables(t.Context(), "linuxmcp_test_")
	if err != nil {
		t.Fatalf("GatherEnvironmentVariables() error = %v, want nil", err)
	}
	got := make(map[string]string, len(out.Variables))
	for _, v := range out.Variables {
		got[v.Name] = v.Value
	}
	want := map[string]string{
		"LINUXMCP_TEST_ACCESS_KEY_ID": "***",
		"LINUXMCP_TEST_URL":           "***",
		"LINUXMCP_TEST_PLAIN":         "visible",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("variables = %v, want %v", got, want)
	}
}
