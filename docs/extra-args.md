# Extra torrc options

`TOR_EXTRA_ARGS` appends arbitrary torrc lines to the generated configuration. It is available for every role and is intended for options that do not have a dedicated environment variable.

This page documents the format, what is allowed, how the lines are ordered, what torinit manages itself, and common examples.

## Format

`TOR_EXTRA_ARGS` is one environment value. Separate entries with semicolons. Each entry becomes one torrc line.

```
TOR_EXTRA_ARGS=MaxMemInQueues 128 MB;DirPort 80
```

The generated torrc contains:

```
MaxMemInQueues 128 MB
DirPort 80
```

Rules:

- The value must be a single line. Newlines are rejected.
- Entries are split on `;`.
- Leading and trailing whitespace around each entry is removed.
- Empty entries are rejected. Do not use `;;`.
- Semicolons cannot appear inside a value. If a value needs a semicolon, use a `*.conf` file in `/etc/tor/torrc.d/` instead.
- The entry is written verbatim after torinit checks its first word against the managed list below. torinit does not parse the value.
- No shell quoting is applied. Write values exactly as they would appear in a torrc file.
- No environment variable expansion is applied. `$HOME` is literal.
- A `#` starts a comment. Both full-line comments and trailing comments work, for example `MaxMemInQueues 128 MB # keep the queue small`.
- A `%include` entry is rejected by default because the image already includes `/etc/tor/torrc.d/`. Set `TOR_ALLOW_UNSAFE_EXTRA_ARGS=1` to allow it. The referenced file must exist and be readable by UID/GID 65532.

## What is allowed

Any directive that tor accepts in a torrc file is allowed, provided it is a single-line entry. The pinned tor release for this image is 0.4.9.17. Its full option list is in the man page:

https://gitlab.torproject.org/tpo/core/tor/-/raw/tor-0.4.9.17/doc/man/tor.1.txt

tor validates the final configuration at startup and in `torinit verify`. If tor rejects a line, the container exits non-zero with tor's error message. torinit does not guess at values.

Common categories that work through `TOR_EXTRA_ARGS`:

| Category | Examples |
| --- | --- |
| Memory and CPU | `MaxMemInQueues 512 MB`, `NumCPUs 2`, `MaxOnionQueueDelay 1750 msec` |
| Connections | `ConnLimit 5000`, `ConstrainedSockets 1`, `ConstrainedSockSize 8192` |
| DNS for exit relays | `ServerDNSResolvConfFile /etc/tor/resolv.conf`, `ServerDNSDetectHijacking 0`, `ServerDNSAllowBrokenConfig 1`, `ServerDNSSearchDomains 1`, `ServerDNSRandomizeCase 1` |
| Directory service | `DirPort 80`, `DirPortFrontPage /etc/tor/exit-notice.html` |
| Accounting | `AccountingMax 1 TB`, `AccountingStart month 1 00:00` |
| Logging | `Log notice file /var/log/tor/notices.log` |
| Exit policy details | `ExitPolicyRejectPrivate 0`, `ExitPolicyRejectLocalInterfaces 1` |
| Testing | `TestingTorNetwork 1`, `DisableNetwork 1` |
| Private addresses | `ExtendAllowPrivateAddresses 1` |
| Hardware | `HardwareAccel 1` |

The categories overlap with dedicated variables in some cases. The table below shows when to use a dedicated variable instead.

## What is not allowed

- Newlines in the environment value. Use semicolons.
- Empty entries, such as `;;` or a trailing `;`.
- Semicolons inside a value. They split the entry.
- Shell syntax. Quotes, backticks, and `$(...)` are literal. Do not wrap values in quotes unless tor expects quotes.
- Multiple directives in one entry without a semicolon. `MaxMemInQueues 128 MB DirPort 80` is one line and tor will reject it.
- Referencing a file that is not mounted or not readable by UID/GID 65532.
- Overriding a directive that torinit must control, unless you have a reason. See the next section.

## Managed directives and the override

The generated torrc already sets these. `TOR_EXTRA_ARGS` rejects a managed directive unless you also set `TOR_ALLOW_UNSAFE_EXTRA_ARGS=1`. Use the dedicated variable where one exists.

The rejected directives are `ControlPort`, `ControlSocket`, `HashedControlPassword`, `SocksPort`, `Transport`, `NATDPort`, `DNSPort`, `HTTPTunnelPort`, `DataDirectory`, `RunAsDaemon`, `User`, `Sandbox`, `SafeLogging`, `MetricsPort`, `MetricsPortPolicy`, and `%include`. The error names the dedicated variable or the reason the image owns the option.

`TOR_ALLOW_UNSAFE_EXTRA_ARGS=1` is an explicit acknowledgment that you are overriding the image's defaults. For example, `TOR_EXTRA_ARGS=Sandbox 0` needs it.

Files in `/etc/tor/torrc.d/` are not blocked. torinit logs one warning per managed directive it finds there, then includes the file.

