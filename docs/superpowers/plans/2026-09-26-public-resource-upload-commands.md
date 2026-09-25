# Public Resource Upload Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [x]`) syntax for tracking.

**Goal:** Support three public-resource uploads in the CLI and its repository skill.

**Architecture:** Extend existing command dispatch and reuse `cmdDataFile` and `Client.UploadDataFile`. Keep HTTP behavior, output envelopes, and error handling shared. Skill routing selects a resource from explicit intent or an exact standard filename.

**Tech Stack:** Go standard library, httptest, Markdown.

---

## Context and file responsibilities

Approved spec: `docs/superpowers/specs/2026-09-26-public-resource-upload-commands-design.md`.
API: https://ap.atlaslabtest.com/openapi.json, Atlas Core HTTP Service v1.0.5, inspected 2026-09-26.
Repository root: `/Users/hans/Desktop/workspace/atlas-ap-cli`.
Paths below are relative to the implementation checkout. Create an isolated feature worktree before implementation; the plan itself is saved alongside the approved spec in the current checkout.

- `internal/cli/commands.go`: command-to-endpoint dispatch.
- `internal/cli/help.go`: discoverability and usage.
- `internal/cli/commands_test.go`: CLI integration and output contracts.
- `internal/cli/help_test.go`: help content.
- `internal/client/client_test.go`: multipart protocol coverage; client production code needs no change.
- `README.md`: user-facing commands and API provenance.
- `skill/atlas-ap-remote/SKILL.md`: agent selection rules and examples.

The checkout currently implements four data-file commands. These three bring the total to seven; do not implement the separately designed template commands. Leave unrelated `.workbuddy/` files untouched. Use only local test servers, never POST to the live service during validation. The module declares Go 1.27.0: use an available compatible toolchain and report an unavailable toolchain rather than changing the module version.

## Task 1: Extend upload and help contracts

**Modify:** `internal/cli/commands_test.go`, `internal/cli/help_test.go`, `internal/client/client_test.go`.

- [x] Add these rows to `TestRun_DataFileCommands`:

```go
{"public-material-catalog", "/data-files/public-material-catalog"},
{"public-onsale-material", "/data-files/public-onsale-material"},
{"public-iccsa-material", "/data-files/public-iccsa-material"},
```

- [x] Add these entries to `TestUploadDataFile_SendsMultipartToAllEndpoints`'s endpoint list:

```go
"/data-files/public-material-catalog",
"/data-files/public-onsale-material",
"/data-files/public-iccsa-material",
```

This existing test checks the method, path, authentication, file field, basename, bytes, and arbitrary response fields. The generic client already supports these paths, so these client cases should pass before production changes.

- [x] Add the following expected strings to `TestHelpText_TopLevelDataFiles` and replace its stale “four” comment with “supported”:

```go
"public-material-catalog", "/data-files/public-material-catalog",
"public-onsale-material", "/data-files/public-onsale-material",
"public-iccsa-material", "/data-files/public-iccsa-material",
```

Append the following strings to the command list in `TestHelpText_DataFile`:

```go
"public-material-catalog", "public-onsale-material", "public-iccsa-material",
```

- [x] Add this integration test to `internal/cli/commands_test.go`; add `sync/atomic` to its imports. It verifies all public commands, environment configuration, required file/help handling, documented response fields, extra fields, and service errors with exactly one request.

