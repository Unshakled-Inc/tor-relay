# Family keys and MyFamily

A Tor relay family is a group of relays run by the same operator. Members
advertise each other so clients can avoid using two relays from the same
family in one circuit. This guide covers the family key, how to distribute
it, how to rotate it, and how to publish a MyFamily list.

## How family identity works

A family has two parts:

- A secret family key. Tor reads it from `DataDirectory/keys` and its
  filename must end in `.secret_family_key`.
- A public family ID. Tor publishes it in `FamilyId` lines. The ID is derived
  from the key and is safe to share.

Every relay in the family needs the same key and the same `FamilyId`. Each
relay also lists the other members in `MyFamily`.

The secret key is not the relay's identity key. Losing it does not change
the relay fingerprint, but it does break family membership until the key is
restored or rotated.

## Generate a family key

Generate the key on a trusted host, not on a shared build machine:

```
mkdir -p family
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v "$PWD/family:/out" \
  docker.io/unshakled/tor-relay:latest \
  keygen-family myfamily
```

This writes two files:

- `family/myfamily.secret_family_key`
- `family/myfamily.public_family_id`

It also prints the line to use in `TOR_FAMILY_ID`, for example:

```
FamilyId AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA
```

Keep the secret key out of git. The repository's `.gitignore` ignores
`*.secret_family_key`.

## Distribute the key as a Docker secret

With Docker Compose, mount the key as a secret:

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
    stop_grace_period: 45s
    logging:
      driver: json-file
      options:
        max-size: "10m"
        max-file: "3"
    pids_limit: 512
    ulimits:
      nofile:
        soft: 65535
        hard: 65535
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

Add the ID from `keygen-family` to the env file:

```
TOR_FAMILY_ID=<family-id>
```

`torinit` copies `/run/secrets/myfamily` to
`/var/lib/tor/keys/myfamily.secret_family_key` with mode 0600. The mounted
secret name does not need the `.secret_family_key` suffix, because `torinit`
normalizes it when `TOR_FAMILY_ID` is set.

For a Docker Swarm deployment, create the secret with `docker secret create`
and reference it as external:

```
secrets:
  myfamily:
    external: true
```

With plain `docker run`, bind-mount the directory:

```
docker run -d \
  --name tor-relay \
  --network host \
  --stop-timeout 45 \
  --log-opt max-size=10m \
  --log-opt max-file=3 \
  -v /srv/tor-relay/family:/run/secrets:ro \
  --env-file .env \
  docker.io/unshakled/tor-relay:latest
```

The same rules apply: every file in the directory is treated as a family
key when `TOR_FAMILY_ID` is set, except files ending in `.public_family_id`.

If `TOR_FAMILY_ID` is set and no key is found, the container exits with an
error. If a key is present and `TOR_FAMILY_ID` is not set, it also exits.
This prevents a relay from silently losing its family membership.

## Multiple family IDs

`TOR_FAMILY_ID` accepts more than one ID, separated by spaces. Each ID needs
its own key in the secret directory:

```
TOR_FAMILY_ID=<id-one> <id-two>
```

This is how rotation works. Tor renders one `FamilyId` line per ID.

## Rotate a family key

Rotation keeps the relay in the family while the new key propagates.

1. Generate a new key with a different name:

```
docker run --rm \
  --user "$(id -u):$(id -g)" \
  -v "$PWD/family:/out" \
  docker.io/unshakled/tor-relay:latest \
  keygen-family myfamily2
```

2. Add `family/myfamily2.secret_family_key` to the secret directory next to
   the old key.

3. Set `TOR_FAMILY_ID` to both IDs, old first, then new:

```
TOR_FAMILY_ID=<old-id> <new-id>
```

4. Restart the container and confirm that both keys were installed:

```
docker compose up -d --force-recreate
docker logs tor-relay | grep -E "installed family key|replaced family key"
```

5. Wait for propagation. Directory authorities include family IDs in the
   consensus, so give them time to pick up the new ID. This can take a few
   hours, depending on the consensus schedule. Confirm that the relay is
   still Running on Relay Search.

6. Remove the old key from the secret directory and remove the old ID from
   `TOR_FAMILY_ID`. Restart the container.

7. Confirm that Relay Search shows the relay Running and that its family
   member fingerprints are still listed.

Do not remove the old key before the new ID appears on Relay Search. A relay
that advertises an ID without the matching key fails its family check.

