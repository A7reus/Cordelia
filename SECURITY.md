# Security Policy

## Supported Versions

Only the latest `v1.x` release receives security fixes. Older `v0.x` pre-releases are not supported.

| Version | Supported |
| ------- | --------- |
| 1.x     | yes       |
| 0.5.x   | no        |
| < 0.5   | no        |

## Reporting a Vulnerability

Do not open a public issue for security reports.

Email `an1ndya@proton.me` with subject `[Cordelia security]` and include:

- Cordelia version (`cordelia version`)
- OS and Go version if built from source
- Steps to reproduce
- Impact

## Security Model

Cordelia is a LAN-only tool. It does not use a public CA.

- **Transport:** All HTTP APIs are served over TLS `1.2+` on `https://host:47777` with a self-signed ECDSA P-256 certificate. The cert and key are generated on first run and stored at `~/.config/cordelia/cert.pem` and `key.pem` (`0600`). Delete both files to rotate.
- **Authentication:** Peers announce `cert_fingerprint` (SHA-256 of the leaf cert, hex lowercased) via UDP multicast `239.255.77.77:47777`. Clients pin the fingerprint: `VerifyPeerCertificate` compares the presented cert's SHA-256 to the expected fingerprint from discovery (`internal/registry`). If no expected fingerprint is known (first contact), the connection is allowed but the fingerprint is logged for Trust-On-First-Use.
- **Discovery:** Announcements are unauthenticated JSON. Do not trust `name` or `fingerprint` for authorization. Only `cert_fingerprint` is used for pinning.
- **File integrity:** `POST /v1/upload` returns `X-Checksum-Sha256` (SHA-256 of bytes as saved). Clients compare it to a local hash of the source file and fail on mismatch.

## Hardening Notes

- Allow only `47777/udp` and `47777/tcp` in your firewall. Cordelia binds to `0.0.0.0:47777` by default; use `-port` to change.
- `~/Downloads/cordelia` is created with `0755`. Files that would overwrite are saved as `name (1).ext`.
- `MaxMessageSize` is 64 KiB, `MaxUploadSize` is 100 MiB. Larger requests get `413`.

## Dependencies

Cordelia has no cgo and no third-party runtime dependencies. Standard library only. Run `go vet` and `go test -race` before release.
