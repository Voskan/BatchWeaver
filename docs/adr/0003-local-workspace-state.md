# ADR 0003: Local workspace state is never published

- Status: Accepted
- Date: 2026-07-29

## Context

Developer tools may maintain local plans, logs, caches, and continuity notes.
This state must never be published because it is tool-specific, noisy, and may
contain sensitive local context.

## Decision

- Keep local tool state under ignored, tool-owned directories.
- Maintain defensive ignore rules for known editor and automation state.
- Treat the committed Git history as the single source of truth for project code
  and documentation.
- Ensure no release artifact includes tool transcripts or local coordination data.

## Consequences

- Repository history stays free of tool-specific state and transcripts.
- Contributors using different tools do not create conflicting tracked files.
- Continuity state can be regenerated locally and is never required to understand
  the committed project.

## Alternatives considered

- **Committing shared tool state for continuity.** Rejected; it risks
  leaking sensitive or tool-specific content and adds noise and merge conflicts.
- **No convention at all.** Rejected; without an explicit ignore policy such
  files are easy to commit by accident.
