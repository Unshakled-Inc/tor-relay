#!/bin/sh
# Print the highest stable tor version on dist.torproject.org, but only if
# its signature file is already published. Otherwise print the current
# VERSION so the watcher does not bump before the signature exists.
set -eu

repo_dir=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
previous=$(cat "$repo_dir/VERSION")

listing=$(curl -fsS --proto =https --tlsv1.2 https://dist.torproject.org/) || {
    echo "warning: could not fetch dist.torproject.org, keeping $previous" >&2
    printf '%s\n' "$previous"
    exit 0
}

latest=$(printf '%s\n' "$listing" \
    | grep -oE 'tor-[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+\.tar\.gz' \
    | sed -E 's/^tor-//; s/\.tar\.gz$//' \
    | sort -uV \
    | tail -1)

if [ -z "$latest" ]; then
    echo "warning: no stable tor version found, keeping $previous" >&2
    printf '%s\n' "$previous"
    exit 0
fi

# Never move backwards if the directory listing is stale.
newest=$(printf '%s\n%s\n' "$previous" "$latest" | sort -V | tail -1)
if [ "$newest" != "$latest" ]; then
    printf '%s\n' "$previous"
    exit 0
fi

code=$(curl -sS --proto =https --tlsv1.2 --head -o /dev/null -w '%{http_code}' \
    "https://dist.torproject.org/tor-$latest.tar.gz.sha256sum.asc") || code=000

if [ "$code" = "200" ]; then
    printf '%s\n' "$latest"
else
    echo "warning: signature for $latest is not available (HTTP $code), keeping $previous" >&2
    printf '%s\n' "$previous"
fi
