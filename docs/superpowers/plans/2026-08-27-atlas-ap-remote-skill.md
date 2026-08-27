# Atlas AP Remote CLI Skill Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a repository-local Codex skill that safely guides use of the Atlas AP Remote CLI, with a hard requirement for a recipe file during safety-assessment generation.

**Architecture:** Use one self-contained `SKILL.md` under `skill/atlas-ap-remote`. It will encode the CLI contract from `README.md`, route user intent to one command, and stop before `submit` when an assessment request lacks a local recipe file. No helper scripts or runtime code are needed.

**Tech Stack:** Markdown, YAML frontmatter, `skill-creator` validation script.

---

### Task 1: Create the skill entrypoint

**Files:**
- Create: `skill/atlas-ap-remote/SKILL.md`

- [ ] **Step 1: Create the skill directory and entrypoint**

Write YAML frontmatter with the name `atlas-ap-remote` and a discriminating description covering submit, status, download, cancel, and safety-assessment requests. Write instructions for explicit server/token precedence, token handling, global flag placement, JSON mode, exit codes, one-request behavior, command templates, safe downloads, and the mandatory recipe-file gate for assessment generation.

- [ ] **Step 2: Review the entrypoint against the README contract**

Confirm that all command names and flags match `README.md`, that the skill does not promise polling or unsupported batch behavior, and that the missing-recipe response tells the user exactly what is required.

### Task 2: Validate and inspect the artifact

**Files:**
- Verify: `skill/atlas-ap-remote/SKILL.md`

- [ ] **Step 1: Run the skill validator**

Run `python3 /Users/hans/.codex/skills/.system/skill-creator/scripts/quick_validate.py /Users/hans/Desktop/workspace/atlas-ap-cli/skill/atlas-ap-remote` and expect a successful validation result.

- [ ] **Step 2: Check the final worktree diff**

Run `git diff --check` and inspect `git status --short`; verify that only the approved design, plan, and skill files are added or changed.
