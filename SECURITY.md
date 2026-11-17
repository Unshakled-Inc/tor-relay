# Security policy

## Supported versions

Security fixes are published for the newest tor version tag and for
`latest`. Older version tags are not patched. If you pin an exact version,
update the tag when a new release is published.

The image version follows the tor release it contains. There is no separate
image version yet.

## Reporting a security issue

Report security issues privately through GitHub private vulnerability
reporting:

https://github.com/Unshakled-Inc/tor-relay/security/advisories/new

Do not open a public issue for a security bug. If private reporting is
unavailable, open an issue with as little detail as possible and say that a
private report will follow.

Include:

- The image tag or digest, and the registry you pulled it from.
- The tor version printed by `torinit version`.
- The host architecture and Docker or Podman version.
- The role and the environment variables that matter, with secrets
  redacted.
- The relevant log lines, and whether the relay had bootstrapped.

Do not include the family key or the contents of `/var/lib/tor/keys` in a
report.

## Response targets

These are targets, not guarantees:

- Acknowledge a report within two business days.
- Triage and confirm the affected versions within five business days.
- Publish a fixed image on the same day as a new upstream tor release that
  contains a security fix, once the upstream signature and checksum verify.

If the fix is in this repository rather than upstream, a patched image is
published as soon as the fix is tested.

## What is in scope

- The Dockerfile and the build stages, including pinned base image digests,
  the tor source signature and checksum verification, and the pluggable
  transport versions.
- `torinit`: configuration parsing, torrc rendering, family key handling,
  MyFamily fetching and reload, signal handling, and the healthcheck.
- The runtime defaults: distroless base, UID/GID 65532, read-only root
  filesystem support, dropped capabilities, `no-new-privileges`, and the
  seccomp sandbox defaults.
- CI: the build workflow, Trivy scan gate, SBOM and provenance generation,
  and keyless cosign signing.

## What is out of scope

- Vulnerabilities in tor itself. Report those to the Tor Project:
  https://www.torproject.org/contact/
- Vulnerabilities in the Docker or Podman engine, the host kernel, or the
  host firewall.
- Vulnerabilities in base image packages that have no upstream fix. Trivy
  ignores unfixed findings, and the weekly rebuild picks up fixes when they
  are published.
- Misconfiguration on the host, such as a relay key file readable by other
  users or an ORPort blocked by a firewall.
- Documentation typos and broken links.

## How images are built and verified

- Tor is built from `https://dist.torproject.org/`. The build downloads the
  tarball, its checksum file, and its signature, verifies the signature with
  `gpgv` against `keys/tor-release.asc`, and then checks the tarball
  checksum. A failed signature or checksum stops the build.
- Base images are pinned by digest. GitHub Actions are pinned by commit SHA.
- Both the runtime and the debug image are scanned with Trivy. The build
  fails on a critical vulnerability with an available fix.
- SBOM and provenance attestations are generated for each image.
- Manifest lists are signed with keyless cosign. Verify with the command in
  the README.
- The release watcher checks `dist.torproject.org` every six hours, and a
  weekly job rebuilds the current version to pick up base image updates.

## Secrets

The image does not contain secrets. Relay identity keys live in the
`/var/lib/tor` volume, and family keys are mounted from `/run/secrets`.
Neither is baked into an image layer.

Never commit `*.secret_family_key`, `secret_id_key`,
`ed25519_master_id_secret_key`, or a `.env` file. The repository's
`.gitignore` covers them.
