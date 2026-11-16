# WebTunnel bridge example

A WebTunnel bridge imitates ordinary HTTPS browsing. It needs a domain you control, a valid TLS certificate, and a reverse proxy in front. Read the Tor Project's WebTunnel guide before you start:

https://community.torproject.org/relay/setup/webtunnel/

The compose file publishes the WebTunnel listener on `127.0.0.1:15000` and sets `TOR_WEBTUNNEL_LISTEN=0.0.0.0:15000` inside the container. The reverse proxy on the host connects to `127.0.0.1:15000`. Tor binds `ORPort 127.0.0.1:auto` inside the container, so no ORPort mapping is needed.

The URL may use a secret path such as `https://example.com/secret-path`, or a root path such as `https://example.com/` when the whole domain is dedicated to WebTunnel. The nginx location must match the URL path.

1. Copy `nginx.conf.example` into your nginx configuration, replace `example.com`, the certificate paths, and the location path, then reload nginx.
2. Set `TOR_WEBTUNNEL_URL` in `.env.example` to the same https URL.
3. Start the bridge:

```
docker compose up -d
```

The bridge line has this shape:

```
webtunnel 10.0.0.2:443 <FINGERPRINT> url=https://example.com/secret-path
```

`10.0.0.2` is a placeholder address required by the pluggable transport specification. Replace `<FINGERPRINT>` with the fingerprint from `/var/lib/tor/fingerprint`, and use your real URL.

`TOR_SANDBOX` defaults to `0` for WebTunnel bridges. Tor 0.4.9.17 reports that managed proxies are not compatible with Sandbox mode, so keep it at `0`.

For extra torrc options such as `MaxMemInQueues`, drop a `*.conf` file into `/etc/tor/torrc.d/`.

## Cloudflare

Cloudflare proxy works in front of the bridge. Keep the origin nginx on 443 with a certificate Cloudflare trusts, allow WebSocket Upgrade, and add Cloudflare's ranges with `set_real_ip_from` plus `real_ip_header CF-Connecting-IP` if you want the real client IP. Cloudflare terminates TLS at its edge, so `TOR_WEBTUNNEL_URL` is the public Cloudflare hostname.

Cloudflare Tunnel also works. Point the tunnel at `http://127.0.0.1:15000`, or at the origin nginx if you keep it. No public IP or open ports are needed. If the tunnel terminates TLS, the origin certificate can be a Cloudflare Origin Certificate.
