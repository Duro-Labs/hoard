#!/usr/bin/env bats
load helpers

setup() { hoard_test_env; today=$(date +%F); inbox="$HOARD_VAULT_ROOT/Agents/personal/_Inbox"; }

hook() { "$HOOKS/after-write.sh" < <(write_payload "$@"); }

@test "loose tmp note is mirrored to _Inbox and agent is told (claude, codex)" {
	for rt in claude codex; do
		rm -rf "$inbox" "$XDG_STATE_HOME"
		echo "v1" > "$TMPDIR/report.md"
		run hook "$rt" "$TMPDIR/report.md"
		[ "$status" -eq 0 ]
		[ "$(cat "$inbox/$today-report.md")" = "v1" ]
		[[ "$(jq -r .hookSpecificOutput.additionalContext <<<"$output")" == *"_Inbox/$today-report.md"* ]]
	done
}

@test "hermes mirrors but prints nothing" {
	echo "v1" > "$TMPDIR/report.md"
	run hook hermes "$TMPDIR/report.md"
	[ "$status" -eq 0 ] && [ -z "$output" ]
	[ -f "$inbox/$today-report.md" ]
}

@test "re-edit overwrites the same copy, no second note" {
	echo "v1" > "$TMPDIR/report.html"
	hook claude "$TMPDIR/report.html" >/dev/null
	echo "v2" > "$TMPDIR/report.html"
	run hook claude "$TMPDIR/report.html"
	[ -z "$output" ]
	[ "$(cat "$inbox/$today-report.html")" = "v2" ]
	[ "$(ls "$inbox" | wc -l)" -eq 1 ]
}

@test "collision with a foreign file gets a suffix" {
	mkdir -p "$inbox"; echo "theirs" > "$inbox/$today-report.md"
	echo "mine" > "$TMPDIR/report.md"
	hook claude "$TMPDIR/report.md" >/dev/null
	[ "$(cat "$inbox/$today-report.md")" = "theirs" ]
	[ "$(cat "$inbox/$today-report-2.md")" = "mine" ]
}

@test "claude scratchpad goes to _Inbox/scratch/<date>-<session8>/ silently" {
	sp="$TMPDIR/claude-1000/-proj/0123456789abcdef/scratchpad/sub"
	mkdir -p "$sp"; echo s > "$sp/notes.md"
	run hook claude "$sp/notes.md"
	[ -z "$output" ]
	[ -f "$inbox/scratch/$today-01234567/sub/notes.md" ]
}

@test "non-note, non-tmp, excluded and disabled paths are ignored" {
	echo x > "$TMPDIR/a.txt"; hook claude "$TMPDIR/a.txt"
	mkdir -p "$BATS_TEST_TMPDIR/elsewhere"; echo x > "$BATS_TEST_TMPDIR/elsewhere/a.md"; hook claude "$BATS_TEST_TMPDIR/elsewhere/a.md"
	mkdir -p "$TMPDIR/skip"; echo x > "$TMPDIR/skip/a.md"
	HOARD_MIRROR_EXCLUDE="$TMPDIR/skip/*" hook claude "$TMPDIR/skip/a.md"
	echo x > "$TMPDIR/b.md"; HOARD_DISABLE=mirror hook claude "$TMPDIR/b.md"
	[ ! -d "$inbox" ]
}

@test "fails open: no vault root, garbage payload" {
	echo x > "$TMPDIR/a.md"
	HOARD_VAULT_ROOT="$BATS_TEST_TMPDIR/missing" run hook claude "$TMPDIR/a.md"
	[ "$status" -eq 0 ] && [ -z "$output" ]
	run "$HOOKS/after-write.sh" <<<'not json'
	[ "$status" -eq 0 ] && [ -z "$output" ]
}

@test "scope follows AGENT_SCOPE" {
	echo x > "$TMPDIR/a.md"
	AGENT_SCOPE=acme hook claude "$TMPDIR/a.md" >/dev/null
	[ -f "$HOARD_VAULT_ROOT/Agents/acme/_Inbox/$today-a.md" ]
}
