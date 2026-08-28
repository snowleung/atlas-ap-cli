# README Agent Skill and Windows Installation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Update the README with a precompiled Windows binary download path and minimal instructions for installing the bundled Agent skill.

**Architecture:** This is a documentation-only change in `README.md`. It preserves the current source-build instructions, adds GitHub Releases as the preferred Windows path, and gives the bundled skill its own section so CLI installation and agent configuration remain distinct.

**Tech Stack:** GitHub Flavored Markdown, PowerShell, POSIX shell

---

### Task 1: Add precompiled Windows installation instructions

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add the precompiled binary subsection**

Under `### Windows (recommended)`, insert this content before the existing PowerShell build commands:

````markdown
#### Precompiled binaries

Download the latest `atlas-ap-remote-windows-amd64.exe` and its optional
SHA256 checksum from the [GitHub Releases page](https://github.com/snowleung/atlas-ap-cli/releases/latest).

Rename the executable if desired, then run it directly or place it on your
`PATH`:

```powershell
Rename-Item .\atlas-ap-remote-windows-amd64.exe atlas-ap-remote.exe
.\atlas-ap-remote.exe --help
```

#### Build from source
````

- [ ] **Step 2: Check heading hierarchy and whitespace**

Run:

```bash
sed -n '20,90p' README.md
```

Expected: `Windows (recommended)` contains `Precompiled binaries` followed by `Build from source`, while the existing cross-compilation and source-running sections remain unchanged.

### Task 2: Add Agent skill installation instructions

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add the Agent skills section**

Insert this section after the final Installation subsection and before `## Configuration`:

````markdown
## Agent skills

The repository includes an [Atlas AP Remote Agent skill](skill/atlas-ap-remote/SKILL.md)
that teaches Codex and compatible agents how to submit files, check or cancel
jobs, and download results safely.

From the repository root, install it for Codex with:

```bash
mkdir -p ~/.codex/skills/atlas-ap-remote
cp skill/atlas-ap-remote/SKILL.md ~/.codex/skills/atlas-ap-remote/SKILL.md
```
````

- [ ] **Step 2: Verify local links, external URL, and Markdown formatting**

Run:

```bash
test -f skill/atlas-ap-remote/SKILL.md
rg -n 'Agent skills|skill/atlas-ap-remote/SKILL.md|releases/latest|Precompiled binaries|Build from source' README.md
git diff --check -- README.md
```

Expected: `test` exits successfully; `rg` finds all five additions; `git diff --check` produces no output and exits successfully.

- [ ] **Step 3: Review the final diff**

Run:

```bash
git diff -- README.md
```

Expected: only the approved Windows installation and Agent skill documentation is added; existing command documentation is unchanged.
