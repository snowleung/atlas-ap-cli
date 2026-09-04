# Report and Safe Material Template Upload Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `report-template` and `safe-material-template` CLI uploads and teach the bundled agent skill the exact source-filename routing.

**Architecture:** Reuse the existing `cmdDataFile` CLI handler and `Client.UploadDataFile` multipart implementation. Extend only the command-to-endpoint mappings, table-driven contract tests, help text, README, and bundled skill; keep filename routing as agent guidance rather than CLI validation.

**Tech Stack:** Go standard library, existing `testing`/`httptest` tests, Markdown documentation, Codex skill frontmatter.

---

## File structure

- `internal/cli/commands.go`: maps the two new top-level command names to their endpoints.
- `internal/cli/commands_test.go`: verifies both commands send the expected multipart POST and preserve existing output behavior.
- `internal/cli/help.go`: lists both commands and endpoints in top-level help.
- `internal/cli/help_test.go`: verifies top-level and command-specific help for all six upload commands.
- `internal/client/client_test.go`: extends the reusable upload protocol contract to the two new endpoint paths; `internal/client/client.go` needs no change.
- `README.md`: documents all six upload commands and shows the two template filenames.
- `skill/atlas-ap-remote/SKILL.md`: routes the two exact basenames to the correct CLI commands and endpoints.

### Task 1: Add failing CLI and help contracts

**Files:**
- Modify: `internal/cli/commands_test.go`
- Modify: `internal/cli/help_test.go`

- [ ] **Step 1: Extend the CLI command table**

Add the two entries to `TestRun_DataFileCommands`:

```go
commands := []struct {
	name string
	path string
}{
	{"material-db", "/data-files/material-db"},
	{"reference-db", "/data-files/reference-db"},
	{"risk-db", "/data-files/risk-db"},
	{"special-materials-config", "/data-files/special-materials-config"},
	{"report-template", "/data-files/report-template"},
	{"safe-material-template", "/data-files/safe-material-template"},
}
```

The existing JSON and human-mode subtests then verify both new commands make a
multipart POST to the exact path and preserve arbitrary response fields.

- [ ] **Step 2: Extend the help expectations**

Add the new command/path pairs to `TestHelpText_TopLevelDataFiles`:

```go
"report-template", "/data-files/report-template",
"safe-material-template", "/data-files/safe-material-template",
```

Replace the command slice in `TestHelpText_DataFile` with:

```go
for _, command := range []string{
	"material-db",
	"reference-db",
	"risk-db",
	"special-materials-config",
	"report-template",
	"safe-material-template",
} {
```

- [ ] **Step 3: Run the focused tests and verify RED**

Run:

```bash
go test ./internal/cli -run 'TestRun_DataFileCommands|TestHelpText_(TopLevelDataFiles|DataFile)' -count=1
```

Expected: FAIL. The `report-template` and `safe-material-template` command
subtests return exit code 2 with `unknown command`, and top-level help lacks
both names and endpoint paths.

### Task 2: Register the commands and expose help

**Files:**
- Modify: `internal/cli/commands.go`
- Modify: `internal/cli/help.go`
- Test: `internal/cli/commands_test.go`
- Test: `internal/cli/help_test.go`

- [ ] **Step 1: Add the command dispatch mappings**

Add these cases beside the existing data-file cases in `Run`:

```go
case "report-template":
	return cmdDataFile(gf, environ, "report-template", "/data-files/report-template", subArgs, stdout, stderr)
case "safe-material-template":
	return cmdDataFile(gf, environ, "safe-material-template", "/data-files/safe-material-template", subArgs, stdout, stderr)
```

Do not add new handlers or filename validation. Both cases must reuse
`cmdDataFile` so configuration, `--file`, `--json`, errors, and exit codes stay
identical to the four existing commands.

- [ ] **Step 2: Add both entries to top-level help**

Append these lines after the existing data-file help entries:

