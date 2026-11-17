# Supply chain

This page records every input to the image, how it is pinned, and how it is
verified. Last reviewed: 2026-11-17.

## Tor

- Source: `https://dist.torproject.org/tor-<version>.tar.gz`, with its
  `.sha256sum` and `.sha256sum.asc`.
- Verification: `scripts/verify-tor-source.sh` runs `gpgv` against the
  vendored keys in `keys/tor-release.asc`, then checks the tarball checksum.
  A failed signature or checksum stops the build. There is no fallback.
- Signing keys: `keys/README.md` lists the fingerprints and where they were
  cross-checked.

## Pluggable transports

Both transports are pinned by tag and by commit. The Dockerfile clones the
tag and fails the build unless `HEAD` equals the pinned commit. A moved tag
therefore cannot silently change the binary.

| Transport | Tag | Commit | Tag type |
| --- | --- | --- | --- |
| lyrebird | `0.10.0` | `59ac7e400c384773f5584dbe366516ad04a33d9f` | annotated |
| webtunnel | `v0.0.7` | `11334a222c1beeb45759d436b7aef963a10600b2` | lightweight |

The commits were obtained on 2026-11-17 with:

```
git ls-remote --tags https://gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/lyrebird.git 'refs/tags/0.10.0*'
git ls-remote --tags https://gitlab.torproject.org/tpo/anti-censorship/pluggable-transports/webtunnel.git 'refs/tags/v0.0.7*'
```

For the annotated lyrebird tag, the `^{}` line is the commit that the tag
resolves to. For the lightweight webtunnel tag, the single line is the
commit.

The versions are in `LYREBIRD_VERSION` and `WEBTUNNEL_VERSION`; the commits
are in `LYREBIRD_COMMIT` and `WEBTUNNEL_COMMIT`. To update a transport,
change all four files together, then run a build. The build fails if the tag
and commit disagree.

The transport builds run with `GOFLAGS=-mod=readonly` and
`GOTOOLCHAIN=local`, so a missing `go.sum` entry or a toolchain download
fails the build instead of silently changing the build inputs.

### Dependency override

lyrebird 0.10.0 pins `golang.org/x/net` v0.59.0. Two HIGH advisories,
CVE-2026-78661 and CVE-2026-78669, are fixed in v0.62.0. The build runs
`go get golang.org/x/net@v0.62.0` in the cloned lyrebird source before
compiling. Remove the override when an upstream lyrebird tag pins v0.62.0 or
newer.

## Go modules

The transports build from their upstream `go.sum` and the Go checksum
database. `torinit` uses only the Go standard library and has no module
dependencies.

## Base images and tools

- The Debian build image, the distroless runtime image, and the Go toolchain
  image are pinned by digest. Dependabot updates the digests.
- GitHub Actions are pinned by full commit SHA with a version comment.
- CI tools are versioned: `actionlint` at `v1.7.12`, Trivy at `v0.70.0`.
- The Dockerfile frontend is not fetched remotely. The file has no `# syntax`
  line, so BuildKit uses its built-in frontend.

## CI identity and registry access

- Images are signed with keyless cosign through GitHub OIDC.
- Publishing is restricted to `main`, and the verification commands use the
  exact workflow identity rather than a regular expression.
- Docker Hub credentials live in the `release` environment, not in
  repository-wide secrets.

## Source of truth

`main` is protected by a ruleset. Changes to the Dockerfile, workflows,
scripts, keys, and version files need review through CODEOWNERS.

## Known gaps

- Debian packages in the build stage are installed by apt with signature
  checking but are not version-pinned. A rebuild is not byte-for-byte
  reproducible. Provenance and the cosign signature cover who built it and
  from which commit.
- The vulnerability scan sees the distroless base and the Go binaries. The
  copied shared libraries and the compiled tor binary are not visible to a
  package scanner. The weekly rebuild and the release watcher are the
  compensating controls.
