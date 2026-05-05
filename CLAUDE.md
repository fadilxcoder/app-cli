# CLAUDE.md

Guidance for Claude Code sessions working on this repo.

## What this is

`myapp` — a Go CLI that authenticates against a **hosted** Supabase project (no local Supabase) and gates a demo command on email verification + a DB-backed permission. Distributed as static binaries via GitHub Releases.

## Stack

- Go 1.22+, stdlib + `cobra`, `joho/godotenv`, `golang.org/x/term`
- Supabase: GoTrue (auth) + PostgREST (data) — REST only, no SDK
- Build: `make` (cross-builds Linux + Darwin, amd64 + arm64)
- CI: `.github/workflows/{ci,release}.yml`

## Build / test / run

```sh
make tidy            # fetch deps
make test            # offline tests (httptest)
make build           # ./dist/myapp for current host
make release         # all 4 binaries + SHA256SUMS
./dist/myapp --help
```

Tests are 100% offline — they spin up `httptest.Server`s, never hit real Supabase.

## Live testing

Requires `.env` with `SUPABASE_URL` + `SUPABASE_ANON_KEY` (anon, not service_role). Schema must be applied (`sql/schema.sql`) and at least one auth user must have a role assigned via `select public.assign_role_by_email('email', 'admin')`. The recipe is in `.claude/commands/smoke.md` — invoke with `/smoke`.

## Repo layout

```
cmd/myapp/main.go             entrypoint (Version overridable via -ldflags)
internal/auth/                Session struct + ~/.myapp/config.json (mode 0600)
internal/cli/                 cobra commands; root.go owns app + requireSession (with token refresh)
internal/config/              env / .env loading, ConfigDir/ConfigFilePath helpers
internal/permissions/         Service.Has / Require / List
internal/supabase/auth.go     SignInWithPassword, GetUser, RefreshSession, Logout
internal/supabase/data.go     ListUserPermissions / ListUserRoles via embedded select
pkg/httpclient/               JSON-over-HTTP helper with typed *Error
sql/schema.sql                tables, FKs, indexes, RLS, seed, assign_role_by_email()
```

## Conventions

- **Errors:** wrap with `fmt.Errorf("...: %w", err)`. Surface `*httpclient.Error` to the user as-is — its `Body` field carries Supabase's JSON error.
- **Contexts:** every command sets a 10–30s timeout via `context.WithTimeout(cmd.Context(), …)`.
- **Tokens:** never log access/refresh tokens. Session file is mode `0600`.
- **No new top-level packages** unless it earns its keep — `pkg/` is reserved for things that could plausibly be imported externally.
- **Commits:** prefer NEW commits over `--amend` (the rare exception is fixing the un-pushed initial commit).

## Hard rules

- ❌ Never use a local Supabase instance — this CLI is REST-only against a hosted project.
- ❌ Never commit `.env` or any `service_role` JWT (`role:service_role` in the payload).
- ❌ Never bypass RLS by using the `service_role` key in the CLI — only `anon` + the user's own JWT.
- ❌ Never skip the email-verification check in `run-secure`.

## Useful slash commands

- `/smoke` — full live login → whoami → run-secure → logout against the configured project
- `/release` — bump tag, push, watch the release workflow

## Useful URLs (this project)

- Repo: https://github.com/fadilxcoder/app-cli
- Releases: https://github.com/fadilxcoder/app-cli/releases
- Supabase project URL is in `.env` (gitignored)

## When changing the schema

If you touch `sql/schema.sql`, also update:
- The seed data block (roles, permissions, role_permissions joins)
- The "Permission matrix" table in README.md
- The integration with `internal/permissions/service.go` if a permission name changes
- Any new permission constant in `internal/permissions/service.go`

Schema changes are not migrations — they're idempotent (uses `if not exists` / `on conflict do nothing`). Apply by re-running the whole file in the Supabase SQL Editor.
