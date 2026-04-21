package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Dixa-public/dixa-cli-public/internal/config"
	"github.com/Dixa-public/dixa-cli-public/internal/spec"
	"github.com/Dixa-public/dixa-cli-public/internal/update"
)

type integrationStore struct{}

func (integrationStore) Get(profile string) (string, error) { return "", config.ErrSecretNotFound }
func (integrationStore) Set(profile, value string) error    { return nil }
func (integrationStore) Delete(profile string) error        { return config.ErrSecretNotFound }

func testEnv(t *testing.T) *Env {
	t.Helper()
	manifest, err := spec.Load()
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	home := t.TempDir()
	return &Env{
		HomeDir:     home,
		Spec:        manifest,
		Config:      config.NewManager(home, integrationStore{}),
		HTTPClient:  &http.Client{Timeout: 2 * time.Second},
		Updater:     nil,
		Stdin:       strings.NewReader(""),
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		IsStdoutTTY: false,
		IsStdinTTY:  false,
		Version:     "0.1.2",
	}
}

func executeCLI(t *testing.T, env *Env, args ...string) (string, string, error) {
	t.Helper()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	env.Stdout = stdout
	env.Stderr = stderr
	cmd := NewRootCmd(env)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestRepresentativeCommandsPerDomain(t *testing.T) {
	t.Parallel()

	type commandCase struct {
		name         string
		args         []string
		method       string
		path         string
		query        url.Values
		body         string
		status       int
		responseBody string
	}

	cases := []commandCase{
		{
			name:         "org get",
			args:         []string{"org", "get"},
			method:       http.MethodGet,
			path:         "/organization",
			status:       http.StatusOK,
			responseBody: `{"data":{"id":"org-1","name":"Acme"}}`,
		},
		{
			name:         "settings list",
			args:         []string{"settings", "list", "--type", "EmailEndpoint"},
			method:       http.MethodGet,
			path:         "/contact-endpoints",
			query:        url.Values{"_type": []string{"EmailEndpoint"}},
			status:       http.StatusOK,
			responseBody: `{"data":[{"name":"Support","_type":"EmailEndpoint"}]}`,
		},
		{
			name:         "agents list",
			args:         []string{"agents", "list", "--page-limit", "5"},
			method:       http.MethodGet,
			path:         "/agents",
			query:        url.Values{"pageLimit": []string{"5"}},
			status:       http.StatusOK,
			responseBody: `{"data":[{"id":"agent-1","displayName":"Alice"}]}`,
		},
		{
			name:         "agents add",
			args:         []string{"--yes", "agents", "add", "--display-name", "Alice", "--email", "alice@example.com"},
			method:       http.MethodPost,
			path:         "/agents",
			body:         `{"displayName":"Alice","email":"alice@example.com"}`,
			status:       http.StatusOK,
			responseBody: `{"data":{"id":"agent-1"}}`,
		},
		{
			name:         "end users get",
			args:         []string{"end-users", "get", "user-1"},
			method:       http.MethodGet,
			path:         "/endusers/user-1",
			status:       http.StatusOK,
			responseBody: `{"data":{"id":"user-1"}}`,
		},
		{
			name:         "end users add",
			args:         []string{"--yes", "end-users", "add", "--display-name", "Jane"},
			method:       http.MethodPost,
			path:         "/endusers",
			body:         `{"displayName":"Jane"}`,
			status:       http.StatusOK,
			responseBody: `{"data":{"id":"user-2"}}`,
		},
		{
			name:         "conversations search",
			args:         []string{"conversations", "search", "--page-limit", "10", "--filters", `{"strategy":"All","conditions":[{"field":{"operator":{"values":["email"],"_type":"IncludesOne"},"_type":"Channel"}}]}`, "--query", `{"value":"refund","exactMatch":false}`},
			method:       http.MethodPost,
			path:         "/search/conversations",
			query:        url.Values{"pageLimit": []string{"10"}},
			body:         `{"filters":{"strategy":"All","conditions":[{"field":{"operator":{"values":["email"],"_type":"IncludesOne"},"_type":"Channel"}}]},"query":{"value":"refund","exactMatch":false}}`,
			status:       http.StatusOK,
			responseBody: `{"data":[{"id":"conv-1"}]}`,
		},
		{
			name:         "conversations search filters only",
			args:         []string{"conversations", "search", "--filters", `{"strategy":"All","conditions":[{"field":{"operator":{"values":["email"],"_type":"IncludesOne"},"_type":"Channel"}}]}`},
			method:       http.MethodPost,
			path:         "/search/conversations",
			body:         `{"filters":{"strategy":"All","conditions":[{"field":{"operator":{"values":["email"],"_type":"IncludesOne"},"_type":"Channel"}}]}}`,
			status:       http.StatusOK,
			responseBody: `{"data":[{"id":"conv-2"}]}`,
		},
		{
			name:         "conversations close",
			args:         []string{"--yes", "conversations", "close", "conv-1", "--user-id", "agent-1"},
			method:       http.MethodPut,
			path:         "/conversations/conv-1/close",
			body:         `{"userId":"agent-1"}`,
			status:       http.StatusOK,
			responseBody: `{"success":true}`,
		},
		{
			name:         "tags list",
			args:         []string{"tags", "list-tags", "--include-deactivated"},
			method:       http.MethodGet,
			path:         "/tags",
			query:        url.Values{"includeDeactivated": []string{"true"}},
			status:       http.StatusOK,
			responseBody: `{"data":[{"id":"tag-1","name":"vip"}]}`,
		},
		{
			name:         "tags add",
			args:         []string{"--yes", "tags", "add", "--name", "vip", "--color", "red"},
			method:       http.MethodPost,
			path:         "/tags",
			body:         `{"name":"vip","color":"red"}`,
			status:       http.StatusOK,
			responseBody: `{"data":{"id":"tag-1"}}`,
		},
		{
			name:         "teams list",
			args:         []string{"teams", "list"},
			method:       http.MethodGet,
			path:         "/teams",
			status:       http.StatusOK,
			responseBody: `{"data":[{"id":"team-1","name":"Support"}]}`,
		},
		{
			name:         "teams add team",
			args:         []string{"--yes", "teams", "add-team", "--name", "Support"},
			method:       http.MethodPost,
			path:         "/teams",
			body:         `{"name":"Support"}`,
			status:       http.StatusOK,
			responseBody: `{"data":{"id":"team-1"}}`,
		},
		{
			name:         "queues list",
			args:         []string{"queues", "list"},
			method:       http.MethodGet,
			path:         "/queues",
			status:       http.StatusOK,
			responseBody: `{"data":[{"id":"queue-1"}]}`,
		},
		{
			name:         "queues assign",
			args:         []string{"--yes", "queues", "assign", "queue-1", "--agent-ids", "agent-1", "--agent-ids", "agent-2"},
			method:       http.MethodPatch,
			path:         "/queues/queue-1/members",
			body:         `{"agentIds":["agent-1","agent-2"]}`,
			status:       http.StatusOK,
			responseBody: `{"success":true}`,
		},
		{
			name:         "knowledge list",
			args:         []string{"knowledge", "list"},
			method:       http.MethodGet,
			path:         "/knowledge/articles",
			status:       http.StatusOK,
			responseBody: `{"data":[{"id":"article-1"}]}`,
		},
		{
			name:         "knowledge add",
			args:         []string{"--yes", "knowledge", "add", "--title", "FAQ", "--content", "Answer"},
			method:       http.MethodPost,
			path:         "/knowledge/articles",
			body:         `{"title":"FAQ","content":"Answer"}`,
			status:       http.StatusOK,
			responseBody: `{"data":{"id":"article-1"}}`,
		},
		{
			name:         "custom attributes list",
			args:         []string{"custom-attributes", "list"},
			method:       http.MethodGet,
			path:         "/custom-attributes",
			status:       http.StatusOK,
			responseBody: `{"data":[{"id":"attr-1"}]}`,
		},
		{
			name:         "custom attributes update conversation",
			args:         []string{"--yes", "custom-attributes", "update-conversation-custom-attributes", "conv-1", "--custom-attributes", `{"attr-1":"gold"}`},
			method:       http.MethodPatch,
			path:         "/conversations/conv-1/custom-attributes",
			body:         `{"attr-1":"gold"}`,
			status:       http.StatusOK,
			responseBody: `{"success":true}`,
		},
		{
			name:         "analytics aggregated data",
			args:         []string{"analytics", "fetch-aggregated-data", "--metric-id", "closed_conversations", "--timezone", "UTC", "--filters", `[{"attribute":"channel","values":["email"]}]`, "--aggregations", `["Count"]`},
			method:       http.MethodPost,
			path:         "/analytics/metrics",
			body:         `{"aggregations":["Count"],"filters":[{"attribute":"channel","values":["email"]}],"id":"closed_conversations","timezone":"UTC"}`,
			status:       http.StatusOK,
			responseBody: `{"data":{"id":"closed_conversations","aggregates":[{"measure":"Count","value":3}]}}`,
		},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != "test-key" {
					t.Fatalf("expected Authorization header to carry raw API key, got %q", got)
				}
				if r.Method != tt.method {
					t.Fatalf("method mismatch: want %s got %s", tt.method, r.Method)
				}
				if r.URL.Path != tt.path {
					t.Fatalf("path mismatch: want %s got %s", tt.path, r.URL.Path)
				}
				if tt.query != nil && r.URL.RawQuery != tt.query.Encode() {
					t.Fatalf("query mismatch: want %s got %s", tt.query.Encode(), r.URL.RawQuery)
				}
				if tt.body != "" {
					defer r.Body.Close()
					var gotBody any
					if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
						t.Fatalf("decode request body: %v", err)
					}
					assertJSONEqual(t, tt.body, gotBody)
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			env := testEnv(t)
			stdout, _, err := executeCLI(t, env, append([]string{"--api-key", "test-key", "--base-url", server.URL, "--output", "json"}, tt.args...)...)
			if err != nil {
				t.Fatalf("execute cli: %v", err)
			}
			if stdout == "" {
				t.Fatalf("expected JSON output")
			}
		})
	}
}

