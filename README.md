# myapp

A production-ready Go CLI demonstrating Supabase email/password auth, email-verification gating, and DB-backed role/permission checks against a **hosted** Supabase project (no local Supabase).

```
myapp login        # email + password → stores token in ~/.myapp/config.json
myapp whoami       # show user, roles, permissions
myapp run-secure   # gated demo command
myapp logout       # clears local + server-side session
```

---

## 1. Project layout

```
.
├── cmd/myapp/                 # entrypoint
├── internal/
│   ├── auth/                  # local session persistence
│   ├── config/                # env loading + paths
│   ├── supabase/              # GoTrue (auth) + PostgREST (data) clients
│   ├── permissions/           # permission service
│   └── cli/                   # cobra commands (login, logout, whoami, run-secure)
├── pkg/httpclient/            # tiny JSON-over-HTTP helper
├── sql/schema.sql             # Supabase schema, RLS, seed data
├── install.sh                 # one-shot installer
├── Makefile
└── README.md
```

## 2. Prerequisites

- Go ≥ 1.22 to build from source
- A hosted Supabase project (https://supabase.com)

## 3. Supabase configuration

1. Create a project at https://supabase.com and grab the API URL + `anon` key from **Project Settings → API**.
2. Open **SQL Editor** and run [`sql/schema.sql`](sql/schema.sql) — this creates `profiles`, `roles`, `permissions`, `user_roles`, `role_permissions`, plus indexes, RLS policies, and seed data.
3. Create users in **Authentication → Users**:
   - Use **"Send invite"** if you want them to receive a confirmation email (production-like).
   - Use **"Add user"** with *Auto Confirm User* checked to skip verification while developing.
4. Assign a role to each user. From the SQL Editor (any service-role context):
   ```sql
   select public.assign_role_by_email('alice@example.com', 'admin');
   select public.assign_role_by_email('bob@example.com',   'user');
   ```

The seeded permission matrix:

| Role  | Permissions                                                |
|-------|------------------------------------------------------------|
| admin | `can_run_protected_command`, `can_view_admin_panel`        |
| user  | `can_run_protected_command`                                |

## 4. Local configuration

The CLI reads two environment variables and also auto-loads a `.env` file in the working directory:

```sh
SUPABASE_URL=https://<your-project-ref>.supabase.co
SUPABASE_ANON_KEY=<your-anon-key>
```

Copy `.env.example` to `.env` to get started.

The local session is persisted to **`~/.myapp/config.json`** (mode `0600`).

## 5. Build & run

```sh
make tidy              # fetch dependencies
make build             # ./dist/myapp for current OS/arch
./dist/myapp --help

# install locally
sudo make install      # copies to /usr/local/bin/myapp
```

Cross-compile for releases:

```sh
make release           # dist/{linux,darwin}-{amd64,arm64} + SHA256SUMS
```

## 6. CLI usage

```sh
# 1. authenticate
myapp login --email alice@example.com
# (prompts for password)

# 2. inspect session
myapp whoami
# User:           alice@example.com
# ID:             8c9d...
# Email verified: true
# Roles:          admin
# Permissions:    can_run_protected_command, can_view_admin_panel

# 3. run the gated command
myapp run-secure
# Permission granted: secure action executed

# 4. log out (also revokes server-side)
myapp logout
```

### Failure modes you can test

| Scenario                                  | Expected output                                              |
|-------------------------------------------|--------------------------------------------------------------|
| Not logged in                             | `not logged in — run `myapp login` first`                    |
| Logged in, email **not** verified         | `Access denied: email not verified`                          |
| Verified, but no `can_run_protected_command` | `Access denied: permission denied: missing "can_run_protected_command"` |
| Verified + permission granted             | `Permission granted: secure action executed`                 |

To force the "no permission" path, add a third user with no `user_roles` row (or remove the seed link for the `user` role).

## 7. Releasing on GitHub

1. Tag and push: `git tag v0.1.0 && git push --tags`
2. Build artifacts: `make release` (produces `dist/myapp-{linux,darwin}-{amd64,arm64}` + `SHA256SUMS`)
3. Create a GitHub release for the tag and upload the four binaries plus `SHA256SUMS`.

The included `install.sh` resolves `https://github.com/<owner>/<repo>/releases/latest/download/myapp-<os>-<arch>`, so as long as the asset names match your release, the installer works untouched.

## 8. Hosting the installer

You have two free options:

- **GitHub raw URL** (recommended — no extra setup):
  ```sh
  curl -fsSL https://raw.githubusercontent.com/<owner>/<repo>/main/install.sh | sh
  ```
- **GitHub Pages** (custom domain / vanity URL): enable Pages on `main` for `/` (or `/docs`) and serve `install.sh` from the resulting `https://<owner>.github.io/<repo>/install.sh` URL.

Override defaults at install time:

```sh
curl -fsSL https://raw.githubusercontent.com/<owner>/<repo>/main/install.sh \
  | MYAPP_REPO=acme/myapp MYAPP_VERSION=v0.2.0 MYAPP_BIN_DIR=$HOME/.local/bin sh
```

## 9. Security notes

- The CLI uses only the **anon** key plus the user's own access token; RLS prevents cross-user reads.
- Tokens are stored at `~/.myapp/config.json` with mode `0600`.
- `myapp logout` calls `/auth/v1/logout` to revoke the session on Supabase, then deletes the local file.
- No credentials are ever hardcoded — all configuration comes from env / `.env`.
