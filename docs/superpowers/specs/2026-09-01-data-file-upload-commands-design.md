# Data File Upload Commands Design

## Context

The Atlas Core HTTP Service exposes four new authenticated endpoints for
replacing or uploading service data files:

- `POST /data-files/material-db`
- `POST /data-files/reference-db`
- `POST /data-files/risk-db`
- `POST /data-files/special-materials-config`

Each endpoint requires a `multipart/form-data` request with one required file
part named `file`. The successful response is an arbitrary JSON object rather
than a stable schema.

## User-facing commands

Add four top-level commands whose names match the endpoint suffixes:

```text
atlas-ap-remote material-db --file <path>
atlas-ap-remote reference-db --file <path>
atlas-ap-remote risk-db --file <path>
atlas-ap-remote special-materials-config --file <path>
```

Each command supports `--file`, `--json`, and `--help`. `--file` is required.
The commands use the existing `--server`/`--token` flags and environment
fallbacks, send one HTTP request, and return exit code 0 on a 2xx response or
1 for an operational/service error. Flag parsing and missing arguments retain
the existing exit code 2/`MISSING_ARG` conventions.

In human mode, print the returned JSON object as indented JSON to stdout. In
JSON mode, wrap it in the standard CLI envelope:

```json
{"success":true,"response":{}}
```

The response object is preserved without assuming particular keys. Errors use
the existing `reportError` path, including service error codes, HTTP status,
timeout/network classification, and token-safe output.

## Architecture

The client package will expose one reusable method, conceptually:

```go
UploadDataFile(ctx context.Context, endpoint, filePath string) (map[string]any, error)
```

The method validates the file path, opens the local file, creates a
`multipart/form-data` request with the `file` field and original basename,
resolves the endpoint against `BaseURL`, applies bearer authentication, and
decodes the successful JSON object. It reuses the existing transport and
service-error handling. The client owns HTTP protocol details; the CLI owns
command names, endpoint mapping, flags, and output.

The four CLI handlers should share a small common data-file command path rather
than duplicating upload logic. A command-to-endpoint mapping will select the
endpoint while preserving command-specific help text. No retries, polling,
temporary result files, or response-schema assumptions are introduced.

## Testing

Add client tests that verify, for each endpoint mapping (or through a table),
the POST method, exact path, multipart `file` field, transmitted file content,
original filename, Authorization header, successful arbitrary JSON decoding,
missing file validation, and non-2xx error propagation.

Add CLI tests covering all four commands in human and JSON modes, required
`--file`, help output, server/token configuration, arbitrary response fields,
and service/network failures. Update help-text tests and documentation tests
as appropriate.

Update the top-level README and the agent skill usage documentation with the
four commands and their output behavior.

