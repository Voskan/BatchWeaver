# ADR 0003: Agent state is local only

- Status: Accepted
- Date: 2026-07-29

## Context

The project is developed with the help of automated coding tools that maintain
local working state (plans, session logs, handoff notes). This state is useful
for continuity across sessions but must never leak into the repository, both to
keep history clean and to avoid committing tool-specific or potentially sensitive
material.

## Decision

- Keep all such continuity files under a single local directory, `.agent/`.
- Ignore `.agent/` and other known tool/agent directories and files in
  `.gitignore` (for example `.claude/`, `.codex/`, `.cursor/`, `AGENTS.md`,
  `CLAUDE.md`, `CODEX.md`, `*.prompt.md`).
- Treat the committed Git history as the single source of truth for project code
  and documentation.
- Ensure no committed artifact states that it was produced by an automated agent.

## Consequences

- Repository history stays free of tool-specific state and transcripts.
- Contributors using different tools do not create conflicting tracked files.
- Continuity state can be regenerated locally and is never required to understand
  the committed project.

## Alternatives considered

- **Committing a shared agent-state file for continuity.** Rejected; it risks
  leaking sensitive or tool-specific content and adds noise and merge conflicts.
- **No convention at all.** Rejected; without an explicit ignore policy such
  files are easy to commit by accident.
