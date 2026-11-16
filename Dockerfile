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
COPY keys/tor-release.asc scripts/verify-tor-source.sh /src/

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
 && make install DESTDIR=/out
