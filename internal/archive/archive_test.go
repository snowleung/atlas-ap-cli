package archive

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildZip constructs an in-memory ZIP file from a name→content map. Each
// entry is stored as a regular file (no directory entries).
func buildZip(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip create %s: %v", name, err)
		}
		if _, err := io.WriteString(w, content); err != nil {
			t.Fatalf("zip write %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip close: %v", err)
	}
	return buf.Bytes()
}

func TestDownloadAndExtract_NormalArchive(t *testing.T) {
	zipData := buildZip(t, map[string]string{
		"a.txt":        "alpha",
		"nested/b.txt": "beta",
	})

	outDir := t.TempDir()
	res, err := DownloadAndExtract(bytes.NewReader(zipData), outDir, false, "job-1")
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	if res.OutputDir != outDir {
		t.Errorf("output_dir mismatch: %q", res.OutputDir)
	}
	if !containsString(res.ExtractedFiles, "a.txt") {
		t.Errorf("expected a.txt in extracted list: %v", res.ExtractedFiles)
	}
	if !containsString(res.ExtractedFiles, filepath.Join("nested", "b.txt")) {
		t.Errorf("expected nested/b.txt in extracted list: %v", res.ExtractedFiles)
	}
	if res.ZipPath != "" {
		t.Errorf("expected no zip path when keepZip=false, got %q", res.ZipPath)
	}
	body, err := os.ReadFile(filepath.Join(outDir, "a.txt"))
	if err != nil {
		t.Fatalf("read a.txt: %v", err)
	}
	if string(body) != "alpha" {
		t.Errorf("a.txt content mismatch: %q", body)
	}
	body, err = os.ReadFile(filepath.Join(outDir, "nested", "b.txt"))
	if err != nil {
		t.Fatalf("read nested/b.txt: %v", err)
	}
	if string(body) != "beta" {
		t.Errorf("nested/b.txt content mismatch: %q", body)
	}
}

func TestDownloadAndExtract_PreservesNestedDirectories(t *testing.T) {
	zipData := buildZip(t, map[string]string{
		"a/b/c/deep.txt": "deep",
	})
	outDir := t.TempDir()
	if _, err := DownloadAndExtract(bytes.NewReader(zipData), outDir, false, "job-n"); err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(outDir, "a", "b", "c", "deep.txt"))
	if err != nil {
		t.Fatalf("read deep.txt: %v", err)
	}
	if string(body) != "deep" {
		t.Errorf("deep.txt content mismatch: %q", body)
	}
}

func TestDownloadAndExtract_RejectsParentTraversal(t *testing.T) {
	zipData := buildZip(t, map[string]string{
		"../escape.txt": "evil",
	})
	outDir := t.TempDir()
	_, err := DownloadAndExtract(bytes.NewReader(zipData), outDir, false, "job-zip")
	if err == nil {
		t.Fatal("expected error for ../escape.txt, got nil")
	}
	if !errors.Is(err, ErrUnsafeZipMember) {
		t.Errorf("expected ErrUnsafeZipMember, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(outDir, "..", "escape.txt")); statErr == nil {
		t.Errorf("escape.txt leaked outside output dir")
	}
}

func TestDownloadAndExtract_RejectsAbsolutePath(t *testing.T) {
	zipData := buildZip(t, map[string]string{
		"/abs.txt": "evil",
	})
	outDir := t.TempDir()
	_, err := DownloadAndExtract(bytes.NewReader(zipData), outDir, false, "job-abs")
	if err == nil {
		t.Fatal("expected error for /abs.txt, got nil")
	}
	if !errors.Is(err, ErrUnsafeZipMember) {
		t.Errorf("expected ErrUnsafeZipMember, got %v", err)
	}
}

func TestDownloadAndExtract_RejectsWindowsBackslashTraversal(t *testing.T) {
	// archive/zip normalizes forward slashes in headers, but a Windows-
	// style backslash path still resolves to a parent on Windows. We
	// reject any path containing a backslash as a backstop.
	zipData := buildZip(t, map[string]string{
		`..\escape.txt`: "evil",
	})
	outDir := t.TempDir()
	_, err := DownloadAndExtract(bytes.NewReader(zipData), outDir, false, "job-bs")
	if err == nil {
		t.Fatal("expected error for ..\\escape.txt, got nil")
	}
	if !errors.Is(err, ErrUnsafeZipMember) {
		t.Errorf("expected ErrUnsafeZipMember, got %v", err)
	}
}

func TestDownloadAndExtract_RejectsArchivesOver500MiB(t *testing.T) {
	// Build a stream of 500 MiB + 1 byte. We don't actually need a valid
	// zip — the size check should fire first.
	limit := int64(500 * 1024 * 1024)
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		chunk := bytes.Repeat([]byte{'A'}, 1<<20) // 1 MiB
		written := int64(0)
		for written <= limit {
			if _, err := pw.Write(chunk); err != nil {
				return
			}
			written += int64(len(chunk))
		}
	}()

	outDir := t.TempDir()
	_, err := DownloadAndExtract(pr, outDir, false, "job-big")
	if err == nil {
		t.Fatal("expected error for >500 MiB archive, got nil")
	}
	if !errors.Is(err, ErrArchiveTooLarge) {
		t.Errorf("expected ErrArchiveTooLarge, got %v", err)
	}
}

func TestDownloadAndExtract_KeepZipRenames(t *testing.T) {
	zipData := buildZip(t, map[string]string{"x.txt": "x"})
	outDir := t.TempDir()
	res, err := DownloadAndExtract(bytes.NewReader(zipData), outDir, true, "job-kz")
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	expectedZip := filepath.Join(outDir, "job-kz.zip")
	if res.ZipPath != expectedZip {
		t.Errorf("zip_path mismatch: got %q want %q", res.ZipPath, expectedZip)
	}
	if _, err := os.Stat(expectedZip); err != nil {
		t.Errorf("expected %s to exist: %v", expectedZip, err)
	}
}

func TestDownloadAndExtract_NoTempFileLeftOnSuccess(t *testing.T) {
	zipData := buildZip(t, map[string]string{"x.txt": "x"})
	outDir := t.TempDir()
	if _, err := DownloadAndExtract(bytes.NewReader(zipData), outDir, false, "job-clean"); err != nil {
		t.Fatalf("extract failed: %v", err)
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".zip.tmp") {
			t.Errorf("temporary zip left behind: %s", e.Name())
		}
	}
}

func TestDownloadAndExtract_PartialExtractionCleaned(t *testing.T) {
	// Inject an unsafe entry by writing a custom zip with a bad member
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("good.txt")
	io.WriteString(w, "good")
	w2, _ := zw.Create("../bad.txt")
	io.WriteString(w2, "bad")
	zw.Close()

	outDir := t.TempDir()
	_, err := DownloadAndExtract(bytes.NewReader(buf.Bytes()), outDir, false, "job-mix")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUnsafeZipMember) {
		t.Errorf("expected ErrUnsafeZipMember, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "good.txt")); err == nil {
		t.Errorf("good.txt should have been cleaned up")
	}
	// No leftover temp zip
	entries, _ := os.ReadDir(outDir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp") {
			t.Errorf("temporary file left behind: %s", e.Name())
		}
	}
}

func TestDownloadAndExtract_NonZipPayloadFails(t *testing.T) {
	outDir := t.TempDir()
	_, err := DownloadAndExtract(bytes.NewReader([]byte("not a zip")), outDir, false, "job-bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func containsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
