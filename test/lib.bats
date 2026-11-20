#!/usr/bin/env bats
load helpers

setup() { hoard_test_env; }

lib() { bash -c ". '$HOOKS/_lib.sh'; $1"; }

@test "config file values load; env wins" {
	mkdir -p "$XDG_CONFIG_HOME/hoard"
	printf 'HOARD_VAULT_ROOT="/from/file/"\nHOARD_DEFAULT_SCOPE=work\n# comment\nOTHER=x\n' > "$XDG_CONFIG_HOME/hoard/config.env"
	unset HOARD_VAULT_ROOT
	run lib 'hoard_load_config; echo "$HOARD_VAULT_ROOT|$HOARD_DEFAULT_SCOPE|$HOARD_SCOPE|${OTHER:-unset}"'
	[ "$output" = "/from/file|work|work|unset" ]
	HOARD_VAULT_ROOT=/from/env run lib 'hoard_load_config; echo "$HOARD_VAULT_ROOT"'
	[ "$output" = "/from/env" ]
}

@test "AGENT_SCOPE sets scope; invalid falls back" {
	AGENT_SCOPE=acme run lib 'hoard_load_config; echo "$HOARD_SCOPE"'
	[ "$output" = "acme" ]
	AGENT_SCOPE='../x' run lib 'hoard_load_config; echo "$HOARD_SCOPE"'
	[ "$output" = "personal" ]
}

@test "tilde root expands" {
	HOARD_VAULT_ROOT='~/v' run lib 'hoard_load_config; echo "$HOARD_VAULT_ROOT"'
	[ "$output" = "$HOME/v" ]
}

@test "hoard_disabled" {
	HOARD_DISABLE=recall,nudge run lib 'hoard_load_config; hoard_disabled nudge && echo y; hoard_disabled mirror || echo n'
	[ "$output" = $'y\nn' ]
}

@test "prompt and paths from every runtime" {
	for rt in "${RUNTIMES[@]}"; do
		run bash -c ". '$HOOKS/_lib.sh'; hoard_read_payload; hoard_prompt" < <(prompt_payload "$rt" "hello there")
		[ "$output" = "hello there" ] || { echo "$rt prompt: $output"; false; }
		run bash -c ". '$HOOKS/_lib.sh'; hoard_read_payload; hoard_paths" < <(write_payload "$rt" "rel/a.md" /work)
		[ "$output" = "/work/rel/a.md" ] || { echo "$rt paths: $output"; false; }
	done
}

@test "context output per runtime" {
	run bash -c ". '$HOOKS/_lib.sh'; hoard_read_payload; hoard_emit_context hi" < <(prompt_payload claude x)
	[ "$(jq -r '.hookSpecificOutput.hookEventName + ":" + .hookSpecificOutput.additionalContext' <<<"$output")" = "UserPromptSubmit:hi" ]
	run bash -c ". '$HOOKS/_lib.sh'; hoard_read_payload; hoard_emit_context hi" < <(prompt_payload hermes x)
	[ "$(jq -r .context <<<"$output")" = "hi" ]
	run bash -c ". '$HOOKS/_lib.sh'; hoard_read_payload; hoard_emit_context hi" < <(write_payload hermes /tmp/a.md)
	[ -z "$output" ]
}

@test "session id is sanitized" {
	run bash -c ". '$HOOKS/_lib.sh'; hoard_read_payload; hoard_session" <<<'{"session_id":"../a b"}'
	[ "$output" = ".._a_b" ]
	run bash -c ". '$HOOKS/_lib.sh'; hoard_read_payload; hoard_session" <<<'{}'
	[ "$output" = "nosession" ]
}
