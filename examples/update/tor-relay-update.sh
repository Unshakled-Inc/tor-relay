#!/bin/sh
# Pull the newest image, verify its cosign signature, and recreate the relay
# with the verified digest. Run this from the directory that holds
# compose.yml. A failed verification exits non-zero and leaves the running
# relay untouched.
set -eu

IMAGE="docker.io/unshakled/tor-relay"
TAG="${TOR_RELAY_TAG:-0.4.9}"
IDENTITY="https://github.com/Unshakled-Inc/tor-relay/.github/workflows/build.yml@refs/heads/main"
ISSUER="https://token.actions.githubusercontent.com"

docker pull -q "$IMAGE:$TAG" >/dev/null

DIGEST="$(docker image inspect "$IMAGE:$TAG" --format '{{index .RepoDigests 0}}')"
if [ -z "$DIGEST" ]; then
    echo "could not resolve the digest of $IMAGE:$TAG" >&2
    exit 1
fi

cosign verify "$DIGEST" \
    --certificate-identity "$IDENTITY" \
    --certificate-oidc-issuer "$ISSUER" >/dev/null

TOR_RELAY_IMAGE="$DIGEST" docker compose up -d
