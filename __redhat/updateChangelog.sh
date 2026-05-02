#!/usr/bin/env bash

set -euo pipefail

SPEC="$(dirname "$0")/dtools2.spec"
DATE=$(date +"%a %b %d %Y")
VERSION=$(rpmspec -q --qf '%{version}\n' "$SPEC" | head -1)
REL=$(rpmspec -q --qf '%{release}\n' "$SPEC" | head -1)
MAINTAINER="Binary package builder <builder@famillegratton.net>"

# Collect git log since last tag, or all commits if no tags yet
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || git rev-list --max-parents=0 HEAD)
ENTRIES=$(git log --oneline "${LAST_TAG}..HEAD" | sed 's/^[a-f0-9]\+ /- /')

if [[ -z "$ENTRIES" ]]; then
  echo "No new commits since last tag, nothing to append."
  exit 0
fi

NEW_ENTRY="* ${DATE} ${MAINTAINER} ${VERSION}-${REL}\n${ENTRIES}"

# Prepend new entry right after the %changelog line
sed -i "s/^%changelog$/%changelog\n${NEW_ENTRY}\n/" "$SPEC"

echo "Changelog updated for ${VERSION}-${REL}"