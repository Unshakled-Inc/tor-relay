# Backup and restore

The relay identity lives in `/var/lib/tor/keys`. Losing it resets the
relay's reputation and forces a new fingerprint. Back it up once, after the
first start, and keep the copy off the host.

Never run two relays from the same backup at the same time.

## What to back up

- `keys/` contains `secret_id_key`, `ed25519_master_id_secret_key`, and any
  `*.secret_family_key`. This is the identity. It is required.
- `pt_state/` contains the obfs4 state for bridge roles. Copy it too, so the
  obfs4 certificate and bridge line do not change.
- `fingerprint` and `fingerprint-ed25519` are useful for comparison but tor
  regenerates them from the keys.
- Do not back up `lock`, `control_auth_cookie`, or `state`. Tor recreates
  them.

## Back up a named volume

The examples use a Docker volume named `tor-data`. Its files are under
`/var/lib/docker/volumes/tor-data/_data` on the host.

Install `age` first:

```
sudo apt install age
```

Generate a key on the host, or use an existing one:

```
age-keygen -o ~/.age/tor-relay.key
```

Back up the keys and the bridge state, encrypted:

```
sudo tar -C /var/lib/docker/volumes/tor-data/_data -czf - keys pt_state \
  | age -r "$(age-keygen -y ~/.age/tor-relay.key)" \
  > tor-relay-keys.tgz.age
```

Move `tor-relay-keys.tgz.age` off the host. Keep the age private key
somewhere else, ideally on a hardware token or in a password manager.

For a bind mount, replace the `tar -C` path with your host directory, for
example `/srv/tor-relay/data`.

## Restore

Stop the container first:

```
docker compose stop
```

Restore into the volume:

```
age -d -i ~/.age/tor-relay.key tor-relay-keys.tgz.age \
  | sudo tar -C /var/lib/docker/volumes/tor-data/_data -xzf -
sudo chown -R 65532:65532 /var/lib/docker/volumes/tor-data/_data/keys
sudo chmod 0700 /var/lib/docker/volumes/tor-data/_data/keys
sudo find /var/lib/docker/volumes/tor-data/_data/keys -type f -exec chmod 0600 {} \;
```

Start the container and compare the fingerprint with the one you recorded
before the backup:

```
docker compose up -d
docker exec tor-relay /usr/local/bin/torinit fingerprint
```

The RSA fingerprint must match. If it does not, the keys were not restored
into `keys/`, or tor could not read them and generated new ones.

## Restore into a fresh host

1. Create the volume with the same Compose project name, or create the
   bind-mount directory with mode 0700 and owner 65532.
2. Restore as above.
3. Start the container with the same nickname, ORPort, and role.
4. Confirm the fingerprint and wait for Relay Search to show the relay
   Running.

The old host must be stopped before the new one starts.

## Offline master key

By default tor stores the ed25519 master key on disk, in
`keys/ed25519_master_id_secret_key`. For a higher bar, tor supports
`OfflineMasterKey 1`, which keeps the master key off the relay and uses a
medium-term signing key instead.

That option changes the key workflow: you generate and sign the medium-term
key on a separate machine. The image does not manage it. See the Tor
Project guide:

https://trac.torproject.org/projects/tor/wiki/doc/TorRelaySecurity/OfflineKeys

If you use it, add `OfflineMasterKey 1` through `TOR_EXTRA_ARGS` (it is not
on the managed list) and back up the medium-term key material.