func TestAnalyticsPrepareMetricQueryWorkflow(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/analytics/metrics/csat":
			_, _ = w.Write([]byte(`{"data":{"description":"CSAT","filters":[{"filterAttribute":"channel","description":"Channel"}],"aggregations":[{"measure":"Count"}],"relatedRecordIds":["ratings"]}}`))
		case "/analytics/filter/channel":
			_, _ = w.Write([]byte(`{"data":[{"value":"EMAIL","label":"Email"}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	env := testEnv(t)
	stdout, _, err := executeCLI(t, env, "--api-key", "test-key", "--base-url", server.URL, "--output", "json", "analytics", "prepare-analytics-metric-query", "--metric-id", "csat")
	if err != nil {
		t.Fatalf("execute cli: %v", err)
	}

	if !strings.Contains(stdout, `"metric_id": "csat"`) || !strings.Contains(stdout, `"attribute": "channel"`) {
		t.Fatalf("unexpected analytics workflow output: %s", stdout)
	}
}

func TestOrgGetAcceptance(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/organization" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":{"id":"org-acceptance"}}`))
	}))
	defer server.Close()

	env := testEnv(t)
	stdout, _, err := executeCLI(t, env, "--api-key", "test-key", "--base-url", server.URL, "--output", "json", "org", "get")
	if err != nil {
		t.Fatalf("org get acceptance: %v", err)
	}
	if !strings.Contains(stdout, `"org-acceptance"`) {
		t.Fatalf("unexpected output: %s", stdout)
	}
}

