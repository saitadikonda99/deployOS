# Device registration

The first real DeployOS feature: an agent registers the machine it runs
on with the control plane, which persists it in Supabase and issues the
agent a signed device token.

## Architecture

```
Agent  --->  DeployOS API (control plane)  --->  Supabase
```

The agent talks only to the DeployOS control plane's HTTP API. It never
holds Supabase credentials and never calls Supabase directly - the
control plane (`cmd/server`, via `internal/auth` and `internal/devices`)
is the only component that does.

## Flow

1. On startup, the agent (`internal/agent`) loads its device ID from
   `<data_dir>/device_id`, generating and persisting a new UUID if the
   file doesn't exist yet. This is what lets the agent reuse the same
   identity across restarts.
2. It collects system info: hostname, OS, architecture, CPU core count,
   and total memory (`internal/agent/sysinfo.go`).
3. It calls `POST /api/v1/devices/register` on the control plane
   (`internal/agent/registrar.go`), sending the device ID, system info,
   and its configured user access token as a bearer token.
4. The control plane (`internal/devices.Handler`) authenticates that
   token against Supabase Auth (`internal/auth.SupabaseAuthenticator`,
   which calls `GET /auth/v1/user`), validates the request body
   (`internal/devices.Service`), and upserts the device into Supabase's
   `devices` table (`internal/devices.PostgresRepository`). Re-registering
   the same device ID is idempotent as long as the owner doesn't change;
   a different owner gets `409 Conflict`.
5. The control plane issues a signed device token (HS256 JWT,
   `internal/devices.JWTTokenIssuer`) and returns it along with the
   device ID.
6. The agent persists the token to `<data_dir>/device_token`.

Verifying that token is intentionally not implemented yet - there's
nothing to authenticate with it until the heartbeat feature lands.

## Endpoints

### `POST /api/v1/devices/register`

Authenticated with `Authorization: Bearer <supabase-user-access-token>`.

Request body:

```json
{
  "device_id": "11111111-1111-1111-1111-111111111111",
  "hostname": "my-server",
  "operating_system": "linux",
  "architecture": "amd64",
  "cpu_cores": 8,
  "memory_bytes": 17179869184,
  "deployos_version": "0.1.0"
}
```

Response (`201 Created`):

```json
{
  "device_id": "11111111-1111-1111-1111-111111111111",
  "token": "<signed JWT>",
  "expires_at": "2027-07-30T12:00:00Z"
}
```

`400` for invalid input, `401` for a missing/invalid access token, `409`
if `device_id` is already registered to a different user.

### `GET /api/v1/devices`

Authenticated the same way. Returns every device owned by the
authenticated user - this is what the dashboard's Devices page reads.

```json
{
  "devices": [
    {
      "id": "11111111-1111-1111-1111-111111111111",
      "hostname": "my-server",
      "operating_system": "linux",
      "architecture": "amd64",
      "status": "registered",
      "created_at": "2026-07-30T12:00:00Z"
    }
  ]
}
```

## Configuration

See [`.env.example`](../.env.example) and
[`config.example.yaml`](../config.example.yaml) for the full list. In
summary:

- **Agent**: `agent.data_dir`, `agent.api_base_url`,
  `agent.user_access_token`.
- **Control plane**: `supabase.url`, `supabase.anon_key`,
  `supabase.database_url`, `device_token.secret`, `device_token.ttl`. The
  control plane fails fast at startup if any of these are unset.

## Database

See [`supabase/migrations/`](../supabase/migrations/) and
[`supabase/README.md`](../supabase/README.md) for the `users`, `devices`,
and `projects` tables and how to apply them.

## Known limitation: no real login flow yet

Both the agent's `user_access_token` and the dashboard's
`DEPLOYOS_API_TOKEN` (see
[`apps/dashboard`](../apps/dashboard/src/lib/devices.ts)) are Supabase
user access tokens obtained out-of-band today (e.g. via Supabase Studio
or a Supabase client library), not through a DeployOS login flow. Adding
one - so an operator can sign in once and have both the agent and
dashboard pick up a session automatically - is future work, not part of
this feature.
