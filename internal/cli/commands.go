package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/atlas-ap/atlas-ap-remote/internal/archive"
	apiclient "github.com/atlas-ap/atlas-ap-remote/internal/client"
)

// Run parses the CLI arguments, dispatches to the appropriate subcommand,
// and returns a process exit code. Tests drive this directly without
// invoking os.Exit.
func Run(args []string, stdout, stderr io.Writer, environ []string) int {
	gf, fs, err := parseGlobalFlags(args, stderr)
	if err != nil {
		// flag already printed usage to stderr
		return 2
	}

	if gf.help {
		printUsage(stdout)
		return 0
	}
	if gf.version {
		fmt.Fprintf(stdout, "atlas-ap-remote %s\n", Version)
		return 0
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		printUsage(stderr)
		return 2
	}

	subcmd := remaining[0]
	subArgs := remaining[1:]

	switch subcmd {
	case "submit":
		return cmdSubmit(gf, environ, subArgs, stdout, stderr)
	case "status":
		return cmdStatus(gf, environ, subArgs, stdout, stderr)
	case "health":
		return cmdHealth(gf, environ, subArgs, stdout, stderr)
	case "cancel":
		return cmdCancel(gf, environ, subArgs, stdout, stderr)
	case "download":
		return cmdDownload(gf, environ, subArgs, stdout, stderr)
	case "material-db":
		return cmdDataFile(gf, environ, "material-db", "/data-files/material-db", subArgs, stdout, stderr)
	case "reference-db":
		return cmdDataFile(gf, environ, "reference-db", "/data-files/reference-db", subArgs, stdout, stderr)
	case "risk-db":
		return cmdDataFile(gf, environ, "risk-db", "/data-files/risk-db", subArgs, stdout, stderr)
	case "special-materials-config":
		return cmdDataFile(gf, environ, "special-materials-config", "/data-files/special-materials-config", subArgs, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n", subcmd)
		return 2
	}
}

type globalFlags struct {
	serverFlag string
	tokenFlag  string
	help       bool
	version    bool
}

// parseGlobalFlags parses --server, --token, --help, and --version from
// the leading global arguments. Remaining positional arguments (the
// subcommand and its own arguments) are returned separately.
func parseGlobalFlags(args []string, stderr io.Writer) (*globalFlags, *flag.FlagSet, error) {
	fs := flag.NewFlagSet("atlas-ap-remote", flag.ContinueOnError)
	fs.SetOutput(stderr)
	gf := &globalFlags{}
	fs.StringVar(&gf.serverFlag, "server", "", "Base URL of the Atlas AP Remote service (default: $ATLAS_REMOTE_URL)")
	fs.StringVar(&gf.tokenFlag, "token", "", "Bearer token (default: $ATLAS_REMOTE_TOKEN)")
	fs.BoolVar(&gf.help, "help", false, "Show help and exit")
	fs.BoolVar(&gf.version, "version", false, "Print version and exit")

	if err := fs.Parse(args); err != nil {
		return nil, nil, err
	}
	return gf, fs, nil
}

// resolveClient applies the flag/env precedence rules and returns a
// configured Atlas AP Remote client.
func resolveClient(gf *globalFlags, environ []string) (*apiclient.Client, error) {
	cfg, err := ResolveConfig(gf.serverFlag, gf.tokenFlag, environ)
	if err != nil {
		return nil, err
	}
	return apiclient.New(cfg.Server, cfg.Token), nil
}

// reportError writes the error envelope to stdout when in JSON mode,
// otherwise writes a human-readable line to stderr. It always returns 1.
func reportError(err error, stdout, stderr io.Writer, jsonMode bool, defaultCode string, httpStatus int) int {
	code := defaultCode
	message := err.Error()
	http := httpStatus

	var se *apiclient.ServiceError
	if errors.As(err, &se) {
		code = se.Code
		message = se.Message
		http = se.HTTPStatus
	} else if errors.Is(err, apiclient.ErrTimeout) {
		code = "TIMEOUT"
	} else if errors.Is(err, apiclient.ErrNetwork) {
		code = "NETWORK_ERROR"
	}

	if jsonMode {
		_ = WriteErrorJSON(stdout, code, message, http)
	} else {
		_ = WriteErrorHuman(stderr, code, message)
	}
	return 1
}

// writeJSON writes a single-line JSON envelope followed by a newline.
func writeJSON(w io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	if _, err := w.Write([]byte("\n")); err != nil {
		return err
	}
	return nil
}

type submitFlags struct {
	file        string
	cosType     string
	bodyParts   string
	productName string
	usageMethod string
	jsonMode    bool
	help        bool
}

func parseSubmitFlags(args []string, stderr io.Writer) (*submitFlags, *flag.FlagSet, error) {
	fs := flag.NewFlagSet("submit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	sf := &submitFlags{}
	fs.StringVar(&sf.file, "file", "", "Path to the file to submit")
	fs.StringVar(&sf.cosType, "cos-type", "驻留", "Cosmetic type")
	fs.StringVar(&sf.bodyParts, "body-parts", "全身", "Body parts")
	fs.StringVar(&sf.productName, "product-name", "", "Product name")
	fs.StringVar(&sf.usageMethod, "usage-method", "", "Usage method")
	fs.BoolVar(&sf.jsonMode, "json", false, "Emit JSON envelope")
	fs.BoolVar(&sf.help, "help", false, "Show help for submit")

	if err := fs.Parse(args); err != nil {
		return nil, fs, err
	}
	return sf, fs, nil
}

func cmdSubmit(gf *globalFlags, environ []string, args []string, stdout, stderr io.Writer) int {
	sf, _, err := parseSubmitFlags(args, stderr)
	if err != nil {
		return 2
	}
	if sf.help {
		printSubmitUsage(stdout)
		return 0
	}

	client, err := resolveClient(gf, environ)
	if err != nil {
		return reportError(err, stdout, stderr, sf.jsonMode, "MISSING_SERVER", 0)
	}

	if sf.file == "" {
		return reportError(errors.New("--file is required"), stdout, stderr, sf.jsonMode, "MISSING_ARG", 0)
	}

	idemKey, err := apiclient.NewUUIDv4()
	if err != nil {
		return reportError(err, stdout, stderr, sf.jsonMode, "INTERNAL_ERROR", 0)
	}

	resp, err := client.Submit(context.Background(), apiclient.SubmitRequest{
		FilePath:       sf.file,
		CosmeticType:   sf.cosType,
		BodyParts:      sf.bodyParts,
		ProductName:    sf.productName,
		UsageMethod:    sf.usageMethod,
		IdempotencyKey: idemKey,
	})
	if err != nil {
		return reportError(err, stdout, stderr, sf.jsonMode, "INTERNAL_ERROR", 0)
	}

	if sf.jsonMode {
		_ = writeJSON(stdout, map[string]any{"success": true, "job_id": resp.JobID})
	} else {
		fmt.Fprintf(stdout, "submitted job_id=%s\n", resp.JobID)
	}
	return 0
}

type healthFlags struct {
	jsonMode bool
	help     bool
}

func parseHealthFlags(args []string, stderr io.Writer) (*healthFlags, *flag.FlagSet, error) {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	fs.SetOutput(stderr)
	hf := &healthFlags{}
	fs.BoolVar(&hf.jsonMode, "json", false, "Emit JSON envelope")
	fs.BoolVar(&hf.help, "help", false, "Show help for health")

	if err := fs.Parse(args); err != nil {
		return nil, fs, err
	}
	return hf, fs, nil
}

func cmdHealth(gf *globalFlags, environ []string, args []string, stdout, stderr io.Writer) int {
	hf, fs, err := parseHealthFlags(args, stderr)
	if err != nil {
		return 2
	}
	if hf.help {
		printHealthUsage(stdout)
		return 0
	}
	if len(fs.Args()) > 0 {
		return reportError(errors.New("health: unexpected argument"), stdout, stderr, hf.jsonMode, "MISSING_ARG", 0)
	}

	client, err := resolveClient(gf, environ)
	if err != nil {
		return reportError(err, stdout, stderr, hf.jsonMode, "MISSING_SERVER", 0)
	}

	resp, err := client.Health(context.Background())
	if err != nil {
		return reportError(err, stdout, stderr, hf.jsonMode, "INTERNAL_ERROR", 0)
	}

	if hf.jsonMode {
		_ = writeJSON(stdout, map[string]any{
			"success": true,
			"status":  resp.Status,
		})
	} else {
		fmt.Fprintf(stdout, "status=%s\n", resp.Status)
	}
	return 0
}

type statusFlags struct {
	jsonMode bool
	help     bool
}

func parseStatusFlags(args []string, stderr io.Writer) (*statusFlags, *flag.FlagSet, error) {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	sf := &statusFlags{}
	fs.BoolVar(&sf.jsonMode, "json", false, "Emit JSON envelope")
	fs.BoolVar(&sf.help, "help", false, "Show help for status")

	if err := fs.Parse(args); err != nil {
		return nil, fs, err
	}
	return sf, fs, nil
}

func cmdStatus(gf *globalFlags, environ []string, args []string, stdout, stderr io.Writer) int {
	sf, fs, err := parseStatusFlags(args, stderr)
	if err != nil {
		return 2
	}

	positional := fs.Args()
	if len(positional) == 0 && !sf.help {
		if _, err := resolveClient(gf, environ); err != nil {
			return reportError(err, stdout, stderr, sf.jsonMode, "MISSING_SERVER", 0)
		}
		return reportError(errors.New("status: job id required"), stdout, stderr, sf.jsonMode, "MISSING_ARG", 0)
	}
	if sf.help && len(positional) == 0 {
		printStatusUsage(stdout)
		return 0
	}
	jobID := positional[0]
	tail := positional[1:]

	if len(tail) > 0 {
		if err := fs.Parse(tail); err != nil {
			return 2
		}
	}

	if sf.help {
		printStatusUsage(stdout)
		return 0
	}

	client, err := resolveClient(gf, environ)
	if err != nil {
		return reportError(err, stdout, stderr, sf.jsonMode, "MISSING_SERVER", 0)
	}

	resp, err := client.Status(context.Background(), jobID)
	if err != nil {
		return reportError(err, stdout, stderr, sf.jsonMode, "INTERNAL_ERROR", 0)
	}

	if sf.jsonMode {
		_ = writeJSON(stdout, map[string]any{
			"success": true,
			"job_id":  resp.JobID,
			"status":  resp.Status,
		})
	} else {
		fmt.Fprintf(stdout, "job_id=%s status=%s\n", resp.JobID, resp.Status)
	}
	return 0
}

type cancelFlags struct {
	jsonMode bool
	help     bool
}

func parseCancelFlags(args []string, stderr io.Writer) (*cancelFlags, *flag.FlagSet, error) {
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	fs.SetOutput(stderr)
	sf := &cancelFlags{}
	fs.BoolVar(&sf.jsonMode, "json", false, "Emit JSON envelope")
	fs.BoolVar(&sf.help, "help", false, "Show help for cancel")

	if err := fs.Parse(args); err != nil {
		return nil, fs, err
	}
	return sf, fs, nil
}

func cmdCancel(gf *globalFlags, environ []string, args []string, stdout, stderr io.Writer) int {
	sf, fs, err := parseCancelFlags(args, stderr)
	if err != nil {
		return 2
	}

	positional := fs.Args()
	if sf.help && len(positional) == 0 {
		printCancelUsage(stdout)
		return 0
	}
	if len(positional) == 0 {
		if _, err := resolveClient(gf, environ); err != nil {
			return reportError(err, stdout, stderr, sf.jsonMode, "MISSING_SERVER", 0)
		}
		return reportError(errors.New("cancel: job id required"), stdout, stderr, sf.jsonMode, "MISSING_ARG", 0)
	}
	jobID := positional[0]
	tail := positional[1:]

	if len(tail) > 0 {
		if err := fs.Parse(tail); err != nil {
			return 2
		}
	}

	if sf.help {
		printCancelUsage(stdout)
		return 0
	}

	client, err := resolveClient(gf, environ)
	if err != nil {
		return reportError(err, stdout, stderr, sf.jsonMode, "MISSING_SERVER", 0)
	}

	resp, err := client.Cancel(context.Background(), jobID)
	if err != nil {
		return reportError(err, stdout, stderr, sf.jsonMode, "INTERNAL_ERROR", 0)
	}

	if sf.jsonMode {
		_ = writeJSON(stdout, map[string]any{
			"success": true,
			"job_id":  resp.JobID,
			"status":  resp.Status,
		})
	} else {
		fmt.Fprintf(stdout, "cancelled job_id=%s\n", resp.JobID)
	}
	return 0
}

type downloadFlags struct {
	outputDir string
	keepZip   bool
	jsonMode  bool
	help      bool
}

func parseDownloadFlags(args []string, stderr io.Writer) (*downloadFlags, *flag.FlagSet, error) {
	fs := flag.NewFlagSet("download", flag.ContinueOnError)
	fs.SetOutput(stderr)
	df := &downloadFlags{}
	fs.StringVar(&df.outputDir, "output-dir", ".", "Directory to extract results into")
	fs.BoolVar(&df.keepZip, "keep-zip", false, "Keep the downloaded ZIP file as <job-id>.zip")
	fs.BoolVar(&df.jsonMode, "json", false, "Emit JSON envelope")
	fs.BoolVar(&df.help, "help", false, "Show help for download")

	if err := fs.Parse(args); err != nil {
		return nil, fs, err
	}
	return df, fs, nil
}

func cmdDownload(gf *globalFlags, environ []string, args []string, stdout, stderr io.Writer) int {
	df, fs, err := parseDownloadFlags(args, stderr)
	if err != nil {
		return 2
	}

	positional := fs.Args()
	if df.help && len(positional) == 0 {
		printDownloadUsage(stdout)
		return 0
	}
	if len(positional) == 0 {
		if _, err := resolveClient(gf, environ); err != nil {
			return reportError(err, stdout, stderr, df.jsonMode, "MISSING_SERVER", 0)
		}
		return reportError(errors.New("download: job id required"), stdout, stderr, df.jsonMode, "MISSING_ARG", 0)
	}
	jobID := positional[0]
	tail := positional[1:]

	if len(tail) > 0 {
		if err := fs.Parse(tail); err != nil {
			return 2
		}
	}

	if df.help {
		printDownloadUsage(stdout)
		return 0
	}

	client, err := resolveClient(gf, environ)
	if err != nil {
		return reportError(err, stdout, stderr, df.jsonMode, "MISSING_SERVER", 0)
	}

	body, err := client.Download(context.Background(), jobID)
	if err != nil {
		return reportError(err, stdout, stderr, df.jsonMode, "INTERNAL_ERROR", 0)
	}
	defer body.Close()

	res, err := archive.DownloadAndExtract(body, df.outputDir, df.keepZip, jobID)
	if err != nil {
		switch {
		case errors.Is(err, archive.ErrUnsafeZipMember):
			return reportError(err, stdout, stderr, df.jsonMode, "UNSAFE_ZIP_MEMBER", 0)
		case errors.Is(err, archive.ErrArchiveTooLarge):
			return reportError(err, stdout, stderr, df.jsonMode, "ARCHIVE_TOO_LARGE", 0)
		case errors.Is(err, archive.ErrIOError):
			return reportError(err, stdout, stderr, df.jsonMode, "IO_ERROR", 0)
		default:
			return reportError(err, stdout, stderr, df.jsonMode, "INTERNAL_ERROR", 0)
		}
	}

	if df.jsonMode {
		payload := map[string]any{
			"success":         true,
			"output_dir":      res.OutputDir,
			"extracted_files": res.ExtractedFiles,
		}
		if res.ZipPath != "" {
			payload["zip_path"] = res.ZipPath
		}
		_ = writeJSON(stdout, payload)
	} else {
		fmt.Fprintf(stdout, "extracted %d files into %s\n", len(res.ExtractedFiles), res.OutputDir)
		if res.ZipPath != "" {
			fmt.Fprintf(stdout, "zip retained at %s\n", res.ZipPath)
		}
	}
	return 0
}

type dataFileFlags struct {
	file     string
	jsonMode bool
	help     bool
}

func parseDataFileFlags(args []string, stderr io.Writer) (*dataFileFlags, *flag.FlagSet, error) {
	fs := flag.NewFlagSet("data-file", flag.ContinueOnError)
	fs.SetOutput(stderr)
	df := &dataFileFlags{}
	fs.StringVar(&df.file, "file", "", "Path to the file to upload")
	fs.BoolVar(&df.jsonMode, "json", false, "Emit JSON envelope")
	fs.BoolVar(&df.help, "help", false, "Show help for this command")

	if err := fs.Parse(args); err != nil {
		return nil, fs, err
	}
	return df, fs, nil
}

func cmdDataFile(gf *globalFlags, environ []string, command, endpoint string, args []string, stdout, stderr io.Writer) int {
	df, _, err := parseDataFileFlags(args, stderr)
	if err != nil {
		return 2
	}
	if df.help {
		printDataFileUsage(stdout, command)
		return 0
	}

	client, err := resolveClient(gf, environ)
	if err != nil {
		return reportError(err, stdout, stderr, df.jsonMode, "MISSING_SERVER", 0)
	}

	if df.file == "" {
		return reportError(errors.New("--file is required"), stdout, stderr, df.jsonMode, "MISSING_ARG", 0)
	}

	resp, err := client.UploadDataFile(context.Background(), endpoint, df.file)
	if err != nil {
		return reportError(err, stdout, stderr, df.jsonMode, "INTERNAL_ERROR", 0)
	}

	if df.jsonMode {
		_ = writeJSON(stdout, map[string]any{"success": true, "response": resp})
	} else {
		data, err := json.MarshalIndent(resp, "", "  ")
		if err != nil {
			return reportError(err, stdout, stderr, false, "INTERNAL_ERROR", 0)
		}
		fmt.Fprintf(stdout, "%s\n", data)
	}
	return 0
}
