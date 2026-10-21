# Banyan Labs

Banyan Labs is a platform-engineering learning environment built around real
services. Its first service is a local Go URL shortener with SQLite storage,
authentication, user management, a minimal HTML UI, and automated tests. The
intended audience is an engineer practicing service delivery and operations.

The longer-term multi-service, Kubernetes, infrastructure-as-code, observability,
and cloud work is roadmap direction, not a claim that every component exists.
The [vision](../docs/banyanlabs-vision.md) and
[roadmap](../docs/platform-roadmap.md) distinguish that direction from today's
service implementation.

Base owns workstation bootstrap and workspace orchestration. Banyan Labs owns
its service code, project-specific tooling declarations, runtime data, and
operational contracts. Do not move service behavior into Base.

Sources: [README](../README.md), [manifest](../base_manifest.yaml), and
[service guide](../services/url-shortener/README.md).
