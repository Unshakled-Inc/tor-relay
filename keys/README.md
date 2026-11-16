# Tor release signing keys

`tor-release.asc` holds the armored public keys allowed to sign tor source releases.
The build dearmors it and verifies `tor-X.tar.gz.sha256sum.asc` with gpgv. Any failure stops the build.

| Signer | Fingerprint |
|---|---|
| Alexander Færøy | 514102454D0A87DB0767A1EBBE6A0531C18A9179 |
| David Goulet | B74417EDDF22AC9F9E90F49142E86A2A11F48D36 |
| Nick Mathewson | 2133BC600AB133E1D826D173FE43009C4607B1FB |

Sources cross-checked:
1. https://support.torproject.org/little-t-tor/verify-little-t-tor/
2. README.md in the tor source tree ("Keys that CAN sign a release")

Key material fetched from keys.openpgp.org by fingerprint. WKD at torproject.org served expired
copies of the Goulet and Mathewson keys and a different Færøy key on 2026-11-15, so it was not used.

Verified 2026-11-15 against tor 0.4.9.17: good signatures from Goulet and Færøy.

Refresh when a release fails verification with an expired or unknown key: re-fetch by fingerprint,
confirm the fingerprints against both sources, re-run the verification, commit.