| Directive | Managed by | Notes |
| --- | --- | --- |
| `DataDirectory` | torinit | Fixed at `/var/lib/tor`. Rejected unless overridden. |
| `RunAsDaemon` | torinit | Fixed at `0`. Rejected unless overridden. |
| `SocksPort` | torinit | Set to `0`. Rejected unless overridden. |
| `Log` | `TOR_LOG_LEVEL` | Use `TOR_LOG_LEVEL` for the level. Extra `Log` lines are additive and allowed. |
| `Nickname` | `TOR_NICKNAME` | Use the dedicated variable. |
| `ContactInfo` | `TOR_CONTACT` | Use the dedicated variable. |
| `ORPort` | `TOR_ORPORT` | WebTunnel uses `ORPort 127.0.0.1:auto`. |
| `Address` | `TOR_ADDRESS` | Use the dedicated variable. |
| `RelayBandwidthRate`, `RelayBandwidthBurst` | `TOR_BANDWIDTH_RATE`, `TOR_BANDWIDTH_BURST` | Use the dedicated variables. |
| `FamilyId` | `TOR_FAMILY_ID` | Additive. Extra `FamilyId` lines add to the list. |
| `MyFamily` | `TOR_MYFAMILY` or `TOR_MYFAMILY_URL` | Additive. Extra `MyFamily` lines add to the list. |
| `ExitRelay`, `ExitPolicy`, `IPv6Exit` | Role and `TOR_EXIT_POLICY`, `TOR_EXIT_IPV6` | `ExitPolicy` is additive. Extra policies combine with the generated ones. |
| `BridgeRelay`, `ServerTransportPlugin`, `ServerTransportListenAddr`, `ServerTransportOptions`, `ExtORPort`, `BridgeDistribution` | Role and `TOR_OBFS4_PORT`, `TOR_BRIDGE_DISTRIBUTION`, `TOR_WEBTUNNEL_*` | Use the dedicated variables where they exist. |
| `AssumeReachable` | WebTunnel role | Already set to `1` for WebTunnel. |
| `Sandbox` | `TOR_SANDBOX` | Use the dedicated variable. Rejected unless overridden. |
| `ControlPort`, `CookieAuthentication` | `TOR_CONTROL_PORT` in the debug variant | Use `TOR_CONTROL_PORT` in the `latest-debug` image. |
| `MetricsPort`, `MetricsPortPolicy` | `TOR_METRICS_PORT`, `TOR_METRICS_POLICY`, `TOR_METRICS_PUBLIC` | Use the dedicated variables. They add the `reject *` fallback and enforce the public acknowledgment. |

If you set a scalar option in both a dedicated variable and `TOR_EXTRA_ARGS`, the extra arg wins because it is rendered later. Managed scalars need `TOR_ALLOW_UNSAFE_EXTRA_ARGS=1` for this to work. For example, `TOR_SANDBOX=1` plus `TOR_ALLOW_UNSAFE_EXTRA_ARGS=1` and `TOR_EXTRA_ARGS=Sandbox 0` produces `Sandbox 0`.

For additive options such as `ExitPolicy`, `MyFamily`, `FamilyId`, `Log`, and `ServerTransportPlugin`, extra lines add to the generated lines rather than replacing them.

## Ordering

`TOR_EXTRA_ARGS` is rendered after the role-specific settings and after `Sandbox`, and before the `/etc/tor/torrc.d/` include:

```
... role-specific settings ...
Sandbox 1
... TOR_EXTRA_ARGS entries ...
%include /etc/tor/torrc.d/
```

Consequences:

- Scalar options that use the last value win. Extra args can override the generated value.
- Additive options add entries. Extra args do not remove generated entries.
- Files in `/etc/tor/torrc.d/` are included last, so they can override both the generated settings and the extra args. Use torrc.d for settings that should take priority over `TOR_EXTRA_ARGS`.
- The generated torrc is rewritten on every start. Do not edit `/var/lib/tor/.torinit/torrc` directly.

## Common examples

### Memory and CPU

```
TOR_EXTRA_ARGS=MaxMemInQueues 512 MB;NumCPUs 2
```

tor warns if `MaxMemInQueues` is set very low. A value like `128 MB` is valid but may reduce relay reliability under load.

### DNS for an exit relay

```
TOR_EXTRA_ARGS=ServerDNSResolvConfFile /etc/tor/resolv.conf;ServerDNSDetectHijacking 0;ServerDNSSearchDomains 1;ServerDNSRandomizeCase 1
```

The resolver file must be mounted into the container. A minimal file for Cloudflare DNS:

```
nameserver 1.1.1.1
nameserver 1.0.0.1
```

You can also use the dedicated variable:

```
TOR_DNS_RESOLV_CONF=/etc/tor/resolv.conf
```

That renders `ServerDNSResolvConfFile /etc/tor/resolv.conf`. Use `TOR_EXTRA_ARGS` for the other `ServerDNS*` options.

### Directory port and exit notice

```
TOR_EXTRA_ARGS=DirPort 80;DirPortFrontPage /etc/tor/exit-notice.html
```

Mount the HTML file into the container. `DirPort` is deprecated since tor 0.4.6.5, but it still works and is useful for the exit notice page.

