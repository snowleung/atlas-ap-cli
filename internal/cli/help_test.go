package cli

import (
	"bytes"
	"strings"
	"testing"
)

// TestHelpText_TopLevel exercises the top-level printUsage text. It must
// mention both environment variables, all four commands, and make clear
// that the CLI does not poll the service.
func TestHelpText_TopLevel(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	out := buf.String()

	for _, want := range []string{
		"atlas-ap-remote",
		"ATLAS_REMOTE_URL",
		"ATLAS_REMOTE_TOKEN",
		"submit",
		"status",
		"download",
		"cancel",
		"health",
		"--server",
		"--token",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("help text missing %q\n---\n%s", want, out)
		}
	}

	if !strings.Contains(strings.ToLower(out), "no polling") &&
		!strings.Contains(strings.ToLower(out), "does not poll") &&
		!strings.Contains(strings.ToLower(out), "single request") &&
		!strings.Contains(strings.ToLower(out), "one request") {
		t.Errorf("help text should explain no-polling behavior\n---\n%s", out)
	}
}

// TestHelpText_TopLevelDataFiles requires the top-level help to list all
// four data-file commands alongside their endpoints.
func TestHelpText_TopLevelDataFiles(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	out := buf.String()

	for _, want := range []string{
		"material-db", "/data-files/material-db",
		"reference-db", "/data-files/reference-db",
		"risk-db", "/data-files/risk-db",
		"special-materials-config", "/data-files/special-materials-config",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("top-level help missing %q\n---\n%s", want, out)
		}
	}
}

// TestHelpText_DataFile requires the command-specific usage to mention the
// command name, --file, --json, --help, and the single-POST contract.
func TestHelpText_DataFile(t *testing.T) {
	for _, command := range []string{"material-db", "reference-db", "risk-db", "special-materials-config"} {
		var buf bytes.Buffer
		printDataFileUsage(&buf, command)
		out := buf.String()
		for _, want := range []string{command, "--file", "--json", "--help", "POST"} {
			if !strings.Contains(out, want) {
				t.Errorf("%s help missing %q\n---\n%s", command, want, out)
			}
		}
	}
}

func TestHelpText_Health(t *testing.T) {
	var buf bytes.Buffer
	printHealthUsage(&buf)
	out := buf.String()
	for _, want := range []string{"health", "GET /health", "--json"} {
		if !strings.Contains(out, want) {
			t.Errorf("health help missing %q\n---\n%s", want, out)
		}
	}
}

// TestHelpText_Submit mentions all submit flags plus --json.
func TestHelpText_Submit(t *testing.T) {
	var buf bytes.Buffer
	printSubmitUsage(&buf)
	out := buf.String()

	for _, want := range []string{
		"submit",
		"--file",
		"--cos-type",
		"--body-parts",
		"--product-name",
		"--usage-method",
		"--json",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("submit help missing %q\n---\n%s", want, out)
		}
	}
}

// TestHelpText_Status shows the positional <job-id> argument.
func TestHelpText_Status(t *testing.T) {
	var buf bytes.Buffer
	printStatusUsage(&buf)
	out := buf.String()
	if !strings.Contains(out, "status") || !strings.Contains(out, "job-id") {
		t.Errorf("status help missing required tokens\n---\n%s", out)
	}
	if !strings.Contains(out, "--json") {
		t.Errorf("status help missing --json\n---\n%s", out)
	}
}

// TestHelpText_Cancel shows the positional <job-id> argument.
func TestHelpText_Cancel(t *testing.T) {
	var buf bytes.Buffer
	printCancelUsage(&buf)
	out := buf.String()
	if !strings.Contains(out, "cancel") || !strings.Contains(out, "job-id") {
		t.Errorf("cancel help missing required tokens\n---\n%s", out)
	}
	if !strings.Contains(out, "--json") {
		t.Errorf("cancel help missing --json\n---\n%s", out)
	}
}

// TestHelpText_Download mentions all download flags.
func TestHelpText_Download(t *testing.T) {
	var buf bytes.Buffer
	printDownloadUsage(&buf)
	out := buf.String()
	for _, want := range []string{
		"download",
		"--output-dir",
		"--keep-zip",
		"--json",
		"job-id",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("download help missing %q\n---\n%s", want, out)
		}
	}
}

// TestHelpText_NoTokenLeak ensures no token environment variable name
// appears twice or carries example values.
func TestHelpText_NoTokenLeak(t *testing.T) {
	var buf bytes.Buffer
	printUsage(&buf)
	out := buf.String()
	// The environment variable name is allowed; literal example token
	// values must not appear.
	for _, banned := range []string{"Bearer ", "secret", "my-token"} {
		if strings.Contains(out, banned) {
			t.Errorf("help text leaks sensitive literal %q\n---\n%s", banned, out)
		}
	}
}
