# shellcheck shell=bash

hoard_load_config() {
	local f="${XDG_CONFIG_HOME:-$HOME/.config}/hoard/config.env" line k v
	if [ -r "$f" ]; then
		while IFS= read -r line || [ -n "$line" ]; do
			case $line in '' | '#'*) continue ;; esac
			k=${line%%=*} v=${line#*=}
			case $k in HOARD_VAULT_ROOT | HOARD_DEFAULT_SCOPE | HOARD_MIRROR_EXCLUDE | HOARD_DISABLE) ;; *) continue ;; esac
			v=${v#\"} v=${v%\"}
			[ -n "${!k+x}" ] || printf -v "$k" '%s' "$v"
		done <"$f"
	fi
	HOARD_VAULT_ROOT=${HOARD_VAULT_ROOT:-}
	case $HOARD_VAULT_ROOT in \~) HOARD_VAULT_ROOT=$HOME ;; \~/*) HOARD_VAULT_ROOT=$HOME/${HOARD_VAULT_ROOT#\~/} ;; esac
	while [ "${#HOARD_VAULT_ROOT}" -gt 1 ] && [ "${HOARD_VAULT_ROOT%/}" != "$HOARD_VAULT_ROOT" ]; do HOARD_VAULT_ROOT=${HOARD_VAULT_ROOT%/}; done
	HOARD_DEFAULT_SCOPE=${HOARD_DEFAULT_SCOPE:-personal}
	[[ $HOARD_DEFAULT_SCOPE =~ ^[a-z0-9][a-z0-9-]*$ ]] || HOARD_DEFAULT_SCOPE=personal
	HOARD_MIRROR_EXCLUDE=${HOARD_MIRROR_EXCLUDE:-}
	HOARD_DISABLE=${HOARD_DISABLE:-}
	HOARD_SCOPE=${AGENT_SCOPE:-$HOARD_DEFAULT_SCOPE}
	[[ $HOARD_SCOPE =~ ^[a-z0-9][a-z0-9-]*$ ]] || HOARD_SCOPE=$HOARD_DEFAULT_SCOPE
	HOARD_STATE=${XDG_STATE_HOME:-$HOME/.local/state}/hoard
	export HOARD_STATE
}

hoard_disabled() { [[ ",$HOARD_DISABLE," == *",$1,"* ]]; }

hoard_read_payload() { HOARD_PAYLOAD=$(tr -d '\0' 2>/dev/null) || HOARD_PAYLOAD=; }

hoard_json() { printf '%s' "$HOARD_PAYLOAD" | jq -r "$1" 2>/dev/null || true; }

hoard_event() { hoard_json '.hook_event_name // empty | strings'; }

hoard_session() {
	local s
	s=$(hoard_json '.session_id // empty | strings')
	s=${s//[^A-Za-z0-9._-]/_}
	case $s in '' | . | ..) s=nosession ;; esac
	printf '%s' "$s"
}

hoard_cwd() {
	local d
	d=$(hoard_json '.tool_input.workdir // .cwd // empty | strings')
	printf '%s' "${d:-$PWD}"
}

hoard_prompt() { hoard_json '.prompt // .extra.user_message // .user_message // empty | strings'; }

hoard_paths() {
	local cwd p
	cwd=$(hoard_cwd)
	hoard_json '
		(.tool_input.file_path // .tool_input.path // empty | strings),
		((if .tool_name == "apply_patch" then .tool_input.command
		  elif .tool_input.mode == "patch" then .tool_input.patch
		  else empty end) // "" | strings | split("\n")[]
		 | capture("^\\*\\*\\* (?:Add File|Update File|Move to): (?<p>.+)$")? | .p)' |
		while IFS= read -r p; do
			[ -z "$p" ] && continue
			case $p in
			"~") p=$HOME ;;
			\~/*) p=$HOME/${p#\~/} ;;
			/*) ;;
			*) p=$cwd/$p ;;
			esac
			printf '%s\n' "$p"
		done
}

hoard_emit_context() {
	local ev
	ev=$(hoard_event)
	case $ev in
	pre_llm_call) jq -cn --arg c "$1" '{context:$c}' ;;
	post_tool_call | '') ;;
	*) jq -cn --arg e "$ev" --arg c "$1" '{hookSpecificOutput:{hookEventName:$e,additionalContext:$c}}' ;;
	esac
}

hoard_mtime() { stat -c %Y -- "$1" 2>/dev/null || stat -f %m -- "$1" 2>/dev/null || echo 0; }