```go
fmt.Fprintln(w, "  report-template")
fmt.Fprintln(w, "                 Upload a data file (one POST /data-files/report-template).")
fmt.Fprintln(w, "  safe-material-template")
fmt.Fprintln(w, "                 Upload a data file (one POST /data-files/safe-material-template).")
```

Also correct the `printUsage` comment so it no longer claims a fixed count of
five commands; describe it as documenting the available commands.

- [ ] **Step 3: Run the focused tests and verify GREEN**

Run:

```bash
go test ./internal/cli -run 'TestRun_DataFileCommands|TestHelpText_(TopLevelDataFiles|DataFile)' -count=1
```

Expected: PASS for the new JSON/human uploads and both help tests.

- [ ] **Step 4: Commit the CLI behavior**

```bash
git add internal/cli/commands.go internal/cli/commands_test.go internal/cli/help.go internal/cli/help_test.go
git commit -m "feat: add template upload commands"
```

### Task 3: Extend protocol coverage and user documentation

**Files:**
- Modify: `internal/client/client_test.go`
- Modify: `README.md`

- [ ] **Step 1: Extend the client endpoint contract table**

Add these paths to `TestUploadDataFile_SendsMultipartToAllEndpoints`:

```go
"/data-files/report-template",
"/data-files/safe-material-template",
```

This is coverage for the already-generic client method, so it is expected to
pass immediately rather than produce a false TDD red state.

- [ ] **Step 2: Run the focused client contract**

Run:

```bash
go test ./internal/client -run TestUploadDataFile_SendsMultipartToAllEndpoints -count=1
```

Expected: PASS while verifying POST method, exact paths, bearer auth,
multipart `file` content and filename, and arbitrary JSON decoding for all six
endpoints.

- [ ] **Step 3: Update the README feature summary and examples**

Change the data-file feature list and command section from four commands to
six. Add these invocations to the existing example block:

```bash
atlas-ap-remote --server https://api.example.com report-template --file './模板文件_勿删.docx' --json
atlas-ap-remote --server https://api.example.com safe-material-template --file './配方的成分安全评估模板_勿删.docx' --json
```

State that those two basenames correspond to the report-template and safe
material-template endpoints respectively. Keep the existing single POST,
required multipart `file`, no polling, no retry, and arbitrary response text.

- [ ] **Step 4: Commit protocol coverage and README**

```bash
git add internal/client/client_test.go README.md
git commit -m "docs: document template upload commands"
```

### Task 4: Update the bundled agent skill

**Files:**
- Modify: `skill/atlas-ap-remote/SKILL.md`

- [ ] **Step 1: Broaden the skill trigger description**

Replace the frontmatter description with this single-line pointer:

```yaml
description: Use when a user wants to submit, inspect, cancel, or download Atlas AP Remote jobs; upload Atlas data files or templates; or generate a safety assessment (安评).
```

This adds the data-file/template branch while retaining the existing job and
safety-assessment branches.

- [ ] **Step 2: Expand the data-file routing section**

Change “Four commands” to “Six commands”, add `report-template` and
`safe-material-template` to the supported list, and add this routing table:

```markdown
| Source filename | Command | Endpoint |
| --- | --- | --- |
| `模板文件_勿删.docx` | `report-template` | `/data-files/report-template` |
| `配方的成分安全评估模板_勿删.docx` | `safe-material-template` | `/data-files/safe-material-template` |
```

Add these copyable invocations:

```bash
atlas-ap-remote --server "$ATLAS_REMOTE_URL" report-template --file '/path/to/模板文件_勿删.docx' --json
atlas-ap-remote --server "$ATLAS_REMOTE_URL" safe-material-template --file '/path/to/配方的成分安全评估模板_勿删.docx' --json
```

After the examples, instruct the agent to compare the source basename with the
table, pass the user-provided path unchanged, and report a mismatch instead of
guessing when neither name matches. State that the CLI itself accepts other
basenames and does not rename the upload.

- [ ] **Step 3: Verify skill routing text mechanically**

Run:

