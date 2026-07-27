# GitHub Workflow

Banyan Labs uses GitHub Issues as the product backlog and Git worktrees for
parallel pull request work. This mirrors
[Base](https://github.com/basefoundry/base#readme) so humans and AI-assisted
development agents follow the same rules across both repositories.

## Labels

Use one primary category label on each issue:

- `bug`
  Unexpected behavior, correctness issues, regressions, and defects.
- `enhancement`
  New capabilities, product improvements, refactors, and most maintenance work.
- `documentation`
  Documentation-only changes.
- `ci`
  GitHub Actions, test automation, release automation, and CI reliability.
- `security`
  Security hardening, dependency pinning, static analysis, and permission
  tightening.

Avoid creating new `type:*` labels.

## Issue Assignment

Issues created by Codex or other automation for Banyan Labs should be assigned
to `codeforester`.

If assignment fails, the automation should mention that in its summary instead
of silently leaving the issue unassigned.

## Preferred GitHub Interface

Use `basectl gh` as the preferred interface for Banyan Labs GitHub workflows
when it supports the operation.

Prefer commands such as:

```bash
basectl gh issue create
basectl gh issue start
basectl gh pr create
basectl gh pr checks
basectl gh branch prune
```

When `basectl` is not on `PATH`, use the sibling Base checkout directly:

```bash
~/work/base/bin/basectl gh issue list
```

Fallbacks are allowed when `basectl gh` does not support the needed operation,
when local GitHub CLI authentication is unavailable, or when a structured
GitHub API is safer for the task. In those cases, use the GitHub connector,
raw `gh`, or `git` as appropriate and keep issue labels, branch names,
assignments, and PR bodies aligned with this policy.

## Branch Names

Branch names should be derived from the issue category:

```text
<category>/<issue>-<YYYYMMDD>-<slug>
```

Examples:

```text
bug/12-20260529-fix-short-code-collision
enhancement/13-20260529-add-url-shortener-api
documentation/14-20260529-document-local-setup
ci/15-20260529-add-go-tests
security/16-20260529-harden-jwt-signing
```

Use `enhancement/` for maintenance work unless the issue is more specifically
`documentation`, `ci`, or `security`.

## Worktrees

All pull request implementation work should happen in a dedicated worktree.

The main checkout can stay as the user's active working copy:

```text
~/work/banyanlabs
```

Issue work should use:

```text
~/work/banyanlabs-worktrees/<slug>
```

Create a worktree from current `origin/main`:

```bash
git fetch origin main
git worktree add -b documentation/14-20260529-document-local-setup \
  ~/work/banyanlabs-worktrees/documentation-14-local-setup origin/main
```

Before creating a worktree, check whether the current checkout is already a
linked worktree for the intended issue:

```bash
git rev-parse --git-dir
git rev-parse --git-common-dir
git branch --show-current
git rev-parse --show-superproject-working-tree
```

When the git directory differs from the common directory, and the checkout is
not a submodule, the current checkout is already a linked worktree. A
non-empty `--show-superproject-working-tree` result means the checkout is a
submodule and should be treated as a normal repository for worktree detection.
Continue in an existing issue worktree instead of creating a nested or
duplicate worktree. If the checkout is a normal repository, create the issue
worktree from current `origin/main`.

Keep the worktree while the pull request is open so review feedback can be
handled on the same branch. Cleanup happens after merge, or after an explicit
discard decision.

After a pull request is merged:

```bash
git -C ~/work/banyanlabs pull --ff-only origin main
git -C ~/work/banyanlabs worktree remove ~/work/banyanlabs-worktrees/<slug>
git -C ~/work/banyanlabs branch -d <branch>
git -C ~/work/banyanlabs push origin --delete <branch>
```

Delete remote branches after merge unless there is a specific reason to keep
one around.

## Pull Requests

Keep each PR scoped to one issue.

PR bodies should include:

- a short summary of the change
- the validation commands that were run
- `Fixes #<issue>` or `Closes #<issue>` when the merge should close the issue
- product or platform impact, or `None` when there is no impact

Prefer small PR trains over large mixed PRs. A train may contain several
worktrees and PRs, but each PR should still close one issue cleanly.

### Review Feedback

Review feedback should be handled as technical input, not as an automatic patch
queue. Before implementing a suggestion, check whether it is correct for Banyan
Labs' product scope, platform direction, service boundaries, and current tests.

Implement clear, correct feedback directly. Push back or ask for clarification
when feedback would:

- move Base-owned workspace behavior into Banyan Labs
- broaden a narrow PR into a platform architecture change
- conflict with existing service contracts or validation rules
- add unused behavior that is not required by the issue
- force one language's library choices onto other services instead of preserving
  cross-language operational contracts

Fix one review item or related group at a time, then rerun the narrowest
verification that proves the change.

## Milestones

Milestones represent release intent, not workflow state.

Suggested Banyan Labs milestones:

- `v0.1.0 - Workflow foundation`
  Repository workflow, CI baseline, issue labels, and contributor guidance.
- `v0.2.0 - URL shortener skeleton`
  Initial Go module, local service entrypoint, tests, and SQLite foundation.
- `v0.3.0 - Authenticated URL management`
  Signup, login, JWT handling, CRUD endpoints, and user URL listing.
- `v0.4.0 - Local product polish`
  Simple frontend, local setup automation, fixtures, and documentation.
- `v1.0.0 - Usable local release`
  Stable local developer experience and complete URL shortener workflow.

Every issue does not need a milestone. Use milestones when the issue contributes
to a concrete release goal.

## Projects

Use one GitHub Project first: `Banyan Labs Roadmap`.

Recommended fields:

- `Status`: Triage, Backlog, Ready, In Progress, In Review, Done
- `Priority`: P0, P1, P2, P3
- `Area`: App, API, Auth, Storage, Frontend, Docs, CI, Infrastructure,
  Security, Product
- `Size`: S, M, L
- `PR Train`: optional text field for batch work

Useful views:

- Board by status
- Priority view
- Release view grouped by milestone
- Bugs and CI
- Ready for PR train

Projects show workflow and prioritization. Milestones show release grouping.
