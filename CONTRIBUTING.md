# Contributing to DeployOS

Thanks for your interest in contributing. DeployOS is infrastructure
software — it runs unattended on real machines — so contributions are held
to a correspondingly high bar: no placeholder code, no fake
implementations, and everything that lands on `main` must install, lint,
and build.

## Getting started

```bash
git clone git@github.com:saitadikonda99/deployOS.git
cd deployOS
./scripts/bootstrap.sh
```

This verifies you have the required toolchain (Node.js >= 20, pnpm, Go
stable) and installs JavaScript dependencies.

## Project layout

See the [repository layout](./README.md#repository-layout) and
[architecture overview](./docs/architecture.md) before making structural
changes. In short:

- The dashboard (TypeScript) lives in `apps/dashboard`.
- Every other component - control plane, agent, CLI - is Go, in a single
  module: entry points in `cmd/`, implementation in `internal/`, and types
  shared across binaries in `pkg/`.
- Documentation lives in `docs/`.

## Development workflow

```bash
pnpm dev             # run the dashboard in dev mode
pnpm lint            # Biome across the JS/TS workspace
pnpm format          # Biome + Prettier, auto-fix
pnpm typecheck       # TypeScript project references
pnpm build           # turbo-orchestrated build
pnpm test            # run JS/TS test suites

go build ./...              # build every Go binary
go vet ./...                  # Go's static checks
gofmt -l .                     # list any unformatted Go files
golangci-lint run ./...        # lint Go code
go test ./...                  # run Go tests
```

A pre-commit hook (via Husky + lint-staged) runs Biome/Prettier on staged
files automatically. CI re-runs the full lint/build/test matrix on every
pull request.

### Changing the protocol

If your change touches anything under `proto/`, see
[`docs/protocol.md`](./docs/protocol.md) for the full workflow. In short:

```bash
buf lint                                      # style/API-design checks
buf breaking --against '.git#branch=main'     # catch accidental breaking changes
buf generate                                  # regenerate gen/go/deployos/v1
```

Commit the regenerated files in `gen/` alongside your `.proto` change -
CI fails if they're out of sync.

## Configuration

The Go binaries (`cmd/agent`, `cmd/server`) read configuration from, in
ascending precedence: defaults, `config.yaml`, `.env`, then environment
variables. Copy [`.env.example`](./.env.example) to `.env` and/or
[`config.example.yaml`](./config.example.yaml) to `config.yaml` to set
local overrides; both are gitignored.

`cmd/server` additionally requires a Supabase project - see
[`supabase/README.md`](./supabase/README.md) and
[`docs/device-registration.md`](./docs/device-registration.md).

### Running the Postgres integration test

`internal/devices` includes a real-database integration test
(`postgres_integration_test.go`) alongside its unit tests. It's skipped
by default; to run it, apply the migrations in
[`supabase/migrations/`](./supabase/migrations/) to a Postgres database
and set:

```bash
DEPLOYOS_TEST_DATABASE_URL=postgres://... go test ./internal/devices/...
```

## Commit messages

This project uses [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(optional scope): <description>

[optional body]
```

Common types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`,
`build`. Example:

```
feat(agent): report memory usage in heartbeat
```

## Changesets

User-facing changes to a published package or app should include a
changeset:

```bash
pnpm changeset
```

Follow the prompts to describe the change and its semver impact. Commit the
generated file in `.changeset/` alongside your PR.

## Pull requests

- Keep PRs focused on a single change; large unrelated changes are harder
  to review and to revert.
- Fill out the pull request template, including the test plan.
- Make sure `pnpm lint`, `pnpm build`, `golangci-lint run ./...`, and
  `go test ./...` all pass locally before requesting review.
- Link any related issue.

## Code style

Formatting and linting are enforced by tooling, not convention:

- TypeScript/JavaScript/JSON: [Biome](https://biomejs.dev/), with Prettier
  covering Markdown and YAML.
- Go: `gofmt` and [`golangci-lint`](https://golangci-lint.run/), both
  required by CI. `cmd/` packages should stay thin; real logic belongs in
  `internal/`.

Don't hand-format code to work around the linter; fix the underlying issue
or, if the rule is genuinely wrong for a case, discuss it in your PR.

## Reporting bugs and requesting features

Please use the issue templates under **New Issue** on GitHub rather than
opening a blank issue — they collect the information needed to triage
quickly.

## Code of Conduct

By participating in this project you agree to abide by the
[Code of Conduct](./CODE_OF_CONDUCT.md).

## Security

Do not open a public issue for security vulnerabilities. See
[SECURITY.md](./SECURITY.md) for how to report them.