func TestConversationSearchAcceptance(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/search/conversations" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"conv-acceptance"}]}`))
	}))
	defer server.Close()

	env := testEnv(t)
	stdout, _, err := executeCLI(t, env,
		"--api-key", "test-key",
		"--base-url", server.URL,
		"--output", "json",
		"conversations", "search",
		"--query", `{"value":"refund","exactMatch":false}`,
	)
	if err != nil {
		t.Fatalf("conversation search acceptance: %v", err)
	}
	if !strings.Contains(stdout, `"conv-acceptance"`) {
		t.Fatalf("unexpected output: %s", stdout)
	}
}

func TestConversationSearchValidation(t *testing.T) {
	t.Parallel()

	env := testEnv(t)
	_, _, err := executeCLI(t, env,
		"--api-key", "test-key",
		"--base-url", "https://dev.dixa.io/v1",
		"--output", "json",
		"conversations", "search",
		"--filters", `[{"attribute":"channel","values":["email"]}]`,
	)
	if err == nil {
		t.Fatalf("expected local validation error for invalid filter shape")
	}
	if !strings.Contains(err.Error(), "\"strategy\" and \"conditions\"") {
		t.Fatalf("unexpected validation message: %v", err)
	}
}

func TestAnalyticsPaginationQuery(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/analytics/records" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("pageKey"); got != "next-page" {
			t.Fatalf("expected pageKey query, got %q", got)
		}
		if got := r.URL.Query().Get("pageLimit"); got != "100" {
			t.Fatalf("expected pageLimit query, got %q", got)
		}
		var gotBody any
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		assertJSONEqual(t, `{"id":"ratings","timezone":"UTC","filters":[{"attribute":"channel","values":["email"]}]}`, gotBody)
		_, _ = w.Write([]byte(`{"data":[{"id":"row-1"}],"pageKey":"next-page"}`))
	}))
	defer server.Close()

	env := testEnv(t)
	stdout, _, err := executeCLI(t, env,
		"--api-key", "test-key",
		"--base-url", server.URL,
		"--output", "json",
		"analytics", "fetch-unaggregated-data",
		"--record-id", "ratings",
		"--timezone", "UTC",
		"--filters", `[{"attribute":"channel","values":["email"]}]`,
		"--page-key", "next-page",
		"--page-limit", "100",
	)
	if err != nil {
		t.Fatalf("execute cli: %v", err)
	}
	if !strings.Contains(stdout, `"pageKey": "next-page"`) {
		t.Fatalf("unexpected pagination output: %s", stdout)
	}
}

