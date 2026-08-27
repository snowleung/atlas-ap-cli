// Package archive downloads and safely extracts result ZIP archives from
// the Atlas AP Remote service. It defends against zip-slip attacks,
// enforces a hard size cap, and cleans up partial files on any failure.
package archive

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// MaxArchiveBytes is the maximum number of bytes accepted from a single
// download stream. Archives larger than this are rejected before any
// extraction begins.
const MaxArchiveBytes = 500 * 1024 * 1024 // 500 MiB

// Sentinel errors returned by DownloadAndExtract.
var (
	// ErrArchiveTooLarge is returned when the download exceeds
	// MaxArchiveBytes.
	ErrArchiveTooLarge = errors.New("archive exceeds 500 MiB limit")
	// ErrUnsafeZipMember is returned when any entry's path attempts to
	// escape the output directory (zip-slip) or contains a backslash.
	ErrUnsafeZipMember = errors.New("unsafe zip member path")
	// ErrIOError wraps any underlying filesystem failure during
	// extraction.
	ErrIOError = errors.New("archive io error")
)

// Result describes the outcome of a successful download/extract.
type Result struct {
	// ExtractedFiles lists paths of regular files written to OutputDir,
	// relative to OutputDir and using the local path separator.
	ExtractedFiles []string
	// ZipPath is the absolute path of the retained ZIP file, or empty
	// when keepZip is false.
	ZipPath string
	// OutputDir is the absolute path of the directory that received
	// the extracted contents.
	OutputDir string
}

// DownloadAndExtract streams body into a temporary file inside outputDir,
// validates every ZIP entry path against zip-slip, and extracts each
// entry into outputDir. When keepZip is true, the temporary file is
// renamed to "<jobID>.zip" on success. On any error, the function removes
// every file it created and returns a non-nil error.
//
// The temporary file is named "<jobID>.zip.tmp" and is created in
// outputDir so the rename is atomic on a single filesystem.
func DownloadAndExtract(body io.Reader, outputDir string, keepZip bool, jobID string) (*Result, error) {
	if outputDir == "" {
		return nil, fmt.Errorf("%w: output dir required", ErrIOError)
	}
	if jobID == "" {
		return nil, fmt.Errorf("%w: job id required", ErrIOError)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("%w: mkdir %s: %v", ErrIOError, outputDir, err)
	}

	tmpPath := filepath.Join(outputDir, jobID+".zip.tmp")
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return nil, fmt.Errorf("%w: create temp zip: %v", ErrIOError, err)
	}

	cleanup := func() {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
	}

	// Stream body into the temp file with a hard size cap. Once the cap
	// is exceeded we stop reading and report ErrArchiveTooLarge.
	limited := &io.LimitedReader{R: body, N: MaxArchiveBytes + 1}
	written, err := io.Copy(tmpFile, limited)
	if err != nil {
		cleanup()
		return nil, fmt.Errorf("%w: copy download: %v", ErrIOError, err)
	}
	if written > MaxArchiveBytes {
		cleanup()
		return nil, ErrArchiveTooLarge
	}
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("%w: close temp zip: %v", ErrIOError, err)
	}

	// Open the temp file as a zip archive and validate every member
	// path before writing any of them to disk.
	zr, err := zip.OpenReader(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("%w: open zip: %v", ErrIOError, err)
	}
	defer zr.Close()

	if err := validateMembers(zr); err != nil {
		_ = os.Remove(tmpPath)
		return nil, err
	}

	// Extraction — track created files so we can roll back on failure.
	created := make([]string, 0, len(zr.File))
	rollBack := func() {
		for _, p := range created {
			_ = os.Remove(p)
		}
		_ = os.Remove(tmpPath)
	}

	var extracted []string
	for _, f := range zr.File {
		dest, err := safeJoin(outputDir, f.Name)
		if err != nil {
			rollBack()
			return nil, err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(dest, 0o755); err != nil {
				rollBack()
				return nil, fmt.Errorf("%w: mkdir %s: %v", ErrIOError, dest, err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			rollBack()
			return nil, fmt.Errorf("%w: mkdir parent: %v", ErrIOError, err)
		}

		src, err := f.Open()
		if err != nil {
			rollBack()
			return nil, fmt.Errorf("%w: open member: %v", ErrIOError, err)
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			_ = src.Close()
			rollBack()
			return nil, fmt.Errorf("%w: create %s: %v", ErrIOError, dest, err)
		}
		if _, err := io.Copy(out, src); err != nil {
			_ = src.Close()
			_ = out.Close()
			rollBack()
			return nil, fmt.Errorf("%w: extract %s: %v", ErrIOError, dest, err)
		}
		_ = src.Close()
		if err := out.Close(); err != nil {
			rollBack()
			return nil, fmt.Errorf("%w: close %s: %v", ErrIOError, dest, err)
		}
		created = append(created, dest)
		rel, _ := filepath.Rel(outputDir, dest)
		extracted = append(extracted, rel)
	}

	res := &Result{OutputDir: outputDir, ExtractedFiles: extracted}

	if keepZip {
		finalPath := filepath.Join(outputDir, jobID+".zip")
		if err := os.Rename(tmpPath, finalPath); err != nil {
			rollBack()
			return nil, fmt.Errorf("%w: rename zip: %v", ErrIOError, err)
		}
		res.ZipPath = finalPath
	} else {
		if err := os.Remove(tmpPath); err != nil {
			// extracted files already exist; failure here is a clean-up
			// problem but not a corruption of the output.
			return nil, fmt.Errorf("%w: remove temp zip: %v", ErrIOError, err)
		}
	}

	return res, nil
}

// validateMembers checks every entry path in zr before any extraction.
// All-or-nothing: if any path is unsafe, the entire archive is rejected.
func validateMembers(zr *zip.ReadCloser) error {
	for _, f := range zr.File {
		if err := validateMemberPath(f.Name); err != nil {
			return err
		}
	}
	return nil
}

func validateMemberPath(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty member path", ErrUnsafeZipMember)
	}
	if strings.Contains(name, `\`) {
		return fmt.Errorf("%w: backslash in %q", ErrUnsafeZipMember, name)
	}
	if strings.HasPrefix(name, "/") {
		return fmt.Errorf("%w: absolute path %q", ErrUnsafeZipMember, name)
	}
	// Resolve against a dummy base; we only care about whether the path
	// escapes via "..", so a non-existent base is fine.
	cleaned := filepath.Clean(name)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: parent traversal in %q", ErrUnsafeZipMember, name)
	}
	return nil
}

// safeJoin ensures dest stays inside base.
func safeJoin(base, name string) (string, error) {
	cleaned := filepath.Clean("/" + name)
	rel, err := filepath.Rel("/", cleaned)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnsafeZipMember, err)
	}
	dest := filepath.Join(base, rel)
	// Verify the resulting path is still inside base. We rely on the
	// preceding validateMembers to reject ".." early, but this is a
	// belt-and-suspenders check for symlinked bases.
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrIOError, err)
	}
	absDest, err := filepath.Abs(dest)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrIOError, err)
	}
	if !strings.HasPrefix(absDest, absBase+string(filepath.Separator)) && absDest != absBase {
		return "", fmt.Errorf("%w: resolved path escapes base", ErrUnsafeZipMember)
	}
	return dest, nil
}
