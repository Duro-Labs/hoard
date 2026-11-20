#!/usr/bin/env bash
set -u
[ "${BASH_VERSINFO[0]}" -ge 4 ] || exit 0
command -v jq >/dev/null 2>&1 || exit 0
. "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/_lib.sh"
hoard_load_config
[ -n "$HOARD_VAULT_ROOT" ] && [ -d "$HOARD_VAULT_ROOT" ] || exit 0
hoard_read_payload
[ -n "$HOARD_PAYLOAD" ] || exit 0

agents=$HOARD_VAULT_ROOT/Agents
inbox=$agents/$HOARD_SCOPE/_Inbox
session=$(hoard_session)
today=$(date +%F)
map_dir=$HOARD_STATE/mirror
map=$map_dir/$session.tsv
notes=()

under_tmp() {
	local r
	local -a roots
	IFS=: read -r -a roots <<<"${HOARD_TEMP_ROOTS:-/tmp:${TMPDIR:-/tmp}}"
	for r in "${roots[@]}"; do
		r=${r%/}
		[ -n "$r" ] && [[ $1 == "$r"/* ]] && return 0
	done
	return 1
}

excluded() {
	local g
	local -a globs
	IFS=: read -r -a globs <<<"$HOARD_MIRROR_EXCLUDE"
	for g in "${globs[@]}"; do
		[ -n "$g" ] || continue
		# shellcheck disable=SC2053
		[[ $1 == $g ]] && return 0
	done
	return 1
}

free_name() {
	local p=$1 stem ext n=2
	[ -e "$p" ] || { printf '%s' "$p"; return; }
	stem=${p%.*} ext=${p##*.}
	while [ -e "$stem-$n.$ext" ]; do n=$((n + 1)); done
	printf '%s' "$stem-$n.$ext"
}

mirror() {
	local src=$1 dest scratch=0
	[ -f "$src" ] || return 0
	if [[ $src =~ /claude-[0-9]+/[^/]+/([^/]+)/scratchpad/(.+)$ ]]; then
		scratch=1
	fi
	dest=$(awk -F'\t' -v s="$src" '$1 == s { d = $2 } END { print d }' "$map" 2>/dev/null)
	if [ -z "$dest" ]; then
		if [ "$scratch" -eq 1 ]; then
			dest=$(free_name "$inbox/scratch/$today-${BASH_REMATCH[1]:0:8}/${BASH_REMATCH[2]}")
		else
			dest=$(free_name "$inbox/$today-$(basename "$src")")
		fi
		mkdir -p "$map_dir" 2>/dev/null && printf '%s\t%s\n' "$src" "$dest" >>"$map"
		[ "$scratch" -eq 0 ] && notes+=("hoard: mirrored $src to $dest. Move it into Agents/$HOARD_SCOPE/<subject>/ once you know its topic.")
	fi
	mkdir -p "$(dirname "$dest")" 2>/dev/null && cp -f "$src" "$dest" 2>/dev/null
	return 0
}

while IFS= read -r p; do
	lp=$(printf '%s' "$p" | tr '[:upper:]' '[:lower:]')
	case $lp in *.md | *.html) ;; *) continue ;; esac
	if under_tmp "$p" && ! hoard_disabled mirror && ! excluded "$p"; then
		mirror "$p"
		continue
	fi
	# nudges
done < <(hoard_paths)

find "$map_dir" -type f -mtime +7 -exec rm -f {} + 2>/dev/null
[ "${#notes[@]}" -eq 0 ] && exit 0
hoard_emit_context "$(printf '%s\n' "${notes[@]}")"
exit 0
