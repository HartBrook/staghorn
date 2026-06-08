---
name: verify
description: Verify a change actually works by running it, not by inspecting it
tags: [workflow, quality]
allowed-tools: Read Grep Glob Bash
args:
  - name: target
    description: The change or feature to verify
    default: the most recent change
  - name: checks
    description: Which checks to run
    default: auto
    options: [auto, build, tests, all]
---

# Verify

Confirm that **{{target}}** actually works by running it and observing the
result — not by reading the code and assuming. A change is not done until it
has been verified.

Checks to run: **{{checks}}**

## How to Verify

1. **Discover the project's own commands.** Don't guess. Look at the files that
   declare them:
   - `package.json` scripts (`build`, `test`, `lint`), `Makefile` targets,
     `pyproject.toml` / `tox.ini`, `go.mod` (`go build ./...`, `go test ./...`),
     `Cargo.toml`, CI config (`.github/workflows/*`).
   - Read the project's docs where build/test steps are written in prose:
     `README.md`, `CONTRIBUTING.md`, `TESTING.md`, or a `docs/` equivalent.
     A documented command is authoritative — prefer it over one you infer.
   - Prefer the command the project already uses in CI over an ad-hoc one.

2. **Run the relevant checks** for `{{checks}}`:
   - `build` — compile or build the project.
   - `tests` — run the test suite (scope to the affected area when the full
     suite is slow, but say so).
   - `all` / `auto` — run build then tests; `auto` skips a step the project
     doesn't have rather than inventing one.

3. **Observe the actual output.** Read exit codes and the real test/build
   results. Passing means the command reported success — not that the code
   looks correct.

4. **If a check fails**, stop and report the failure with the actual output.
   Do not work around it silently or describe it as a minor issue.

## Reporting

Report only what you observed:

- **What ran** — the exact commands.
- **Result** — pass or fail, with the real output for any failure.
- **Not covered** — anything you could not verify (no test exists, couldn't run
  the app, environment missing), stated plainly.

Never claim a change works without having run something that demonstrates it.
If verification wasn't possible, say exactly what was and wasn't checked rather
than implying confidence you don't have.
