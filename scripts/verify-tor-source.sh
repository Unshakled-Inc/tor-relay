#!/bin/sh
# Verify a tor source release before building it.
#
# Arguments: <version> <keyring> <workdir>
#
# Downloads tor-<version>.tar.gz, its .sha256sum, and the .asc signature over
# that checksum file from dist.torproject.org, then checks the signature with
# gpgv and the tarball with sha256sum. Any failure exits non-zero: there is no
# fallback path.
set -eu

if [ "$#" -ne 3 ]; then
    echo "usage: $0 <version> <keyring> <workdir>" >&2
    exit 2
fi

version=$1
keyring=$2
workdir=$3

# Reject anything that is not four dot-separated numeric parts, so a typo or a
# stray suffix cannot silently download the wrong release.
if ! printf '%s\n' "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'; then
    echo "error: version must be four numeric parts, got '$version'" >&2
    exit 2
fi

if [ ! -s "$keyring" ]; then
    echo "error: keyring missing or empty: $keyring" >&2
    echo "error: dearmor keys/tor-release.asc first" >&2
    exit 1
fi

# Resolve relative paths against the caller's directory before cd, so a
# relative keyring or work dir keeps working after the cd below.
case "$keyring" in
    /*) ;;
    *) keyring=$(pwd)/$keyring ;;
esac
case "$workdir" in
    /*) ;;
    *) workdir=$(pwd)/$workdir ;;
esac

mkdir -p "$workdir"
cd "$workdir"

base=https://dist.torproject.org
for name in "tor-$version.tar.gz" "tor-$version.tar.gz.sha256sum" "tor-$version.tar.gz.sha256sum.asc"; do
    curl -fsS --proto =https --tlsv1.2 -o "$name" "$base/$name" || {
        echo "error: download failed: $base/$name" >&2
        exit 1
    }
done

gpgv --keyring "$keyring" \
    "tor-$version.tar.gz.sha256sum.asc" \
    "tor-$version.tar.gz.sha256sum" || {
    echo "error: signature verification failed for tor-$version" >&2
    exit 1
}

sha256sum -c "tor-$version.tar.gz.sha256sum" || {
    echo "error: checksum verification failed for tor-$version" >&2
    exit 1
}

printf 'VERIFIED tor-%s\n' "$version"
