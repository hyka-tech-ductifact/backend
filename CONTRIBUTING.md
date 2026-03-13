# Contributing Guide

This backend repository uses a QA-friendly workflow:

- Short-lived topic branches (`feat/*`, `fix/*`, etc.)
- Pull Requests into `main`
- Automatic deploy to **staging** from `main`
- Promotion to **production** by tagging a release from `main`

The goal is to keep `main` always in a deployable state, while giving QA a stable environment (staging) to validate changes before production.

---

## 1) Branches and naming

- `main`: integration branch; always expected to be deployable.
- `release`: tracks what's in production; hotfixes go here. There is only
  one `release` branch at any time — it always points to the current
  production version. Version history is preserved by tags, not by branches.
- Topic branches (short-lived):
  - `feat/<short-topic>`
  - `fix/<short-topic>`
  - `chore/<short-topic>`
  - `docs/<short-topic>`
  - `hotfix/<short-topic>` (only for urgent production fixes — targets `release`, not `main`)

Examples:

- `feat/payment-webhook`
- `fix/auth-npe`
- `hotfix/critical-login-500`

---

## 2) Environments (Dev / Staging / Production)

- **Dev**: local development (your laptop).
- **Staging**: pre-production environment for QA. It should be as close as practical to production:
  - similar runtime and dependencies
  - separate secrets/credentials
  - safe data (seeded/anonymous), not real user data
- **Production**: real environment serving end users.

**Key rule:**

- Every merge to `main` deploys to **staging**.
- Only tagged releases are deployed to **production**.

---

## 3) Pull Request (PR) process

### Required gates (recommended)

Protect `main` with:

- No direct pushes to `main`
- At least 1 approval
- CI checks required (tests/lint/build)

### Keeping PRs small

- Prefer PRs that can be reviewed in 5–20 minutes.
- If a feature is large, split it into incremental PRs.

### Merge strategy

Use **Squash merge** for PRs into `main`:

- 1 PR → 1 commit on `main`
- Easier rollbacks and simpler history

---

## 4) Day-to-day workflow

### Create a branch

```bash
git checkout main
git pull

git checkout -b feat/add-login
```

### Commit and push

```bash
git add -A
git commit -m "feat: add login endpoint"

git push -u origin feat/add-login
```

### Stay up to date with `main`

Before opening the PR (or whenever you fall behind):

```bash
git fetch origin
git rebase origin/main
```

If you prefer merges instead of rebases, agree as a team and do it consistently.

---

## 5) QA on staging

Once your PR is merged into `main`, it will be deployed to **staging**.

QA validates on staging:

- smoke tests
- regression checks
- feature acceptance
- migrations (if any)

If QA finds issues:

- open a `fix/*` branch
- PR to `main`
- validate again on staging after merge

---

## 6) Releases and versioning

### Versioning

Use **SemVer** tags:

- `vMAJOR.MINOR.PATCH` (example: `v0.4.0`)
- `PATCH`: bugfixes
- `MINOR`: backward-compatible features
- `MAJOR`: breaking changes

### Release flow (promote `main` to production)

When staging is approved by QA, create a tag and update the `release` branch:

```bash
git checkout main
git pull

# Tag the release
git tag -a v0.4.0 -m "Release v0.4.0"
git push origin v0.4.0

# First release? Create the branch. Otherwise, fast-forward it.
git checkout -B release v0.4.0
git push origin release --force-with-lease
```

Production deploys from the tag (`v0.4.0`), not from "latest `main`".
The `release` branch exists so that hotfixes can be applied without
pulling in unvalidated code from `main` (see section 7).

> `git checkout -B release v0.4.0` creates the branch if it doesn't exist,
> or resets it to the tag if it already exists. This way there's always
> a single `release` branch pointing to the current production version.

---

## 7) Release branch and hotfixes

### Why a release branch?

When you tag `v0.4.0` and deploy to production, `main` keeps moving forward
with new PRs that QA hasn't validated yet. If a production bug appears, you
can't tag from `main` — it contains unvalidated code.

The `release` branch is a single, long-lived branch that always tracks
what's in production. Hotfixes go there, get tagged, and production deploys
from those tags. Meanwhile, `main` keeps evolving independently.

Version history is preserved by **tags** (v0.4.0, v0.4.1, v0.5.0...), not
by branch names. There's never more than one `release` branch.

```
main:      A ── B ── C ── H1' ── H2' ── D ── E
                                                ↑
                                           QA validates
                                           tag v0.5.0
                                           release branch moves here

release:   v0.4.0 ── H1 ── H2              v0.5.0
                      ↑      ↑              ↑
                   v0.4.1  v0.4.2     (branch reset to v0.5.0)

(H1', H2' = hotfixes merged back into main so they're not lost)
(tags v0.4.0, v0.4.1, v0.4.2 remain in git history forever)
```

### Lifecycle of the release branch

1. QA validates staging → tag `v0.4.0` on `main` → `release` points here
2. Production deploys from the tag
3. If production breaks → hotfix on `release` → tag `v0.4.1`
4. More hotfixes? → same branch → tag `v0.4.2`, `v0.4.3`, etc.
5. QA validates the next batch on staging → tag `v0.5.0` → `release` resets to `v0.5.0`
6. Old tags (v0.4.x) stay in git for history and rollback

### Hotfix flow (single hotfix)

1. Branch from `release` (not from `main`)
2. Fix, test, PR into `release`
3. Tag the new PATCH version on `release`
4. Merge the fix back into `main` so it's not lost

```bash
# 1. Create the hotfix branch from the release branch
git fetch origin
git checkout -b hotfix/fix-login-500 origin/release

# 2. Fix and test
git add -A
git commit -m "fix: prevent login 500 when user missing"
git push -u origin hotfix/fix-login-500

# 3. Open PR → merge into release (not main!)
#    After merge:
git checkout release
git pull

# 4. Tag the hotfix release on the release branch
git tag -a v0.4.1 -m "Hotfix v0.4.1"
git push origin v0.4.1
# → Production deploys v0.4.1 (only v0.4.0 + the fix, no unvalidated code)

# 5. Merge the fix back into main
git checkout main
git pull
git merge release
git push origin main
# → Staging gets updated with the fix too
```

### Multiple hotfixes

If another production bug appears before the next release, repeat the same
process — hotfixes accumulate on `release`:

```bash
git checkout -b hotfix/fix-signup-timeout origin/release
# fix, test, PR into release

git checkout release
git pull
git tag -a v0.4.2 -m "Hotfix v0.4.2"
git push origin v0.4.2

# Merge back into main
git checkout main
git pull
git merge release
git push origin main
```

The `release` branch accumulates hotfixes until the next release from
`main` resets it (see section 6, release flow).

### Summary: where does each thing go?

| Situation | Branch from | PR target | Tag on | Deploys to |
|-----------|-------------|-----------|--------|------------|
| New feature / fix | `main` | `main` | — | staging |
| New release | — | — | `main` | production |
| Hotfix | `release` | `release` | `release` | production |
| Hotfix backport | — | `main` | — | staging |

---

## 8) Commit message convention (recommended)

Use a simple Conventional Commits style:

- `feat: ...`
- `fix: ...`
- `chore: ...`
- `docs: ...`
- `test: ...`
- `refactor: ...`

This helps generate changelogs later if you choose.
