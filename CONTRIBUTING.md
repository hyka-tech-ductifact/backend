# Contributing Guide

## Workflow overview

- Short-lived topic branches → PR into `main` → deploy to **staging**
- Tag a release on `main` → deploy to **production**
- Hotfixes go to `release` branch → tag → deploy to production → merge back into `main`

---

## 1) Branches

| Branch | Purpose | Example |
|--------|---------|---------|
| `main` | Integration branch, always deployable. Deploys to staging on merge | — |
| `release` | Tracks production. Hotfixes go here. Reset on each new release | — |
| `feat/` | New features | `feat/add-login` |
| `fix/` | Bug fixes | `fix/null-pointer-crash` |
| `chore/` | Everything else (docs, tests, refactor, deps, CI) | `chore/update-deps` |
| `hotfix/` | Urgent production fixes (targets `release`, not `main`) | `hotfix/fix-login-500` |

---

## 2) Day-to-day workflow

```bash
git checkout main && git pull
git checkout -b feat/add-login

# work, commit, push
git push -u origin feat/add-login
```

Open a PR into `main`. Use **squash merge** (1 PR = 1 commit).

Stay up to date before merging:

```bash
git fetch origin && git rebase origin/main
```

---

## 3) Releases

Use **SemVer**: `vMAJOR.MINOR.PATCH`

```bash
# 1. Create release branch
git checkout main && git pull
git checkout -b chore/release-v0.4.0

# 2. Generate changelog
make changelog VERSION=v0.4.0
git add CHANGELOG.md && git commit -m "chore(release): v0.4.0"
git push -u origin chore/release-v0.4.0

# 3. Open PR "chore(release): v0.4.0" → squash merge into main

# 4. Tag the merged commit
git checkout main && git pull
git tag -a v0.4.0 -m "Release v0.4.0"
git push origin v0.4.0

# 5. Reset release branch
git checkout -B release v0.4.0
git push origin release --force-with-lease
```

The `make tag` command creates the annotated tag and pushes it.
Production deploys from the **tag**, not from `main`.

---

## 4) Hotfixes

For urgent production bugs — branch from `release`, not `main`:

```bash
git checkout -b hotfix/fix-login-500 origin/release

# fix, commit, push, PR into release

git checkout release && git pull
git tag -a v0.4.1 -m "Hotfix v0.4.1"
git push origin v0.4.1

# merge back into main
git checkout main && git pull
git merge release && git push origin main
```

```
main:      A ── B ── C ── H1' ── D ── E
                                        ↑ tag v0.5.0, release resets here

release:   v0.4.0 ── H1 ── H2
                      ↑      ↑
                   v0.4.1  v0.4.2
```

### Summary

| Situation | Branch from | PR target | Tag on | Deploys to |
|-----------|-------------|-----------|--------|------------|
| Feature / fix | `main` | `main` | — | staging |
| Release | — | — | `main` | production |
| Hotfix | `release` | `release` | `release` | production |
| Hotfix backport | — | `main` | — | staging |

---

## 5) Commit messages & PR titles

We use [Conventional Commits](https://www.conventionalcommits.org/). Since we do
**squash merge**, the PR title becomes the commit message on `main`. CI validates
the PR title format automatically.

### Format

```
<type>(<scope>): <description>
```

### Types

| Type | When to use | In changelog? |
|------|------------|---------------|
| `feat` | New feature | ✅ |
| `fix` | Bug fix | ✅ |
| `chore` | Everything else (docs, deps, refactor, CI, tests...) | ❌ |

### Examples

```
feat(auth): add refresh token rotation
fix(client): prevent duplicate client names
chore: update Go to 1.24
feat(api)!: change error response format   ← breaking change
```

### Breaking changes

Add `!` after the scope to indicate a breaking change:

```
feat(api)!: change pagination response format
```

---

## 6) PR rules

- No direct pushes to `main`
- CI must pass (tests, lint, build, **PR title validation**)
- PR title must follow Conventional Commits format (see §5)
- Keep PRs small and focused
- Use **squash merge** (1 PR = 1 commit)
