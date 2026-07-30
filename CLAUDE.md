# CLAUDE.md

Working conventions for this repository. Follow these for every task.

## General Principles

- Always prioritize code quality over speed.
- Write production-ready code.
- Keep functions and files small and focused.
- Prefer readability over cleverness.
- Never introduce breaking changes without explanation.
- Never leave TODOs unless explicitly requested.

## Git Workflow

- Never commit directly to the `main` branch.
- Always create a new branch for every task.
- Branch naming:
  - `feature/<name>`
  - `fix/<name>`
  - `refactor/<name>`
  - `docs/<name>`
  - `chore/<name>`

  Examples: `feature/deployment-engine`, `fix/docker-timeout`,
  `docs/getting-started`.

- Commit frequently with meaningful commit messages.
- Push the branch after completing the task.
- Never merge into `main` directly.
- Open a Pull Request after pushing.

## Pull Requests

Every PR should include:

- Summary of changes
- Reason for the change
- Screenshots (if UI)
- Testing performed
- Breaking changes (if any)

## Code Style

- Follow Go best practices.
- Prefer composition over inheritance.
- Use interfaces only when needed.
- Avoid unnecessary abstractions.
- Keep nesting shallow.
- Return errors early.
- Use descriptive names.

Bad:

```go
func Do()
```

Good:

```go
func DeployContainer()
```

## Project Structure

- Keep folders organized.
- Do not place unrelated code together.
- Every package should have one responsibility.

## Error Handling

Never ignore errors.

Bad:

```go
_, _ = os.ReadFile(...)
```

Good:

```go
data, err := os.ReadFile(...)
if err != nil {
    return err
}
```

Always return contextual errors.

## Logging

- Use structured logging.
- Do not use `fmt.Println` for production code.
- Every important operation should have logs.

## Testing

Every feature should include tests.

Minimum:

- unit tests

Where appropriate:

- integration tests

Tests must pass before pushing.

## Documentation

Whenever introducing:

- API
- CLI command
- Configuration
- Environment variable

update documentation. Keep README current.

## Security

Never hardcode:

- passwords
- API keys
- secrets
- tokens

Use environment variables. Validate all user input. Never trust external
input.

## Performance

Avoid premature optimization. However:

- avoid unnecessary allocations
- avoid duplicated work
- use goroutines only when beneficial
- benchmark before optimizing

## Dependencies

Before adding a dependency, ask:

- Can the standard library solve this?
- Is the dependency actively maintained?
- Is it worth the added complexity?

Prefer fewer dependencies.

## Docker

Docker images should be:

- multi-stage
- minimal
- reproducible

Do not run containers as root.

## API Design

REST endpoints should be:

```
/deployments
/deployments/{id}
/projects
/servers
```

Use proper HTTP status codes. Return consistent JSON responses.

## Database

- Write migrations.
- Never modify old migrations.
- Add new migrations.
- Use transactions where appropriate.
- Avoid N+1 queries.

## Configuration

Configuration should come from:

- environment variables
- config files

Never hardcode values.

## CI/CD

Code must pass formatting, linting, and tests before merge.

## Before Finishing Any Task

Claude should verify:

- Builds successfully
- Tests pass
- Lint passes
- Documentation updated
- No dead code
- No commented-out code
- No debugging prints

## Communication

Explain what changed, why, and important design decisions. If multiple
approaches exist, briefly explain why the chosen one is preferred. Do not
make unnecessary architectural changes. Stay within the requested scope.

## Definition of Done

A task is complete only when:

- Feature works
- Code builds
- Tests pass
- Documentation updated
- Branch pushed
- Pull Request ready
