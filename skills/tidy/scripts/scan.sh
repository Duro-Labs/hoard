#!/usr/bin/env bash
set -euo pipefail
. "${HOARD_HOME:-${XDG_DATA_HOME:-$HOME/.local/share}/hoard}/hooks/_lib.sh"
hoard_load_config
[ -n "$HOARD_VAULT_ROOT" ] || { echo 'FAIL no-vault-root' >&2; exit 2; }
ROOT=${1:-$HOARD_VAULT_ROOT/Agents/$HOARD_SCOPE}
CDPATH='' cd -- "$ROOT"

day_of() {
  date -d "@$1" +%F 2>/dev/null || date -r "$1" +%F
}

emit_file() {
  local day
  day=$(day_of "$(hoard_mtime "$1")")
  printf 'file\t%s\t1\t%s\n' "$1" "$day"
}

emit_dir() {
  local d=$1 f count=0 newest=0 m day
  while IFS= read -r f; do
    count=$((count + 1))
    m=$(hoard_mtime "$f")
    [ "$m" -gt "$newest" ] && newest=$m
  done < <(find "./$d" -type f)
  [ "$count" -gt 0 ] || newest=$(hoard_mtime "$d")
  day=$(day_of "$newest")
  printf 'dir\t%s\t%s\t%s\n' "$d" "$count" "$day"
}

while IFS= read -r f; do emit_file "$f"; done < <(find . -mindepth 1 -maxdepth 1 -type f | sed 's|^\./||' | LC_ALL=C sort)
while IFS= read -r d; do emit_dir "$d"; done < <(find . -mindepth 1 -maxdepth 1 -type d ! -name _Archive | sed 's|^\./||' | LC_ALL=C sort)
