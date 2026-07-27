# Codex Guidance

This file gives coding agents the repository-specific rules for Banyan Labs. It
is a navigation layer over `CONTRIBUTING.md`, `docs/github-workflow.md`, and
`skills.md`, not a replacement for them.

## Working Agreement

- Keep Banyan Labs focused on services, application behavior, delivery,
  observability, infrastructure, and platform engineering practice.
- Keep shared workstation setup, repository discovery, and Base-managed
  workspace orchestration in the sibling
  [Base](https://github.com/basefoundry/base#readme) repository.
- Adopt external agent workflow ideas only after translating them into
  Banyan Labs-specific guidance. Do not vendor or require a third-party
  methodology when a smaller repo-native rule is enough.
- When the user explicitly says a session is design-only or asks for no code
  changes, stay in discussion mode and do not edit files.
- Surface unresolved product, platform, or architecture decisions instead of
  silently choosing defaults for broad changes.

## GitHub Workflow

- Create or choose a GitHub issue before implementation work.
- Use one primary category label: `bug`, `enhancement`, `documentation`, `ci`,
  or `security`.
- Do not create or apply `type:*` issue labels.
- Assign Codex-created issues to `codeforester` when GitHub allows it.
- Prefer `basectl gh` for supported issue, branch, PR, check, and cleanup
  operations. Fall back to the GitHub connector, raw `gh`, or `git` when
  `basectl gh` does not support the needed operation.
- Branch from `origin/main` with `<category>/<issue>-<YYYYMMDD>-<slug>`.
- Use a dedicated worktree under `~/work/banyanlabs-worktrees/<slug>` for PR
  work.
- Before creating a worktree, check whether the current checkout is already a
  linked worktree for the intended issue.
- Link PRs with `Fixes #<issue>` or `Closes #<issue>` when merge should close
  the issue.
- After merge, sync `main`, remove the worktree, and delete local and remote
  branches.

## Validation

- Run the narrowest relevant checks first, then broaden when shared service,
  workflow, or platform behavior is touched.
- Do not claim work is fixed or complete without fresh verification output from
  the current checkout or worktree.
- For documentation-only changes, run `git diff --check`.
- For repository baseline or workflow docs, run `tests/validate.sh` when it is
  affected.
- For Go service changes, run the relevant `go test`, `go vet`, and
  `go build` checks in the changed module. Use `CGO_ENABLED=0` unless the
  change explicitly requires CGO.
- For API behavior, run the Hurl/API smoke tests when relevant.
- If a required check cannot be run locally, say so in the PR and final
  summary.

## Change Boundaries

- Keep Base-managed workspace setup in Base; keep Banyan Labs project setup in
  `base_manifest.yaml`, `Brewfile`, `.mise.toml`, and project docs.
- Keep service code, tests, and service-local docs close together under the
  owning service directory.
- Define cross-language operational contracts in Banyan Labs docs, while
  letting each language use idiomatic libraries.
- Use review feedback as technical input. Verify suggestions against Banyan
  Labs' product and platform boundaries before implementing them.
