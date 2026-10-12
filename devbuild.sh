#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_PATH="${CANNON_CONFIG:-$ROOT_DIR/sites.json}"

if [[ ! -f "$CONFIG_PATH" ]]; then
  echo "sites.json not found: $CONFIG_PATH" >&2
  exit 1
fi

EXTENSIONS_DIR="$(
  CONFIG_PATH="$CONFIG_PATH" python3 - <<'PY'
import json
import os

config_path = os.environ["CONFIG_PATH"]
with open(config_path, "r", encoding="utf-8") as fh:
    cfg = json.load(fh)

ext_dir = (cfg.get("extensions") or {}).get("dir")
if not ext_dir:
    ext_dir = os.path.join(cfg.get("data_root") or "./data", "extensions")

if not os.path.isabs(ext_dir):
    ext_dir = os.path.abspath(os.path.join(os.path.dirname(config_path), ext_dir))

print(ext_dir)
PY
)"

mkdir -p "$EXTENSIONS_DIR"

shopt -s nullglob
extensions=("$ROOT_DIR"/../cannon-ext-*)

if (( ${#extensions[@]} == 0 )); then
  echo "No sibling cannon-ext-* directories found."
else
  for ext_dir in "${extensions[@]}"; do
    if [[ ! -d "$ext_dir" ]]; then
      continue
    fi

    name="$(basename "$ext_dir")"
    if [[ ! -f "$ext_dir/go.mod" ]]; then
      echo "Skipping $name: no go.mod found."
      continue
    fi

    echo "Building $name -> $EXTENSIONS_DIR/$name"
    (
      cd "$ext_dir"
      go build -o "$EXTENSIONS_DIR/$name" .
    )
  done
fi

echo "Starting Cannon..."
cd "$ROOT_DIR"
exec go run cmd/cannon/main.go "$@"