```go
func TestRun_PublicResourceContract(t *testing.T) {
    resources := []struct{ command, filename string }{
        {"public-material-catalog", "已使用化妆品原料目录.xlsx"},
        {"public-onsale-material", "已上市产品原料使用信息.xlsx"},
        {"public-iccsa-material", "《国际化妆品安全评估数据索引》.xlsx"},
    }
    for _, resource := range resources {
        t.Run(resource.command, func(t *testing.T) {
            var out, errOut bytes.Buffer
            if code := Run([]string{resource.command, "--help"}, &out, &errOut, nil); code != 0 || !strings.Contains(out.String(), "--file") {
                t.Fatalf("help: code=%d stdout=%s stderr=%s", code, &out, &errOut)
            }
            for _, jsonMode := range []bool{false, true} {
                for _, status := range []int{200, 404, 413} {
                    var requests atomic.Int32
                    errorCode := "DATA_FILE_NOT_FOUND"
                    if status == 413 { errorCode = "UPLOAD_TOO_LARGE" }
                    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                        requests.Add(1)
                        if r.Method != http.MethodPost || r.URL.Path != "/data-files/"+resource.command {
                            t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
                        }
                        if r.Header.Get("Authorization") != "Bearer test-token" { t.Error("missing auth") }
                        w.Header().Set("Content-Type", "application/json")
                        w.WriteHeader(status)
                        if status != 200 {
                            _ = json.NewEncoder(w).Encode(map[string]any{"code": errorCode, "message": "upload rejected"})
                            return
                        }
                        _ = json.NewEncoder(w).Encode(map[string]any{
                            "file_type": resource.command, "filename": resource.filename,
                            "size": 7, "applied_at": "2026-09-26T12:00:00+08:00", "extra": true,
                        })
                    }))
                    environ := []string{"ATLAS_REMOTE_URL="+srv.URL, "ATLAS_REMOTE_TOKEN=test-token"}
                    out.Reset(); errOut.Reset()
                    code := Run([]string{resource.command, "--json"}, &out, &errOut, environ)
                    if code != 1 || !strings.Contains(out.String(), "MISSING_ARG") || requests.Load() != 0 {
                        t.Fatalf("missing file: code=%d output=%s requests=%d", code, &out, requests.Load())
                    }
                    args := []string{resource.command, "--file", writeTempFile(t, "payload")}
                    if jsonMode { args = append(args, "--json") }
                    out.Reset(); errOut.Reset()
                    code = Run(args, &out, &errOut, environ)
                    srv.Close()
                    if requests.Load() != 1 { t.Fatalf("requests=%d", requests.Load()) }
                    if status != 200 {
                        if code != 1 || !strings.Contains(out.String()+errOut.String(), errorCode) {
                            t.Fatalf("error: code=%d stdout=%s stderr=%s", code, &out, &errOut)
                        }
                        if jsonMode {
                            var result map[string]any
                            if err := json.Unmarshal(out.Bytes(), &result); err != nil { t.Fatal(err) }
                            if result["http_status"] != float64(status) || result["success"] != false || result["message"] != "upload rejected" {
                                t.Fatalf("error envelope: %v", result)
                            }
                        }
                        continue
                    }
                    if code != 0 { t.Fatalf("code=%d stderr=%s", code, &errOut) }
                    var result map[string]any
                    if err := json.Unmarshal(out.Bytes(), &result); err != nil { t.Fatal(err) }
                    if jsonMode {
                        if result["success"] != true { t.Fatalf("envelope: %v", result) }
                        response, ok := result["response"].(map[string]any)
                        if !ok { t.Fatalf("missing response: %v", result) }
                        result = response
                    }
                    if result["file_type"] != resource.command || result["filename"] != resource.filename || result["size"] != float64(7) || result["applied_at"] != "2026-09-26T12:00:00+08:00" || result["extra"] != true {
                        t.Fatalf("response: %v", result)
                    }
                }
            }
        })
    }
}
```

- [x] Run `go test ./internal/cli ./internal/client -run 'DataFile|PublicResource' -count=1`. Expect CLI failures for unknown public commands and missing top-level help; existing client cases pass. Resolve test compilation errors before interpreting a failure as the intended red result.

## Task 2: Implement dispatch and help

**Modify:** `internal/cli/commands.go`, `internal/cli/help.go`.

- [x] Add these cases before the command dispatch default:

```go
case "public-material-catalog":
    return cmdDataFile(gf, environ, "public-material-catalog", "/data-files/public-material-catalog", subArgs, stdout, stderr)
case "public-onsale-material":
    return cmdDataFile(gf, environ, "public-onsale-material", "/data-files/public-onsale-material", subArgs, stdout, stderr)
case "public-iccsa-material":
    return cmdDataFile(gf, environ, "public-iccsa-material", "/data-files/public-iccsa-material", subArgs, stdout, stderr)
```

- [x] Add top-level help lines alongside existing data-file commands:

```go
fmt.Fprintln(w, "  public-material-catalog")
fmt.Fprintln(w, "                 Upload a public resource (one POST /data-files/public-material-catalog).")
fmt.Fprintln(w, "  public-onsale-material")
fmt.Fprintln(w, "                 Upload a public resource (one POST /data-files/public-onsale-material).")
fmt.Fprintln(w, "  public-iccsa-material")
fmt.Fprintln(w, "                 Upload a public resource (one POST /data-files/public-iccsa-material).")
```

Command-specific help is already provided by `cmdDataFile`; no second parser or client method is needed. Preserve the current one-request behavior and error mapping, including missing-file exit code 1, invalid syntax exit code 2, and arbitrary filenames.

