#!/usr/bin/env sh

set -e

BRANCH=$(git rev-parse --abbrev-ref HEAD)
BRANCH=$(echo "$BRANCH" | tr '/' '_')

BINARY=dtools
OUTPUT=/opt/bin
COMPLETION=false
BINARY_OVERRIDE=false

# Parse arguments
while [ "$#" -gt 0 ]; do
    case "$1" in
        -b|--binary)
            shift
            BINARY="$1"
            BINARY_OVERRIDE=true
            ;;
        *)
            OUTPUT="$1"
            ;;
    esac
    shift
done

if [ "$BINARY_OVERRIDE" = true ]; then
    FULLNAME="$BINARY"
else
    if [ "$BRANCH" = "master" ] || [ "$BRANCH" = "main" ] || [ "$BRANCH" = "develop" ]; then
        FULLNAME="$BINARY"
    else
        FULLNAME="$BINARY-$BRANCH"
    fi
fi

echo "Building ${OUTPUT}/${FULLNAME}"
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -buildid=" -o "${OUTPUT}/${FULLNAME}" .

# Enable tab completion
