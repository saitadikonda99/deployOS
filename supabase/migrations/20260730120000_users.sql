-- Profile table extending Supabase's built-in auth.users. Application
-- tables (devices, projects) reference this table rather than auth.users
-- directly, so application data stays in the public schema.
create table if not exists public.users (
  id uuid primary key references auth.users (id) on delete cascade,
  email text not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now()
);

alter table public.users enable row level security;

create policy "Users can view their own profile"
  on public.users for select
  using (auth.uid() = id);

create policy "Users can update their own profile"
  on public.users for update
  using (auth.uid() = id);

-- Shared trigger function: stamps updated_at on every UPDATE. Reused by
-- the projects and devices migrations.
create or replace function public.set_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

create trigger set_users_updated_at
  before update on public.users
  for each row execute function public.set_updated_at();

-- Keeps public.users in sync with auth.users, so a row always exists
-- here for every authenticated user without the application having to
-- create it explicitly.
create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer set search_path = public
as $$
begin
  insert into public.users (id, email)
  values (new.id, new.email)
  on conflict (id) do update set email = excluded.email, updated_at = now();
  return new;
end;
$$;

create or replace trigger on_auth_user_created
  after insert or update of email on auth.users
  for each row execute function public.handle_new_user();
