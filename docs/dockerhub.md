# tor-relay

Hardened, signed, multi-arch Docker image for Tor relay and bridge
operators, published by Unshakled Inc.

Tor is compiled from the signed upstream source tarball, never from distro
packages. The runtime is distroless, runs as UID/GID 65532, and works with a
read-only root filesystem, all capabilities dropped, and
`no-new-privileges`.

## What it does

- Middle, exit, obfs4 bridge, and WebTunnel bridge roles.
- Family keys and `MyFamily`, including an https URL sync mode.
- A `latest-debug` variant with nyx for live monitoring.
- Prometheus metrics on a policy-controlled port.
- Multi-arch images for `linux/amd64`, `linux/arm64`, and `linux/arm/v7`.

It does not make you anonymous: it is a server for the Tor network, not a
client or proxy. `SocksPort` is off.

## Verify the image

Images are signed with keyless cosign. Verify the exact workflow identity:

```
cosign verify docker.io/unshakled/tor-relay:latest \
  --certificate-identity 'https://github.com/Unshakled-Inc/tor-relay/.github/workflows/build.yml@refs/heads/main' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

Every image has SBOM and provenance attestations. The build fails on a HIGH
or CRITICAL vulnerability with an available fix, and the current version is
rebuilt weekly.

## Quick start

1. Create an env file:

```
TOR_ROLE=middle
TOR_NICKNAME=ExampleRelay
TOR_CONTACT=email:support[]example.org url:example.org
TOR_ORPORT=9001
```

2. Verify the configuration:

```
docker run --rm --env-file .env docker.io/unshakled/tor-relay:latest verify
```

3. Start it with Compose. A named volume cannot be forgotten:

```
services:
  tor-relay:
    image: docker.io/unshakled/tor-relay:latest
    network_mode: host
    env_file:
      - .env
    read_only: true
    cap_drop:
      - ALL
    security_opt:
      - no-new-privileges:true
    restart: unless-stopped
    stop_grace_period: 45s
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    volumes:
      - tor-data:/var/lib/tor

volumes:
  tor-data:
```

Open the ORPort in the host firewall. With host networking, the relay sees
the host interfaces and its reachability self-test works without manual port
forwarding.

## Tags

| Tag | Moves? | Use it for |
| --- | --- | --- |
| `latest` | Yes | Trying the image |
| `0.4.9` | Yes, within the series | Single relays on auto-update |
| `0.4.9.17` | Yes, weekly rebuild | Pinning a Tor version with OS fixes |
| `0.4.9.17-20261117` | No | Rollback and audit |
| `latest-debug` | Yes | Troubleshooting only |

## Documentation

- [README](https://github.com/Unshakled-Inc/tor-relay/blob/main/README.md)
- [Roles and configuration](https://github.com/Unshakled-Inc/tor-relay/blob/main/README.md#roles)
- [Systemd migration](https://github.com/Unshakled-Inc/tor-relay/blob/main/docs/migrate-from-systemd.md)
- [Family keys](https://github.com/Unshakled-Inc/tor-relay/blob/main/docs/family-keys.md)
- [Auto-update](https://github.com/Unshakled-Inc/tor-relay/blob/main/docs/auto-update.md)
- [Backup](https://github.com/Unshakled-Inc/tor-relay/blob/main/docs/backup.md)
- [Host checklist](https://github.com/Unshakled-Inc/tor-relay/blob/main/docs/host-checklist.md)
- [Supply chain](https://github.com/Unshakled-Inc/tor-relay/blob/main/docs/supply-chain.md)
- [Security policy](https://github.com/Unshakled-Inc/tor-relay/blob/main/SECURITY.md)

BSD-3-Clause. Source: https://github.com/Unshakled-Inc/tor-relay
