# WebTunnel bridge example

A WebTunnel bridge imitates ordinary HTTPS browsing. It needs a domain you control, a valid TLS certificate, and a reverse proxy in front. Read the Tor Project's WebTunnel guide before you start:

https://community.torproject.org/relay/setup/webtunnel/

The compose file uses host networking so the reverse proxy on the host can reach the WebTunnel listener at `127.0.0.1:15000`. Tor binds `ORPort 127.0.0.1:auto` and uses `AssumeReachable 1`, so no public ORPort is needed.

1. Copy `nginx.conf.example` into your nginx configuration, replace `example.com`, the certificate paths, and `/secret-path` with your values, then reload nginx.
2. Set `TOR_WEBTUNNEL_URL` in `.env.example` to the same https URL, including the secret path.
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
