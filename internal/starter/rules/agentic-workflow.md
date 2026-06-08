# Agentic Workflow

How Claude should operate while working in this codebase. These rules govern
process, not just the code that results from it.

## Verification

- Verify before claiming done; run the code, tests, or build and report the
  actual output. Never state that something works from inspection alone.
- When a check fails, say so plainly and show the output. Don't paper over a
  failing test or a skipped step.
- If verification isn't possible, say what was and wasn't checked rather than
  implying full confidence.

## Planning and Scope

- Plan large changes before editing. For multi-file or architectural work, lay
  out the approach and get agreement before making wide edits.
- Solve the task that was asked, not adjacent refactors. Surface tangents you
  notice; don't silently act on them.
- Don't over-engineer. Build for the problem at hand, not hypothetical future
  ones.

## Working in the Codebase

- Match the surrounding code. Mirror the file's existing naming, comment
  density, and idioms over global defaults when they conflict locally.
- Delegate broad searches. Fan out wide exploration so you keep the conclusion,
  not the raw file dumps.
- Make independent tool calls in parallel rather than serially when there is no
  dependency between them.
- Prefer the project's own skills and slash commands over ad-hoc equivalents;
  they encode local conventions.

## Safe Actions

- Confirm irreversible or outward-facing actions before taking them: pushes,
  deletes, deploys, and anything that leaves the machine.
- Look at the target before deleting or overwriting it. If what you find
  contradicts how it was described, surface that instead of proceeding.
- Approval in one context doesn't extend to the next; re-confirm when the stakes
  or surface change.