### Accounting

```
TOR_EXTRA_ARGS=AccountingMax 1 TB;AccountingStart month 1 00:00
```

### Logging to a file

```
TOR_EXTRA_ARGS=Log notice file /var/log/tor/notices.log
```

The runtime root filesystem is read-only. Mount a writable volume at `/var/log/tor`, or keep the default stdout logging.

### Control port for nyx

The debug variant has a dedicated variable:

```
TOR_CONTROL_PORT=9051
```

It renders `ControlPort 127.0.0.1:9051` and `CookieAuthentication 1`. The runtime image rejects `TOR_CONTROL_PORT`, and `TOR_EXTRA_ARGS` rejects `ControlPort` because the image owns it. If you need a control port in a runtime image, override the guard:

```
TOR_ALLOW_UNSAFE_EXTRA_ARGS=1
TOR_EXTRA_ARGS=ControlPort 127.0.0.1:9051;CookieAuthentication 1
```

### Metrics

The dedicated variables are preferred:

```
TOR_METRICS_PORT=0.0.0.0:9035
TOR_METRICS_POLICY=accept 10.0.0.0/8
```

`TOR_EXTRA_ARGS` can also set the same options, but it is rejected unless you override the guard, and it does not add the `reject *` fallback or enforce `TOR_METRICS_PUBLIC`:

```
TOR_ALLOW_UNSAFE_EXTRA_ARGS=1
TOR_EXTRA_ARGS=MetricsPort 0.0.0.0:9035;MetricsPortPolicy accept 10.0.0.0/8;MetricsPortPolicy reject *
```

### Testing

```
TOR_EXTRA_ARGS=TestingTorNetwork 1;DisableNetwork 1
```

Use these only on a test relay. `TestingTorNetwork` changes many defaults.

## Cloudflare

Cloudflare is not a torrc directive. It is an infrastructure choice in front of the relay or bridge.

- Cloudflare DNS: use `TOR_DNS_RESOLV_CONF` with a resolver file containing `nameserver 1.1.1.1`, or use `TOR_EXTRA_ARGS=ServerDNSResolvConfFile /path/to/resolv.conf`.
- Cloudflare proxy in front of a WebTunnel bridge: configure nginx with Cloudflare's client IP ranges and `real_ip_header CF-Connecting-IP`. See `examples/webtunnel/nginx.conf.example`.
- Cloudflare Tunnel in front of a WebTunnel bridge: point the tunnel at `http://127.0.0.1:15000`. No public IP or open ports are needed. See `examples/webtunnel/README.md`.
- Do not put Cloudflare API tokens or tunnel tokens in `TOR_EXTRA_ARGS`. They belong in the environment of the tunnel or proxy process, not in the torrc.

## Validation and troubleshooting

Before restarting, inspect the final configuration:

```
docker run --rm --env-file .env docker.io/unshakled/tor-relay:latest render
```

Then validate it:

```
docker run --rm --env-file .env docker.io/unshakled/tor-relay:latest verify
```

Common errors:

- `TOR_EXTRA_ARGS contains an empty entry`: remove the extra semicolon.
- `TOR_EXTRA_ARGS must be a single line`: separate entries with semicolons, not newlines.
- `TOR_EXTRA_ARGS <directive> is managed by the image`: use the dedicated variable named in the error, or set `TOR_ALLOW_UNSAFE_EXTRA_ARGS=1` if you really need to override it.
- `torinit: warning: <file>:<line>: <directive> is managed by the image`: a `torrc.d` file sets an option the image also sets. The file is included last, so it wins. Remove the line if that is not what you want.
- `Couldn't parse line` or `Unknown option`: tor rejected an entry. Check the directive name and value against the pinned man page.
- `No such file or directory`: a referenced file is not mounted or not readable by UID/GID 65532.
- `MaxMemInQueues is set to a low value`: the value is valid but small. Raise it if the relay is unreliable under load.

If tor rejects a line, the container exits non-zero. Read the tor log for the exact line and error. `torinit verify` reports the same error without starting the relay.

## Security notes

- `TOR_EXTRA_ARGS` is not evaluated by a shell. Quotes and variables are literal.
- The generated torrc is written with mode 0600 and owned by UID/GID 65532.
- Values end up in `/var/lib/tor/.torinit/torrc`. Do not put long-lived secrets there; use Docker secrets for family keys.
- Referenced files must be mounted into the container and readable by UID/GID 65532.
- The root filesystem is read-only at runtime. Extra args that write files, such as `Log ... file`, need a writable mount.

## Reference

- Pinned tor man page for 0.4.9.17: https://gitlab.torproject.org/tpo/core/tor/-/raw/tor-0.4.9.17/doc/man/tor.1.txt
- Tor Project relay documentation: https://community.torproject.org/relay/
- WebTunnel guide: https://community.torproject.org/relay/setup/webtunnel/
- obfs4 bridge guide: https://community.torproject.org/relay/setup/bridge/
- Exit relay guide: https://community.torproject.org/relay/setup/exit/
