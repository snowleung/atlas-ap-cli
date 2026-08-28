# atlas-ap-remote

A single-shot command-line client for the Atlas AP Remote service, written
in dependency-free Go and distributable as a single Windows executable
(`atlas-ap-remote.exe`).

The client performs exactly one HTTP request per command and does **not**
poll the service. Pass `--json` to receive a single-line JSON envelope on
stdout; otherwise the client prints human-readable text on stdout and
errors on stderr.

## Features

- **Single binary** — no runtime dependencies, no CGO, no installer.
- **Single request per command** — health, submit, status, download, cancel each
  make exactly one HTTP call.
- **Token safe** — the bearer token is sent only in the Authorization
  header; it is never written to logs, error envelopes, or output files.
- **Idempotent submits** — every `submit` generates a fresh UUIDv4
  idempotency key, so retries from the client never produce duplicates.
- **Safe extraction** — the `download` command enforces a 500 MiB cap
  and rejects ZIP entries that attempt to escape the output directory
  (`../escape.txt`, absolute paths, Windows-style `..\escape.txt`).
- **JSON mode** — every command supports `--json` for scriptable use.

## Installation

### Windows (recommended)

Use PowerShell 5+ on Windows 10/11 or Windows Server 2019+.

#### Precompiled binaries

Download the latest `atlas-ap-remote-windows-amd64.exe` and its optional
SHA256 checksum from the [GitHub Releases page](https://github.com/snowleung/atlas-ap-cli/releases/latest).

Rename the executable if desired, then run it directly or place it on your
`PATH`:

```powershell
Rename-Item .\atlas-ap-remote-windows-amd64.exe atlas-ap-remote.exe
.\atlas-ap-remote.exe --help
```

#### Build from source

```powershell
# from the repository root
.\build.ps1
# -> dist\atlas-ap-remote.exe

# Formal release build with an explicit version
.\build.ps1 -Version 0.2.0
```

To target a different architecture:

```powershell
.\build.ps1 -Arch arm64
```

The script pins `GOOS=windows`, `CGO_ENABLED=0`, uses `-ldflags "-s -w"`
to strip the symbol table, and sets `-trimpath` so the binary contains no
absolute paths. The produced `.exe` runs on any Windows machine without
additional runtime files.

### Cross-compile from macOS or Linux

```bash
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 \
  go build -trimpath -ldflags "-s -w -X github.com/atlas-ap/atlas-ap-remote/internal/cli.Version=0.2.0" \
  -o dist/atlas-ap-remote.exe ./cmd/atlas-ap-remote
```

### Run from source

```bash
go run ./cmd/atlas-ap-remote --help
```

`go run` does not require an installed Go runtime — it shells out to the
`go` binary — but it is intended for development only.

## Agent skills

The repository includes an [Atlas AP Remote Agent skill](skill/atlas-ap-remote/SKILL.md)
that teaches Codex and compatible agents how to submit files, check or cancel
jobs, and download results safely.

From the repository root, install it for Codex with:

```bash
mkdir -p ~/.codex/skills/atlas-ap-remote
cp skill/atlas-ap-remote/SKILL.md ~/.codex/skills/atlas-ap-remote/SKILL.md
```

## Configuration

Two environment variables configure the client when the flags are not
passed. They are intentionally named after the global CLI so shell
completion stays simple.

| Variable             | Purpose                                  |
|----------------------|------------------------------------------|
| `ATLAS_REMOTE_URL`   | Default value for `--server`.            |
| `ATLAS_REMOTE_TOKEN` | Default value for `--token`.             |

> ⚠️ The token grants full access to your Atlas AP Remote account. Treat
> it like a password. Do not paste it into shell histories, source code,
> or bug reports.

Flag precedence: `--server` / `--token` always win over the environment
variables. The server URL is normalized by trimming a single trailing
`/`. The token has only surrounding whitespace stripped; internal
whitespace is preserved.

## Commands

### `health`

Check whether the remote service is alive. It performs one `GET /health`
request and sends the configured bearer token when one is available.

```bash
atlas-ap-remote --server https://api.example.com health
atlas-ap-remote --server https://api.example.com --token "$ATLAS_REMOTE_TOKEN" health --json
```

Human mode:

```text
status=ok
```

JSON mode:

```json
{"status":"ok","success":true}
```

### `submit`

Upload a file to the remote service. Generates a fresh UUIDv4
idempotency key per call.

```bash
atlas-ap-remote --server https://api.example.com submit \
  --file ./image.jpg \
  --cos-type 驻留 \
  --body-parts 全身 \
  --product-name 面霜 \
  --usage-method daily \
  --json
```

Success envelope (with `--json`):

```json
{"success":true,"job_id":"abcd-1234"}
```

Human mode:

```
submitted job_id=abcd-1234
```

### `status`

Read the state of a single job.

```bash
atlas-ap-remote status abcd-1234 --json
```

Success envelope:

```json
{"job_id":"abcd-1234","status":"succeeded","success":true}
```

### `download`

Stream the result ZIP for a completed job, validate it against zip-slip,
and extract it into a local directory. By default the ZIP is discarded
after extraction; pass `--keep-zip` to retain it as `<job-id>.zip` in
the output directory.

```bash
atlas-ap-remote download abcd-1234 \
  --output-dir ./results \
  --keep-zip \
  --json
```

Success envelope:

```json
{
  "extracted_files": ["a.txt","nested/b.txt"],
  "output_dir": "/home/me/results",
  "success": true,
  "zip_path": "/home/me/results/abcd-1234.zip"
}
```

If you do **not** pass `--keep-zip`, `zip_path` is omitted from the
envelope.

### `cancel`

Request cancellation of a running job. The server may still finish the
job if the cancellation arrives after completion.

```bash
atlas-ap-remote cancel abcd-1234 --json
```

Success envelope:

```json
{"job_id":"abcd-1234","status":"cancelled","success":true}
```

## Output and errors

Every command exits with one of three codes:

| Code | Meaning                                                       |
|------|---------------------------------------------------------------|
| `0`  | Success.                                                      |
| `1`  | Failure. The error envelope is on stdout (JSON mode) or stderr (human mode). |
| `2`  | Bad usage. The CLI prints usage information and exits.        |

Common error codes you may see:

| Code                  | When                                                    |
|-----------------------|---------------------------------------------------------|
| `MISSING_SERVER`      | Neither `--server` nor `ATLAS_REMOTE_URL` was provided. |
| `MISSING_ARG`         | A required positional argument was omitted.             |
| `TIMEOUT`             | The HTTP request exceeded the 30-second timeout.        |
| `NETWORK_ERROR`       | The connection failed (DNS, refused, TLS, etc.).        |
| `UNSAFE_ZIP_MEMBER`   | The download archive contains an unsafe path.           |
| `ARCHIVE_TOO_LARGE`   | The download exceeded the 500 MiB cap.                  |
| `IO_ERROR`            | A local filesystem operation failed.                    |
| `INTERNAL_ERROR`      | The server returned non-JSON or unknown error shape.    |
| Server-defined codes  | The server's `code` field is preserved verbatim.        |

JSON error envelope example:

```json
{"code":"TIMEOUT","message":"request timed out","success":false}
```

## Development

The repository has no third-party Go dependencies. Run the tests with:

```bash
go test ./...
```

Format and vet before committing:

```bash
gofmt -w .
go vet ./...
```

The project layout:

```
cmd/atlas-ap-remote/main.go    Entry point — calls cli.Run
internal/cli/                  Flag parsing, env fallback, JSON envelopes
internal/client/               HTTP protocol, multipart upload, error mapping
internal/archive/              Safe ZIP streaming, size cap, path validation
build.ps1                      PowerShell build script for Windows
```
