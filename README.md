# tor-relay

Hardened, signed, multi-arch Docker image for Tor relay and bridge operators, published by Unshakled Inc. Tor is compiled from the signed upstream source tarball, never from distro packages.

Status: the middle and exit relay roles, the obfs4 and WebTunnel bridge roles, family keys, and MyFamily are available now. The debug variant and metrics are in progress.

## What the image provides

- Tor built from the signed source release, with the signature checked against `keys/tor-release.asc` before compilation.
- A distroless runtime on `gcr.io/distroless/cc-debian13`, with no shell and no package manager.
- Runs as UID/GID 65532 and works with a read-only root filesystem.
- Works with all Linux capabilities dropped and `no-new-privileges`.
- `torinit` renders the torrc from environment variables, writes it to `/var/lib/tor/.torinit`, starts tor, forwards signals, and records the PID.
- Installs family keys from `/run/secrets` and renders `FamilyId` lines.
- Renders `MyFamily` from an env var, or syncs it from an https URL and reloads tor when the list changes.
- Ships lyrebird for obfs4 bridges and the webtunnel server for WebTunnel bridges, and prints the client bridge line.
- Multi-arch images for `linux/amd64`, `linux/arm64`, and `linux/arm/v7`.
- Keyless cosign signatures, SBOM, and provenance attestations from CI.
- Images published to `docker.io/unshakled/tor-relay` and `ghcr.io/unshakled-inc/tor-relay`.

## Quick start: middle relay

Requirements:

- Docker Engine with Buildx support.
- A host with a public IPv4 or IPv6 address.
- ORPort reachable from the internet. Host networking is recommended.

Replace the nickname and contact with your own values. Nicknames are not unique on the Tor network, but a distinct name makes the relay easier to find.

1. Create a working directory and an env file:

```
mkdir tor-relay
cd tor-relay
cat > .env <<'EOF'
TOR_ROLE=middle
TOR_NICKNAME=ExampleRelay
TOR_CONTACT=operator@example.com
TOR_ORPORT=9001
EOF
```

2. Verify the rendered configuration before starting:

```
docker run --rm --env-file .env docker.io/unshakled/tor-relay:latest verify
```

3. Start the relay:

```
docker run -d \
  --name tor-relay \
  --network host \
  --read-only \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --restart unless-stopped \
  -v tor-data:/var/lib/tor \
  --env-file .env \
  docker.io/unshakled/tor-relay:latest
```

4. Follow the log:

```
docker logs -f tor-relay
```

5. Confirm the relay is bootstrapping and note the fingerprint:

Look for `Bootstrapped 100%` and `Your Tor server's identity key fingerprint`. The fingerprint is also written to `/var/lib/tor/fingerprint`.

### Docker Compose

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
    volumes:
      - tor-data:/var/lib/tor

volumes:
  tor-data:
