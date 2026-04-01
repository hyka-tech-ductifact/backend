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
git checkout main && git pull

git tag -a v0.4.0 -m "Release v0.4.0"
git push origin v0.4.0

git checkout -B release v0.4.0
git push origin release --force-with-lease
```

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

## 5) Commit messages

[Conventional Commits](https://www.conventionalcommits.org/):

`feat:`, `fix:`, `chore:`

---

## 6) PR rules

- No direct pushes to `main`
- CI must pass (tests, lint, build)
- Keep PRs small and focused
