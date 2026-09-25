# Body Parts Help Reference Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show the nine standard Chinese `--body-parts` values in submit help and teach the bundled skill to use those text values.

**Architecture:** Keep the values advisory: extend the existing hand-written submit usage output and skill instructions without adding client-side validation or changing request serialization. Protect the CLI help contract with the existing unit test.

**Tech Stack:** Go standard library, Go `testing`, Markdown

---

### Task 1: Add the body-parts reference to CLI help

**Files:**
- Modify: `internal/cli/help_test.go`
- Modify: `internal/cli/help.go`

- [ ] **Step 1: Write the failing help test**

Extend `TestHelpText_Submit` so its expected strings include the Chinese-text instruction and every reference value:

```go
for _, want := range []string{
	"submit",
	"--file",
	"--cos-type",
	"--body-parts",
	"Chinese text",
	"全身",
	"躯干部位",
	"面部（含颈部）",
	"手足",
	"头部",
	"头发",
	"口唇",
	"眼部",
	"指（趾）甲",
	"--product-name",
	"--usage-method",
	"--json",
} {
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `go test ./internal/cli -run TestHelpText_Submit -count=1`

Expected: FAIL because the current help lacks `Chinese text` and eight non-default reference values.

- [ ] **Step 3: Add the reference block to submit help**

Append this output after the existing usage lines in `printSubmitUsage`:

```go
fmt.Fprintln(w)
fmt.Fprintln(w, "Body parts")
fmt.Fprintln(w, "  --body-parts accepts Chinese text (default: 全身). Reference values:")
fmt.Fprintln(w, "    全身")
fmt.Fprintln(w, "    躯干部位")
fmt.Fprintln(w, "    面部（含颈部）")
fmt.Fprintln(w, "    手足")
fmt.Fprintln(w, "    头部")
fmt.Fprintln(w, "    头发")
fmt.Fprintln(w, "    口唇")
fmt.Fprintln(w, "    眼部")
fmt.Fprintln(w, "    指（趾）甲")
```

- [ ] **Step 4: Run the focused test and verify it passes**

Run: `go test ./internal/cli -run TestHelpText_Submit -count=1`

Expected: PASS.

### Task 2: Update the bundled skill guidance

**Files:**
- Modify: `skill/atlas-ap-remote/SKILL.md`

- [ ] **Step 1: Add body-parts guidance beside the submit workflow**

After the submit command example, add:

```markdown
For `--body-parts`, pass the Chinese text value, not a numeric key. The standard reference values are:

- `全身` (default)
- `躯干部位`
- `面部（含颈部）`
- `手足`
- `头部`
- `头发`
- `口唇`
- `眼部`
- `指（趾）甲`

Treat this as guidance rather than a closed enum; if the user supplies another server-supported value, pass it through unchanged.
```

- [ ] **Step 2: Verify the skill contains every value exactly once in the new reference list**

Run:

```bash
for value in '全身' '躯干部位' '面部（含颈部）' '手足' '头部' '头发' '口唇' '眼部' '指（趾）甲'; do
  rg -q --fixed-strings -- "- \`$value" skill/atlas-ap-remote/SKILL.md || exit 1
done
```

Expected: exit code 0.

### Task 3: Verify the complete change

**Files:**
- Verify: `internal/cli/help.go`
- Verify: `internal/cli/help_test.go`
- Verify: `skill/atlas-ap-remote/SKILL.md`

- [ ] **Step 1: Format Go files**

Run: `gofmt -w internal/cli/help.go internal/cli/help_test.go`

Expected: exit code 0.

- [ ] **Step 2: Run the full test suite**

Run: `go test ./... -count=1`

Expected: all packages pass.

- [ ] **Step 3: Check patch hygiene and scope**

Run: `git diff --check && git status --short && git diff -- internal/cli/help.go internal/cli/help_test.go skill/atlas-ap-remote/SKILL.md`

Expected: no whitespace errors; only the plan, help test, help output, and skill guidance are pending changes after the already committed design document.

- [ ] **Step 4: Commit the implementation**

```bash
git add docs/superpowers/plans/2026-09-03-body-parts-help.md internal/cli/help.go internal/cli/help_test.go skill/atlas-ap-remote/SKILL.md
git commit -m "feat: document body parts reference values"
```
