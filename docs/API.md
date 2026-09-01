# Cordelia API v1

Frozen as of `v1.0.0`. All endpoints are served over TLS on `https://host:port`. Old paths without `/v1` (`/info`, `/peers`, `/message`, `/upload`) remain as aliases but should be considered deprecated. New clients must use `/v1`.

Base URL: `https://host:port/v1`

All JSON is `application/json`. File upload is `multipart/form-data`.

## GET /v1/info

Returns the receiver's identity.

**Request:**

```
GET /v1/info HTTP/1.1
```

**Response `200`:**

```json
{ "name": "tux", "fingerprint": "69a34fd2577a916f61e2105207309a3d" }
```

## GET /v1/peers

Returns the current peer registry snapshot. Each peer was seen via UDP multicast and has not yet expired (TTL = 10s).

**Response `200`:**

```json
[
  {
    "name": "tux",
    "fingerprint": "69a34fd2577a916f61e2105207309a3d",
    "cert_fingerprint": "d83ecd753b52d8871f0829224ea3ff7e81b528dcbbdb50753ba40afaa1d98af5",
    "addr": "192.168.1.5",
    "tcp_port": 47777,
    "last_seen": "2026-08-31T20:45:21Z"
  }
]
```

`cert_fingerprint` is SHA-256 of the peer's leaf certificate (hex, lowercased). Empty for peers from pre-v0.4.0.

## POST /v1/message

Sends a text message. Body is JSON, max 64 KiB.

**Request:**

```
POST /v1/message HTTP/1.1
Content-Type: application/json

{"from":"tux","text":"hello world"}
```

**Response `204`** on success, `400` for invalid JSON or empty `text`, `413` for too large.

## POST /v1/upload

Sends a file. Body is `multipart/form-data` with a single part named `file`. Max request 110 MiB (100 MiB file + overhead), max stored file 100 MiB.

**Request (example via curl):**

```
curl -sk -F "file=@photo.jpg" https://host:47777/v1/upload
```

**Response `204`** on success, `400` for missing part or invalid filename, `413` for too large.

**Headers on success:**

```
X-Checksum-Sha256: 56b4f65e49489fe0a89885f8a7a5881adf7aea66d43d4af6b5df4473e7cb60f9
```

The checksum is SHA-256 of the bytes as saved. Clients should compare it to a local hash of the source file.

## Discovery

Not part of the HTTP API. UDP multicast `239.255.77.77:47777`, JSON:

```json
{
  "name": "tux",
  "fingerprint": "...",
  "cert_fingerprint": "...",
  "tcp_port": 47777
}
```

Announced every 3s, expired after 10s.

## TLS

Self-signed ECDSA P-256 cert at `~/.config/cordelia/cert.pem` / `key.pem` (or `-data-dir`). Clients pin the `cert_fingerprint` from discovery; `curl` needs `-k` to skip CA verification.
