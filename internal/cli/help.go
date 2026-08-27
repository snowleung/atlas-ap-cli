package cli

import (
	"fmt"
	"io"
)

// printUsage writes the top-level help text to w. It describes all four
// commands, the environment-variable fallbacks, the global flags, and
// the single-request (no polling) contract.
func printUsage(w io.Writer) {
	fmt.Fprintln(w, "atlas-ap-remote — single-shot CLI for the Atlas AP Remote service")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage")
	fmt.Fprintln(w, "  atlas-ap-remote [--server URL] [--token TOKEN] [--help|--version] <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "GLOBAL OPTIONS")
	fmt.Fprintln(w, "  --server URL   Base URL of the Atlas AP Remote service.")
	fmt.Fprintln(w, "                 Falls back to $ATLAS_REMOTE_URL when omitted.")
	fmt.Fprintln(w, "  --token TOKEN  bearer token sent in the Authorization header.")
	fmt.Fprintln(w, "                 Falls back to $ATLAS_REMOTE_TOKEN when omitted.")
	fmt.Fprintln(w, "  --help         Print this help text and exit.")
	fmt.Fprintln(w, "  --version      Print the version string and exit.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "ENVIRONMENT")
	fmt.Fprintln(w, "  ATLAS_REMOTE_URL    Default base URL when --server is not given.")
	fmt.Fprintln(w, "  ATLAS_REMOTE_TOKEN  Default bearer token when --token is not given.")
	fmt.Fprintln(w, "  The token is used only for the Authorization header; it is never")
	fmt.Fprintln(w, "  written to logs, error messages, or output files.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "COMMANDS")
	fmt.Fprintln(w, "  submit    Upload a file to the remote service (one POST /jobs).")
	fmt.Fprintln(w, "  status    Read the state of a job (one GET /jobs/{id}).")
	fmt.Fprintln(w, "  download  Stream results for a completed job (one GET /jobs/{id}/download).")
	fmt.Fprintln(w, "  cancel    Request cancellation of a running job (one POST /jobs/{id}/cancel).")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "OUTPUT")
	fmt.Fprintln(w, "  Each command performs exactly one HTTP request. The CLI does not poll")
	fmt.Fprintln(w, "  the service; pass --json to receive a single-line JSON envelope on stdout.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Use --help with any command for command-specific options.")
}

func printSubmitUsage(w io.Writer) {
	fmt.Fprintln(w, "submit — upload a file to the remote service")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage")
	fmt.Fprintln(w, "  atlas-ap-remote submit --file <path>")
	fmt.Fprintln(w, "         [--cos-type 驻留]")
	fmt.Fprintln(w, "         [--body-parts 全身]")
	fmt.Fprintln(w, "         [--product-name NAME]")
	fmt.Fprintln(w, "         [--usage-method METHOD]")
	fmt.Fprintln(w, "         [--json]")
	fmt.Fprintln(w, "         [--help]")
}

func printStatusUsage(w io.Writer) {
	fmt.Fprintln(w, "status — read the state of a job")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage")
	fmt.Fprintln(w, "  atlas-ap-remote status <job-id> [--json] [--help]")
}

func printCancelUsage(w io.Writer) {
	fmt.Fprintln(w, "cancel — request cancellation of a running job")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage")
	fmt.Fprintln(w, "  atlas-ap-remote cancel <job-id> [--json] [--help]")
}

func printDownloadUsage(w io.Writer) {
	fmt.Fprintln(w, "download — stream results for a completed job")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage")
	fmt.Fprintln(w, "  atlas-ap-remote download <job-id>")
	fmt.Fprintln(w, "         [--output-dir DIR]")
	fmt.Fprintln(w, "         [--keep-zip]")
	fmt.Fprintln(w, "         [--json]")
	fmt.Fprintln(w, "         [--help]")
}
