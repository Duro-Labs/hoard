#!/usr/bin/env bats
load helpers

setup() { hoard_test_env; A="$HOARD_VAULT_ROOT/Agents"; }

ctx() { "$HOOKS/after-write.sh" < <(write_payload claude "$1") | jq -r '.hookSpecificOutput.additionalContext // empty'; }

@test "draft-shaped name at scope root is nudged" {
	for n in pr-123.md issue-body.md draft-x.html fix-reply.md 245-closing-comment.md; do
		out=$(ctx "$A/personal/$n")
		[[ "$out" == *"Drafts/<slug>/"* ]] || { echo "no nudge for $n"; false; }
	done
}

@test "durable names and subject-folder files are not nudged" {
	for p in "$A/personal/report-api.md" "$A/personal/api-245/pr-123.md" "$A/personal/api-245/Drafts/x/reply.md"; do
		[ -z "$(ctx "$p")" ] || { echo "nudged $p"; false; }
	done
}

@test "write into another scope is nudged" {
	out=$(ctx "$A/acme/billing/report.md")
	[[ "$out" == *"scope 'acme'"* && "$out" == *"active scope is 'personal'"* ]]
}

@test "nudge can be disabled; write is never blocked" {
	run bash -c "HOARD_DISABLE=nudge '$HOOKS/after-write.sh'" < <(write_payload claude "$A/acme/x.md")
	[ "$status" -eq 0 ] && [ -z "$output" ]
	run "$HOOKS/after-write.sh" < <(write_payload codex "$A/acme/x.md")
	[ "$status" -eq 0 ]
}
