#!/usr/bin/env bats

setup() {
	REPO=$(cd "$BATS_TEST_DIRNAME/.." && pwd)
	work="$BATS_TEST_TMPDIR/repo"
	git init -q "$work"
	mkdir -p "$work/scripts"
	cp "$REPO/scripts/public-check.sh" "$work/scripts/"
	git -C "$work" add scripts
}

@test "clean tree passes" {
	echo "hello" > "$work/a.md"; git -C "$work" add a.md
	run "$work/scripts/public-check.sh"
	[ "$status" -eq 0 ]
}

@test "absolute home path fails" {
	printf 'see /home/%s/notes\n' alice > "$work/a.md"; git -C "$work" add a.md
	run "$work/scripts/public-check.sh"
	[ "$status" -eq 1 ]
	[[ "$output" == *"a.md"* ]]
}

@test "private pattern from env fails" {
	printf '%s%s internal\n' AC ME > "$work/a.md"; git -C "$work" add a.md
	HOARD_PRIVATE_PATTERNS='acme' run "$work/scripts/public-check.sh"
	[ "$status" -eq 1 ]
}
