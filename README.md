# myapp

A Go CLI that authenticates against a hosted Supabase project and gates a demo command on email-verification + a database-backed permission.

```
myapp login        email + password → token in ~/.myapp/config.json (mode 0600)
myapp whoami       show user, roles, permissions
myapp run-secure   succeeds only if email verified AND has can_run_protected_command
myapp logout       revoke server-side + clear local file
```

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/fadilxcoder/app-cli/main/install.sh | sh
```

Or build from source:

```sh
make build && sudo make install
```

## Setup

1. Create a project at https://supabase.com.
2. Open SQL Editor, paste [`sql/schema.sql`](sql/schema.sql), run it.
3. Authentication → Users → **Add user** (check *Auto Confirm* for fast testing).
4. SQL Editor: `select public.assign_role_by_email('you@example.com', 'admin');`
5. Project Settings → API → copy `Project URL` and **`anon` public** key into `.env`:

```
SUPABASE_URL=https://<ref>.supabase.co
SUPABASE_ANON_KEY=eyJ...
```

Never commit `.env` or paste the `service_role` key anywhere — only the `anon` key.

## Use

```sh
myapp login --email you@example.com
myapp whoami
myapp run-secure   # → "Permission granted: secure action executed"
```

## Permission matrix

| Role  | Permissions                                          |
|-------|------------------------------------------------------|
| admin | `can_run_protected_command`, `can_view_admin_panel`  |
| user  | `can_run_protected_command`                          |

## Development

```sh
make tidy && make test     # offline tests
make build                 # ./dist/myapp
make release               # cross-build linux+darwin amd64+arm64 + SHA256SUMS
```

CI runs on every push; tagging `v*` triggers the release workflow which uploads binaries to GitHub Releases.

## Layout

```
cmd/myapp/        entrypoint
internal/auth/    local session (~/.myapp/config.json)
internal/cli/     cobra commands
internal/config/  env / .env loader
internal/permissions/  role → permission resolution
internal/supabase/     GoTrue + PostgREST clients
pkg/httpclient/   tiny JSON-over-HTTP helper
sql/schema.sql    tables, RLS, seed, helper functions
```
