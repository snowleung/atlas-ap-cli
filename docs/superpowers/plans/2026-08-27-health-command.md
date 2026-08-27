# Health Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `health` subcommand that calls `GET /health`, reports the server status, and preserves existing authentication and error behavior.

**Architecture:** Add `Client.Health` beside the existing HTTP methods and route a new CLI command through the existing configuration, output envelope, and error mapping layers. No TLS behavior changes, retries, polling, or additional endpoints.

**Tech Stack:** Go standard library, `net/http`, `httptest`, existing `flag` CLI and JSON output helpers.

---

### Task 1: Add the HTTP client health method

**Files:**
- Modify: `internal/client/client.go`
- Modify: `internal/client/client_test.go`

- [ ] **Step 1: Write the failing client test.** Add an `httptest.Server` test that returns `{"status":"ok"}`, calls `New(server.URL, "token").Health(context.Background())`, and asserts one `GET /health` request with `Authorization: Bearer token` and response status `ok`.

- [ ] **Step 2: Add the no-token assertion.** In the same client test file, add a test server that fails if an Authorization header is present; call `New(server.URL, "").Health(...)` and assert the status is `ok`.

- [ ] **Step 3: Run the focused tests and verify failure.** Run `go test ./internal/client -run 'TestHealth' -count=1`; expect compilation failure because `Health` and its response type do not exist.

- [ ] **Step 4: Implement the minimal client API.** Add:

```go
type HealthResponse struct {
    Status string `json:"status"`
}

func (c *Client) Health(ctx context.Context) (*HealthResponse, error)
```

Construct `GET /health` with `buildURL`, `http.NewRequestWithContext`, `applyAuth`, and `do`, matching `Status`/`Cancel` error behavior.

- [ ] **Step 5: Run the focused tests.** Run `go test ./internal/client -run 'TestHealth' -count=1`; expect all health tests to pass.

- [ ] **Step 6: Commit the client change.** Run `git add internal/client && git commit -m "feat: add health API client"`.

### Task 2: Wire the `health` CLI command and output

**Files:**
- Modify: `internal/cli/commands.go`
- Modify: `internal/cli/commands_test.go`

- [ ] **Step 1: Write failing CLI tests.** Add tests using the existing test-server helpers to assert `Run([]string{"health"}, ...)` returns exit code `0` and prints `status=ok`, and `Run([]string{"health", "--json"}, ...)` returns JSON with `status: "ok"` and `success: true`.

- [ ] **Step 2: Add the command dispatch.** Add `case "health": return runHealth(...)` to `Run` and implement a `runHealth` handler that resolves config, constructs `client.New`, calls `Health`, and maps errors through the existing `reportError` function.

- [ ] **Step 3: Implement health flags.** Add a `parseHealthFlags` helper supporting `--json` and `--help`, requiring no positional job ID. `--help` prints command usage and exits successfully.

- [ ] **Step 4: Implement output envelopes.** Human mode prints `status=<value>`. JSON mode marshals a response containing `status` and `success: true`, using the same output conventions as other commands.

- [ ] **Step 5: Run CLI tests and the full suite.** Run `go test ./internal/cli -run 'TestRun_Health' -count=1` followed by `go test ./...`; expect all tests to pass.

- [ ] **Step 6: Commit the CLI change.** Run `git add internal/cli && git commit -m "feat: add health CLI command"`.

### Task 3: Update help text and documentation

**Files:**
- Modify: `internal/cli/help.go`
- Modify: `internal/cli/help_test.go`
- Modify: `README.md`

- [ ] **Step 1: Extend help tests.** Require top-level help to contain `health`, and add a command-help assertion for `GET /health` and `--json`.

- [ ] **Step 2: Update help text.** Add `health` to the top-level command list and add `printHealthUsage` describing `atlas-ap-remote health [--json]`.

- [ ] **Step 3: Update README.** Add `health` to the command list and document ordinary and JSON examples, including that it performs one `GET /health` request and optionally sends the configured Bearer token.

- [ ] **Step 4: Run formatting and verification.** Run `gofmt -w internal/client internal/cli`, `go test ./...`, `go vet ./...`, and `git diff --check`; expect all commands to pass.

- [ ] **Step 5: Commit documentation and help changes.** Run `git add internal/cli/help.go internal/cli/help_test.go README.md && git commit -m "docs: document health command"`.

