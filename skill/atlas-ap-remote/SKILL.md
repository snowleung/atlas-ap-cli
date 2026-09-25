---
name: atlas-ap-remote
description: Use when the user wants to submit, inspect, cancel, or download Atlas AP Remote jobs, upload data files or public resources, or generate safety assessments (安评).
metadata:
  short-description: Operate Atlas jobs and upload data files
---

# Atlas AP Remote CLI

Use this skill when the user asks to operate Atlas AP Remote, upload data files or public resources, or generate an 安评/安全评估/安评报告 through this CLI.

## Hard gate for 安评

Before generating an 安评, verify that the user has supplied a local recipe/formula file that can be passed to `--file`. A text description of a formula is not a substitute for the required file.

If no recipe file is available, stop and tell the user:

> 生成安评必须提供配方文件，请上传配方文件后再继续。

Do not call `submit` until the file exists and the user has supplied or confirmed the required product metadata. Ask for missing metadata one item at a time as needed:

- `--cos-type`
- `--body-parts`
- `--product-name`
- `--usage-method`

## Configuration and safety

Use the installed `atlas-ap-remote` executable when available. When working from this repository during development, `go run ./cmd/atlas-ap-remote` is an acceptable fallback.

Global options must appear before the subcommand:

```text
atlas-ap-remote --server <url> [--token <token>] <command> ...
```

Resolve configuration in this order:

1. Explicit `--server` and `--token` flags.
2. `ATLAS_REMOTE_URL` and `ATLAS_REMOTE_TOKEN` environment variables.

Prefer the environment variable for the token to reduce shell-history exposure. Never print, log, echo, or include the token in a generated report. The server URL may have one trailing `/` removed; token whitespace handling belongs to the CLI.

Each command makes exactly one HTTP request. Do not implement polling, loops, or background retries. If the user wants to wait for completion, explain that they must request a later `status` check.

Use `--json` by default when the result will be interpreted by Codex or another script. JSON success output contains `"success":true`; JSON failures contain `"success":false` and a `code`/`message`. Exit codes are:

- `0`: success
- `1`: command or remote failure
- `2`: invalid usage; show or preserve the CLI usage output

## Command routing

### Submit a file

Use this for uploading a recipe file to start an assessment or another remote job. Keep the user-provided path unchanged unless it is relative to the current working directory and needs resolution.

```bash
atlas-ap-remote --server "$ATLAS_REMOTE_URL" submit \
  --file <recipe-file> \
  --cos-type <value> \
  --body-parts <value> \
  --product-name <value> \
  --usage-method <value> \
  --json
```

For 安评, report the returned `job_id` clearly. Do not claim that the assessment is finished merely because submission succeeded.

### Check a job

Use one request when the user asks for the current job state:

```bash
atlas-ap-remote --server <url> status <job-id> --json
```

### Download results

Only download when the user asks for the result or explicitly authorizes it. Require a concrete output directory and create/use that directory according to the user's request. The CLI streams and extracts the ZIP, caps the download at 500 MiB, and rejects unsafe paths; do not bypass these checks or manually extract the response.

```bash
atlas-ap-remote --server <url> download <job-id> \
  --output-dir <directory> [--keep-zip] --json
```

Tell the user where files were extracted. Mention the ZIP path only when `--keep-zip` was used.

### Upload a data file

Seven commands upload a single local data file to its dedicated Atlas Core
endpoint: `material-db`, `reference-db`, `risk-db`,
`special-materials-config`, `public-material-catalog`,
`public-onsale-material`, and `public-iccsa-material`. Each sends one multipart
POST with a required `file` part; the user must provide the file path.

```bash
atlas-ap-remote --server "$ATLAS_REMOTE_URL" material-db --file <path> --json
atlas-ap-remote --server "$ATLAS_REMOTE_URL" reference-db --file <path> --json
atlas-ap-remote --server "$ATLAS_REMOTE_URL" risk-db --file <path> --json
atlas-ap-remote --server "$ATLAS_REMOTE_URL" special-materials-config --file <path> --json
```

The command does not poll or retry; it performs exactly one POST request.
The response is an arbitrary JSON object, reported verbatim (in `--json`
mode as `{"success":true,"response":{...}}`). Do not assume particular
response keys or invent upload results beyond what the server returned.

#### Select a public resource

Require a user-provided local file path. Use the resource or command explicitly
selected by the user. Otherwise match the source basename exactly:

| Source basename | Command |
| --- | --- |
| `已使用化妆品原料目录.xlsx` | `public-material-catalog` |
| `已上市产品原料使用信息.xlsx` | `public-onsale-material` |
| `《国际化妆品安全评估数据索引》.xlsx` | `public-iccsa-material` |

When no basename matches and the user has not selected a resource, ask which
resource they intend before uploading. Pass the provided path unchanged.
This routing rule selects the command; the CLI accepts arbitrary filenames.

These uploads replace the selected shared resource. Report the server's
returned fields; success confirms replacement, not workbook validation.
A timeout or disconnect may leave the replacement applied. Do not automatically
retry; explain the uncertain result and let the user decide the next action.

```bash
atlas-ap-remote --server "$ATLAS_REMOTE_URL" public-material-catalog --file './已使用化妆品原料目录.xlsx' --json
atlas-ap-remote --server "$ATLAS_REMOTE_URL" public-onsale-material --file './已上市产品原料使用信息.xlsx' --json
atlas-ap-remote --server "$ATLAS_REMOTE_URL" public-iccsa-material --file './《国际化妆品安全评估数据索引》.xlsx' --json
```

### Cancel a job

Use one request when the user asks to cancel a job:

```bash
atlas-ap-remote --server <url> cancel <job-id> --json
```

Cancellation can race with completion; report the server's returned status without assuming cancellation succeeded before reading the response.

## Error handling

On failure, preserve the CLI's error code and message, then give the smallest useful next action:

- `MISSING_SERVER`: ask for the server URL or `ATLAS_REMOTE_URL`.
- `MISSING_ARG`: ask for the missing required argument.
- `TIMEOUT`: let the user retry explicitly; do not retry automatically.
- `NETWORK_ERROR`: check URL, connectivity, TLS, and credentials.
- `IO_ERROR`: check the recipe path or output-directory permissions.
- `UNSAFE_ZIP_MEMBER` / `ARCHIVE_TOO_LARGE`: stop and report that the result was rejected by the safety checks.
- `INTERNAL_ERROR` or a server-defined code: show the returned message without exposing headers or tokens.

Do not invent job IDs, server responses, completed assessment content, or unsupported CLI flags.
