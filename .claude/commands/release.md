---
description: Cut a new release — tag, push, watch the GitHub Actions workflow, verify the published binaries. Asks for the version if not given (e.g. v0.2.0).
---

Cut a new release of `myapp`.

## Prerequisites (verify, abort cleanly otherwise)

- Working tree is clean — run `git status --short`. If anything is dirty, ask the user whether to commit/stash first.
- `gh auth status` succeeds.
- The current branch is `main` and is up to date with `origin/main` (`git fetch && git status -sb`).

## Steps

1. Ask for the new version (e.g. `v0.2.0`) if not provided. It must match `^v\d+\.\d+\.\d+$`.
2. Confirm the tag does not already exist locally or on origin: `git tag -l "$VERSION"`, `gh api repos/fadilxcoder/app-cli/git/refs/tags/$VERSION`.
3. Run `make test` — abort if any test fails.
4. Tag and push: `git tag $VERSION && git push origin $VERSION`.
5. Find the run id with `gh run list -R fadilxcoder/app-cli --workflow=release.yml -L 1`, then `gh run watch <id> -R fadilxcoder/app-cli --exit-status`.
6. After it succeeds, run `gh release view $VERSION -R fadilxcoder/app-cli` and confirm all 5 assets are present:
   - `myapp-linux-amd64`, `myapp-linux-arm64`, `myapp-darwin-amd64`, `myapp-darwin-arm64`, `SHA256SUMS`.
7. Smoke-test the public installer into a temp dir (commit-pinned URL to bypass `raw.githubusercontent.com` cache):
   ```sh
   SHA=$(git rev-parse HEAD)
   TMP=$(mktemp -d)
   curl -fsSL "https://raw.githubusercontent.com/fadilxcoder/app-cli/$SHA/install.sh" \
     | MYAPP_BIN_DIR="$TMP" sh
   "$TMP/myapp" --version  # expect: myapp version $VERSION
   ```
8. Print the release URL.

## On failure

- Workflow red → run `gh run view <id> --log-failed -R fadilxcoder/app-cli` to surface the failing step. Common causes: vet/test regression, asset-upload permission (token scope), Go version drift.
- Asset missing → check the Makefile `release` target and the `files:` block in `.github/workflows/release.yml`.
- Installer 404 → the release is draft; run `gh release edit $VERSION --draft=false -R fadilxcoder/app-cli`.

If the workflow fails after the tag is pushed, prefer rolling forward with a new patch version over deleting + retagging.
