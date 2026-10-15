#!/usr/bin/env bash
# Build static, versioned release archives using only Go and system archivers.
set -euo pipefail
version="${1:?usage: bash scripts/package.sh vMAJOR.MINOR.PATCH}"
if [[ ! "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "expected stable version vMAJOR.MINOR.PATCH" >&2
  exit 1
fi
# Refuse to overwrite previous local build results.
if [[ -e dist ]]; then
  echo "dist already exists; move it aside before packaging" >&2
  exit 1
fi
mkdir dist
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64; do
  os="${target%/*}"
  arch="${target#*/}"
  name="banner-fingerprint_${version}_${os}_${arch}"
  package="$work/$name"
  mkdir -p "$package"
  suffix=""
  if [[ "$os" == windows ]]; then suffix=".exe"; fi
  for cmd in client server; do
    CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" go build \
      -trimpath -ldflags='-s -w' -o "$package/banner-$cmd$suffix" "./cmd/$cmd"
  done
  cp README.md "$package/"
  cp -R examples docs "$package/"
  if [[ "$os" == windows ]]; then
    destination="$PWD/dist/$name.zip"
    (cd "$work" && zip -qr "$destination" "$name")
  else
    tar -czf "dist/$name.tar.gz" -C "$work" "$name"
  fi
done
(cd dist && sha256sum ./*.tar.gz ./*.zip > SHA256SUMS)
