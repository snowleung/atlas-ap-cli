# README Agent Skill and Windows Installation Design

## Goal

Make the README point Windows users to the precompiled release artifact and
show agent users how to install the bundled Atlas AP Remote skill.

## README changes

1. At the start of **Windows (recommended)**, add a **Precompiled binaries**
   subsection linking to the repository's latest GitHub Release. Tell users to
   download `atlas-ap-remote-windows-amd64.exe`, optionally verify it with the
   adjacent `.sha256` file, and rename it or place it on `PATH` as desired.
2. Keep the existing PowerShell build instructions under a **Build from
   source** subsection so users can still compile the executable locally.
3. Add a top-level **Agent skills** section after Installation. Briefly explain
   that the bundled skill teaches Codex and compatible agents to submit files,
   inspect or cancel jobs, and download results safely.
4. Link to `skill/atlas-ap-remote/SKILL.md` and include a minimal shell command
   that copies it from the repository root into
   `~/.codex/skills/atlas-ap-remote/SKILL.md`.

## Validation

- Confirm all relative Markdown links resolve inside the repository.
- Confirm the release URL matches the repository configured as `origin`.
- Review the rendered heading hierarchy and fenced command examples.