func TestMutatingCommandRequiresYes(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("server should not be called when confirmation fails")
	}))
	defer server.Close()

	env := testEnv(t)
	_, stderr, err := executeCLI(t, env, "--api-key", "test-key", "--base-url", server.URL, "--output", "json", "tags", "add", "--name", "vip")
	if err == nil {
		t.Fatalf("expected write command to fail without --yes in non-interactive mode")
	}
	if !strings.Contains(stderr, "[write] tags.add_tag") {
		t.Fatalf("expected confirmation summary in stderr, got %s", stderr)
	}
}

func TestHTTPErrorsSurface(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "client error", statusCode: http.StatusNotFound, body: `{"error":"missing"}`},
		{name: "server error", statusCode: http.StatusInternalServerError, body: `{"error":"boom"}`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			env := testEnv(t)
			_, _, err := executeCLI(t, env, "--api-key", "test-key", "--base-url", server.URL, "--output", "json", "org", "get")
			if err == nil {
				t.Fatalf("expected HTTP error")
			}
			if !strings.Contains(err.Error(), tt.body) {
				t.Fatalf("expected error to include response body, got %v", err)
			}
		})
	}
}

func assertJSONEqual(t *testing.T, expected string, actual any) {
	t.Helper()

	var want any
	if err := json.Unmarshal([]byte(expected), &want); err != nil {
		t.Fatalf("parse expected json: %v", err)
	}
	gotBytes, err := json.Marshal(actual)
	if err != nil {
		t.Fatalf("marshal actual json: %v", err)
	}
	var got any
	if err := json.Unmarshal(gotBytes, &got); err != nil {
		t.Fatalf("parse actual json: %v", err)
	}
	if !jsonEqual(want, got) {
		t.Fatalf("json mismatch:\nwant %s\ngot  %s", expected, string(gotBytes))
	}
}