## MyFamily

`MyFamily` lists the other relays in the family. Tor uses it to keep
same-family relays out of the same circuit. Set it directly or sync it from
an https URL.

### Direct mode

```
TOR_MYFAMILY=$0123456789ABCDEF0123456789ABCDEF01234567,$89ABCDEF0123456789ABCDEF0123456789ABCDEF
```

The value is a comma-separated list of 40-hex fingerprints. A leading `$` is
optional on each entry. `torinit` normalizes the fingerprints to upper case
and renders one `MyFamily` line.

### URL mode

URL mode keeps the list current without restarting the container:

```
TOR_MYFAMILY_URL=https://example.com/family.txt
TOR_MYFAMILY_REFRESH=1h
TOR_MYFAMILY_MAX_DROP_PCT=20
```

The URL must be https. The file contains one 40-hex fingerprint per line.
Blank lines and `#` comments are ignored, and a leading `$` is optional.
Setting both `TOR_MYFAMILY` and `TOR_MYFAMILY_URL` is an error.

In URL mode:

- `torinit` fetches the list at startup with a 15 second timeout. Redirects
  to http are refused.
- It writes `/var/lib/tor/.torinit/family.conf` atomically and excludes this
  relay's own fingerprint when `/var/lib/tor/fingerprint` exists.
- If the startup fetch fails, an existing `family.conf` is kept. If there is
  none, an empty file is written. Tor always starts, and the failure is
  logged.
- A background loop refetches every `TOR_MYFAMILY_REFRESH` (minimum `10m`).
- An update is rejected when the response does not parse, when it is empty
  while the current list is not, or when it drops more than
  `TOR_MYFAMILY_MAX_DROP_PCT` percent of the current entries. The old file is
  kept and the reason is logged.
- When the content changes, `family.conf` is rewritten and tor receives
  `SIGHUP`, which reloads the family list.

Do not set `MyFamily` on a bridge. Tor's man page recommends it for relays
only, because listing other relays would reveal them.

## Publishing a MyFamily list

Serve a plain text file over https. One fingerprint per line, optionally
prefixed with `$`:

```
# Family list for ExampleRelay
$0123456789ABCDEF0123456789ABCDEF01234567
$89ABCDEF0123456789ABCDEF0123456789ABCDEF
```

A static file on an existing web server is enough. For example, with nginx:

```
server {
    listen 443 ssl;
    server_name family.example.com;

    location = /family.txt {
        default_type text/plain;
        root /var/www/family;
    }
}
```

Then set:

```
TOR_MYFAMILY_URL=https://family.example.com/family.txt
```

The file must stay behind https. `torinit` refuses http redirects, so make
sure the server does not redirect to a plain http URL.

Add every relay fingerprint in the family. `torinit` removes the local
relay's own fingerprint automatically, so listing all members, including the
local one, is safe.

## Verify the result

Check the rendered configuration:

```
docker run --rm --env-file .env docker.io/unshakled/tor-relay:latest render
```

The output should contain one `FamilyId` line per ID and either a `MyFamily`
line or an include for `family.conf`.

Check the running relay:

```
docker exec tor-relay /usr/local/bin/torinit fingerprint
```

Use that fingerprint in Relay Search, and confirm the family IDs and members
are listed.

## Troubleshooting

- `FamilyId set but no *.secret_family_key found in /run/secrets`: the secret
  is not mounted, or `TOR_FAMILY_KEY_DIR` points at the wrong directory.
- `family key present but TOR_FAMILY_ID not set`: add `TOR_FAMILY_ID` with the
  ID printed by `keygen-family`.
- `TOR_FAMILY_ID "..." must be base64-like with no whitespace`: separate
  multiple IDs with a single space and do not add punctuation.
- `myfamily: startup fetch failed`: tor still starts. Check the URL, DNS, and
  outbound https access. An existing `family.conf` is kept; otherwise the
  file is empty until the next successful refresh.
- `myfamily: refresh failed: refusing update`: the fetched list was empty or
  dropped too many entries. Check the file at `TOR_MYFAMILY_URL`. Raise
  `TOR_MYFAMILY_MAX_DROP_PCT` only if the drop is intentional.
- The relay is not shown as a family member: confirm that every member has
  the same key and ID, that `MyFamily` lists the other members, and that the
  consensus has had time to update.
