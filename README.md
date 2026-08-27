# Cordelia

Cordelia is a LocalSend-inspired file and message sharing tool for local networks. It is written in Go as a learning project focused on networking fundamentals. Every device runs the same binary, discovers peers automatically on the LAN, and exchanges messages over a simple HTTP API.

Current status: v0.1.0 -> device identity, LAN discovery, peer registry, and text messaging. File transfer, TLS, and GUI are planned for later releases.

## How it works

- Each instance generates a persistent identity (name and fingerprint) on first run and stores it under the OS config directory.
- Discovery uses UDP multicast on `239.255.77.77:47777`. Instances announce themselves every 3 seconds and listen for announcements from others.
- A peer registry keeps the list of recently seen devices. Entries expire after 10 seconds of silence and are swept every 3 seconds. The registry is protected by a mutex because it is accessed from multiple goroutines.
- The HTTP API runs on TCP `47777` by default and exposes:
  - `GET /info` -> returns the local identity as JSON
  - `GET /peers` -> returns the current peer list
  - `POST /message` -> receives a JSON message `{from, text}`

All of it is implemented with the Go standard library only.

## Requirements

- Go 1.22 or newer (tested on Go 1.27)
- Linux, macOS, or Windows. The code is pure Go with `CGO_ENABLED=0`, so it cross-compiles without extra toolchains.
- For LAN discovery, UDP inbound on port `47777` must be allowed. The HTTP API also needs TCP `47777`.

## Development setup

1. Clone the repository:

   ```bash
   git clone https://github.com/A7reus/Cordelia.git
   cd Cordelia
   ```

2. Run directly:

   ```bash
   go run .
   ```

   On first run, this creates an identity and starts the API server. Logs show the address being served.

3. Common commands while developing:

   ```bash
   go vet ./...          # static checks
   go run -race .        # run with the race detector
   go run . probe localhost:47777
   go run . peers
   go run . send-text localhost:47777 "hello world"
   go run . version      # prints the embedded version, defaults to "dev"
   ```

4. Running two instances on one machine (for testing discovery):

   ```bash
   # terminal A
   go run .

   # terminal B -> different port and different config directory
   go run . -port 47778 -data-dir /tmp/cordelia-b
   ```

   The second instance gets its own fingerprint and announces on its own TCP port. Both discover each other over multicast loopback. If you use UFW or another firewall, allow the ports first:

   ```bash
   sudo ufw allow in 47777/udp
   sudo ufw allow in 47777/tcp
   ```

5. Formatting: the project uses `gofmt`. Run `gofmt -w .` before committing.

## Production setup

### Building locally

```bash
go build -trimpath -ldflags "-s -w -X main.version=v0.1.0" -o cordelia .
./cordelia version
./cordelia
```

The `-ldflags` flag embeds the release version into the binary. Without it, `version` reports `dev`. `-trimpath` and `-s -w` make the build reproducible and smaller.

### Cross-compiling

Because the project has no cgo dependencies, cross-compilation is a single command per target:

```bash
CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=v0.1.0" -o dist/cordelia-linux-amd64 .
CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.version=v0.1.0" -o dist/cordelia-linux-arm64 .
CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=v0.1.0" -o dist/cordelia-darwin-amd64 .
CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.version=v0.1.0" -o dist/cordelia-darwin-arm64 .
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w -X main.version=v0.1.0" -o dist/cordelia-windows-amd64.exe .
CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "-s -w -X main.version=v0.1.0" -o dist/cordelia-windows-arm64.exe .
```

### Automated releases (GitHub Actions)

Releases are automated. Pushing a tag matching `v*` triggers `.github/workflows/release.yml`, which builds all targets above and publishes them to the GitHub Releases page via `gh release create`. To cut a new release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Artifacts are named `cordelia-<os>-<arch>` (with `.exe` on Windows) and a `checksums.txt` is attached.

## Project structure

```
.
├── main.go                 # CLI dispatch, HTTP handlers, client commands
├── internal/
│   ├── identity/           # fingerprint generation and persistence
│   ├── discovery/          # UDP multicast announce and listen
│   └── registry/           # peer registry with TTL
├── go.mod
└── README.md
```

## Roadmap

- v0.1.0 (current) -> identity, discovery, peer registry, text messaging
- Next -> file transfer over multipart upload, progress reporting
- Later -> TLS with self-signed certificates, graceful shutdown, optional TUI

## License

MIT. See [LICENSE](LICENSE).
