# obfs4 bridge example

An obfs4 bridge helps censored users reach the Tor network. Both the ORPort and the obfs4 port must be reachable from the internet. Read the Tor Project's bridge guide before you start:

https://community.torproject.org/relay/setup/bridge/

The example uses ORPort 9001 and obfs4 port 8443. Change both to ports that suit your host. Ports below 1024 need `cap_add: [NET_BIND_SERVICE]` in the compose file, because the container drops all capabilities.

Start the example:

```
docker compose up -d
```

After tor has started and lyrebird has generated its state, print the client bridge line:

```
docker compose exec tor-relay /usr/local/bin/torinit bridgeline
```

The line fills in the fingerprint from `/var/lib/tor/fingerprint`. Set `TOR_ADDRESS` in the env file to also fill in the public IP address and obfs4 port. Otherwise the `<IP ADDRESS>:<PORT>` placeholder remains for you to replace.

Do not set `MyFamily` on a bridge. Tor's man page recommends it for relays only.
