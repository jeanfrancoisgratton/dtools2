#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")"

sudo rm -rf src pkg

PKGDEST=/data makepkg --force --clean --syncdeps --noconfirm

sudo rm -rf src pkg