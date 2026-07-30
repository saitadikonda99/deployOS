-- Foundation schema only: no deployment logic references this table yet.
create table if not exists public.projects (
  id uuid primary key default gen_random_uuid(),
  user_id uuid not null references public.users (id) on delete cascade,
  name text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

create index if not exists projects_user_id_idx on public.projects (user_id);

alter table public.projects enable row level security;

create policy "Users can manage their own projects"
  on public.projects for all
  using (auth.uid() = user_id)
  with check (auth.uid() = user_id);

create trigger set_projects_updated_at
  before update on public.projects
  for each row execute function public.set_updated_at();
