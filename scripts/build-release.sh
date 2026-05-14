#!/usr/bin/env sh
set -eu

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || printf dev)}"
DIST="${DIST:-dist}"
PKG="./cmd/git-preflight"

rm -rf "$DIST"
mkdir -p "$DIST"

build_one() {
  goos="$1"
  goarch="$2"
  name="git-preflight-${VERSION}-${goos}-${goarch}"
  work="$DIST/$name"
  mkdir -p "$work"

  bin="git-preflight"
  if [ "$goos" = "windows" ]; then
    bin="git-preflight.exe"
  fi

  printf 'building %s/%s\n' "$goos" "$goarch"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags "-s -w -X main.version=$VERSION" \
    -o "$work/$bin" \
    "$PKG"

  cp README.md "$work/"
  mkdir -p "$work/man/man1"
  cp docs/man/git-preflight.1 "$work/man/man1/"

  if [ "$goos" = "windows" ]; then
    if command -v zip >/dev/null 2>&1; then
      (cd "$DIST" && zip -qr "$name.zip" "$name")
    elif command -v python3 >/dev/null 2>&1; then
      (cd "$DIST" && python3 -c 'import os, sys, zipfile
name = sys.argv[1]
with zipfile.ZipFile(name + ".zip", "w", zipfile.ZIP_DEFLATED) as z:
    for root, _, files in os.walk(name):
        for file in files:
            path = os.path.join(root, file)
            z.write(path, path)
' "$name")
    else
      printf 'error: zip or python3 is required to package Windows artifacts\n' >&2
      exit 1
    fi
  else
    (cd "$DIST" && tar -czf "$name.tar.gz" "$name")
  fi
  rm -rf "$work"
}

build_one linux amd64
build_one linux arm64
build_one darwin amd64
build_one darwin arm64
build_one windows amd64

(cd "$DIST" && sha256sum *.tar.gz *.zip > checksums.txt)
