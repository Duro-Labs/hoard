#!/usr/bin/env bash
set -uo pipefail

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
  echo "usage: apply.sh moves.tsv [scope-root]" >&2
  exit 2
fi
PLAN="$1"
. "${HOARD_HOME:-${XDG_DATA_HOME:-$HOME/.local/share}/hoard}/hooks/_lib.sh"
hoard_load_config
[ -n "$HOARD_VAULT_ROOT" ] || { echo 'FAIL no-scope-root' >&2; exit 2; }
ROOT="${2:-$HOARD_VAULT_ROOT/Agents/$HOARD_SCOPE}"
LOG_DIR="$HOARD_STATE/tidy"

case "$PLAN" in /*) ;; *) PLAN="$PWD/$PLAN" ;; esac
case "$LOG_DIR" in /*) ;; *) LOG_DIR="$PWD/$LOG_DIR" ;; esac
LOG="$LOG_DIR/moves.log"

if [ ! -f "$PLAN" ] || [ ! -r "$PLAN" ]; then
  echo "FAIL plan-unreadable: $PLAN" >&2
  exit 2
fi

if ! mkdir -p -- "$LOG_DIR"; then
  echo "FAIL log-dir: $LOG_DIR" >&2
  exit 2
fi

if ! CDPATH='' cd -- "$ROOT"; then
  echo "FAIL bad-root: $ROOT" >&2
  exit 2
fi

bad_path() {
  case "$1" in
    '' | /*) return 0 ;;
    . | ..) return 0 ;;
    ../* | */../* | */..) return 0 ;;
  esac
  case "$1" in
    ./* | */./* | */.) return 0 ;;
    *//*) return 0 ;;
  esac
  case "$1" in *$'\r'*) return 0 ;; esac
  return 1
}

strip_slashes() {
  local p="$1"
  while [ "${#p}" -gt 1 ] && [ "${p%/}" != "$p" ]; do p="${p%/}"; done
  printf '%s' "$p"
}

fail=0
while IFS=$'\t' read -r src dest extra || [ -n "${src:-}" ]; do
  [ -z "$src" ] && continue
  case "$src" in '#'*) continue ;; esac
  if [ -z "${dest:-}" ] || [ -n "${extra:-}" ]; then
    echo "FAIL bad-row: $src" >&2; fail=1; continue
  fi
  src=$(strip_slashes "$src")
  dest=$(strip_slashes "$dest")
  if bad_path "$src" || bad_path "$dest"; then
    echo "FAIL bad-path: $src -> $dest" >&2; fail=1; continue
  fi
  if [ ! -e "$src" ] && [ ! -L "$src" ]; then
    echo "FAIL missing-src: $src" >&2; fail=1; continue
  fi
  if [ -e "$dest" ] || [ -L "$dest" ]; then
    echo "FAIL dest-exists: $src -> $dest" >&2; fail=1; continue
  fi
  case "$dest/" in
    "$src"/*) echo "FAIL dest-inside-src: $src -> $dest" >&2; fail=1; continue ;;
  esac
  destdir=$(dirname -- "$dest")
  if ! mkdir -p -- "$destdir"; then
    echo "FAIL mkdir: $destdir" >&2; fail=1; continue
  fi
  if ! mv -n -- "$src" "$dest"; then
    echo "FAIL mv: $src -> $dest" >&2; fail=1; continue
  fi
  if [ -e "$src" ] || [ -L "$src" ]; then
    echo "FAIL mv-noop: $src -> $dest" >&2; fail=1; continue
  fi
  echo "moved: $src -> $dest"
  if ! ts=$(date "+%Y-%m-%dT%H:%M:%S%z"); then
    echo "FAIL log-timestamp: $src -> $dest" >&2; fail=1; continue
  fi
  if ! printf '%s\t%s\t%s\n' "$ts" "$src" "$dest" >>"$LOG"; then
    echo "FAIL log-append: $src -> $dest" >&2; fail=1; continue
  fi
done <"$PLAN"
exit "$fail"
