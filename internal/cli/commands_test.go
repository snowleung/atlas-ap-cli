package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// withServer runs the supplied args with --server prepended pointing at
// srv, returning captured stdout/stderr and the exit code.
func withServer(t *testing.T, srv *httptest.Server, args ...string) (string, string, int) {
	t.Helper()
	args = append([]string{"--server", srv.URL}, args...)
	var stdout, stderr bytes.Buffer
	code := Run(args, &stdout, &stderr, []string{})
	return stdout.String(), stderr.String(), code
}

// jsonSubmitHandler captures a multipart /jobs POST and replies with the
// given job id. It records whether the request carried a Bearer token.
type recordingHandler struct {
	gotMethod      string
	gotPath        string
	gotAuth        string
	gotContentType string
	gotField       map[string]string
	gotFile        string
}

func newSubmitHandler(rec *recordingHandler, jobID string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rec.gotMethod = r.Method
		rec.gotPath = r.URL.Path
		rec.gotAuth = r.Header.Get("Authorization")
		rec.gotContentType = r.Header.Get("Content-Type")
		if err := r.ParseMultipartForm(1 << 20); err == nil {
			rec.gotField = map[string]string{}
			for k, v := range r.MultipartForm.Value {
				if len(v) > 0 {
					rec.gotField[k] = v[0]
				}
			}
			for _, files := range r.MultipartForm.File {
				for _, fh := range files {
					f, _ := fh.Open()
					data, _ := io.ReadAll(f)
					rec.gotFile = string(data)
					f.Close()
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"job_id": jobID})
	}
}

func TestRun_HelpFlagExitsZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--help"}, &stdout, &stderr, []string{})
	if code != 0 {
		t.Errorf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "atlas-ap-remote") {
		t.Errorf("expected help text, got: %q", stdout.String())
	}
}

func TestRun_VersionFlagExitsZero(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--version"}, &stdout, &stderr, []string{})
	if code != 0 {
		t.Errorf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), Version) {
		t.Errorf("expected version %q in stdout, got %q", Version, stdout.String())
	}
}

func TestRun_HealthHumanOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/health" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	stdout, stderr, code := withServer(t, srv, "health")
	if code != 0 {
		t.Fatalf("expected 0, got %d (stderr=%s)", code, stderr)
	}
	if stdout != "status=ok\n" {
		t.Errorf("unexpected health output: %q", stdout)
	}
}

func TestRun_HealthJSONOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	stdout, stderr, code := withServer(t, srv, "health", "--json")
	if code != 0 {
		t.Fatalf("expected 0, got %d (stderr=%s)", code, stderr)
	}
	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if envelope["status"] != "ok" || envelope["success"] != true {
		t.Errorf("unexpected health envelope: %v", envelope)
	}
}

func TestRun_NoArgsReturnsTwo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{}, &stdout, &stderr, []string{})
	if code != 2 {
		t.Errorf("expected 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage") {
		t.Errorf("expected usage on stderr, got: %q", stderr.String())
	}
}

func TestRun_UnknownCommandReturnsTwo(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--server", "http://x", "bogus"}, &stdout, &stderr, []string{})
	if code != 2 {
		t.Errorf("expected 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Errorf("expected 'unknown command' on stderr, got %q", stderr.String())
	}
}

func TestRun_EnvFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"job_id": "job-env"})
	}))
	defer srv.Close()

	tmp := writeTempFile(t, "x")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"submit", "--file", tmp, "--json"}, &stdout, &stderr,
		[]string{"ATLAS_REMOTE_URL=" + srv.URL})
	if code != 0 {
		t.Fatalf("expected 0, got %d (stderr=%s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), `"job_id":"job-env"`) {
		t.Errorf("expected job_id in stdout, got: %s", stdout.String())
	}
}

func TestRun_MissingServerReturnsOne(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"status", "job-x"}, &stdout, &stderr, []string{})
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "MISSING_SERVER") {
		t.Errorf("expected MISSING_SERVER on stderr, got: %q", stderr.String())
	}
	if strings.Contains(stdout.String(), "Token") || strings.Contains(stdout.String(), "Bearer") {
		t.Errorf("token leaked: %s", stdout.String())
	}
}

