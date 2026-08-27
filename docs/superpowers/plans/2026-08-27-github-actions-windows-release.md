# GitHub Actions Windows EXE Release Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Publish a Windows amd64 `atlas-ap-remote` executable and SHA256 file when a `v*` Git tag is pushed.

**Architecture:** Keep `build.ps1` as the single local/CI build entry point and make its version injectable. Add one Windows-only GitHub Actions workflow triggered exclusively by `v*` tags; it builds, verifies, hashes, and publishes a GitHub Release in one job.

**Tech Stack:** PowerShell, Go 1.27, GitHub Actions on `windows-latest`, GitHub CLI, SHA256.

---

### Task 1: Make the PowerShell build script version-aware and single-pass

**Files:**
- Modify: `build.ps1`
- Modify: `README.md`

- [ ] **Step 1: Add the version parameter.** Add `[string]$Version = "dev"` beside the existing `-Arch` and `-Out` parameters; document `.\build.ps1 -Version 0.2.0` in the examples.

- [ ] **Step 2: Remove the duplicate build.** Keep one `go build` call using:

```powershell
$ldflags = "-s -w -X github.com/atlas-ap/atlas-ap-remote/internal/cli.Version=$Version"
& go build -trimpath -ldflags $ldflags -o $exePath ./cmd/atlas-ap-remote
```

Preserve strict error handling and print the selected version after success.

- [ ] **Step 3: Update the README.** Explain that local builds default to `dev`, formal builds pass `-Version`, and replace the fixed `0.1.0` cross-compile example with a release-version example.

- [ ] **Step 4: Verify locally.** Run `git diff --check` and `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -X github.com/atlas-ap/atlas-ap-remote/internal/cli.Version=0.2.0" -o /private/tmp/atlas-ap-remote.exe ./cmd/atlas-ap-remote`; expect no diff errors and a non-empty executable.

- [ ] **Step 5: Commit.** Run `git add build.ps1 README.md && git commit -m "build: make Windows version configurable"`.

### Task 2: Add the tag-only Release workflow

**Files:**
- Create: `.github/workflows/release.yml`

- [ ] **Step 1: Define trigger and permissions.** Use `push.tags: ["v*"]`, `permissions: contents: write`, and `concurrency.group: release-${{ github.ref }}` with `cancel-in-progress: false`. Do not add PR, branch-push, schedule, or validation triggers/jobs.

- [ ] **Step 2: Configure the Windows job.** Use `runs-on: windows-latest`, checkout the pushed ref, install Go `1.27.x` with `actions/setup-go`, and use PowerShell for build steps.

- [ ] **Step 3: Derive the version.** Remove only the leading `v` from `${{ github.ref_name }}`, fail if empty, and expose the result as `RELEASE_VERSION` for later steps. Do not use runtime service secrets.

- [ ] **Step 4: Build and rename the asset.** Run `.\build.ps1 -Version $env:RELEASE_VERSION -Out dist`; fail if `dist\atlas-ap-remote.exe` is missing or empty; use `Move-Item dist\atlas-ap-remote.exe dist\atlas-ap-remote-windows-amd64.exe` for the final asset name.

- [ ] **Step 5: Generate the checksum.** Use `Get-FileHash -Algorithm SHA256` on the final EXE and write its hash plus the exact filename to `dist/atlas-ap-remote-windows-amd64.exe.sha256`.

- [ ] **Step 6: Publish the Release.** Set `GH_TOKEN: ${{ github.token }}` and run `gh release create $env:GITHUB_REF_NAME dist/atlas-ap-remote-windows-amd64.exe dist/atlas-ap-remote-windows-amd64.exe.sha256 --title $env:GITHUB_REF_NAME --generate-notes`. Do not use `--prerelease`.

- [ ] **Step 7: Commit.** Run `git add .github/workflows/release.yml && git commit -m "ci: publish Windows executable from version tags"`.

### Task 3: Verify the release contract

**Files:**
- Verify: `build.ps1`
- Verify: `.github/workflows/release.yml`
- Verify: `README.md`

- [ ] **Step 1: Inspect workflow syntax.** Run `ruby -e 'require "yaml"; YAML.load_file(".github/workflows/release.yml")'` and inspect the workflow with GitHub Actions validation. Confirm PowerShell paths and `${{ }}` expressions are valid.

- [ ] **Step 2: Run repository checks.** Run `gofmt -l .`, `go test ./...`, `go vet ./...`, `git diff --check`, and `git status --short`; expect no gofmt output and passing tests/vet.

- [ ] **Step 3: Run a controlled release test after merge.** Push a new tag such as `v0.2.0`; verify the workflow creates the matching Release with both assets and the checksum validates the downloaded EXE. Verify a normal fix commit does not trigger this workflow.
