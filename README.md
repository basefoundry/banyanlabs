# Banyan Labs

Banyan Labs is a realistic infrastructure and platform engineering lab.

## License

Copyright (C) 2019-2026 Ramesh Padmanabhaiah.

Banyan Labs is licensed under the GNU Affero General Public License v3.0 or
later. See [LICENSE](LICENSE).

Previous versions released under the MIT License remain available under MIT as
licensed at the time. New versions from this relicensing change forward are
licensed under AGPL v3.

The purpose is to build deep DevOps, SRE, and platform engineering knowledge by
assembling real services and infrastructure tools into a meaningful environment.
Certifications can introduce AWS, GCP, Azure, Kubernetes, Terraform, CI/CD,
monitoring, and distributed systems, but Banyan Labs is about learning those
ingredients by building with them.

The long-term goal is to grow toward the complexity of a medium-sized
engineering organization: multiple services, multiple languages, local and
remote environments, CI/CD, infrastructure as code, observability, Kubernetes,
and eventually multi-cloud patterns based on real open source tools.

The first concrete service is a Go URL shortener with local SQLite storage,
authentication, user management, tests, and a minimal HTML UI from day one.

See [Banyan Labs Vision](docs/banyanlabs-vision.md) for the broader direction.
See [Platform Roadmap](docs/platform-roadmap.md) for the phased path from the
local service to observability, delivery, Kubernetes, infrastructure as code,
and cloud environments.

## Local Setup

Banyan Labs uses Base for workspace orchestration and `mise` for language
runtime versions.

From the repository root:

```bash
basectl setup banyanlabs
```

That setup path uses:

- `Brewfile` to install `mise` and API test tooling
- `.mise.toml` to install the Go runtime
- `base_manifest.yaml` to connect those tools to Base

After setup:

```bash
basectl test banyanlabs
basectl run banyanlabs api-test
basectl run banyanlabs build
basectl run banyanlabs dev
basectl run banyanlabs status
basectl run banyanlabs stop
```

`basectl run banyanlabs dev` starts local services in the background. To run
the URL shortener in the foreground instead:

```bash
basectl run banyanlabs dev -- --foreground
```

API contracts and black-box smoke tests live with the repo:

- `services/url-shortener/openapi.yaml`
- `tests/api/url-shortener/*.hurl`

## Relationship to Base

The shared developer bootstrap layer no longer lives here.

That foundational layer now belongs in
[`base`](https://github.com/codeforester/base#readme), including:

- shell environment bootstrap
- shared Bash wrapper and command conventions
- common shell libraries
- workspace-level setup and test entrypoints

The intended local workspace shape is:

```text
work/
  base/
  banyanlabs/
  other-project/
```

In that model:

- `base` owns shared developer tooling and workspace orchestration
- `banyanlabs` owns Banyan Labs-specific code, manifests, infrastructure, and
  project behavior

## What This Repo Will Focus On

Banyan Labs focuses on project-level and product-level concerns such as:

- project source code
- infrastructure definitions
- service code
- manifests and configuration specific to Banyan Labs
- project-specific documentation
- project-specific tests and automation

## Status

This repo is being rebuilt around a local-first platform lab.

The shared CLI/bootstrap artifacts moved into the sibling `base` repo. Banyan
Labs now owns the services, docs, tests, and project-specific infrastructure
that will grow on top of that workspace model.

## Development Workflow

Banyan Labs follows Base's issue-first GitHub workflow:

- track work in GitHub Issues before implementation starts
- prefer `basectl gh` for supported GitHub issue and pull request tasks
- use one worktree per pull request
- keep each PR scoped to one issue and link it with `Fixes #<issue>`

See [CONTRIBUTING.md](CONTRIBUTING.md) and
[docs/github-workflow.md](docs/github-workflow.md) for the full workflow.