func TestRun_SubmitJSONEnvelope(t *testing.T) {
	rec := &recordingHandler{}
	srv := httptest.NewServer(newSubmitHandler(rec, "job-1"))
	defer srv.Close()

	tmp := writeTempFile(t, "data!")
	stdout, stderr, code := withServer(t, srv,
		"submit", "--file", tmp,
		"--cos-type", "驻留", "--body-parts", "全身",
		"--product-name", "面霜", "--usage-method", "daily",
		"--json",
	)
	if code != 0 {
		t.Fatalf("expected 0, got %d (stderr=%s)", code, stderr)
	}
	if rec.gotMethod != http.MethodPost || rec.gotPath != "/jobs" {
		t.Errorf("unexpected request: %s %s", rec.gotMethod, rec.gotPath)
	}
	if rec.gotField["cosmetic_type"] != "驻留" {
		t.Errorf("cosmetic_type mismatch: %q", rec.gotField["cosmetic_type"])
	}
	if rec.gotField["body_parts"] != "全身" {
		t.Errorf("body_parts mismatch: %q", rec.gotField["body_parts"])
	}
	if rec.gotField["product_name"] != "面霜" {
		t.Errorf("product_name mismatch: %q", rec.gotField["product_name"])
	}
	if rec.gotField["usage_method"] != "daily" {
		t.Errorf("usage_method mismatch: %q", rec.gotField["usage_method"])
	}
	if !strings.HasPrefix(rec.gotContentType, "multipart/form-data") {
		t.Errorf("expected multipart, got %q", rec.gotContentType)
	}
	if rec.gotField["idempotency_key"] == "" {
		t.Error("expected non-empty idempotency_key")
	}
	if !looksLikeUUIDv4(rec.gotField["idempotency_key"]) {
		t.Errorf("idempotency_key is not UUIDv4-shaped: %q", rec.gotField["idempotency_key"])
	}
	if rec.gotFile != "data!" {
		t.Errorf("file content mismatch: %q", rec.gotFile)
	}
	if rec.gotAuth != "" {
		t.Errorf("unexpected Authorization: %q", rec.gotAuth)
	}

	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &env); err != nil {
		t.Fatalf("stdout is not JSON: %v (%q)", err, stdout)
	}
	if env["success"] != true {
		t.Errorf("expected success=true, got %v", env["success"])
	}
	if env["job_id"] != "job-1" {
		t.Errorf("expected job_id=job-1, got %v", env["job_id"])
	}
}

func TestRun_SubmitHumanText(t *testing.T) {
	srv := httptest.NewServer(newSubmitHandler(&recordingHandler{}, "job-2"))
	defer srv.Close()

	tmp := writeTempFile(t, "x")
	stdout, _, code := withServer(t, srv, "submit", "--file", tmp)
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout, "job_id=job-2") {
		t.Errorf("expected human line, got %q", stdout)
	}
}

func TestRun_SubmitBearerHeaderWhenToken(t *testing.T) {
	rec := &recordingHandler{}
	srv := httptest.NewServer(newSubmitHandler(rec, "job-3"))
	defer srv.Close()

	tmp := writeTempFile(t, "x")
	var stdout, stderr bytes.Buffer
	code := Run([]string{"--server", srv.URL, "--token", "tk-1",
		"submit", "--file", tmp, "--json"}, &stdout, &stderr, []string{})
	if code != 0 {
		t.Fatalf("expected 0, got %d stderr=%s", code, stderr.String())
	}
	if rec.gotAuth != "Bearer tk-1" {
		t.Errorf("expected Bearer tk-1, got %q", rec.gotAuth)
	}
}

func TestRun_StatusJSONEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/jobs/job-s" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"job_id": "job-s", "status": "succeeded"})
	}))
	defer srv.Close()

	stdout, stderr, code := withServer(t, srv, "status", "job-s", "--json")
	if code != 0 {
		t.Fatalf("expected 0, got %d stderr=%s", code, stderr)
	}
	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &env); err != nil {
		t.Fatalf("not JSON: %v (%q)", err, stdout)
	}
	if env["success"] != true {
		t.Errorf("expected success=true, got %v", env["success"])
	}
	if env["status"] != "succeeded" {
		t.Errorf("expected status=succeeded, got %v", env["status"])
	}
}

func TestRun_StatusHumanText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"job_id": "job-h", "status": "queued"})
	}))
	defer srv.Close()

	stdout, _, code := withServer(t, srv, "status", "job-h")
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout, "status=queued") {
		t.Errorf("expected human line, got %q", stdout)
	}
}

