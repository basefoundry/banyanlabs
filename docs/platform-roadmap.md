# Banyan Labs Platform Roadmap

Banyan Labs should grow as a learning environment by adding platform complexity
only after there is product behavior that needs it. The roadmap starts with one
real local service, then layers delivery, observability, Kubernetes,
multi-service operation, infrastructure as code, and cloud environments around
that service.

This document is guidance, not a fixed implementation backlog. Detailed issues
should be created when a phase is close enough to execute. Until then, later
phases should stay as tracking issues so they do not become stale task lists.

## Roadmap Principles

- Start local-first and keep the first product slice real.
- Add infrastructure because it solves a product or operations problem.
- Prefer one working path before adding a second option.
- Make operational contracts explicit across languages.
- Practice detection, diagnosis, recovery, and change management, not just tool
  installation.
- Keep shared workstation/bootstrap ownership in Base and keep Banyan Labs
  focused on project services, infrastructure, delivery, and operations.

## Phase 1: Local Product Foundation

Goal: make the URL shortener a usable local product before expanding the
platform surface.

Current issue train:

- [#26](https://github.com/basefoundry/banyanlabs/issues/26) Create Go URL
  shortener service skeleton
- [#27](https://github.com/basefoundry/banyanlabs/issues/27) Implement URL
  shortener user management
- [#28](https://github.com/basefoundry/banyanlabs/issues/28) Implement URL
  shortening and redirect behavior
- [#29](https://github.com/basefoundry/banyanlabs/issues/29) Add minimal HTML
  UI for URL shortener
- [#30](https://github.com/basefoundry/banyanlabs/issues/30) Add JWT
  authentication for API clients

Entry criteria:

- Repository workflow, CI baseline, and project ownership boundaries exist.
- The URL shortener architecture has enough shape to guide implementation.

Completion criteria:

- A developer can run the service locally.
- Users can sign up, sign in, create short URLs, manage URLs, and follow
  redirects.
- Tests, structured logs, health checks, migrations, and local documentation are
  present.
- The service is useful enough to justify platform work around it.

## Phase 2: Local Platform Composition

Tracking issue: [#33](https://github.com/basefoundry/banyanlabs/issues/33)

Goal: compose the local product with realistic runtime dependencies and
operational tooling.

Likely scope:

- Container image build for the URL shortener.
- Local multi-container stack using Docker Compose or an equivalent local-first
  approach.
- Postgres when a network database is needed.
- Redis when session, cache, queue, or token behavior justifies it.
- Prometheus, Grafana, OpenTelemetry Collector, and log aggregation once there
  is enough behavior to observe.
- Local smoke checks for the composed environment.

Completion criteria:

- One local command starts the product and its local operational stack.
- The stack is documented and suitable for development and smoke testing.
- Local dependencies are configured as services, not as hidden manual setup.

## Phase 3: Observability Foundation

Tracking issue: [#34](https://github.com/basefoundry/banyanlabs/issues/34)

Goal: make service behavior visible through logs, metrics, traces, dashboards,
and alerts.

Likely scope:

- Structured JSON logs using the Banyan Labs logging contract.
- Request metrics for rate, errors, latency, and saturation.
- Database and dependency health signals.
- Basic traces for request flows when OpenTelemetry is introduced.
- Grafana dashboards and initial alert rules.
- Operational notes that explain how to diagnose common failures.

Completion criteria:

- A local operator can answer what is failing, where it is failing, and whether
  the failure is user-visible.
- Dashboards and alerts are tied to real service behavior.
- Logs, metrics, and traces include correlation fields where relevant.

## Phase 4: CI/CD And Supply Chain

Tracking issue: [#35](https://github.com/basefoundry/banyanlabs/issues/35)

Goal: expand the current test baseline into automated build, scan, package, and
release discipline.

Likely scope:

- Go, Java, Python, shell, and security checks as the repo grows.
- Container image builds.
- Dependency and vulnerability scanning.
- Database migration checks.
- Release tags, changelog discipline, and reproducible release artifacts.
- Optional SBOM and provenance once images are part of the delivery path.

Completion criteria:

- Pull requests prove test, build, and security quality.
- Release artifacts can be reproduced from source.
- The delivery workflow is documented and does not depend on local-only manual
  steps.

## Phase 5: Local Kubernetes

Tracking issue: [#36](https://github.com/basefoundry/banyanlabs/issues/36)

Goal: run Banyan Labs locally in Kubernetes after the local service and platform
stack are useful.

Likely scope:

- A local Kubernetes cluster, with kind as the default first candidate unless a
  later decision chooses another tool.
- Kubernetes manifests, Kustomize, Helm, or another packaging approach.
- ConfigMaps, Secrets, Services, Ingress, probes, and resource requests.
- Local deployment docs, smoke tests, and rollback practice.
- Observability integration for workloads running in the cluster.

Completion criteria:

- The URL shortener runs locally in Kubernetes.
- Deployment, rollout, rollback, health checking, and troubleshooting are
  documented.
- The Kubernetes path reuses the same operational contracts as the local
  container stack.

## Phase 6: Multi-Service And Multi-Language Expansion

Tracking issue: [#37](https://github.com/basefoundry/banyanlabs/issues/37)

Goal: add multiple services and languages to practice platform consistency
across different application stacks.

Likely scope:

- A Java service and a Python service after the Go service is operationally
  useful.
- Realistic service boundaries such as click analytics, audit events,
  background jobs, notification workflows, or admin reporting.
- Shared service standards for config, health, logs, metrics, traces, tests,
  and deployment.
- Inter-service communication and local failure scenarios.

Completion criteria:

- Multiple services run together locally.
- Different languages follow the same operational contracts without forcing one
  shared application framework.
- Service boundaries, ownership, and failure behavior are clear enough for SRE
  practice.

## Phase 7: Infrastructure As Code And Cloud Environments

Tracking issue: [#38](https://github.com/basefoundry/banyanlabs/issues/38)

Goal: model repeatable dev, staging, and production environments after local
deployment and observability patterns are proven.

Likely scope:

- Terraform or OpenTofu modules for networking, Kubernetes, databases, DNS,
  secrets, and observability.
- One cloud provider first.
- Dev, staging, and production environment modeling.
- Deployment and destruction workflows for non-production environments.
- Cloud cost and blast-radius controls.

Completion criteria:

- A non-production cloud environment can be provisioned, deployed, observed, and
  destroyed from documented automation.
- Differences between dev, staging, and production are explicit.
- One cloud path works end to end before a second cloud is introduced.

## Phase 8: Reliability Practice And Multi-Cloud Patterns

Goal: make Banyan Labs useful as an SRE and platform engineering lab, not just a
collection of tools.

Likely scope:

- Service-level objectives and error-budget thinking.
- Alerts, runbooks, incident drills, and postmortems.
- Backup, restore, load testing, and capacity exercises.
- Failure injection and dependency outage scenarios.
- A second cloud provider after one cloud path is mature.
- Crossplane or a similar control-plane pattern only when there is a clear need
  to model multi-cloud infrastructure behind Kubernetes APIs.

Completion criteria:

- A failure can be detected, diagnosed, mitigated, and reviewed using Banyan
  Labs artifacts.
- The same product can run in more than one environment with documented
  tradeoffs.
- Multi-cloud work teaches real differences between providers instead of hiding
  them too early.

## Completion Model

Banyan Labs is complete at the learning-environment level when it can
demonstrate the full operational loop:

1. Build and test services.
2. Run them locally with realistic dependencies.
3. Observe logs, metrics, traces, dashboards, and alerts.
4. Deploy through automated delivery.
5. Run in Kubernetes.
6. Provision infrastructure from code.
7. Operate at least one cloud environment.
8. Practice incident response and recovery.
9. Extend the system across languages, services, and eventually clouds.

The project can keep evolving after that point, but this loop is the threshold
where Banyan Labs has become a credible platform engineering and SRE lab.
