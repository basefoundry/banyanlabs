# Workflows

From a configured Base workspace, `basectl setup banyanlabs` prepares the
manifest's tools. Review and approve manifest-declared commands through Base's
trust workflow before execution when prompted.

```bash
basectl test banyanlabs
basectl run banyanlabs api-test
basectl run banyanlabs build
basectl run banyanlabs dev
basectl run banyanlabs status
basectl run banyanlabs stop
```

`dev` starts background services; `dev -- --foreground` runs in the foreground.
Service development can also use `CGO_ENABLED=0 go test ./...`, `go vet ./...`,
and `go build ./...` from the Go module after preparing its toolchain.

Use an issue, a single category label, and a dedicated issue worktree based on
`origin/main`. Link the PR to its issue and report fresh validation. Documentation
changes require `git diff --check`; workflow/baseline changes also use
`tests/validate.sh`. Service/API changes require their matching checks.

Sources: [contribution guide](../CONTRIBUTING.md),
[GitHub workflow](../docs/github-workflow.md),
[repository validation](../tests/validate.sh), and [agent guidance](../AGENTS.md).
