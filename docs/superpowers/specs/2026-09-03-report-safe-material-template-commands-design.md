# Report and Safe Material Template Upload Commands Design

## Context

The Atlas Core HTTP Service exposes two additional authenticated data-file
upload endpoints:

- `POST /data-files/report-template`
- `POST /data-files/safe-material-template`

The service OpenAPI document specifies that both endpoints require a
`multipart/form-data` body with one required file part named `file`. A
successful request returns an arbitrary JSON object with HTTP status 200.

The CLI already supports four endpoints with the same protocol through the
shared `Client.UploadDataFile` method and `cmdDataFile` handler. This change
extends that existing behavior without introducing a second upload path.

## User-facing commands

Add two top-level commands whose names match the endpoint suffixes:

```text
atlas-ap-remote report-template --file <path>
atlas-ap-remote safe-material-template --file <path>
```

Each command supports `--file`, `--json`, and `--help`. The `--file` flag is
required. Global `--server` and `--token` flags and their environment-variable
fallbacks continue to work unchanged.

Each invocation performs exactly one multipart POST. A 2xx response exits 0;
an operational or service error exits 1; invalid flags or invocation syntax
exit 2 according to existing CLI conventions.

Human mode prints the returned object as indented JSON. JSON mode wraps the
object in the existing success envelope:

```json
{"success":true,"response":{}}
```

Errors continue through the shared reporting path, preserving service error
codes, HTTP status, timeout and network classifications, and token-safe output.

## Architecture and data flow

Extend command dispatch with these mappings:

| Command | Endpoint |
| --- | --- |
| `report-template` | `/data-files/report-template` |
| `safe-material-template` | `/data-files/safe-material-template` |

Both mappings call the existing shared `cmdDataFile` handler. That handler
parses and validates flags, resolves configuration, then calls
`Client.UploadDataFile`. The client opens the local file, constructs the
multipart `file` part using the original basename, applies bearer
authentication, sends the POST, and decodes the arbitrary JSON response.

No new client method, response type, retry, polling, temporary-file behavior,
file-extension restriction, or response-schema assumption is added.

## Testing

Use the existing table-driven tests as the primary contract:

- Add both endpoint paths to the client upload table, verifying HTTP method,
  exact path, multipart field, filename, content, Authorization header, and
  arbitrary JSON decoding.
- Add both command/path pairs to the CLI upload table, covering human and JSON
  output through the existing shared cases.
- Add both command names and endpoint paths to top-level and command-specific
  help tests.
- Keep the existing missing-file and service-error tests on the shared handler;
  they already cover behavior used by all mapped commands.

Run the focused tests first during each red-green cycle, then run formatting,
`git diff --check`, `go vet ./...`, and `go test ./... -count=1` before declaring
the work complete.

## Agent skill integration

Treat `skill/atlas-ap-remote/SKILL.md` as a first-class deliverable of this
change. Update its frontmatter description so requests to upload Atlas data
files or templates reliably activate the skill, while retaining the existing
job and safety-assessment triggers.

In the skill's data-file routing section:

- Change the supported-command count from four to six.
- Add `report-template` and `safe-material-template` to the supported command
  list, using the same names exposed by the CLI.
- Add this deterministic filename-to-command routing table:

  | Source filename | CLI command | Endpoint |
  | --- | --- | --- |
  | `模板文件_勿删.docx` | `report-template` | `/data-files/report-template` |
  | `配方的成分安全评估模板_勿删.docx` | `safe-material-template` | `/data-files/safe-material-template` |

- Add copyable invocations for both commands using those filenames, the
  required `--file <path>`, and the existing default `--json` guidance.
- Require a user-provided local file path before invoking either command.
- Select the command from the source file's basename according to the table,
  pass the path through unchanged, and report a mismatch instead of guessing
  when neither basename matches.
- Preserve the instruction to make exactly one POST, avoid polling and
  automatic retries, and report only response fields returned by the server.

The skill remains a concise routing and safety guide. It does not duplicate
the OpenAPI schema. The exact basenames above guide agent routing only; the CLI
does not reject other names, infer a template type from file contents, or
rename the uploaded file.

### CLI update workflow

The skill also handles requests to update or upgrade the installed
`atlas-ap-remote` CLI. When that intent is present, the agent must first read
the current [Agent skills](https://github.com/snowleung/atlas-ap-cli#agent-skills)
section rather than relying on installation details cached in the skill.

The agent follows the page's current link to the latest GitHub release,
compares that release with `atlas-ap-remote --version`, and avoids reinstalling
when the installed version is already current. When an update is needed, it
uses the platform asset and checksum named by the current instructions,
verifies SHA256 before replacement, installs the binary on `PATH`, and refreshes
the installed skill from the repository instructions. It then runs
`atlas-ap-remote --version` again and reports the verified installed version.

The skill must not hardcode a release number or treat its own installation
examples as authoritative update metadata. Downloading and replacing an
installed executable or skill remains an external mutation that requires the
usual user authorization at execution time.

## Other documentation

Update the README so its feature summary, data-file upload list, and examples
include both commands. State that the user provides the file path and that
each command performs one request without polling or retrying.