func jsonEqual(a, b any) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return bytes.Equal(left, right)
}

func TestAuthShowUsesMaskedKey(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	store := &authStore{values: map[string]string{"default": "super-secret"}}
	manager := config.NewManager(home, store)
	if err := manager.UpsertProfile("default", config.Profile{BaseURL: "https://api.example", Output: "json"}, "super-secret", true); err != nil {
		t.Fatalf("upsert profile: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	manifest, err := spec.Load()
	if err != nil {
		t.Fatalf("load spec: %v", err)
	}
	env := &Env{
		HomeDir:     home,
		Spec:        manifest,
		Config:      manager,
		HTTPClient:  &http.Client{},
		Updater:     nil,
		Stdin:       strings.NewReader(""),
		Stdout:      &bytes.Buffer{},
		Stderr:      &bytes.Buffer{},
		IsStdoutTTY: false,
		IsStdinTTY:  false,
		Version:     "0.1.2",
	}

	stdout, _, err := executeCLI(t, env, "--output", "json", "auth", "show")
	if err != nil {
		t.Fatalf("auth show: %v", err)
	}
	if !strings.Contains(stdout, `"api_key": "sup******ret"`) {
		t.Fatalf("expected masked API key, got %s", stdout)
	}
}

func TestUpdateCommandUsesUpdaterWithoutAPIKey(t *testing.T) {
	env := testEnv(t)
	env.Updater = &fakeUpdater{
		updateFn: func(ctx context.Context, currentVersion string) (update.UpdateResult, error) {
			return update.UpdateResult{
				Status:         "updated",
				Message:        "Updated dixa to 0.1.3.",
				CurrentVersion: currentVersion,
				LatestVersion:  "0.1.3",
				LatestTag:      "v0.1.3",
				ExecutablePath: "/usr/local/bin/dixa",
			}, nil
		},
	}

	stdout, stderr, err := executeCLI(t, env, "--output", "json", "update")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr, got %s", stderr)
	}
	if !strings.Contains(stdout, `"status": "updated"`) || !strings.Contains(stdout, `"latest_version": "0.1.3"`) {
		t.Fatalf("unexpected update output: %s", stdout)
	}
}

func TestUpdateNoticeAppearsOnEligibleCommands(t *testing.T) {
	env := testEnv(t)
	env.Updater = &fakeUpdater{
		checkFn: func(ctx context.Context, currentVersion string) (update.CheckResult, error) {
			return update.CheckResult{
				CurrentVersion:  currentVersion,
				LatestVersion:   "0.1.3",
				LatestTag:       "v0.1.3",
				UpdateAvailable: true,
			}, nil
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"id":"org-1","name":"Acme"}}`))
	}))
	defer server.Close()

	stdout, stderr, err := executeCLI(t, env, "--api-key", "test-key", "--base-url", server.URL, "--output", "json", "org", "get")
	if err != nil {
		t.Fatalf("org get: %v", err)
	}
	if !json.Valid([]byte(stdout)) {
		t.Fatalf("expected stdout to remain valid JSON, got %s", stdout)
	}
	if !strings.Contains(stderr, "Run 'dixa update' to install it.") {
		t.Fatalf("expected update notice in stderr, got %s", stderr)
	}
}

func TestUpdateNoticeSkipsHelpCompletionVersionAndUpdate(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "help command", args: []string{"help"}},
		{name: "completion", args: []string{"completion", "bash"}},
		{name: "version flag", args: []string{"--version"}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			env := testEnv(t)
			fake := &fakeUpdater{
				checkFn: func(ctx context.Context, currentVersion string) (update.CheckResult, error) {
					return update.CheckResult{
						CurrentVersion:  currentVersion,
						LatestVersion:   "0.1.3",
						UpdateAvailable: true,
					}, nil
				},
			}
			env.Updater = fake

			_, stderr, err := executeCLI(t, env, tt.args...)
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if fake.checkCalls != 0 {
				t.Fatalf("expected no update check calls, got %d", fake.checkCalls)
			}
			if strings.Contains(stderr, "Run 'dixa update'") {
				t.Fatalf("expected no update notice, got %s", stderr)
			}
		})
	}

	env := testEnv(t)
	fake := &fakeUpdater{
		checkFn: func(ctx context.Context, currentVersion string) (update.CheckResult, error) {
			return update.CheckResult{
				CurrentVersion:  currentVersion,
				LatestVersion:   "0.1.3",
				UpdateAvailable: true,
			}, nil
		},
		updateFn: func(ctx context.Context, currentVersion string) (update.UpdateResult, error) {
			return update.UpdateResult{
				Status:         "up_to_date",
				Message:        "dixa is already up to date.",
				CurrentVersion: currentVersion,
				LatestVersion:  currentVersion,
				LatestTag:      "v" + currentVersion,
			}, nil
		},
	}
	env.Updater = fake

	_, stderr, err := executeCLI(t, env, "--output", "json", "update")
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if fake.checkCalls != 0 {
		t.Fatalf("expected no check call for update command, got %d", fake.checkCalls)
	}
	if fake.updateCalls != 1 {
		t.Fatalf("expected one update call, got %d", fake.updateCalls)
	}
	if strings.Contains(stderr, "Run 'dixa update'") {
		t.Fatalf("expected no update notice during update command, got %s", stderr)
	}
}

func TestUpdateCheckFailureDoesNotFailCommand(t *testing.T) {
	env := testEnv(t)
	env.Updater = &fakeUpdater{
		checkFn: func(ctx context.Context, currentVersion string) (update.CheckResult, error) {
			return update.CheckResult{}, errors.New("boom")
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"id":"org-1"}}`))
	}))
	defer server.Close()

	stdout, stderr, err := executeCLI(t, env, "--api-key", "test-key", "--base-url", server.URL, "--output", "json", "org", "get")
	if err != nil {
		t.Fatalf("org get: %v", err)
	}
	if !strings.Contains(stdout, `"id": "org-1"`) {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if strings.Contains(stderr, "Run 'dixa update'") {
		t.Fatalf("expected no update notice, got %s", stderr)
	}
}

type authStore struct {
	values map[string]string
}

func (a *authStore) Get(profile string) (string, error) {
	value, ok := a.values[profile]
	if !ok {
		return "", config.ErrSecretNotFound
	}
	return value, nil
}

func (a *authStore) Set(profile, value string) error {
	if a.values == nil {
		a.values = map[string]string{}
	}
	a.values[profile] = value
	return nil
}

func (a *authStore) Delete(profile string) error {
	delete(a.values, profile)
	return nil
}

type fakeUpdater struct {
	checkCalls  int
	updateCalls int
	checkFn     func(context.Context, string) (update.CheckResult, error)
	updateFn    func(context.Context, string) (update.UpdateResult, error)
}

func (f *fakeUpdater) Check(ctx context.Context, currentVersion string) (update.CheckResult, error) {
	f.checkCalls++
	if f.checkFn != nil {
		return f.checkFn(ctx, currentVersion)
	}
	return update.CheckResult{}, nil
}

func (f *fakeUpdater) SelfUpdate(ctx context.Context, currentVersion string) (update.UpdateResult, error) {
	f.updateCalls++
	if f.updateFn != nil {
		return f.updateFn(ctx, currentVersion)
	}
	return update.UpdateResult{}, nil
}
