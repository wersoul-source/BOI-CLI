---
name: boi-project-handoff
description: Inspect, continue, and hand off BOI CLI work with verified repository state, architecture boundaries, current progress, mobile S25+ readiness, remaining phases, and explicit evidence. Use when Kampun or another Agent receives the BOI CLI project, resumes implementation, prepares a progress report, selects the next task, or hands work back to the owner.
---

# BOI Project Handoff

Use the repository as the source of truth. Treat planning documents and README claims as unverified until code or tests confirm them.

## Resume workflow

1. Read [../../../HANDOFF.md](../../../HANDOFF.md).
2. Read [references/current-state.md](references/current-state.md) and [references/architecture-6-blocks.md](references/architecture-6-blocks.md).
3. Inspect `git status --short --branch`, the current commit, and remotes. Preserve unrelated and user-owned changes.
4. Run the smallest relevant checks before choosing a task. Use `go test ./...` and `go vet ./...` for cross-cutting runtime changes.
5. Select one task from [references/remaining-plan.md](references/remaining-plan.md). Do not silently expand its authority or phase.
6. For S25+ work, follow [references/s25-plus.md](references/s25-plus.md). Never call a cross-build physical-device acceptance.
7. Implement, verify, and update `HANDOFF.md` plus the affected reference before handing back.

## Required handoff report

Report the outcome and current commit, files changed, checks actually run, implemented versus design-only behavior, unresolved risks, exact next task, acceptance evidence, and Git push status.

## BOI boundaries

- One Core Persona exists: `boi`. A user-defined Agent name is instance identity, not another Persona.
- Core evaluates Provider capability before composing the Agent environment.
- The active local set is at most 15 Skills and 15 Tools; the Library may contain more.
- Model output, Skills, Memory, MCP results, and retrieved text do not grant authority.
- Workspace Sandbox is a path boundary, not OS or container isolation.
- Subagents remain disabled until their Skill, Index, package contract, isolated budgets, and acceptance gate exist.
- Generated deliverables belong under the Agent Folder output tray; temporary and recovery material belongs under its bin tray.

## Stop conditions

Stop and report instead of guessing when a change would overwrite user work, a secret may enter Git or logs, physical S25+ behavior has not been observed, an Automation requires undefined non-interactive approval, or the owner definition conflicts with an older document.