```bash
rg -n 'upload data files or templates|Six commands|模板文件_勿删\.docx|配方的成分安全评估模板_勿删\.docx|report-template|safe-material-template|does not poll or retry' skill/atlas-ap-remote/SKILL.md
```

Expected: every routing phrase, filename, command, and safety constraint is
present; both filenames appear in the table and example commands.

- [ ] **Step 4: Commit the skill update**

```bash
git add skill/atlas-ap-remote/SKILL.md
git commit -m "docs: teach skill template upload routing"
```

### Task 5: Teach the skill to update the CLI from current instructions

**Files:**
- Modify: `skill/atlas-ap-remote/SKILL.md`

- [ ] **Step 1: Run a failing skill baseline scenario**

Give an independent agent the current skill and this read-only scenario:

```text
The user asks you to update the installed atlas-ap-remote CLI. Explain the
source you will check for the latest version, how you decide whether an update
is needed, what integrity check you perform, whether the installed skill is
also refreshed, and how you verify completion. Do not perform the update.
```

Expected: the current skill has no update branch, so the agent cannot derive
all required decisions from it—especially checking the live Agent skills page,
comparing installed/latest versions, and refreshing the installed skill.

- [ ] **Step 2: Add the minimal update workflow**

Extend the frontmatter trigger with CLI installation and update intent, then
add this section after `## Configuration and safety`:

```markdown
## Install or update the CLI

When the user asks to install, update, or upgrade `atlas-ap-remote`, first read
the current [Agent skills](https://github.com/snowleung/atlas-ap-cli#agent-skills)
section. Treat that page and its latest-release link as the source of truth;
do not rely on a version number or asset name cached in this skill.

Run `atlas-ap-remote --version` when the executable is installed and compare it
with the latest GitHub release. Stop without reinstalling when it is current.
When an update is needed, follow the page's current platform instructions,
verify the downloaded asset against its published SHA256 checksum before
replacement, install it on `PATH`, and refresh the installed
`atlas-ap-remote` skill as directed by the page. Obtain user authorization
before downloading or replacing installed files. Run
`atlas-ap-remote --version` afterward and report the verified version.
```

Use this updated frontmatter description:

```yaml
description: Use when a user wants to install or update the Atlas AP Remote CLI; submit, inspect, cancel, or download its jobs; upload Atlas data files or templates; or generate a safety assessment (安评).
```

- [ ] **Step 3: Validate and forward-test the updated skill**

Run:

```bash
python3 /Users/hans/.codex/skills/.system/skill-creator/scripts/quick_validate.py skill/atlas-ap-remote
```

Then rerun the exact scenario from Step 1 with an independent agent reading
the modified skill.

Expected: the skill validator succeeds, and the agent identifies the Agent
skills page as live source of truth, checks `atlas-ap-remote --version`, follows
the page to the latest release, skips an unnecessary reinstall, otherwise
verifies SHA256 before replacement, refreshes the installed skill, obtains
authorization for mutations, and verifies the installed version afterward.

- [ ] **Step 4: Commit the skill workflow**

```bash
git add skill/atlas-ap-remote/SKILL.md
git commit -m "docs: teach skill CLI update workflow"
```

### Task 6: Complete verification

**Files:**
- No source changes expected.

- [ ] **Step 1: Format the changed Go files**

Run:

```bash
gofmt -w internal/cli/commands.go internal/cli/commands_test.go internal/cli/help.go internal/cli/help_test.go internal/client/client_test.go
```

Expected: files are formatted with no command errors.

- [ ] **Step 2: Check whitespace and static analysis**

Run:

```bash
git diff --check
go vet ./...
```

Expected: both commands exit 0 with no diagnostics.

- [ ] **Step 3: Run the full test suite**

Run:

```bash
go test ./... -count=1
```

Expected: all packages pass with zero failures.

- [ ] **Step 4: Review final scope**

Run:

```bash
git status --short --branch
git log --oneline --decorate -8
```

Confirm the feature changes are limited to the two command mappings, table
coverage, help, README, bundled skill routing/update workflows, and the
approved design/plan documents.
