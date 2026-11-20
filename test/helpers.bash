HOOKS=$(cd "$BATS_TEST_DIRNAME/../hooks" && pwd)
RUNTIMES=(claude codex hermes)

hoard_test_env() {
	export HOME="$BATS_TEST_TMPDIR/home"
	export XDG_CONFIG_HOME="$HOME/.config" XDG_STATE_HOME="$HOME/.local/state" XDG_DATA_HOME="$HOME/.local/share"
	export HOARD_VAULT_ROOT="$BATS_TEST_TMPDIR/vault"
	export TMPDIR="$BATS_TEST_TMPDIR/tmp"
	export HOARD_TEMP_ROOTS="$TMPDIR"
	unset AGENT_SCOPE HOARD_DEFAULT_SCOPE HOARD_MIRROR_EXCLUDE HOARD_DISABLE
	mkdir -p "$HOME" "$HOARD_VAULT_ROOT/Agents/personal" "$TMPDIR"
}

# prompt_payload <runtime> <prompt> [cwd]
prompt_payload() {
	local rt=$1 p=$2 d=${3:-$BATS_TEST_TMPDIR}
	case $rt in
	claude) jq -nc --arg p "$p" --arg d "$d" '{session_id:"s1",transcript_path:"/x",cwd:$d,permission_mode:"default",hook_event_name:"UserPromptSubmit",prompt:$p}' ;;
	codex) jq -nc --arg p "$p" --arg d "$d" '{session_id:"s1",turn_id:"t1",transcript_path:null,cwd:$d,hook_event_name:"UserPromptSubmit",model:"gpt-5.5",permission_mode:"default",prompt:$p}' ;;
	hermes) jq -nc --arg p "$p" --arg d "$d" '{hook_event_name:"pre_llm_call",session_id:"s1",cwd:$d,extra:{user_message:$p,is_first_turn:true,model:"m",platform:"cli"}}' ;;
	esac
}

# write_payload <runtime> <path> [cwd]
write_payload() {
	local rt=$1 f=$2 d=${3:-$BATS_TEST_TMPDIR}
	case $rt in
	claude) jq -nc --arg f "$f" --arg d "$d" '{session_id:"s1",transcript_path:"/x",cwd:$d,hook_event_name:"PostToolUse",tool_name:"Write",tool_input:{file_path:$f,content:"x"},tool_response:{success:true},tool_use_id:"toolu_1"}' ;;
	codex) jq -nc --arg f "$f" --arg d "$d" '{session_id:"s1",turn_id:"t1",transcript_path:null,cwd:$d,hook_event_name:"PostToolUse",model:"gpt-5.5",tool_name:"apply_patch",tool_input:{command:("*** Begin Patch\n*** Add File: " + $f + "\n+x\n*** End Patch\n")},tool_response:"ok",tool_use_id:"call_1"}' ;;
	hermes) jq -nc --arg f "$f" --arg d "$d" '{hook_event_name:"post_tool_call",tool_name:"write_file",tool_input:{path:$f,content:"x"},session_id:"s1",cwd:$d,extra:{tool_call_id:"c1",result:"ok"}}' ;;
	esac
}
