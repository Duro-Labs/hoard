#!/usr/bin/env bats
load helpers

setup() {
	hoard_test_env
	export HOARD_HOME=$(cd "$BATS_TEST_DIRNAME/.." && pwd)
	S="$HOARD_VAULT_ROOT/Agents/personal"
	T="$HOARD_HOME/skills/tidy/scripts"
	mkdir -p "$S/api-245" "$S/_Archive/old" "$S/_Inbox/scratch/2026-01-01-abcd1234"
	echo a > "$S/api-245/report.md"; echo b > "$S/loose.md"; echo c > "$S/_Archive/old/x.md"
	echo d > "$S/_Inbox/scratch/2026-01-01-abcd1234/s.md"
}

@test "scan inventories the active scope, skips _Archive, lists _Inbox" {
	run "$T/scan.sh"
	[ "$status" -eq 0 ]
	[[ "$output" == *$'file\tloose.md\t1\t'* ]]
	[[ "$output" == *$'dir\tapi-245\t1\t'* ]]
	[[ "$output" == *$'dir\t_Inbox\t1\t'* ]]
	[[ "$output" != *"_Archive"* ]]
}

@test "apply moves, refuses overwrite, logs" {
	printf 'loose.md\tapi-245/loose.md\napi-245/report.md\tapi-245/loose.md\n' > "$BATS_TEST_TMPDIR/moves.tsv"
	run "$T/apply.sh" "$BATS_TEST_TMPDIR/moves.tsv"
	[ "$status" -eq 1 ]
	[ -f "$S/api-245/loose.md" ] && [ -f "$S/api-245/report.md" ]
	grep -q $'\tloose.md\tapi-245/loose.md$' "$XDG_STATE_HOME/hoard/tidy/moves.log"
}

@test "apply rejects bad paths" {
	printf '../x\ty\n' > "$BATS_TEST_TMPDIR/moves.tsv"
	run "$T/apply.sh" "$BATS_TEST_TMPDIR/moves.tsv"
	[ "$status" -eq 1 ] && [[ "$output" == *"bad-path"* ]]
}
