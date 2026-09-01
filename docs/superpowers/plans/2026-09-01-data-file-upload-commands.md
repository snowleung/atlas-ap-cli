# Data File Upload Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add four direct CLI commands that upload a local file to the four Atlas Core data-file endpoints.

**Architecture:** Add one reusable multipart upload method to `internal/client`, returning an arbitrary JSON object. Add one shared CLI handler with a command-to-endpoint mapping for `material-db`, `reference-db`, `risk-db`, and `special-materials-config`; retain existing configuration and error-reporting behavior. Update help and user-facing documentation.

**Tech Stack:** Go standard library (`net/http`, `mime/multipart`, `encoding/json`, `testing`), existing CLI/client test helpers, Markdown documentation.

---

### Task 1: Add the reusable data-file client upload

**Files:**
- Modify: `internal/client/client.go` near `Submit` and the shared request helpers
- Test: `internal/client/client_test.go`

- [ ] **Step 1: Write the failing client contract test**

Add a table-driven `httptest.Server` test that sends a temporary file through
`Client.UploadDataFile` for all four endpoint paths. The handler must assert
`POST`, parse multipart form data, assert the part is named `file`, verify its
filename and content, verify `Authorization: Bearer test-token`, and return an
arbitrary JSON object such as `{"accepted":true,"count":3}`. Assert that each
call returns the decoded keys and no error.

Also add focused tests for an empty file path and a non-2xx JSON service error.
Use the intended signature:

```go
func (c *Client) UploadDataFile(ctx context.Context, endpoint, filePath string) (map[string]any, error)
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run:

```bash
go test ./internal/client -run 'TestClient_UploadDataFile' -count=1
```

Expected: FAIL because `UploadDataFile` does not exist yet.

- [ ] **Step 3: Implement the minimal multipart upload method**

Open `filePath`, build a multipart body with one `file` part using
`filepath.Base(filePath)`, close the multipart writer, resolve `endpoint` with
the existing `buildURL`, create a context-bound POST request, set the
multipart content type, apply bearer auth, and decode the successful response
into `map[string]any` through the existing `do` method. Return descriptive
errors prefixed with `data-file:` for local request construction failures.
Reject an empty endpoint or file path before opening/building the request.
Do not add retries, new error types, or endpoint-specific response structs.

- [ ] **Step 4: Run the focused client tests**

Run:

```bash
go test ./internal/client -run 'TestClient_UploadDataFile' -count=1
```

Expected: PASS, including multipart content, auth, arbitrary JSON, validation,
and service-error cases.

- [ ] **Step 5: Commit the client change**

```bash
git add internal/client/client.go internal/client/client_test.go
git commit -m "feat: add data file upload client"
```

### Task 2: Add the four CLI commands and output behavior

**Files:**
- Modify: `internal/cli/commands.go`
- Test: `internal/cli/commands_test.go`

- [ ] **Step 1: Write failing CLI tests for all command mappings**

Add table-driven tests using the existing `withServer` helper. For each command,
create a temporary file, assert the request is `POST` to its exact endpoint and
has a multipart `file` part, return `{"accepted":true,"kind":"<command>"}`,
and assert exit code 0. Test both:

```text
<command> --file <path> --json
```

with exact envelope fields `success=true` and `response`, and human mode with
indented JSON containing the returned fields. Add a missing-`--file` test,
`--help` test, and service-error test confirming the existing error envelope
behavior.

- [ ] **Step 2: Run the focused CLI tests and verify they fail**

Run:

```bash
go test ./internal/cli -run 'TestRun_(DataFile|MaterialDB|ReferenceDB|RiskDB|SpecialMaterials)' -count=1
```

Expected: FAIL because the new commands are not dispatched.

- [ ] **Step 3: Add command dispatch and shared flags/handler**

Extend `Run` with these cases:

```go
case "material-db":
    return cmdDataFile(gf, environ, "material-db", "/data-files/material-db", subArgs, stdout, stderr)
