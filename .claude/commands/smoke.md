---
description: Live smoke test against the configured Supabase project — login, whoami, run-secure, logout. Asks for the test email + password if not provided.
---

Run a full end-to-end happy-path test against the Supabase project configured in `.env`.

## Prerequisites (verify first, abort with a clear message if missing)

- `.env` exists in the repo root and contains `SUPABASE_URL` + `SUPABASE_ANON_KEY`
- Schema is applied — probe with `curl -sS -H "apikey: $KEY" "$URL/rest/v1/roles?select=name&limit=1"`. A 200 means schema is live; 404 with `PGRST205` means run `sql/schema.sql` first.
- The binary is built — run `make build` if `dist/myapp` is missing or older than the source.

## Steps

1. Ask the user for `TEST_EMAIL` and `TEST_PASSWORD` if they weren't passed as arguments.
2. Run `./dist/myapp logout` first to clear any stale local session (suppress the "no local session" error).
3. Run `./dist/myapp login --email "$TEST_EMAIL" --password "$TEST_PASSWORD"`. Show output verbatim. Abort on non-zero exit.
4. Run `./dist/myapp whoami`. Confirm:
   - `Email verified: true`
   - At least one role
   - `can_run_protected_command` in permissions
5. Run `./dist/myapp run-secure`. Expect: `Permission granted: secure action executed`.
6. Run `./dist/myapp logout`.
7. Print a summary: which user, which roles, whether the gate passed.

## On failure

- 400 on login → wrong password or email not in auth.users
- "Access denied: email not verified" → user exists but `email_confirmed_at` is null; tell the user to toggle *Auto Confirm* or click the verification link
- "missing can_run_protected_command" → user has no role; tell them to run `select public.assign_role_by_email('<email>', 'admin')` in the Supabase SQL Editor
- 404 on whoami → schema not applied; tell them to paste `sql/schema.sql` into the SQL Editor

Never commit secrets. Never echo tokens. The session file is mode `0600` — leave it that way.
