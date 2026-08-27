// Command atlas-ap-remote is the command-line client for the Atlas AP
// Remote service. See the project README for full usage; the high-level
// commands are submit, status, download, and cancel.
package main

import (
	"os"

	"github.com/atlas-ap/atlas-ap-remote/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr, os.Environ()))
}
