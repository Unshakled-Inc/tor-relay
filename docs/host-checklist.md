# Host checklist

The image cannot fix the host. Before you run a relay, check these on the
host itself.

## 1. SSH

- Use key authentication only. Set `PasswordAuthentication no` and
  `PermitRootLogin no` in `/etc/ssh/sshd_config`.
- Restrict SSH to your own addresses in the firewall if you can.
- Keep the host's SSH port off the public internet if a bastion is
  available.

## 2. Firewall

- Default deny inbound.
- Allow the ORPort (TCP) for middle and exit relays.
- Allow the ORPort and the obfs4 port for obfs4 bridges.
- WebTunnel bridges need 443 for the reverse proxy, not the ORPort.
- Allow outbound connections. Tor needs them for bootstrap and operation.
- If the provider has a cloud firewall, configure it too. A host firewall
  rule alone does not open the port.

## 3. Updates

- Enable `unattended-upgrades` for security updates.
- Schedule automatic reboots in a quiet window, or reboot by hand after a
  kernel update.
- Check `apt list --upgradable` and reboot if the kernel is pending. A
  running relay on an old kernel is a known risk.

## 4. Time

- Run `chrony` or `systemd-timesyncd`.
- Confirm the clock is correct. Tor depends on a correct clock for
  certificates and consensus.

## 5. Docker

- Install Docker Engine from Docker's official repository, not a distro
  package.
- Membership of the `docker` group is equivalent to root. Keep it to
  administrators.
- Do not expose the Docker socket to containers unless they need it.

## 6. Logs and disk

- Set Docker log rotation: `max-size: 10m` and `max-file: 3`, or the
  `--log-opt` equivalents.
- Watch free disk space. A full disk stops the relay.

## 7. Keys

- Back up `/var/lib/tor/keys` after the first start. See
  [backup.md](backup.md).
- Store the backup off the host.
- Never run two relays from the same backup at the same time.

## 8. One identity per container

- Run one relay identity per container.
- If you run more than one relay, declare the family on all of them.
- For bridges, never put the bridge on the same IP as a public relay, and
  never publish the bridge line.

## 9. Resource limits

- A busy relay needs many file descriptors. Set `nofile` explicitly, for
  example `65535`.
- Set `pids_limit` to a sane cap, for example `512`.
- Leave memory uncapped by default. Tor sizes `MaxMemInQueues` from system
  memory, and a cgroup limit below that causes OOM kills. On a small host,
  set `MaxMemInQueues` through `TOR_EXTRA_ARGS`.

## 10. Monitoring

- Watch the Docker healthcheck and the tor log for `Bootstrapped 100%`.
- Add the relay to Relay Search and check that it stays Running.
- Consider Tor Weather or an uptime monitor for the ORPort.
