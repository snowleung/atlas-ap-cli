package cli

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func buildZipBytes(entries map[string]string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(content))
	}
	_ = zw.Close()
	return buf.Bytes()
}

func TestRun_DownloadJSONEnvelope(t *testing.T) {
	payload := buildZipBytes(map[string]string{"a.txt": "alpha"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/jobs/job-dl/download" {
			t.Errorf("expected /jobs/job-dl/download, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/zip")
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	outDir := t.TempDir()
	stdout, stderr, code := withServer(t, srv, "download", "job-dl",
		"--output-dir", outDir, "--json")
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
	if env["output_dir"] != outDir {
		t.Errorf("output_dir mismatch: got %v want %v", env["output_dir"], outDir)
	}
	files, _ := env["extracted_files"].([]any)
	if len(files) != 1 || files[0] != "a.txt" {
		t.Errorf("extracted_files mismatch: %v", files)
	}
	if _, ok := env["zip_path"]; ok {
		t.Errorf("expected no zip_path without --keep-zip, got %v", env["zip_path"])
	}
	body, err := os.ReadFile(filepath.Join(outDir, "a.txt"))
	if err != nil {
		t.Fatalf("read a.txt: %v", err)
	}
	if string(body) != "alpha" {
		t.Errorf("content mismatch: %q", body)
	}
}

func TestRun_DownloadHumanText(t *testing.T) {
	payload := buildZipBytes(map[string]string{"a.txt": "alpha", "b.txt": "beta"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	outDir := t.TempDir()
	stdout, stderr, code := withServer(t, srv, "download", "job-h",
		"--output-dir", outDir)
	if code != 0 {
		t.Fatalf("expected 0, got %d stderr=%s", code, stderr)
	}
	if !strings.Contains(stdout, "extracted 2 files") {
		t.Errorf("expected human summary, got %q", stdout)
	}
}

func TestRun_DownloadKeepZip(t *testing.T) {
	payload := buildZipBytes(map[string]string{"x.txt": "x"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	outDir := t.TempDir()
	stdout, stderr, code := withServer(t, srv, "download", "job-kz",
		"--output-dir", outDir, "--keep-zip", "--json")
	if code != 0 {
		t.Fatalf("expected 0, got %d stderr=%s", code, stderr)
	}
	var env map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(stdout)), &env); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	wantZip := filepath.Join(outDir, "job-kz.zip")
	if env["zip_path"] != wantZip {
		t.Errorf("zip_path mismatch: got %v want %v", env["zip_path"], wantZip)
	}
	if _, err := os.Stat(wantZip); err != nil {
		t.Errorf("zip file missing: %v", err)
	}
}

func TestRun_DownloadUnsafeZipMember(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("../escape.txt")
	_, _ = w.Write([]byte("evil"))
	_ = zw.Close()
	payload := buf.Bytes()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	outDir := t.TempDir()
	stdout, stderr, code := withServer(t, srv, "download", "job-bad",
		"--output-dir", outDir, "--json")
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, "UNSAFE_ZIP_MEMBER") {
		t.Errorf("expected UNSAFE_ZIP_MEMBER, got stdout=%q stderr=%q", stdout, stderr)
	}
}

func TestRun_DownloadIOError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not a zip"))
	}))
	defer srv.Close()

	// output-dir pointing at a path inside a file -> mkdir will fail
	tmpFile := filepath.Join(t.TempDir(), "regular-file")
	if err := os.WriteFile(tmpFile, []byte("x"), 0o600); err != nil {
		t.Fatalf("setup: %v", err)
	}
	stdout, stderr, code := withServer(t, srv, "download", "job-io",
		"--output-dir", filepath.Join(tmpFile, "subdir"), "--json")
	if code != 1 {
		t.Errorf("expected 1, got %d", code)
	}
	combined := stdout + stderr
	if !strings.Contains(combined, "IO_ERROR") {
		t.Errorf("expected IO_ERROR, got stdout=%q stderr=%q", stdout, stderr)
	}
}
