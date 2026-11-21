## Agent skills

### Issue tracker

Issues live as markdown under `.scratch/<feature>/`. See `docs/agents/issue-tracker.md`.

### Triage labels

Default vocabulary: `needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`. See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: root `GLOSSARY.md` + `docs/adr/`. See `docs/agents/domain.md`.

## Backend architecture

Before designing, implementing, or reviewing backend changes, read and follow the [backend architecture pattern](docs/patterns/architecture.md). Preserve its dependency direction, module boundaries, and transaction ownership.

### Backend API pattern

When adding or changing Gin endpoints, request/response DTOs, error envelopes, or cursor pagination, see the [HTTP API pattern](docs/patterns/http.md).

When defining or wrapping typed backend errors, see the [application error pattern](docs/patterns/errors.md).
