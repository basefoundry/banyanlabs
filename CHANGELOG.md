# Changelog

All notable changes to banyanlabs will be documented in this file.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and versions are tracked in the repo-root `VERSION` file.

## [Unreleased]

### Changed

- Aligned Banyan Labs local and CI Go toolchains on Go 1.25 for patched
  standard-library security coverage.
- Updated Banyan Labs repository references, roadmap links, CODEOWNERS, and Go
  module coordinates for the Base Foundry organization migration.
- Moved the README license notice out of the opening project summary.
- Relicensed Banyan Labs prospectively from MIT to AGPL-3.0-or-later.

### Added

- Added expired session cleanup for URL shortener browser sessions.
- Added a Go toolchain guard for local URL shortener build and API smoke-test
  paths.
- Added the GitHub Project intake configuration and workflow to the repo
  baseline.
- Added Go vulnerability scanning to CI with `govulncheck`.
- Added the AGPL-3.0-or-later application notice to `LICENSE`.
- Added URL shortener package tests for command and app package coverage.
- Added URL shortener auth storage and app use cases with bcrypt password
  hashes and hashed opaque session tokens.
- Added cookie-backed URL shortener auth endpoints for signup, login, and
  logout, including OpenAPI and Hurl smoke-test coverage.
- Initialized the repository with the Base-managed repo baseline.
- Documented the Banyan Labs platform lab vision and first service direction.
- Linked Banyan Labs documentation references to the Base README.
- Added the Banyan Labs platform roadmap with phased completion criteria.
- Added the initial Go URL shortener service skeleton.
- Added mise-managed Go runtime setup for Banyan Labs.
- Added the URL shortener OpenAPI contract and Hurl API smoke tests.
- Added Banyan Labs agent workflow guidance and a pull request template.
- Added local service lifecycle commands for background dev, status, and stop.