func TestRun_StatusServiceErrorJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"BAD_INPUT","message":"bad job id"}`))
	}))
	defer srv.Close()

	stdout, _, code := withServer(t, srv, "status", "job-bad", "--json")
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &env); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if env["code"] != "BAD_INPUT" {
		t.Errorf("expected code=BAD_INPUT, got %v", env["code"])
	}
	if env["http_status"] != float64(400) {
		t.Errorf("expected http_status=400, got %v", env["http_status"])
	}
}

func TestRun_CancelJSONEnvelope(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/jobs/job-c/cancel" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"job_id": "job-c", "status": "cancelled"})
	}))
	defer srv.Close()

	stdout, stderr, code := withServer(t, srv, "cancel", "job-c", "--json")
	if code != 0 {
		t.Fatalf("expected 0, got %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, `"status":"cancelled"`) {
		t.Errorf("expected cancelled in stdout, got %q", stdout)
	}
}

func TestRun_CancelHumanText(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"job_id": "job-c2", "status": "cancelled"})
	}))
	defer srv.Close()

	stdout, _, code := withServer(t, srv, "cancel", "job-c2")
	if code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout, "cancelled") {
		t.Errorf("expected human line, got %q", stdout)
	}
}

func TestRun_MissingFileReturnsOne(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	stdout, stderr, code := withServer(t, srv, "submit", "--json")
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, "MISSING_ARG") {
		t.Errorf("expected MISSING_ARG, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestRun_GlobalFlagsAfterSubcommandIgnored(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	tmp := writeTempFile(t, "x")
	var stdout, stderr bytes.Buffer
	// --server placed after submit should fail (unknown flag for submit)
	code := Run([]string{"submit", "--server", srv.URL, "--file", tmp}, &stdout, &stderr, []string{})
	if code == 0 {
		t.Errorf("expected non-zero exit when --server follows submit, got 0")
	}
}

func TestRun_HelpFlagForSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"submit", "--help"}, &stdout, &stderr, []string{})
	if code != 0 {
		t.Errorf("expected 0, got %d", code)
	}
	if stdout.Len() == 0 {
		t.Error("expected help output")
	}
}

func TestRun_DataFileCommands(t *testing.T) {
	commands := []struct {
		name string
		path string
	}{
		{"material-db", "/data-files/material-db"},
		{"reference-db", "/data-files/reference-db"},
		{"risk-db", "/data-files/risk-db"},
		{"special-materials-config", "/data-files/special-materials-config"},
		{"report-template", "/data-files/report-template"},
		{"safe-material-template", "/data-files/safe-material-template"},
	}

	for _, tc := range commands {
		t.Run(tc.name+"-json", func(t *testing.T) {
			var gotMethod, gotPath, gotContentType, gotFile string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod = r.Method
				gotPath = r.URL.Path
				gotContentType = r.Header.Get("Content-Type")
				if err := r.ParseMultipartForm(1 << 20); err == nil {
					for _, files := range r.MultipartForm.File {
						for _, fh := range files {
							f, _ := fh.Open()
							data, _ := io.ReadAll(f)
							gotFile = string(data)
							f.Close()
						}
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true, "kind": tc.name})
			}))
			defer srv.Close()

			tmp := writeTempFile(t, "datafile-"+tc.name)
			stdout, stderr, code := withServer(t, srv, tc.name, "--file", tmp, "--json")
			if code != 0 {
				t.Fatalf("expected 0, got %d (stderr=%s)", code, stderr)
			}
			if gotMethod != http.MethodPost || gotPath != tc.path {
				t.Errorf("unexpected request: %s %s", gotMethod, gotPath)
			}
			if !strings.HasPrefix(gotContentType, "multipart/form-data") {
				t.Errorf("expected multipart, got %q", gotContentType)
			}
			if gotFile != "datafile-"+tc.name {
				t.Errorf("file content mismatch: %q", gotFile)
			}

			var env map[string]any
			if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &env); err != nil {
				t.Fatalf("stdout is not JSON: %v (%q)", err, stdout)
			}
			if env["success"] != true {
				t.Errorf("expected success=true, got %v", env["success"])
			}
			resp, ok := env["response"].(map[string]any)
			if !ok {
				t.Fatalf("expected response object, got %T", env["response"])
			}
			if resp["accepted"] != true || resp["kind"] != tc.name {
				t.Errorf("unexpected response: %v", resp)
			}
		})

		t.Run(tc.name+"-human", func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(map[string]any{"accepted": true, "kind": tc.name})
			}))
			defer srv.Close()

			tmp := writeTempFile(t, "x")
			stdout, stderr, code := withServer(t, srv, tc.name, "--file", tmp)
			if code != 0 {
				t.Fatalf("expected 0, got %d (stderr=%s)", code, stderr)
			}
			if !strings.Contains(stdout, `"kind": "`+tc.name+`"`) {
				t.Errorf("expected indented JSON with kind, got %q", stdout)
			}
			if !strings.Contains(stdout, "\n") {
				t.Errorf("expected multi-line indented JSON, got %q", stdout)
			}
		})
	}
}

func TestRun_DataFileMissingFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	stdout, stderr, code := withServer(t, srv, "material-db", "--json")
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, "MISSING_ARG") {
		t.Errorf("expected MISSING_ARG, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestRun_DataFileHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := Run([]string{"material-db", "--help"}, &stdout, &stderr, []string{})
	if code != 0 {
		t.Errorf("expected 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "--file") {
		t.Errorf("expected --file in help, got %q", stdout.String())
	}
}

func TestRun_DataFileServiceError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"BAD_FILE","message":"bad file"}`))
	}))
	defer srv.Close()

	tmp := writeTempFile(t, "x")
	stdout, stderr, code := withServer(t, srv, "risk-db", "--file", tmp, "--json")
	if code != 1 {
		t.Errorf("expected 1, got %d (stderr=%s)", code, stderr)
	}
	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &env); err != nil {
		t.Fatalf("stdout is not JSON: %v (%q)", err, stdout)
	}
	if env["code"] != "BAD_FILE" {
		t.Errorf("expected code=BAD_FILE, got %v", env["code"])
	}
	if env["http_status"] != float64(400) {
		t.Errorf("expected http_status=400, got %v", env["http_status"])
	}
}

// looksLikeUUIDv4 mirrors the helper in internal/client/client_test.go;
// duplicated here to keep this test package self-contained.
func looksLikeUUIDv4(s string) bool {
	if len(s) != 36 {
		return false
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	if s[14] != '4' {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "submit-*")
	if err != nil {
		t.Fatalf("tempfile: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return f.Name()
}
