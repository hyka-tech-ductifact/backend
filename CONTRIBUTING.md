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
- Topic branches (short-lived):
  - `feat/<short-topic>`
  - `fix/<short-topic>`
  - `chore/<short-topic>`
  - `docs/<short-topic>`
  - `hotfix/<short-topic>` (only for urgent production fixes)

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

When staging is approved by QA, create an annotated tag from `main`:

```bash
git checkout main
git pull

git tag -a v0.4.0 -m "Release v0.4.0"
git push origin v0.4.0
```

Production should deploy from the tag (`v0.4.0`), not from “latest `main`”.

---

## 7) Hotfixes (urgent production issues)

A hotfix is a small, focused change for an urgent production problem.

### Hotfix flow

1) Create the hotfix branch from the production tag (or the exact production commit).
2) Fix, test, PR.
3) Squash merge into `main`.
4) Tag a new PATCH release and deploy it.

```bash
git fetch --tags

git checkout -b hotfix/fix-login-500 v0.4.0
# edit, test...

git add -A
git commit -m "fix: prevent login 500 when user missing"

git push -u origin hotfix/fix-login-500
# open PR -> squash merge into main

git checkout main
git pull

git tag -a v0.4.1 -m "Hotfix v0.4.1"
git push origin v0.4.1
```

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
