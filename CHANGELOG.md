# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

The project version follows the tor release in the image. `torinit` is
versioned with the image until its first standalone release.

## [0.4.9.17] - 2026-11-17

### Added

- Multi-arch images for `linux/amd64`, `linux/arm64`, and `linux/arm/v7`,
  published to `docker.io/unshakled/tor-relay` and
  `ghcr.io/unshakled-inc/tor-relay`.
- Tor 0.4.9.17 compiled from the signed upstream source release.
- `torinit` entrypoint with `render`, `verify`, `run`, `version`,
  `healthcheck`, `keygen-family`, `bridgeline`, `fingerprint`, and `envdoc`
  subcommands. `torinit version` reports the image version, and the image
  carries the `org.opencontainers.image.revision` label.
- Middle relay role.
- Exit relay role with an explicit `I_UNDERSTAND_EXIT=true` acknowledgment
  and an operator-supplied exit policy.
- obfs4 bridge role with lyrebird 0.10.0, a configurable obfs4 port, bridge
  distribution, and a client bridge line.
- WebTunnel bridge role with webtunnel v0.0.7 and support for a dedicated
  domain path or a secret path.
- Family key support: keys are installed from `/run/secrets`, normalized to
  `*.secret_family_key`, and rendered as `FamilyId` lines.
- `MyFamily` direct mode and https URL mode, with atomic writes, refresh,
  drop protection, own-fingerprint exclusion, and `SIGHUP` reload.
- `-debug` image variant with nyx, a shell, and `TOR_CONTROL_PORT`.
- Prometheus metrics through `TOR_METRICS_PORT`, `TOR_METRICS_POLICY`, and
  `TOR_METRICS_PUBLIC`, with a `reject *` fallback.
- `TOR_EXTRA_ARGS` for arbitrary torrc lines and `TOR_DNS_RESOLV_CONF` for a
  resolver configuration file.
- Structured `.env.example` files for middle, exit, bridge, and WebTunnel
  roles.
- Operator documentation: README, extra torrc arguments, systemd migration,
  family keys, auto-update, and this security policy.

### Security

- Tor source signature verification with `gpgv` and a tarball checksum check
  before compilation.
- Distroless runtime with no shell and no package manager.
- Runs as UID/GID 65532, with a 0700 data directory.
- Compatible with a read-only root filesystem, dropped capabilities, and
  `no-new-privileges`.
- Seccomp sandbox enabled by default for middle and exit relays.
- Trivy scan gate on critical vulnerabilities with available fixes, for both
  the runtime and debug images.
- SBOM and provenance attestations for every image.
- Keyless cosign signatures on both registries.

### Known limitations

- The seccomp sandbox is disabled by default for obfs4 and WebTunnel bridges,
  because tor 0.4.9.17 reports that managed proxies are not compatible with
  sandbox mode.
- The metrics endpoint is unencrypted HTTP. Use a private address, a
  restricted policy, or a private network.
- Snowflake is out of scope. Use the Tor Project's official
  `thetorproject/snowflake-proxy` image.
