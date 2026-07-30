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

This verifies you have the required toolchain (Node.js >= 20, pnpm, Rust
stable) and installs JavaScript dependencies.

## Project layout

See the [repository layout](./README.md#repository-layout) and
[architecture overview](./docs/architecture.md) before making structural
changes. In short:

- TypeScript apps live in `apps/`, shared TypeScript packages in `packages/`.
- Rust crates live in `crates/`.
- Documentation lives in `docs/`.

## Development workflow

```bash
pnpm dev             # run apps in dev mode
pnpm lint            # Biome across the workspace
pnpm format          # Biome + Prettier, auto-fix
pnpm typecheck       # TypeScript project references
pnpm build           # turbo-orchestrated build
pnpm test            # run test suites

cargo fmt --all               # format Rust code
cargo clippy --workspace       # lint Rust code
cargo test --workspace         # run Rust tests
```

A pre-commit hook (via Husky + lint-staged) runs Biome/Prettier on staged
files automatically. CI re-runs the full lint/build/test matrix on every
pull request.

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
- Make sure `pnpm lint`, `pnpm build`, `cargo clippy --workspace`, and
  `cargo test --workspace` all pass locally before requesting review.
- Link any related issue.

## Code style

Formatting and linting are enforced by tooling, not convention:

- TypeScript/JavaScript/JSON: [Biome](https://biomejs.dev/), with Prettier
  covering Markdown and YAML.
- Rust: `rustfmt` and `clippy`, both required by CI.

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
