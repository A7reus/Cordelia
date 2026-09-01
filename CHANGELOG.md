# Changelog

All notable changes to Cordelia will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.0] - 2026-09-01

### Added

- Frozen API under `/v1` (`/v1/info`, `/v1/peers`, `/v1/message`, `/v1/upload`) with old paths kept as aliases
- `docs/API.md`, `SECURITY.md`, `CHANGELOG.md` 1.1.0
- Persistent config file (`internal/config`) with `port`, `out_dir`, `ttl` and flag overrides
- Tests for `registry`, `config`, `certs` with `go test -race` and CI gate (`.github/workflows/ci.yml`)

## [0.5.0] - 2025-09-01

### Added

- Graceful shutdown on `SIGINT`/`SIGTERM` with 5s timeout
- SHA-256 checksums for file uploads (`X-Checksum-Sha256` header, verified by client)
- Interactive peer picker for `send-text` and `send-file` when no host is given
- Retry with backoff (500ms, 1s, 2s) for `send-text` and `send-file`

## [0.4.0] - 2025-08-31

### Added

- TLS with self-signed ECDSA P-256 certificates (`~/.config/cordelia/cert.pem`/`key.pem`)
- Cert fingerprint in discovery (`cert_fingerprint`) and pinning in client (`VerifyPeerCertificate`)

## [0.3.0] - 2025-08-27

### Added

- Progress reporting for `send-file` (10% steps)
- Collision-safe download naming (`file (1).ext`)
- Multi-file `send-file` (`send-file host file1 file2`)
- Custom download directory (`-out` flag)
- Project layout refactor to `cmd/cordelia` + `internal/client` + `internal/server` per `golang-standards/project-layout`
- Quick start for binary users in `README.md`

## [0.2.0] - 2025-08-27

### Added

- File transfer via `POST /upload` multipart streaming (`send-file`)
- `README.md` with dev and production setup, cross-compilation

## [0.1.0] - 2025-08-27

### Added

- Device identity with persistent fingerprint
- UDP multicast discovery (`239.255.77.77:47777`) and peer registry with TTL
- HTTP API `GET /info`, `GET /peers`, `POST /message`
- Automated releases via GitHub Actions

[Unreleased]: https://github.com/A7reus/Cordelia/compare/v1.0.0...HEAD
[1.0.0]: https://github.com/A7reus/Cordelia/compare/v0.5.0...v1.0.0
[0.5.0]: https://github.com/A7reus/Cordelia/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/A7reus/Cordelia/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/A7reus/Cordelia/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/A7reus/Cordelia/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/A7reus/Cordelia/releases/tag/v0.1.0
