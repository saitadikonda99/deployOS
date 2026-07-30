# packages/

Shared TypeScript packages consumed by the apps in [`apps/`](../apps). Nothing
lives here yet — this directory exists so that code shared between
`dashboard` and `control-plane` (UI components, API clients, shared configs,
type definitions) has an obvious home as soon as it needs to be extracted.

Each package added here should:

- Be published as `@deployos/<name>` and added to the pnpm workspace
  automatically (see [`pnpm-workspace.yaml`](../pnpm-workspace.yaml)).
- Declare its own `package.json`, `tsconfig.json` (extending
  [`tsconfig.base.json`](../tsconfig.base.json)), and `lint`/`build` scripts so
  Turborepo can pipeline it.
