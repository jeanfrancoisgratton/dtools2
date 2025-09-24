#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./check-cycles.sh [MODULE_PREFIX]
#
# If MODULE_PREFIX is omitted, we default to the current module path
# (same as the first 'module ...' line in go.mod).

mod_prefix="${1:-}"

if [[ -z "${mod_prefix}" ]]; then
  # Try to get the module path from Go (works anywhere inside the module)
  if ! mod_prefix="$(go list -m -f '{{.Path}}' 2>/dev/null)"; then
    # Fallback: read go.mod directly (requires running at module root)
    if [[ -f go.mod ]]; then
      mod_prefix="$(awk '/^module[[:space:]]+/ {print $2; exit}' go.mod)"
    fi
  fi
fi

if [[ -z "${mod_prefix}" ]]; then
  echo "Unable to determine module path. Pass it explicitly, e.g.:"
  echo "  $0 github.com/you/yourmod"
  exit 1
fi

# Build edges (package -> imported package within same module) and feed to tsort.
# Note: tsort expects whitespace-separated pairs, not '->'.
go list -json ./... \
| jq --arg mod "$mod_prefix" -r '
    .ImportPath as $p
    | ( (.Imports // [])
        + (.TestImports // [])
        + (.XTestImports // []) )       # include test-only deps too
    | map(select(startswith($mod + "/")))
    | .[]
    | "\($p)\t\(.)"
' | tsort