```

Start it with:

```
docker compose up -d
```

The repository also has `examples/middle/compose.yml`, which uses `tor-relay:dev` for local builds. It shows the commented-out bridge networking alternative and the trade-off.

## Roles

| Role | Status | Required variables |
| --- | --- | --- |
| `middle` | available | `TOR_ROLE=middle`, `TOR_NICKNAME`, `TOR_CONTACT` |
| `exit` | available | `TOR_ROLE=exit`, `I_UNDERSTAND_EXIT=true`, and `TOR_EXIT_POLICY` or an `ExitPolicy` line in `/etc/tor/torrc.d/*.conf` |
| `bridge` | available | `TOR_ROLE=bridge`, `TOR_OBFS4_PORT` |
| `webtunnel` | available | `TOR_ROLE=webtunnel`, `TOR_WEBTUNNEL_URL` |

Only `middle`, `exit`, `bridge`, and `webtunnel` are accepted by the current release. Any other `TOR_ROLE` exits non-zero with a clear message.

## Configuration reference

The current release supports these variables:

| Variable | Required | Default | Rule |
| --- | --- | --- | --- |
| `TOR_ROLE` | yes | | `middle`, `exit`, `bridge`, or `webtunnel` |
| `TOR_NICKNAME` | yes | | `^[A-Za-z0-9]{1,19}$` |
| `TOR_CONTACT` | yes | | Non-empty, single line |
| `TOR_ORPORT` | no | `9001` | 1 to 65535 |
| `TOR_ADDRESS` | no | | Passed through to `Address` |
| `TOR_BANDWIDTH_RATE` | no | | Passed to `RelayBandwidthRate`, for example `50 MBytes` |
| `TOR_BANDWIDTH_BURST` | no | | Passed to `RelayBandwidthBurst` |
| `TOR_SANDBOX` | no | `1` | `0` or `1` |
| `TOR_LOG_LEVEL` | no | `notice` | `debug`, `info`, `notice`, `warn`, or `err` |
| `TOR_FAMILY_ID` | no | | One or more space-separated family IDs from `keygen-family` |
| `TOR_FAMILY_KEY_DIR` | no | `/run/secrets` | Directory containing family key files |
| `TOR_MYFAMILY` | no | | Comma-separated 40-hex fingerprints, optional `$` prefix |
| `TOR_MYFAMILY_URL` | no | | https URL to a fingerprint list, one per line |
| `TOR_MYFAMILY_REFRESH` | no | `1h` | Refresh interval, minimum `10m` |
| `TOR_MYFAMILY_MAX_DROP_PCT` | no | `20` | 0 to 100, maximum percent of entries that may disappear in one update |
| `I_UNDERSTAND_EXIT` | exit only | | Must be exactly `true` |
| `TOR_EXIT_POLICY` | exit, unless set in `torrc.d` | | Semicolon-separated exit policy entries, rendered as `ExitPolicy` lines |
| `TOR_EXIT_IPV6` | no | `0` | `0` or `1`, renders `IPv6Exit 1` when set |
| `TOR_OBFS4_PORT` | bridge only | | 1 to 65535, must differ from `TOR_ORPORT` |
| `TOR_BRIDGE_DISTRIBUTION` | no | `any` | `any`, `none`, `https`, `email`, or `settings` |
| `TOR_WEBTUNNEL_URL` | webtunnel only | | https URL with a host; the path may be `/` for a dedicated domain |
| `TOR_WEBTUNNEL_LISTEN` | no | `127.0.0.1:15000` | host:port for the webtunnel listener |
| `TOR_DNS_RESOLV_CONF` | no | | Path rendered as `ServerDNSResolvConfFile`, mainly for exit relays |
| `TOR_EXTRA_ARGS` | no | | Semicolon-separated torrc lines, rendered verbatim at the end |

Any other variable starting with `TOR_` is rejected with an error naming it. This catches typos instead of silently ignoring them.

`TOR_SANDBOX=1` enables tor's seccomp sandbox. It is the default. Set it to `0` only if you are debugging a sandbox-related failure.

### Extra torrc options

`TOR_EXTRA_ARGS` appends arbitrary torrc lines. Separate entries with semicolons, for example:

```
TOR_EXTRA_ARGS=MaxMemInQueues 128 MB;DirPort 80
```

It is rendered after the role-specific settings and after `Sandbox`, and before the `/etc/tor/torrc.d/` include. Any single-line directive from the pinned tor man page is allowed. `torinit` does not parse the directive; tor validates it at startup and in `torinit verify`.

`TOR_DNS_RESOLV_CONF` renders `ServerDNSResolvConfFile <path>`, mainly for exit relays and Cloudflare DNS resolver files.

See [docs/extra-args.md](https://github.com/Unshakled-Inc/tor-relay/blob/main/docs/extra-args.md) for the full format, allowed directives, ordering rules, managed directives, and examples for memory, DNS, DirPort, accounting, logging, metrics, and Cloudflare.

### Rendered torrc

The middle relay torrc is rendered in a fixed order. A minimal configuration looks like this:

```
# generated by torinit, do not edit
DataDirectory /var/lib/tor
Log notice stdout
RunAsDaemon 0
SocksPort 0
Nickname ExampleRelay
ContactInfo operator@example.com
ORPort 9001
ExitRelay 0
ExitPolicy reject *:*
ExitPolicy reject6 *:*
Sandbox 1
```

For `TOR_ROLE=exit`, `ExitRelay 1` replaces `ExitRelay 0`, the middle relay reject policies are not added, and each `TOR_EXIT_POLICY` entry becomes an `ExitPolicy` line. `TOR_EXIT_IPV6=1` adds `IPv6Exit 1`.

For `TOR_ROLE=bridge`, the middle relay reject policies are not added. The torrc adds `BridgeRelay 1`, `ServerTransportPlugin obfs4 exec /usr/local/bin/lyrebird`, `ServerTransportListenAddr obfs4 0.0.0.0:<TOR_OBFS4_PORT>`, `ExtORPort auto`, and `BridgeDistribution <value>`. `TOR_SANDBOX` defaults to `0` for bridges.

For `TOR_ROLE=webtunnel`, the torrc uses the directives from the Tor Project's WebTunnel guide: `BridgeRelay 1`, `ORPort 127.0.0.1:auto`, `AssumeReachable 1`, `ServerTransportPlugin webtunnel exec /usr/local/bin/webtunnel`, `ServerTransportListenAddr webtunnel <TOR_WEBTUNNEL_LISTEN>`, `ServerTransportOptions webtunnel url=<TOR_WEBTUNNEL_URL>`, and `ExtORPort auto`. Relay-only options such as `Address`, bandwidth, `FamilyId`, and `MyFamily` are not rendered for this role. `TOR_SANDBOX` defaults to `0`.

If `/etc/tor/torrc.d/` exists and contains at least one `*.conf` file, torinit appends:

```
%include /etc/tor/torrc.d/
```

Use that directory for operator-specific torrc options. Keep the generated file at `/var/lib/tor/.torinit/torrc` untouched.

If `TOR_MYFAMILY` is set, the torrc contains a single line listing the fingerprints, for example:

```
MyFamily $0123456789ABCDEF0123456789ABCDEF01234567,$89ABCDEF0123456789ABCDEF0123456789ABCDEF
```

If `TOR_MYFAMILY_URL` is set, the torrc contains:

```
%include /var/lib/tor/.torinit/family.conf
```

`torinit` keeps that file in sync and reloads tor when its content changes.

### torinit subcommands

The image entrypoint is `torinit`. The current subcommands are:

- `render`: print the rendered torrc to stdout.
- `verify`: render to a throwaway DataDirectory and run `tor --verify-config`.
- `run`: the default. Validate, write the config, start tor, and forward signals.
- `version`: print the torinit version and run `tor --version`.
- `healthcheck`: read the PID file and exit 0 only if `/proc/<pid>` exists.
- `keygen-family`: generate a family key in `/out` or the directory given by `--out`, print the `FamilyId` line, and refuse to overwrite an existing key unless `--force` is set.
- `bridgeline`: print the obfs4 client bridge line from `/var/lib/tor/pt_state/obfs4_bridgeline.txt`, with the fingerprint filled in from `/var/lib/tor/fingerprint`.

Examples:

```
docker run --rm --env-file .env docker.io/unshakled/tor-relay:latest render
docker run --rm --env-file .env docker.io/unshakled/tor-relay:latest verify
docker run --rm docker.io/unshakled/tor-relay:latest version
mkdir -p family
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD/family:/out" docker.io/unshakled/tor-relay:latest keygen-family myfamily
```

## Family keys

A family key lets a group of relays advertise that they belong to the same family. Generate a key on the host:

```
mkdir -p family
docker run --rm --user "$(id -u):$(id -g)" -v "$PWD/family:/out" docker.io/unshakled/tor-relay:latest keygen-family myfamily
```

This writes `family/myfamily.secret_family_key` and `family/myfamily.public_family_id`, and prints the `FamilyId` line. Add the printed ID to your env file:

```
TOR_FAMILY_ID=<id>
```

Mount the secret key as a Docker secret. With Compose:

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
    volumes:
      - tor-data:/var/lib/tor
    secrets:
      - myfamily

secrets:
  myfamily:
    file: ./family/myfamily.secret_family_key

volumes:
  tor-data:
```

The secret is mounted at `/run/secrets/myfamily`. `torinit` installs it as `myfamily.secret_family_key` in `/var/lib/tor/keys` with mode 0600.

If `TOR_FAMILY_ID` is set and no key file is found, the container exits with an error. If a key file is present and `TOR_FAMILY_ID` is not set, it also exits with an error. This prevents a relay from silently losing its family membership.

The `.public_family_id` file is not a secret. Keep it with the key file for reference and for publishing the ID to other operators.

## MyFamily

`MyFamily` tells tor which other relays belong to your family. You can set it directly or sync it from a URL.

Direct mode, in the env file:

```
TOR_MYFAMILY=$0123456789ABCDEF0123456789ABCDEF01234567,$89ABCDEF0123456789ABCDEF0123456789ABCDEF
```

The value is a comma-separated list of 40-hex fingerprints. A leading `$` is optional on each entry. `torinit` normalizes the fingerprints to upper case and renders one `MyFamily` line.

URL mode, in the env file:

```
TOR_MYFAMILY_URL=https://example.com/family.txt
TOR_MYFAMILY_REFRESH=1h
TOR_MYFAMILY_MAX_DROP_PCT=20
```

The URL must be https. The file contains one 40-hex fingerprint per line. Blank lines and `#` comments are ignored, and a leading `$` is optional. Setting both `TOR_MYFAMILY` and `TOR_MYFAMILY_URL` is an error.

In URL mode:

- `torinit` fetches the list at startup with a 15 second timeout. Redirects to http are refused.
- It writes `/var/lib/tor/.torinit/family.conf` atomically and excludes this relay's own fingerprint when `/var/lib/tor/fingerprint` exists.
- If the startup fetch fails, an existing `family.conf` is kept. If there is none, an empty file is written. Tor always starts, and the failure is logged.
- A background loop refetches every `TOR_MYFAMILY_REFRESH` (minimum `10m`).
- An update is rejected when the response does not parse, when it is empty while the current list is not, or when it drops more than `TOR_MYFAMILY_MAX_DROP_PCT` percent of the current entries. The old file is kept and the reason is logged.
- When the content changes, `family.conf` is rewritten and tor receives `SIGHUP`, which reloads the family list.

Do not set `MyFamily` on a bridge. Tor's man page recommends it for relays only, because listing other relays would reveal them.

## Exit relay

An exit relay forwards traffic from the Tor network to the internet. Run it on a dedicated host, and read the Tor Project's exit relay guidelines before you start:

https://community.torproject.org/relay/setup/exit/

Set the role and acknowledge the exit behavior:

```
TOR_ROLE=exit
I_UNDERSTAND_EXIT=true
```

Provide an exit policy either in the env file or in a `torrc.d` file. The env value is semicolon-separated and each entry becomes one `ExitPolicy` line:

```
TOR_EXIT_POLICY=accept *:80;accept *:443;reject *:*
```

The image does not add a default exit policy. If `TOR_EXIT_POLICY` is empty, `torinit` looks for an `ExitPolicy` line in `/etc/tor/torrc.d/*.conf`. If neither exists, the container exits with an error.

`TOR_EXIT_IPV6=1` adds `IPv6Exit 1`.

See `examples/exit/` for a minimal configuration.

## obfs4 bridge

An obfs4 bridge helps censored users reach the Tor network. Read the Tor Project's bridge guide before you start:

https://community.torproject.org/relay/setup/bridge/

Set the role and the obfs4 port:

```
TOR_ROLE=bridge
TOR_OBFS4_PORT=8443
```

Both the ORPort and the obfs4 port must be reachable from the internet, and the ports must differ. The image ships lyrebird and renders:

```
BridgeRelay 1
ServerTransportPlugin obfs4 exec /usr/local/bin/lyrebird
ServerTransportListenAddr obfs4 0.0.0.0:<port>
ExtORPort auto
BridgeDistribution <distribution>
```

`TOR_BRIDGE_DISTRIBUTION` defaults to `any`. The recognized values are `any`, `none`, `https`, `email`, and `settings`.

`TOR_SANDBOX` defaults to `0` for bridges. Tor 0.4.9.17 reports that managed proxies are not compatible with Sandbox mode, so obfs4 bridges must keep it at `0`.

After tor starts, print the client bridge line:

```
docker compose exec tor-relay /usr/local/bin/torinit bridgeline
```

Set `TOR_ADDRESS` to fill in the public IP address and obfs4 port. Otherwise the `<IP ADDRESS>:<PORT>` placeholder remains for you to replace.

Do not set `MyFamily` on a bridge. Tor's man page recommends it for relays only.

See `examples/bridge/` for a minimal configuration.

## WebTunnel bridge

A WebTunnel bridge imitates ordinary HTTPS browsing. It needs a domain you control, a valid TLS certificate, and a reverse proxy in front. Read the Tor Project's WebTunnel guide before you start:

https://community.torproject.org/relay/setup/webtunnel/

Set the role and the public URL. The path may be a secret path, or `/` when the whole domain is dedicated to WebTunnel:

```
TOR_ROLE=webtunnel
TOR_WEBTUNNEL_URL=https://example.com/secret-path
```

For a dedicated domain, `https://example.com/` is also valid. The nginx location must match the URL path.

The torrc uses the directives from the official guide:

```
BridgeRelay 1
ORPort 127.0.0.1:auto
AssumeReachable 1
ServerTransportPlugin webtunnel exec /usr/local/bin/webtunnel
ServerTransportListenAddr webtunnel 127.0.0.1:15000
ServerTransportOptions webtunnel url=https://example.com/secret-path
ExtORPort auto
```

`TOR_WEBTUNNEL_LISTEN` defaults to `127.0.0.1:15000`. The example compose publishes `127.0.0.1:15000` and sets the listener to `0.0.0.0:15000` inside the container. If you use host networking instead, keep the default `127.0.0.1:15000`.

`TOR_SANDBOX` defaults to `0` for WebTunnel bridges. Tor 0.4.9.17 reports that managed proxies are not compatible with Sandbox mode, so keep it at `0`.

For extra torrc options such as `MaxMemInQueues`, drop a `*.conf` file into `/etc/tor/torrc.d/`.

The bridge line has this shape:

```
webtunnel 10.0.0.2:443 <FINGERPRINT> url=https://example.com/secret-path
```

`10.0.0.2` is a placeholder address required by the pluggable transport specification. Use the fingerprint from `/var/lib/tor/fingerprint`.

See `examples/webtunnel/` for a compose file and an `nginx.conf.example`.

Cloudflare proxy and Cloudflare Tunnel work in front of this bridge. If Cloudflare proxies the hostname, keep the origin nginx on 443 with a certificate Cloudflare trusts, allow WebSocket Upgrade, and trust Cloudflare's client IP ranges in nginx. If you use Cloudflare Tunnel, point it at `http://127.0.0.1:15000`; no public IP or open ports are needed. Cloudflare terminates TLS at its edge, so the URL is the public Cloudflare hostname.

## Hardening

The runtime image is intentionally minimal:

- Distroless `cc-debian13` base, with no shell and no package manager.
- UID/GID 65532, owned data directory, mode 0700.
- Read-only root filesystem compatible.
- All capabilities can be dropped.
- `no-new-privileges` compatible.
- Tor sandbox enabled by default.

The image does not contain secrets. All configuration comes from environment variables and the `/var/lib/tor` volume.

## Data and persistence

`/var/lib/tor` holds the relay keys, the fingerprint, tor state, and the generated configuration under `.torinit/`.

- Use a named volume or a bind mount for `/var/lib/tor`.
- Back up the volume before moving or recreating the relay.
- The directory must be writable by UID/GID 65532.
- Never run two containers with the same keys at the same time.

The image declares `/var/lib/tor` as a volume. The runtime creates it with mode 0700 and ownership 65532:65532.

## Healthcheck

The image defines a Docker healthcheck that runs `torinit healthcheck`. It reads `/var/lib/tor/.torinit/pid` and checks whether `/proc/<pid>` exists.

This confirms that the tor process is alive. It does not confirm that the relay has bootstrapped or that the ORPort is reachable. Check the logs for `Bootstrapped 100%` and the relay's self-test results.

## Updating

Pull the new image and recreate the container:

```
docker pull docker.io/unshakled/tor-relay:latest
docker compose up -d
```

For a plain `docker run` deployment, stop and remove the old container, then start a new one with the same volume and env file.

The release watcher checks `dist.torproject.org` every six hours and starts a build when a new stable tor release is published. The weekly job rebuilds the current version without cache to pick up Debian and distroless security updates.

## Tags

CI publishes these tags:

- `<version>`, for example `0.4.9.17`
- `<series>`, for example `0.4.9`
- `latest`
- `<version>-<YYYYMMDD>`, for example `0.4.9.17-20261116`

The `-debug` variant will use the same tags with a `-debug` suffix once the debug image is implemented.

## Verifying signatures

Images are signed by CI with keyless cosign. Verify the Docker Hub image with:

```
cosign verify docker.io/unshakled/tor-relay:latest \
  --certificate-identity-regexp 'https://github.com/Unshakled-Inc/tor-relay/.github/workflows/.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

For GHCR:

```
cosign verify ghcr.io/unshakled-inc/tor-relay:latest \
  --certificate-identity-regexp 'https://github.com/Unshakled-Inc/tor-relay/.github/workflows/.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

The signature covers the multi-arch manifest digest. SBOM and provenance attestations are generated by the build workflow.

## Building from source

Clone the repository and build the runtime target:

```
git clone https://github.com/Unshakled-Inc/tor-relay.git
cd tor-relay
docker buildx build \
  --target runtime \
  --build-arg TOR_VERSION=$(cat VERSION) \
  -t tor-relay:dev \
  --load .
```

Verify the local image with the example env file:

```
docker run --rm --env-file examples/middle/.env.example tor-relay:dev verify
```

The builder downloads `tor-<version>.tar.gz`, its checksum file, and its signature from `https://dist.torproject.org/`. It verifies the signature with `gpgv` against `keys/tor-release.asc`, then checks the tarball checksum. A failed signature or checksum stops the build.

## Development

Run the Go tests and vet:

```
go test ./...
go vet ./...
```

Check formatting:

```
gofmt -l .
```

Lint the scripts and workflows:

```
shellcheck scripts/*.sh
actionlint
```

## Troubleshooting

- `Unable to find IPv4 address for ORPort`: set `TOR_ADDRESS` to the public address, or use host networking so tor can see the host interfaces.
- Bootstrapping stays at `5% (conn)`: outbound TCP connections to Tor ORPorts are blocked. Check the host firewall and the network policy.
- `Permission denied` on `/var/lib/tor`: the volume must be writable by UID/GID 65532. Fix ownership on the host before starting the container.
- `FamilyId set but no *.secret_family_key found in /run/secrets`: mount the family key as a Docker secret, or set `TOR_FAMILY_KEY_DIR` to the directory that contains it.
- `family key present but TOR_FAMILY_ID not set`: add `TOR_FAMILY_ID` to the env file with the ID printed by `keygen-family`.
- `TOR_MYFAMILY and TOR_MYFAMILY_URL cannot both be set`: pick direct mode or URL mode, not both.
- `myfamily: startup fetch failed`: tor still starts. Check the URL, DNS, and outbound https access. An existing `family.conf` is kept; otherwise the file is empty until the next successful refresh.
- `myfamily: refresh failed: refusing update`: the fetched list was empty or dropped too many entries. Check the file at `TOR_MYFAMILY_URL`. Raise `TOR_MYFAMILY_MAX_DROP_PCT` only if the drop is intentional.
- `TOR_ROLE=exit requires I_UNDERSTAND_EXIT=true`: set `I_UNDERSTAND_EXIT=true` in the env file. The value must be exactly `true`.
- `TOR_ROLE=exit requires TOR_EXIT_POLICY or an ExitPolicy line in /etc/tor/torrc.d/*.conf`: add an exit policy to the env file or to a `*.conf` file in the torrc.d directory.
- `TOR_ROLE=bridge requires TOR_OBFS4_PORT`: add the obfs4 port to the env file.
- `TOR_OBFS4_PORT ... must differ from TOR_ORPORT`: pick a different port for obfs4.
- `bridgeline: read bridge line ...`: tor has not started lyrebird yet, or the data directory does not contain `pt_state/obfs4_bridgeline.txt`. Check that the bridge is running.
- `bridgeline: read fingerprint ...`: tor has not written `/var/lib/tor/fingerprint` yet. Wait for the first start to finish.
- `Managed proxies are not compatible with Sandbox mode`: remove `TOR_SANDBOX=1` from an obfs4 or WebTunnel bridge. The bridge default is already `0`.
- `TOR_ROLE=webtunnel requires TOR_WEBTUNNEL_URL`: add the public https URL, including the path, to the env file.
- `TOR_WEBTUNNEL_LISTEN ... must be host:port`: use a value such as `127.0.0.1:15000` or `0.0.0.0:15000`.
- The reverse proxy returns 502: check that the bridge is running, that it listens on the address in `TOR_WEBTUNNEL_LISTEN`, and that the nginx location path matches the path in `TOR_WEBTUNNEL_URL`.
- `TOR_EXTRA_ARGS contains an empty entry`: remove the extra semicolon between entries.
- `TOR_EXTRA_ARGS must be a single line`: separate entries with semicolons, not newlines.
- `TOR_DNS_RESOLV_CONF must be a single line`: use one path to a resolver file.
- The container exits immediately: run `torinit verify` with the same env file. The error message names the invalid variable or value.
- `exec: "/bin/sh": stat /bin/sh: no such file or directory`: expected. The runtime image has no shell. Use `torinit` subcommands or `docker exec` with the torinit binary.
- Healthcheck is unhealthy: check the tor log and the PID file at `/var/lib/tor/.torinit/pid`.

## Roadmap

- Phase 2: debug variant with nyx, metrics.
- Phase 3: systemd migration guide, family keys guide, auto-update guide, security policy, changelog.

## Security

Report security issues privately through GitHub private vulnerability reporting:

`https://github.com/Unshakled-Inc/tor-relay/security/advisories/new`

The target is to publish a new image on the same day as a new upstream tor release. No secrets are baked into the image.

## License

BSD-3-Clause. See `LICENSE`.
