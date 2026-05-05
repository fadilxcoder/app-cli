-- =============================================================================
-- myapp — Supabase schema, RLS policies, and seed data
-- =============================================================================
-- Apply via Supabase SQL Editor on your hosted project.
--
-- Tables:
--   public.profiles          (1:1 with auth.users)
--   public.roles             (catalog)
--   public.permissions       (catalog)
--   public.user_roles        (auth.users.id -> roles.id)
--   public.role_permissions  (roles.id -> permissions.id)
--
-- RLS allows authenticated users to read THEIR OWN role/permission rows so the
-- CLI can compute permissions client-side using the user's access token.
-- Mutations are blocked by default — assign roles via the service_role key
-- (e.g. from the Supabase dashboard or a trusted backend).
-- =============================================================================

-- ----------------------------------------------------------------------------
-- 1. Profiles
-- ----------------------------------------------------------------------------
create table if not exists public.profiles (
    id           uuid primary key references auth.users (id) on delete cascade,
    email        text not null,
    full_name    text,
    created_at   timestamptz not null default now(),
    updated_at   timestamptz not null default now()
);

create index if not exists profiles_email_idx on public.profiles (email);

-- Auto-create a profile row whenever a new auth user is created.
create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
begin
    insert into public.profiles (id, email)
    values (new.id, new.email)
    on conflict (id) do nothing;
    return new;
end;
$$;

drop trigger if exists on_auth_user_created on auth.users;
create trigger on_auth_user_created
    after insert on auth.users
    for each row execute function public.handle_new_user();

-- ----------------------------------------------------------------------------
-- 2. Roles & Permissions catalog
-- ----------------------------------------------------------------------------
create table if not exists public.roles (
    id          bigserial primary key,
    name        text not null unique,
    description text,
    created_at  timestamptz not null default now()
);

create table if not exists public.permissions (
    id          bigserial primary key,
    name        text not null unique,
    description text,
    created_at  timestamptz not null default now()
);

-- ----------------------------------------------------------------------------
-- 3. Role <-> Permission join
-- ----------------------------------------------------------------------------
create table if not exists public.role_permissions (
    role_id       bigint not null references public.roles (id)       on delete cascade,
    permission_id bigint not null references public.permissions (id) on delete cascade,
    created_at    timestamptz not null default now(),
    primary key (role_id, permission_id)
);

create index if not exists role_permissions_permission_idx on public.role_permissions (permission_id);

-- ----------------------------------------------------------------------------
-- 4. User <-> Role join
-- ----------------------------------------------------------------------------
create table if not exists public.user_roles (
    user_id    uuid   not null references auth.users (id) on delete cascade,
    role_id    bigint not null references public.roles (id) on delete cascade,
    created_at timestamptz not null default now(),
    primary key (user_id, role_id)
);

create index if not exists user_roles_role_idx on public.user_roles (role_id);

-- ----------------------------------------------------------------------------
-- 5. Row-Level Security
-- ----------------------------------------------------------------------------
alter table public.profiles         enable row level security;
alter table public.roles            enable row level security;
alter table public.permissions      enable row level security;
alter table public.role_permissions enable row level security;
alter table public.user_roles       enable row level security;

-- profiles: a user can read & update their own row.
drop policy if exists "profiles_select_self"  on public.profiles;
drop policy if exists "profiles_update_self"  on public.profiles;
create policy "profiles_select_self" on public.profiles
    for select using (auth.uid() = id);
create policy "profiles_update_self" on public.profiles
    for update using (auth.uid() = id);

-- user_roles: a user can read their own assignments.
drop policy if exists "user_roles_select_self" on public.user_roles;
create policy "user_roles_select_self" on public.user_roles
    for select using (auth.uid() = user_id);

-- roles & permissions catalogs are readable by any authenticated user
-- (no sensitive data — just labels).
drop policy if exists "roles_select_authenticated"       on public.roles;
drop policy if exists "permissions_select_authenticated" on public.permissions;
create policy "roles_select_authenticated" on public.roles
    for select to authenticated using (true);
create policy "permissions_select_authenticated" on public.permissions
    for select to authenticated using (true);

-- role_permissions: an authenticated user may read entries for roles they hold.
drop policy if exists "role_permissions_select_for_user_roles" on public.role_permissions;
create policy "role_permissions_select_for_user_roles" on public.role_permissions
    for select to authenticated using (
        exists (
            select 1 from public.user_roles ur
            where ur.role_id = role_permissions.role_id
              and ur.user_id = auth.uid()
        )
    );

-- ----------------------------------------------------------------------------
-- 6. Seed data
-- ----------------------------------------------------------------------------
insert into public.roles (name, description) values
    ('admin', 'Full administrative access'),
    ('user',  'Standard authenticated user')
on conflict (name) do nothing;

insert into public.permissions (name, description) values
    ('can_run_protected_command', 'May execute the demo `myapp run-secure` command'),
    ('can_view_admin_panel',      'May view administrative dashboards')
on conflict (name) do nothing;

-- admin: every permission
insert into public.role_permissions (role_id, permission_id)
select r.id, p.id
from public.roles r
cross join public.permissions p
where r.name = 'admin'
on conflict do nothing;

-- user: only the demo permission
insert into public.role_permissions (role_id, permission_id)
select r.id, p.id
from public.roles r
join public.permissions p on p.name = 'can_run_protected_command'
where r.name = 'user'
on conflict do nothing;

-- ----------------------------------------------------------------------------
-- 7. Helper to assign a role by email (run with service_role)
-- ----------------------------------------------------------------------------
-- Example: select public.assign_role_by_email('alice@example.com', 'admin');
create or replace function public.assign_role_by_email(p_email text, p_role text)
returns void
language plpgsql
security definer
set search_path = public
as $$
declare
    v_user_id uuid;
    v_role_id bigint;
begin
    select id into v_user_id from auth.users where email = p_email;
    if v_user_id is null then
        raise exception 'no auth user with email %', p_email;
    end if;

    select id into v_role_id from public.roles where name = p_role;
    if v_role_id is null then
        raise exception 'no role named %', p_role;
    end if;

    insert into public.user_roles (user_id, role_id)
    values (v_user_id, v_role_id)
    on conflict do nothing;
end;
$$;
