#!/bin/sh
# Copy the shared libraries a binary needs into a staging rootfs.
#
# Arguments: <binary> <staging-rootfs>
#
# The staging rootfs is later copied into a distroless image that already
# ships the glibc core libraries, so those are skipped. Every other library
# is copied at its resolved path, and the SONAME symlink that ldd printed is
# recreated pointing at that copy.
set -eu

if [ "$#" -ne 2 ]; then
    echo "usage: $0 <binary> <staging-rootfs>" >&2
    exit 2
fi

binary=$1
staging=$2

if [ ! -x "$binary" ]; then
    echo "error: binary not executable: $binary" >&2
    exit 1
fi

mkdir -p "$staging"

tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT

ldd "$binary" > "$tmp" || {
    echo "error: ldd failed for $binary" >&2
    exit 1
}

while IFS= read -r line; do
    case "$line" in
        *"=>"*) ;;
        *) continue ;;
    esac

    lib=${line#*=> }
    lib=${lib%% (*}

    # ldd prints "not found" as a bare name with no path.
    case "$lib" in
        /*) ;;
        *)
            echo "error: library not found: $lib" >&2
            exit 1
            ;;
    esac

    base=${lib##*/}

    # Distroless cc-debian13 already ships these glibc core files.
    case "$base" in
        ld-linux*|libc.so*|libm.so*|libpthread*|libdl*|librt*|libgcc_s*|libstdc++*)
            continue
            ;;
    esac

    real=$(readlink -f "$lib")
    if [ ! -f "$real" ]; then
        echo "error: cannot resolve $lib" >&2
        exit 1
    fi

    dest="$staging$real"
    mkdir -p "${dest%/*}"
    cp -a "$real" "$dest"

    if [ "$lib" != "$real" ]; then
        link="$staging$lib"
        mkdir -p "${link%/*}"
        ln -sfn "$real" "$link"
    fi
done < "$tmp"
