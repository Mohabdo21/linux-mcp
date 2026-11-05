package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDockerResponsesOmitSecrets(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			var body string
			switch strings.TrimPrefix(r.URL.Path, "/v1.56") {
			case "/_ping":
				w.Header().Set("API-Version", "1.56")
				body = "OK"
			case "/containers/json":
				body = `[{"Id":"container123456","Names":["/app"],"Image":"app:latest","Status":"Up","Command":"run list-command-placeholder"}]`
			case "/containers/app/json":
				body = `{"Id":"container123456","Name":"/app","Image":"image123456789","Path":"/private/path-command-placeholder","Args":["--auth=argument-placeholder"],"Config":{"Env":["REDIS_URL=redis-url-placeholder","ARBITRARY=arbitrary-env-placeholder","TOKEN=known-env-placeholder"],"Cmd":["config-cmd-placeholder"],"Entrypoint":["config-entrypoint-placeholder"]},"State":{"Status":"running"}}`
			case "/images/app/json":
				body = `{"Id":"image123456789","RepoTags":["app:latest"],"RepoDigests":[],"Architecture":"amd64","Os":"linux","Config":{"Env":["REDIS_URL=image-redis-placeholder","ARBITRARY=image-arbitrary-placeholder"],"Cmd":["sh","-c","image-cmd-placeholder"],"Entrypoint":["sh","-c","image-entrypoint-placeholder"]},"RootFS":{"Type":"layers","Layers":["sha256:layer"]}}`
			case "/images/app/history":
				body = `[{"Id":"layer123456789","Created":123,"CreatedBy":"/bin/sh -c REDIS_URL=history-command-placeholder run","Size":42,"Tags":["app:latest"]}]`
			case "/containers/app/top":
				if r.URL.Query().
					Get("ps_args") ==
					"-eo pid,ppid,user,stat,comm" {
					body = `{"Titles":["PID","PPID","USER","STAT","COMMAND"],"Processes":[["1","0","root","S","app"]]}`
				} else {
					body = `{"Titles":["PID","COMMAND"],"Processes":[["1","run --auth=top-command-placeholder REDIS_URL=top-env-placeholder"]]}`
				}
			default:
				http.Error(w, "unexpected Docker API path", http.StatusNotFound)
				t.Errorf("unexpected Docker API path: %s", r.URL.Path)
				return
			}
			if _, err := fmt.Fprint(w, body); err != nil {
				t.Errorf("write fixture: %v", err)
			}
		}),
	)
	t.Cleanup(server.Close)
	t.Setenv("DOCKER_HOST", "tcp://"+strings.TrimPrefix(server.URL, "http://"))
	t.Setenv("DOCKER_API_VERSION", "1.56")
	t.Setenv("DOCKER_CONTEXT", "")
	t.Setenv("DOCKER_CONFIG", t.TempDir())
	t.Setenv("DOCKER_TLS_VERIFY", "")
	t.Setenv("DOCKER_CERT_PATH", "")

	for _, tt := range []struct {
		name   string
		gather func(context.Context) (any, error)
		want   []string
	}{
		{
			"container list",
			func(ctx context.Context) (any, error) { return ListDockerContainers(ctx) },
			[]string{`"name":"app"`, `"status":"Up"`},
		},
		{
			"container inspect",
			func(ctx context.Context) (any, error) { return GatherContainerDetail(ctx, "app") },
			[]string{`"args":[]`, `"path":"***"`, `"REDIS_URL=***"`, `"ARBITRARY=***"`, `"status":"running"`},
		},
		{
			"image inspect",
			func(ctx context.Context) (any, error) { return GatherImageDetail(ctx, "app") },
			[]string{`"REDIS_URL=***"`, `"ARBITRARY=***"`, `"architecture":"amd64"`, `"layers":["sha256:layer"]`},
		},
		{
			"image history",
			func(ctx context.Context) (any, error) { return GatherImageHistory(ctx, "app") },
			[]string{`"created_by":"***"`, `"created":123`, `"size":"42 B"`},
		},
		{
			"container top default",
			func(ctx context.Context) (any, error) { return GatherContainerTop(ctx, "app", nil) },
			[]string{`"1","0","root","S","app"`},
		},
		{
			"container top custom environment format",
			func(ctx context.Context) (any, error) { return GatherContainerTop(ctx, "app", []string{"auxeww"}) },
			[]string{`"1","0","root","S","app"`},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, err := tt.gather(t.Context())
			if err != nil {
				t.Fatalf("gather: %v", err)
			}
			data, err := json.Marshal(out)
			if err != nil {
				t.Fatalf("marshal response: %v", err)
			}
			if strings.Contains(string(data), "-placeholder") {
				t.Errorf("response exposes a secret: %s", data)
			}
			for _, want := range tt.want {
				if !strings.Contains(string(data), want) {
					t.Errorf("response %s does not contain %s", data, want)
				}
			}
		})
	}
}
