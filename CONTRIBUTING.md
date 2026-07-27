# Contributing to Banyan Labs

Banyan Labs uses the same issue-first GitHub workflow as the sibling
[Base](https://github.com/basefoundry/base#readme) repository. Keep work
visible in GitHub Issues, keep pull requests small, and use Base's `basectl gh`
helper when it supports the operation.

Coding agents should also follow [AGENTS.md](AGENTS.md). Repeatable
AI-assisted workflows live in [skills.md](skills.md).

## Workflow

1. Create or choose a GitHub issue before starting implementation work.
2. Apply one primary category label:
   - `bug` for defects, regressions, or correctness issues.
   - `enhancement` for new capabilities, product improvements, refactors, and
     most maintenance work.
   - `documentation` for documentation-only work.
   - `ci` for GitHub Actions, tests, release automation, or CI reliability.
   - `security` for security hardening, dependency pinning, static analysis, or
     permission tightening.
3. Create a branch from the issue using this convention:

   ```text
   <category>/<issue>-<YYYYMMDD>-<slug>
   ```

   Example:

   ```text
   enhancement/42-20260529-add-url-shortener-api
   ```

4. Use an isolated Git worktree for each pull request:

   ```bash
   git fetch origin main
   git worktree add ../banyanlabs-worktrees/<slug> -b <branch> origin/main
   ```

5. Before creating a worktree, check whether the current checkout is already a
   linked worktree for the issue. Do not create nested or duplicate worktrees.
6. Keep the PR scoped to one issue. Avoid unrelated refactors.
7. Link the PR back to the issue with `Fixes #<issue>`.
8. After merge, sync `main`, remove the worktree, and delete the local and
   remote branches.

For the full policy, including milestone and GitHub Project guidance, see
[GitHub Workflow](docs/github-workflow.md).

## Base Tooling

Use the sibling Base checkout as the shared workspace control plane:

```text
~/work/base
~/work/banyanlabs
```

When `basectl` is on `PATH`, prefer:

```bash
basectl gh issue create --category enhancement --title "Add ..."
basectl gh issue start <issue-number>
basectl gh pr create
basectl gh pr checks
```

If `basectl` is not yet on `PATH`, invoke it from the sibling checkout:

```bash
~/work/base/bin/basectl gh issue list
```

Fallback to `gh` only when `basectl gh` does not support the needed operation.

## Validation

Run the narrowest relevant checks first, then broaden when the change touches
shared behavior.

Current general checks:

```bash
tests/validate.sh
git diff --check
```

For Go service changes, run the relevant `go test`, `go vet`, and `go build`
checks in the changed module. Use `CGO_ENABLED=0` unless the change explicitly
requires CGO. For API behavior, run the Hurl/API smoke tests when relevant.

Do not claim work is fixed or complete without fresh verification output from
the current checkout or worktree. If a required check cannot be run locally,
state that in the PR.

## Pull Request Checklist

Before opening a PR:

- The branch name follows `<category>/<issue>-<YYYYMMDD>-<slug>`.
- The PR is scoped to one issue.
- The PR body explains what changed and how it was validated.
- Validation commands were run from the current checkout or worktree, or
  unavailable checks are explained.
- Documentation is updated when behavior, setup, or workflow changes.
- The PR includes `Fixes #<issue>` when it should close the issue.
