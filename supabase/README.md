# supabase/

SQL migrations for the DeployOS control plane's Supabase project, applied
in filename order:

- `20260730120000_users.sql` - a `public.users` profile table kept in
  sync with `auth.users`, plus the shared `set_updated_at()` trigger
  function reused by the other migrations.
- `20260730120100_projects.sql` - foundation schema for projects. Nothing
  in the codebase writes to this table yet.
- `20260730120200_devices.sql` - registered devices. Rows are written by
  the control plane (`internal/devices`) on
  `POST /api/v1/devices/register`; the `id` column has no default
  because the agent generates and supplies it.

## Applying migrations

Using the [Supabase CLI](https://supabase.com/docs/guides/cli):

```bash
supabase link --project-ref <your-project-ref>
supabase db push
```

Or apply the files directly with `psql` against your project's
connection string, in filename order.

## Local development

The control plane needs:

- `DEPLOYOS_SUPABASE_URL` - your project URL.
- `DEPLOYOS_SUPABASE_ANON_KEY` - your project's anon/public API key
  (used only to verify user access tokens against Supabase Auth).
- `DEPLOYOS_SUPABASE_DATABASE_URL` - the Postgres connection string
  (Project Settings -> Database -> Connection string).

See [`.env.example`](../.env.example) and
[`config.example.yaml`](../config.example.yaml) at the repo root.

Row Level Security is enabled on every table here as defense in depth.
The control plane connects with a Postgres role that bypasses RLS (the
`postgres` role in Supabase's default connection string), so these
policies matter primarily if something else - a future dashboard feature
using `supabase-js` directly, for example - ever queries these tables
under a user's own session.
