# Migrating a relay from systemd to the container image

This guide moves an existing tor relay that runs under systemd onto the
`tor-relay` image without changing its identity. The relay keeps the same
fingerprint, nickname, and ORPort.

> **Warning:** never run the systemd instance and the container at the same
> time with the same keys. Two tor processes that share an identity confuse
> the directory authorities and can get the relay marked down. Stop and
> disable the systemd unit before you start the container.

## Before you start

- Docker Engine with the Compose plugin, or plain `docker run`.
- The same public IP address and ORPort as the systemd relay.
- Root or sudo on the relay host.
- A copy of the current tor configuration, so you can carry over the exit
  policy, bridge settings, and any extra options.

The examples below use `/var/lib/tor` for the systemd DataDirectory and
`/srv/tor-relay/data` for the container data directory. Adjust both if your
system uses different paths.

## 1. Back up the keys

Back up the whole data directory. Tor does not rewrite the identity keys
while it is running, so a live copy is safe, but a backup taken with the
service stopped is easier to reason about.

```
sudo cp -a /var/lib/tor "/root/tor-backup-$(date +%Y%m%d)"
sudo ls -la "/root/tor-backup-$(date +%Y%m%d)/keys"
```

Keep this backup until the relay has been running on the image for at least
a week and Relay Search shows it as Running.

## 2. Stop and disable systemd tor

On Debian and Ubuntu the packaged unit is `tor@default`:

```
sudo systemctl disable --now tor@default
sudo systemctl status tor@default
```

Confirm the unit is inactive and that no tor process is left:

```
pgrep -a tor
```

`pgrep` should print nothing. If your distribution uses a different unit
name, stop that unit instead. Do not continue until tor is stopped.

## 3. Create the data directory

The container runs as UID/GID 65532. Create the directory with the same
ownership and mode:

```
sudo install -d -m 0700 -o 65532 -g 65532 /srv/tor-relay/data
sudo install -d -m 0700 -o 65532 -g 65532 /srv/tor-relay/data/keys
```

A Docker named volume also works. A fresh named volume inherits the
ownership and mode of `/var/lib/tor` from the image, which is already
65532:65532 and 0700. If you copy keys into an existing named volume and
the ownership is wrong, fix it with a one-off container:

```
docker run --rm -v tor-data:/var/lib/tor debian:trixie-slim \
  chown -R 65532:65532 /var/lib/tor
```

## 4. Copy the identity keys

Copy the RSA identity key, the ed25519 master key, and any family keys:

```
sudo install -m 0600 -o 65532 -g 65532 \
  /var/lib/tor/keys/secret_id_key \
  /srv/tor-relay/data/keys/
sudo install -m 0600 -o 65532 -g 65532 \
  /var/lib/tor/keys/ed25519_master_id_secret_key \
  /srv/tor-relay/data/keys/
```

If the relay belongs to a family, copy the family keys too:

```
sudo install -m 0600 -o 65532 -g 65532 \
  /var/lib/tor/keys/*.secret_family_key \
  /srv/tor-relay/data/keys/
```

For an obfs4 or WebTunnel bridge, also copy the pluggable transport state so
the obfs4 certificate and bridge line stay the same:

```
sudo cp -a /var/lib/tor/pt_state /srv/tor-relay/data/
sudo chown -R 65532:65532 /srv/tor-relay/data/pt_state
```

If the relay also runs an onion service, copy `secret_onion_key` and
`secret_onion_key_ntor` as well. Relay identity does not need them.

Confirm the result:

```
sudo ls -la /srv/tor-relay/data/keys
```

Every key must be owned by 65532:65532 and mode `-rw-------`. The keys
directory must be mode `drwx------`.

Do not copy `lock`, `state`, or `control_auth_cookie`. Tor recreates them.
Do not copy `fingerprint` or `fingerprint-ed25519` either; tor regenerates
them from the keys on the next start.

## 5. Start the container with the same nickname and ORPort

Create an env file with the same nickname, ORPort, and contact information
as the systemd relay:

```
TOR_ROLE=middle
TOR_NICKNAME=YourRelayName
TOR_CONTACT=operator@example.com
TOR_ORPORT=9001
```

If the systemd relay is an exit, carry over `I_UNDERSTAND_EXIT=true` and the
exit policy. If it is a bridge, carry over the obfs4 or WebTunnel settings.
See the README role sections and `examples/` for the full variables.

Verify the configuration before starting:

```
docker run --rm --env-file .env \
  -v /srv/tor-relay/data:/var/lib/tor \
  docker.io/unshakled/tor-relay:latest verify
```

Start the relay with host networking so it sees the same public address and
can bind the same ORPort:

```
docker run -d \
  --name tor-relay \
  --network host \
  --read-only \
  --cap-drop ALL \
  --security-opt no-new-privileges \
  --restart unless-stopped \
  --stop-timeout 45 \
  --log-opt max-size=10m \
  --log-opt max-file=3 \
  -v /srv/tor-relay/data:/var/lib/tor \
  --env-file .env \
  docker.io/unshakled/tor-relay:latest
```

Then follow the log:

```
docker logs -f tor-relay
```

## 6. Confirm the fingerprint

Compare the fingerprint on the image with the fingerprint in the backup.
Print both fingerprints from the running container:

```
docker exec tor-relay /usr/local/bin/torinit fingerprint
```

The output has two lines:

```
RSA: <40 hex characters>
ED25519: <40 characters>
```

Compare the RSA fingerprint with the first fingerprint in the backup:

```
sudo cat "/root/tor-backup-<date>/fingerprint"
```

The RSA fingerprint must match exactly. If it does not, tor generated new
keys, which means the copied keys are missing, in the wrong directory, or
not readable by UID 65532. Stop the container, fix the keys, and start it
again before the relay publishes the new identity.

The `fingerprint` subcommand reads the files that tor writes after its first
start. If it reports a missing file, wait for tor to finish starting and run
it again.

## 7. Watch Relay Search

Look up the fingerprint on Relay Search:

```
https://metrics.torproject.org/rs.html#search/<RSA-FINGERPRINT>
```

The relay should return with the same fingerprint and the same nickname. It
can take a few hours to reappear after the move. Check that it reaches
`Bootstrapped 100%` in the log and that its self-test succeeds.

Flags such as `Running` and `Fast` return after the directory authorities
observe the relay. A short gap is normal. If the relay stays absent for more
than a day, check that the ORPort is reachable from the internet and that
the advertised address matches the host's public address.

## Rolling back

If you need to return to systemd:

```
docker stop tor-relay
sudo systemctl enable --now tor@default
```

The systemd instance reads `/var/lib/tor`, which still has the original
keys. Do not run both at once.

## Troubleshooting

- `Permission denied` on `/var/lib/tor`: the data directory and key files
  must be owned by 65532:65532, with mode 0700 on directories and 0600 on
  files.
- The fingerprint changed: the RSA key was not copied, or tor could not read
  it and generated a new one. Restore the backup and copy
  `keys/secret_id_key` again.
- `Unable to find IPv4 address for ORPort`: use `--network host`, or set
  `TOR_ADDRESS` to the public address.
- The relay does not reappear on Relay Search: confirm the ORPort is open
  and that no firewall rule still points at the old process.
- Family membership is missing: copy `*.secret_family_key` into
  `keys/` and set `TOR_FAMILY_ID` to the family ID from the old torrc.
