# Public resource upload commands

## Approved direction

Add three CLI commands and update the repository's Atlas AP Remote skill.
Reuse the existing data-file upload handler and HTTP client. Separate public
resource handlers would duplicate the same protocol without adding behavior.

## API source

Source: https://ap.atlaslabtest.com/openapi.json

Inspected on 2026-09-26: `Atlas Core HTTP Service`, version `v1.0.5`.
This records the version observed during design; the URL is mutable.

| Command | POST endpoint | Fixed server filename |
| --- | --- | --- |
| `public-material-catalog` | `/data-files/public-material-catalog` | `已使用化妆品原料目录.xlsx` |
| `public-onsale-material` | `/data-files/public-onsale-material` | `已上市产品原料使用信息.xlsx` |
| `public-iccsa-material` | `/data-files/public-iccsa-material` | `《国际化妆品安全评估数据索引》.xlsx` |

All three require Bearer authentication and a multipart body with one required
binary part named `file`. The server ignores the uploaded filename and chooses
the target from the endpoint. HTTP 200 returns a JSON object with `file_type`,
`filename`, `size`, and `applied_at`; preserve additional response fields.

The service replaces an existing file atomically. HTTP 200 confirms replacement,
not validation of Excel contents or workbook structure. The default content
limit is 20 MiB, configurable on the service; proxies may impose lower limits.
Do not hardcode that default as a CLI limit. The target directory and file must
already exist. A timeout or disconnect can occur after replacement succeeds.

## CLI behavior and architecture

Each command accepts required `--file <path>`, optional `--json`, and `--help`.
Global server and token flags and environment fallbacks retain their existing
behavior. Example:

```sh
atlas-ap-remote --server "$ATLAS_REMOTE_URL" public-material-catalog \
  --file './已使用化妆品原料目录.xlsx' --json
```

Extend command dispatch to call `cmdDataFile` with the corresponding endpoint.
That handler parses arguments and calls `Client.UploadDataFile`, which builds
the multipart file part, applies authentication, performs one POST, and decodes
the JSON object. No new client method or response type is needed.

Human output remains indented JSON. Machine output retains
`{"success":true,"response":{...}}`. Preserve the existing exit-code and error
mapping behavior, including HTTP status and service `code`/`message` fields.
Relevant API errors include `UNAUTHORIZED`, `DATA_FILE_NOT_FOUND`,
`UPLOAD_TOO_LARGE`, `INVALID_REQUEST`, `http_error`, and `internal_error`.
The shared handler remains responsible for non-JSON remote errors.

The CLI accepts arbitrary source filenames and does not inspect Excel contents.
Each invocation performs one request without automatic retries or polling.

## Skill and documentation

Update `skill/atlas-ap-remote/SKILL.md` and README alongside the CLI.
Extend the skill's description to cover data-file and public-resource uploads,
while preserving its existing job and safety-assessment triggers.

For public-resource requests, require a user-provided local file path. When the
user explicitly selects a resource or command, use that selection. Otherwise,
match the source basename exactly against the fixed filenames in the table.
If no filename matches, ask which of the three resource types the user intends
before uploading. Preserve the supplied path; filename matching guides the
agent and is not a CLI restriction.

Include examples for all three commands using `--json`. Explain that the
operation replaces the selected shared resource, that success does not validate
the workbook, and that a timeout may leave the replacement applied. Report only
the returned result and do not automatically retry an uncertain upload.

Update top-level and command-specific CLI help and the README command list.
The current checkout implements four existing data-file commands; this change
adds three, giving seven. Template-upload designs present in the repository do
not imply those commands are implemented and are outside this change's scope.
This change updates the repository skill, not a separately installed skill copy.

## Validation

Extend existing table-driven client and CLI tests to cover the three mappings:

- Exact POST path, multipart `file` name, original upload basename and bytes,
  and Bearer authentication.
- Human and JSON output preserving the documented response fields and extras.
- Required file argument, help output, configuration fallback, and one request
  per invocation.
- Service errors, including missing target and upload-size rejection, and
  existing timeout/network error behavior without retries.

Use local test servers; validation must not replace live public resources.
Run focused tests, then `go test ./...`, `go vet ./...`, and `git diff --check`.
Review skill routing against explicit resource selection, each standard
filename, and an unmatched filename requiring clarification.

## Scope

No new job behavior, batch uploads, multi-file transactions, server changes,
Excel validation, or unrelated refactoring. Implementation planning begins
after the user reviews this written specification.
