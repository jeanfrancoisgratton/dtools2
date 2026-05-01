#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")"

rm -rf src pkg

PKGDEST=/data makepkg \
  --force \
  --clean \
  --syncdeps \
  --noconfirm

rm -rf src pkg