package cli

import (
	"fmt"
	"io"
)

// printUsage writes the top-level help text to w. The full content is
// expanded in Task 4 (help_test.go); this stub keeps the dispatcher
// compilable so the rest of Task 2 can be validated.
func printUsage(w io.Writer) {
	fmt.Fprintln(w, "atlas-ap-remote — single-shot CLI for the Atlas AP Remote service")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  atlas-ap-remote [--server URL] [--token TOKEN] <command> [args]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  submit    Upload a file to the remote service")
	fmt.Fprintln(w, "  status    Read the state of a job")
	fmt.Fprintln(w, "  download  Stream results for a completed job")
	fmt.Fprintln(w, "  cancel    Request cancellation of a running job")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Use --help with any command for command-specific help.")
}

func printSubmitUsage(w io.Writer) {
	fmt.Fprintln(w, "submit --file <path> [--cos-type ...] [--body-parts ...] [--product-name ...] [--usage-method ...] [--json]")
}

func printStatusUsage(w io.Writer) {
	fmt.Fprintln(w, "status <job-id> [--json]")
}

func printCancelUsage(w io.Writer) {
	fmt.Fprintln(w, "cancel <job-id> [--json]")
}