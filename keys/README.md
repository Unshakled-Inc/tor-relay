# Tor release signing keys

`tor-release.asc` holds the armored public keys of the people who sign Tor
releases. The image build verifies `tor-X.tar.gz.sha256sum.asc` with `gpgv`
against this keyring before it compiles anything, so a bad or missing
signature fails the build closed.

A human verified the fingerprints in this keyring. They were taken from the
Tor Project's official "verify little-t-tor" instructions and cross-checked
against a second Tor Project source. The fingerprints and the verification
date are recorded below.

## Verified fingerprints

Not recorded yet. They are added, with the verification date, in the
`build: pin tor release signing keys` commit that adds `tor-release.asc`.
