# Contributing Guide

## Workflow overview

- Short-lived topic branches → PR into `main`
- Every merge into `main` produces an immutable candidate image
- Promotion to production is decided and executed from `infra` (GitOps)

---

## 1) Branches

| Branch | Purpose | Example |
|--------|---------|---------|
| `main` | Single integration branch, always deployable | — |
| `feat/` | New features | `feat/add-login` |
| `fix/` | Bug fixes | `fix/null-pointer-crash` |
| `chore/` | Everything else (docs, tests, refactor, deps, CI) | `chore/update-deps` |

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

## 3) CD model (main-only)

- We do **not** run a backend release process (no release branch, no hotfix branch, no release tags).
- Every merge into `main` publishes an immutable image (candidate artifact).
- Production is promoted from the `infra` repository by updating production manifests to a tested image.
- Backend contributors focus on shipping validated changes to `main`; promotion timing is owned by infra.

### Summary

| Situation | Branch from | PR target | Artifact outcome | Production decision |
|-----------|-------------|-----------|------------------|---------------------|
| Feature / fix | `main` | `main` | New immutable candidate image | `infra` promotion PR |
| Urgent fix | `main` | `main` | New immutable candidate image | `infra` promotion PR |

---

## 4) Commit messages & PR titles

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
chore: update Go to 1.26
feat(api)!: change error response format   ← breaking change
```

### Breaking changes

Add `!` after the scope to indicate a breaking change:

```
feat(api)!: change pagination response format
```

---

## 5) PR rules

- No direct pushes to `main`
- CI must pass (tests, lint, build, **PR title validation**)
- PR title must follow Conventional Commits format (see §5)
- Keep PRs small and focused
- Use **squash merge** (1 PR = 1 commit)
