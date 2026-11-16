# Exit relay example

An exit relay forwards Tor traffic to the internet. Run it on a dedicated host, and read the Tor Project's exit relay guidelines before you start:

https://community.torproject.org/relay/setup/exit/

No default exit policy ships in this image. Either set `TOR_EXIT_POLICY` or put an `ExitPolicy` line in a `*.conf` file under `/etc/tor/torrc.d/`.

`I_UNDERSTAND_EXIT=true` is required. It is not prefixed with `TOR_` because it is an acknowledgment, not a tor setting.

Start the example:

```
docker compose up -d
```

The compose file uses host networking and a named volume. Review the DNS guidance on the exit relay page, and point the host at a local caching resolver.
