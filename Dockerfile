# syntax=docker/dockerfile:1

ARG TOR_VERSION

# Tor is compiled from the signed upstream tarball, never from distro packages.
# The base is pinned by digest so a rebuild cannot silently pick up a new image.
FROM debian:trixie-slim@sha256:a72b4433de1cf7d6bae0c4cee74d0547f58601a643b3375f070c51600d63b6b2 AS tor-build
ARG TOR_VERSION

RUN apt-get update \
 && apt-get install -y --no-install-recommends \
      build-essential \
      pkg-config \
      ca-certificates \
      curl \
      gpg \
      gpgv \
      libevent-dev \
      libssl-dev \
      zlib1g-dev \
      libzstd-dev \
      liblzma-dev \
      libseccomp-dev \
      libcap-dev \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY keys/tor-release.asc scripts/verify-tor-source.sh scripts/collect-libs.sh /src/

RUN gpg --dearmor < /src/tor-release.asc > /src/tor-release.gpg \
 && chmod +x /src/verify-tor-source.sh \
 && /src/verify-tor-source.sh "$TOR_VERSION" /src/tor-release.gpg /src/work

# Fail closed: the configure summary must show seccomp and zstd support.
RUN tar -xzf "/src/work/tor-$TOR_VERSION.tar.gz" -C /src \
 && cd "/src/tor-$TOR_VERSION" \
 && ./configure \
      --prefix=/usr \
      --sysconfdir=/etc \
      --localstatedir=/var \
      --disable-asciidoc \
      --disable-manpage \
      --disable-html-manual \
      --disable-unittests \
      2>&1 | tee /src/configure.out \
 && grep -Eq 'libseccomp \(--disable-seccomp\):[[:space:]]+yes' /src/configure.out \
 && grep -Eq 'libzstd \(--enable-zstd\):[[:space:]]+yes' /src/configure.out \
 && make -j"$(nproc)" \
 && make install DESTDIR=/out \
 && chmod +x /src/collect-libs.sh \
 && /src/collect-libs.sh /out/usr/bin/tor /rootfs \
 && mkdir -p /rootfs/usr/bin \
 && cp -a /out/usr/bin/tor /rootfs/usr/bin/tor \
 && mkdir -p /rootfs/usr/share/tor \
 && cp -a /out/usr/share/tor/geoip /out/usr/share/tor/geoip6 /rootfs/usr/share/tor/ \
 && mkdir -p /var/lib/tor \
 && chown 65532:65532 /var/lib/tor \
 && chmod 0700 /var/lib/tor

# torinit is a static Go binary. No cgo, no external modules.
FROM golang:1.27.4-trixie@sha256:b54b2639de8c1bfe538ec5ffc45f29b1932235ac5f6d63c059853646677112d9 AS torinit-build
ARG TORINIT_VERSION=dev

WORKDIR /src
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/

RUN CGO_ENABLED=0 go build -trimpath \
      -ldflags "-s -w -X main.version=${TORINIT_VERSION}" \
      -o /torinit ./cmd/torinit

FROM gcr.io/distroless/cc-debian13:nonroot@sha256:cb87acc574abe719b6e5c547384defa2f7b1f24af9605d3f618cf852495d8be2 AS runtime
ARG TOR_VERSION

LABEL org.opencontainers.image.source="https://github.com/Unshakled-Inc/tor-relay" \
      org.opencontainers.image.licenses="BSD-3-Clause" \
      org.opencontainers.image.vendor="Unshakled Inc." \
      org.opencontainers.image.version="${TOR_VERSION}" \
      org.opencontainers.image.title="tor-relay"

COPY --from=tor-build /rootfs/ /
COPY --from=torinit-build /torinit /usr/local/bin/torinit
COPY --from=tor-build --chown=65532:65532 --chmod=0700 /var/lib/tor /var/lib/tor

USER 65532:65532
VOLUME /var/lib/tor
EXPOSE 9001

ENTRYPOINT ["/usr/local/bin/torinit"]
CMD ["run"]

HEALTHCHECK --interval=60s --timeout=5s --start-period=120s CMD ["/usr/local/bin/torinit","healthcheck"]

