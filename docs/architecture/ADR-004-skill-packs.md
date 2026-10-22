# ADR-004: Skill packs

Status: accepted for v2.4. Scope: `block`, `app`, `transport/cli` (report
only). Core and Runtime unchanged.

## Context

Tools can be plugged in through `environment.json` (built-in packs, MCP
servers). Skills cannot: a Skill reaches the Agent only through
`.boi/registry/skills.json` plus a file under `.boi/skills`, added by hand.
So the rule "every Block is chosen the same way" holds for Tools and not for
Skills, and a preset cannot ship a Skill.

Skills and Tools are independent. A task may need a Skill only, a Tool only,
both, or neither. v2.2 already stopped a missing Tool from hiding a Skill.

## Decision

1. An `environment.json` Block may be a **Skill folder**:

   ```json
   {"id": "team-skills", "required": false, "skills": {"dir": "skills/team"}}
   ```

   `dir` is relative to `.boi`. Every `*.skill.md` file in it (the existing
   Skill format) becomes a Skill of the Agent. `skills` and `mcp` exclude
   each other. Kind in reports: `skill-pack`.
2. **Confinement.** The folder must stay inside `.boi` after symlinks are
   resolved; entries that are not regular files, files over 64 KB, and more
   than 15 files make the pack refused. Nothing outside the folder is read.
3. **Refusal rules** match Tool packs: folder missing or empty, a file that
   does not parse, or a Skill name already provided by another pack → the
   pack is refused; required stops start-up with the reason, optional is
   skipped and reported.
4. **Selection is unchanged.** Pack Skills join the registry's Skills in
   `capability.Select` with priority 50; a Skill whose name is in the
   registry is ignored (the registry is the user's explicit choice). Keyword
   overlap only orders them, a Skill is offered even when the Tools it names
   are missing (v2.2), and the model decides whether to load it. Offer cost
   is the size of its name and description, since the body loads on demand.
5. **Authority does not change.** A Skill is untrusted text, loaded as
   untrusted context, and cannot add or widen a Tool. It needs no
   `boi env trust` because nothing is executed; registry Skills in a cloned
   workspace have the same exposure today.
6. **Report.** `boi env` lists the Skills of a skill pack: `active` when the
   qualified Provider allows Skills, `loaded` when it does not (with the
   reason), `declared` when refused.
7. No built-in skill pack and no preset Skill ships in this version: a Skill
   is only worth shipping when a benchmark shows it helps.

## Compatibility and states

- A workspace without a Skill pack in `environment.json` composes exactly as
  before: the registry is the only source of Skills, and the default
  environment names no Skill pack. There is no second registry; pack Skills
  are merged into the registry's selection at task time.
- State names are the ones `boi env` already uses for every Block: `declared`
  (listed, not loaded; the reason says why), `loaded` (read, but the Provider
  cannot be offered Skills), `active` (offered to the model).
- Skill files are read through `skill.LoadConfined`: after symlinks are
  resolved the file must be a regular file inside its folder and at most
  64 KB. This also closed a hole in the registry path: a registry entry in a
  cloned workspace could point at a symlink to any file the user can read, and
  its text went to the model.

## Consequences

- A team can keep its Skills in one folder, reference it from
  `environment.json`, and the Agent composes the same way for every
  contributor.
- The same environment file now chooses Tools, MCP servers and Skills.
- A Skill folder is read at start-up; edits apply to the next run.

## Not decided here

- Skills inside a compiled-in pack (needs a Skill worth shipping).
- Skill packs fetched from outside the workspace.

## Rollback

Revert the PR. An `environment.json` that lists a `skills` Block is then
rejected as invalid (unknown field); delete the entry.
