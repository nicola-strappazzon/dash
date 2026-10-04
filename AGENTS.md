# AGENTS.md

Behavioral guidelines for coding agents. The general guidance applies to any
project; project-specific instructions belong in their own section below.

These guidelines reduce common coding mistakes and bias toward caution over
speed. Use judgment for trivial tasks.

## General guidelines

### Think before coding

Do not assume or hide uncertainty. State material assumptions and tradeoffs.
When several interpretations would materially change the implementation,
present them or ask for direction rather than choosing silently. Prefer the
simpler viable approach and explain a necessary new abstraction before adding
it.

### Engineering principles

1. Prefer the smallest change that solves the problem.
2. Do not introduce abstractions for hypothetical future requirements.
3. Avoid duplicating code. Reuse existing patterns, and extract shared code only when it has more than one real use.
4. Do not add defensive checks for states that are impossible by construction or database constraints.
5. Let unexpected failures propagate to the existing error boundary.
6. Do not catch exceptions unless this layer can meaningfully recover from them.
7. Prefer existing patterns over introducing new architecture.
8. Do not refactor unrelated code while implementing a feature.
9. Three similar lines of code are preferable to a premature abstraction.
10. Before creating a new class, service, interface, or module, explain what concrete problem it solves.
11. Optimize for readability of the code that remains, not theoretical extensibility.
12. Prefer the Go standard library and idiomatic language features over custom helpers or dependencies when they solve the problem clearly.
13. Prefer concrete domain types with real behavior over generic frameworks. Do not introduce an abstraction merely to hold repeated data.
14. Keep queries, parsing, and transformations explicit and easy to audit; avoid hidden execution paths and magic configuration.
15. Do not add error handling for impossible scenarios.
16. If you write 200 lines and it could be 50, rewrite it. Prefer fewer lines whenever clarity and required behavior are preserved.

### Before adding code

Ask:

1. Is this required by the specification?
2. Does the repository already solve this problem?
3. Is this state actually possible?
4. Is this error recoverable at this layer?
5. Does this abstraction have more than one real use?
6. Can the solution be simpler?

### Surgical changes

Touch only what the request requires. Match the existing style and do not
refactor, reformat, or delete adjacent code merely because it could be
improved. Mention unrelated issues instead.

Remove only imports, variables, and functions made unused by your own change.
Every changed line should trace directly to the request.

### Goal-driven execution

Define observable success criteria before implementing. For a bug, reproduce
it in a test where practical; for a feature, test its requested behavior.
For multi-step work, give a short plan with a verification for each step.
Continue until the criteria are verified rather than stopping at an
unconfirmed implementation.

### Public interfaces

Treat command-line behavior as a public API. Command names, flags, examples,
completion, exit behavior, and terminal output require the same care as Go
APIs. Prefer small, deterministic tests for parsing, normalization, formatting,
and boundary behavior.

### Legacy code

Learn from existing and legacy code, but do not reproduce a historical
compromise unless its original constraint still exists.

### Commit messages

Use Conventional Commits: `type(scope): imperative summary`. Use the smallest
scope that describes the change, keep the summary concise and without a final
period, and make each commit atomic. Prefer `feat`, `fix`, `docs`, `test`,
`refactor`, or `chore` as appropriate.

## Project-specific instructions: Dash

Dash is a Go CLI for inspecting operational data from ClickHouse,
Elasticsearch, and MySQL.

### Project layout

- `cmd/<service>/`: Cobra commands and terminal rendering.
- `cmd/<service>/internal/runner/`: connection lifecycle and section registry.
- `internal/driver/<service>/`: queries, parsing, and service-specific models.
- `internal/command/`: shared CLI options and repeat/watch behavior.

Keep data retrieval in drivers and presentation in commands. New dashboard
sections should be registered through the service runner and exposed as Cobra
subcommands.

Each section should answer one operational question with a concise,
information-dense terminal view. Prefer a concrete driver model per section
(for example, `Index`, `Table`, or `InnoDBStatus`) over untyped maps. Keep SQL
in the driver and name result fields after their operational meaning.

### Table output

Tables use `github.com/nicola-strappazzon/go-table`, replaced locally by
`../go-table` in `go.mod`. Use `FitWidth(table.TerminalWidth())` with
`Column.MaxWidth` for text columns that may overflow; do not truncate numeric
metrics unless explicitly requested.

For optional database capabilities or privileges, degrade only when the
remaining output is still meaningful, and make unavailable information clear.

### Development

- Format changed Go files with `gofmt`.
- Verify changes with `go test ./...`, `go vet ./...`, and `git diff --check`.
- Preserve existing user changes and avoid unrelated formatting or refactors.
- Never include credentials, connection strings, or production data in source,
  tests, commits, or output.