case "reference-db":
    return cmdDataFile(gf, environ, "reference-db", "/data-files/reference-db", subArgs, stdout, stderr)
case "risk-db":
    return cmdDataFile(gf, environ, "risk-db", "/data-files/risk-db", subArgs, stdout, stderr)
case "special-materials-config":
    return cmdDataFile(gf, environ, "special-materials-config", "/data-files/special-materials-config", subArgs, stdout, stderr)
```

Create shared `dataFileFlags` and parser with `--file`, `--json`, and `--help`.
Implement `cmdDataFile` to validate `--file`, resolve the client, call
`UploadDataFile(context.Background(), endpoint, file)`, and route errors through
`reportError`. Marshal human-mode responses with `json.MarshalIndent` followed
by a newline. JSON mode must write `{"success":true,"response":<object>}`
through the existing `writeJSON` helper. Preserve exit codes and avoid logging
the token or file contents.

- [ ] **Step 4: Add command-specific help output**

Add `printDataFileUsage(w, command)` to `internal/cli/help.go` with the command
name, `--file <path>`, `--json`, and `--help`. Keep the wording explicit that
the command uploads one data file and performs one POST request.

- [ ] **Step 5: Run the focused CLI tests**

Run:

```bash
go test ./internal/cli -run 'TestRun_(DataFile|MaterialDB|ReferenceDB|RiskDB|SpecialMaterials)' -count=1
```

Expected: PASS for all four endpoints, output modes, validation, help, and
service errors.

- [ ] **Step 6: Commit the CLI change**

```bash
git add internal/cli/commands.go internal/cli/help.go internal/cli/commands_test.go
git commit -m "feat: add data file upload commands"
```

### Task 3: Update help and documentation

**Files:**
- Modify: `internal/cli/help.go` top-level command list
- Test: `internal/cli/help_test.go`
- Modify: `README.md`
- Modify: `skill/atlas-ap-remote/SKILL.md`

- [ ] **Step 1: Extend top-level help tests**

Require all four command names and their corresponding `/data-files/...` paths
in the output returned by `printUsage`.

- [ ] **Step 2: Update top-level help**

List the four commands in the `COMMANDS` section and state that each performs
one POST upload. Ensure command-specific help is reachable through each new
command.

- [ ] **Step 3: Document command usage and output**

Add README examples for all four commands, explain that `--file` is a required
multipart upload, and show both ordinary JSON output and the `--json` envelope.
Add the same safe invocation guidance to `skill/atlas-ap-remote/SKILL.md`,
including that the user must provide the file path and that the command does
not poll or retry.

- [ ] **Step 4: Run help tests and commit documentation**

Run:

```bash
go test ./internal/cli -run 'TestHelpText|TestRun_Help' -count=1
```

Expected: PASS with all existing and new help assertions.

```bash
git add internal/cli/help.go internal/cli/help_test.go README.md skill/atlas-ap-remote/SKILL.md
git commit -m "docs: document data file upload commands"
```

### Task 4: Run the complete verification suite

**Files:**
- No source changes expected; inspect the complete diff and test results.

- [ ] **Step 1: Run formatting and static checks**

```bash
gofmt -w internal/client/client.go internal/client/client_test.go internal/cli/commands.go internal/cli/commands_test.go internal/cli/help.go internal/cli/help_test.go
git diff --check
go vet ./...
```

Expected: no formatting diff after `gofmt`, no whitespace errors, and `go vet`
exits 0.

- [ ] **Step 2: Run the complete test suite**

```bash
go test ./... -count=1
```

Expected: PASS for `internal/archive`, `internal/client`, and `internal/cli`.

- [ ] **Step 3: Verify the final scope and status**

```bash
git diff HEAD~3..HEAD --stat
git status --short --branch
```

Confirm the three feature commits contain only the client, CLI, tests, and
documentation changes described above. Preserve the pre-existing untracked
`atlas-ap-remote` artifact; do not add or remove it as part of this feature.

