-- Devices are registered by the DeployOS control plane on behalf of an
-- agent (see internal/devices and POST /api/v1/devices/register); the
-- id is generated and persisted by the agent itself, so it has no
-- default here.
create table if not exists public.devices (
  id uuid primary key,
  user_id uuid not null references public.users (id) on delete cascade,
  hostname text not null,
  operating_system text not null,
  architecture text not null,
  cpu_cores integer not null check (cpu_cores > 0),
  memory_bytes bigint not null check (memory_bytes > 0),
  deployos_version text not null,
  status text not null default 'registered',
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists devices_user_id_idx on public.devices (user_id);

alter table public.devices enable row level security;

create policy "Users can view their own devices"
  on public.devices for select
  using (auth.uid() = user_id);

create trigger set_devices_updated_at
  before update on public.devices
  for each row execute function public.set_updated_at();
