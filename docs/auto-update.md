# Auto-updating the image

CI publishes a new image when the Tor Project releases a new stable tor
version, and rebuilds the current version weekly to pick up base image
security updates. This guide shows three ways to keep a relay on the newest
image.

Auto-update works only with a moving tag such as `latest` or `0.4.9`. If you
pin an exact version such as `0.4.9.17`, no updater will move you to a new
release. Choose a moving tag for automatic updates, or update the pinned tag
yourself.

Every method below recreates the container, so the relay restarts. A relay
restart is normal and does not change the fingerprint, but choose a quiet
time and avoid updating several relays at once.

## Recommended: systemd timer with Compose

This method needs no extra container and no access to the Docker socket. It
runs `docker compose pull` and `docker compose up -d` on a timer.

Create a Compose file at `/srv/tor-relay/compose.yml`:

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
    volumes:
      - tor-data:/var/lib/tor

volumes:
  tor-data:
```

Create the service unit at `/etc/systemd/system/tor-relay-update.service`:

```
[Unit]
Description=Update the tor-relay image and recreate the container
Requires=docker.service
After=docker.service network-online.target
Wants=network-online.target

[Service]
Type=oneshot
WorkingDirectory=/srv/tor-relay
ExecStart=/usr/bin/docker compose pull
ExecStart=/usr/bin/docker compose up -d
```

Create the timer at `/etc/systemd/system/tor-relay-update.timer`:

```
[Unit]
Description=Daily tor-relay image update

[Timer]
OnCalendar=daily
RandomizedDelaySec=1h
Persistent=true
Unit=tor-relay-update.service

[Install]
WantedBy=timers.target
```

Enable and start it:

```
sudo systemctl daemon-reload
sudo systemctl enable --now tor-relay-update.timer
```

Check that the timer is scheduled:

```
systemctl list-timers tor-relay-update.timer
```

Test one run by hand:

```
sudo systemctl start tor-relay-update.service
journalctl -u tor-relay-update.service -n 50
```

`RandomizedDelaySec=1h` spreads the pull across the hour, which avoids
hitting registry rate limits when several relays share a host or network.
`Persistent=true` runs the update after a missed schedule, for example after
a reboot.

If the relay is not managed by Compose, replace the two `ExecStart` lines
with the equivalent `docker pull` and `docker run` commands, or move the
deployment to Compose.

## Watchtower

Watchtower watches containers and recreates them when a new image appears.
It needs access to the Docker socket, which is effectively root access on
the host. Use it only if that trade-off is acceptable.

Start Watchtower with label filtering so it only touches containers that opt
in:

```
docker run -d \
  --name watchtower \
  --restart unless-stopped \
  -v /var/run/docker.sock:/var/run/docker.sock \
  containrrr/watchtower:1.7.1 \
  --label-enable \
  --cleanup
```

Add the opt-in label to the relay container:

```
services:
  tor-relay:
    image: docker.io/unshakled/tor-relay:latest
    labels:
      - "com.centurylinklabs.watchtower.enable=true"
```

Without `--label-enable`, Watchtower watches every container on the host.
With it, only containers that carry the enable label are updated.

`--cleanup` removes the old image after a successful update. Keep it if disk
space matters; drop it if you want to roll back to the previous image.

Watchtower checks for a new image periodically and recreates the container
when the digest changes. It does not use Compose, so any Compose-only
settings must be present as labels or as the same `docker run` flags.

## Podman auto-update

Podman can update containers that run under systemd. The container must
carry the `io.containers.autoupdate=registry` label, or the Quadlet unit must
set `AutoUpdate=registry`.

Create a Quadlet unit at `/etc/containers/systemd/tor-relay.container`:

```
[Unit]
Description=tor-relay

[Container]
Image=docker.io/unshakled/tor-relay:latest
Network=host
AutoUpdate=registry
Volume=tor-data:/var/lib/tor
EnvironmentFile=/srv/tor-relay/.env
ReadOnly=true
DropCapability=ALL
NoNewPrivileges=true

[Install]
WantedBy=default.target
```

Reload systemd and start the unit:

```
sudo systemctl daemon-reload
sudo systemctl start tor-relay.service
```

Podman ships a `podman-auto-update.timer` that runs daily at midnight. Enable
it if it is not already active:

```
sudo systemctl enable --now podman-auto-update.timer
```

Check for a pending update without changing anything:

```
sudo podman auto-update --dry-run --format "{{.Image}} {{.Updated}}"
```

Run the update:

```
sudo podman auto-update
```

The registry policy requires a fully qualified image reference, which
`docker.io/unshakled/tor-relay:latest` is. Podman compares the local digest
with the registry digest, pulls the new image, and restarts the systemd unit.
It rolls back to the previous image by default if the restart fails.

## Choosing an update window

A relay restart drops existing circuits. Tor clients retry through other
relays, so a short restart is not harmful, but avoid restarting many relays
at the same moment.

- Use `RandomizedDelaySec` in the systemd timer.
- Stagger Watchtower instances with `--schedule` if you run several.
- Update exit relays one at a time, and check Relay Search afterwards.

## Verifying an update

After any update, confirm that the container is running and that the
fingerprint did not change:

```
docker exec tor-relay /usr/local/bin/torinit fingerprint
docker logs tor-relay | grep "Bootstrapped 100%"
```

The fingerprint must match the value from before the update. If it changed,
the data volume was not preserved or the keys are no longer readable by UID
65532.

Verify the image signature before trusting a new pull:

```
cosign verify docker.io/unshakled/tor-relay:latest \
  --certificate-identity-regexp 'https://github.com/Unshakled-Inc/tor-relay/.github/workflows/.*' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

## Troubleshooting

- The timer runs but the container is not recreated: check
  `journalctl -u tor-relay-update.service`. `docker compose pull` may have
  failed because the Compose file is in the wrong directory, or because the
  registry returned an error.
- Watchtower does nothing: the container must use a moving tag and carry
  `com.centurylinklabs.watchtower.enable=true`, and Watchtower must run with
  `--label-enable`.
- Podman reports `no containers to update`: the Quadlet unit must set
  `AutoUpdate=registry` or the container must carry the
  `io.containers.autoupdate=registry` label, and it must run under systemd.
- The relay fingerprint changed after an update: the `/var/lib/tor` volume
  was not mounted, or its ownership changed. Restore the volume from backup
  and confirm that it is owned by 65532:65532 with mode 0700.
