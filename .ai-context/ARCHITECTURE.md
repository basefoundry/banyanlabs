# Architecture

The Go module under `services/url-shortener/` separates process startup in
`cmd/url-shortener/` from application/authentication logic in `internal/app/`,
HTTP handling in `internal/httpserver/`, and persistence behind `internal/storage/`.
The SQLite implementation owns migrations under `internal/storage/sqlite/`.
Configuration and structured logging have dedicated internal packages.

The service's OpenAPI description is the HTTP contract; Hurl scenarios in
`tests/api/url-shortener/` exercise the service from outside the process. Go
unit tests live beside their implementations.

`base_manifest.yaml`, `Brewfile`, and `.mise.toml` describe project setup and
commands. `scripts/services.sh` manages local service start/status/stop.
The documented defaults are loopback port 8080 and a database under
`var/url-shortener/` when started through the repository's dev workflow.
Use the service's configuration variables for address, database, and log level.
Local databases, credentials, and logs do not belong in exported context.

Sources: [service entrypoint](../services/url-shortener/cmd/url-shortener/main.go),
[OpenAPI](../services/url-shortener/openapi.yaml),
[service lifecycle](../scripts/services.sh), and
[service guide](../services/url-shortener/README.md).