- [x] Run `gofmt -w internal/cli/commands.go internal/cli/help.go internal/cli/commands_test.go internal/cli/help_test.go internal/client/client_test.go`.
- [x] Run `go test ./internal/cli ./internal/client -run 'DataFile|PublicResource' -count=1`. Expect PASS.
- [x] Commit only these five files with message `feat: support public resource uploads`.

## Task 3: Document commands and agent routing

**Modify:** `README.md`, `skill/atlas-ap-remote/SKILL.md`.

- [x] Expand README's data-file feature summary and upload list to include all seven commands. Add this subsection under its data-file usage section:

```markdown
### Public resources

These commands follow Atlas Core HTTP Service v1.0.5's
[OpenAPI description](https://ap.atlaslabtest.com/openapi.json), inspected on
2026-09-26. Each uses Bearer authentication and one multipart POST with a
required `file` part.

| Command | Endpoint | Fixed server file |
| --- | --- | --- |
| `public-material-catalog` | `/data-files/public-material-catalog` | `已使用化妆品原料目录.xlsx` |
| `public-onsale-material` | `/data-files/public-onsale-material` | `已上市产品原料使用信息.xlsx` |
| `public-iccsa-material` | `/data-files/public-iccsa-material` | `《国际化妆品安全评估数据索引》.xlsx` |

The endpoint selects the resource; the local filename can be arbitrary.
Uploads replace an existing shared resource. The target file must already
exist. The server's default content limit is 20 MiB and is configurable;
proxies may impose lower limits. Supply a complete compatible workbook.
Success confirms replacement, not validation of Excel contents. The response
includes `file_type`, `filename`, `size`, and `applied_at`, preserved inside the
CLI's `response` object in JSON mode. A timeout may occur after replacement;
the CLI does not automatically retry or poll.
```

Add these executable examples in a shell code block:

```sh
atlas-ap-remote --server "$ATLAS_REMOTE_URL" public-material-catalog --file './已使用化妆品原料目录.xlsx' --json
atlas-ap-remote --server "$ATLAS_REMOTE_URL" public-onsale-material --file './已上市产品原料使用信息.xlsx' --json
atlas-ap-remote --server "$ATLAS_REMOTE_URL" public-iccsa-material --file './《国际化妆品安全评估数据索引》.xlsx' --json
```

- [x] Update skill frontmatter description to:

```yaml
description: Use the Atlas AP Remote CLI to submit, inspect, cancel, or download jobs, upload data files or public resources, or generate safety assessments (安评) from a user-provided recipe file.
```

Update its opening applicability sentence to include data-file and public-resource uploads. Change the data-file paragraph from four to seven commands, appending the three public commands. Add this routing subsection and the same three executable examples above:

```markdown
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
```

- [x] Review these skill scenarios manually: each of the three standard filenames selects its command; `new.xlsx` without a resource asks for a resource; `new.xlsx` with explicit catalog intent uses catalog; a missing path asks for the local file; a timeout produces no automatic retry. Keep the existing assessment and job rules intact.
- [x] Run `git diff --check`, review README and skill for stale command counts, and commit these two files with message `docs: explain public resource uploads and skill routing`.

## Task 4: Verify and deliver

- [x] Run `go test ./... -count=1` and `go vet ./...`. Expect both to exit successfully. Existing shared timeout/network tests cover unchanged transport behavior.
- [x] Run `git diff --check` and inspect `git status --short` for unintended changes.
- [x] Run `go run ./cmd/atlas-ap-remote --help` and `go run ./cmd/atlas-ap-remote public-material-catalog --help`. Expect the new command list and required file usage without a server request.
- [x] Review the branch diff against the approved spec: three mappings, shared upload behavior, output/error preservation, help, README provenance, skill routing, and test coverage.
- [x] Report implementation location, verification results, and any actual limitations. Do not imply the separately installed skill or a released CLI binary has been updated. No publication or live upload is part of this plan.

## Execution results

Implemented on `codex/public-resource-upload`. Baseline tests passed. New CLI
tests failed on unknown commands before implementation and passed afterward.
Full `go test ./... -count=1`, `go vet ./...`, help smoke checks, and
`git diff --check` passed. Tests required sandbox escalation for local ports
and Go cache access; no live upload was performed. Skill routing was manually
reviewed for the scenarios above, consistent with the requested inline execution.
The repository skill is updated; separately installed copies remain unchanged.

Before PR creation, merged `origin/main` at `ce7d928`. Upstream had added the
report-template and safe-material-template commands. Resolved overlapping
command lists, tests, help, README, and skill text by retaining both sets of
commands and the upstream installation/update guidance. The integrated result
supports nine data-file commands. Post-merge tests, vet, and build passed.
