#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if [[ $# -lt 1 ]]; then
  echo "Usage: $0 \"commit message\"" >&2
  exit 1
fi

commit_message="$1"

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

    echo "Adding/committing extension $name"
    (
      cd "$ext_dir"
      git add .
      if git diff --cached --quiet; then
        echo "No staged changes for $name; skipping."
        exit 0
      fi
      git commit -m "$commit_message"
      git push origin main
    )
  done
fi
