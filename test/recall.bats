#!/usr/bin/env bats
load helpers

setup() {
	hoard_test_env
	V=$HOARD_VAULT_ROOT
	mkdir -p "$V/Agents/personal/billing-retry" "$V/Agents/acme/billing-secret" "$V/Agents/personal/_Archive/old-billing" \
		"$V/Notes/Private" "$V/Agents/personal/_Inbox/scratch/2026-01-01-abcd1234" "$V/Notes/Web"
	printf '# Billing retry report\nidempotency backoff webhook retries\n' > "$V/Agents/personal/billing-retry/report.md"
	printf '# Acme secret\nidempotency backoff webhook retries\n' > "$V/Agents/acme/billing-secret/x.md"
	printf '# Old\nidempotency backoff webhook retries\n' > "$V/Agents/personal/_Archive/old-billing/x.md"
	printf '# Private\nidempotency backoff webhook retries\n' > "$V/Notes/Private/p.md"
	printf '# Scratch\nidempotency backoff webhook retries\n' > "$V/Agents/personal/_Inbox/scratch/2026-01-01-abcd1234/s.md"
	printf '<html><head><title>Webhook page</title></head><body>idempotency backoff webhook retries</body></html>\n' > "$V/Notes/Web/page.html"
	printf 'Notes/Private/\n' > "$V/.ignore"
	PROMPT="fix the webhook idempotency backoff retries"
}

recall() { "$HOOKS/recall.sh" < <(prompt_payload "$1" "$2"); }
ctx_of() { jq -r '.hookSpecificOutput.additionalContext // .context // empty' <<<"$1"; }

@test "surfaces own-scope subject and html note, not other scope, archive, ignored or scratch" {
	out=$(ctx_of "$(recall claude "$PROMPT")")
	[[ "$out" == *"Agents/personal/billing-retry/"* ]]
	[[ "$out" == *"Webhook page"* ]]
	[[ "$out" != *"acme"* && "$out" != *"_Archive"* && "$out" != *"Private"* && "$out" != *"scratch"* ]]
}

@test "every runtime gets its context shape" {
	[ -n "$(recall codex "$PROMPT" | jq -r .hookSpecificOutput.additionalContext)" ]
	rm -rf "$XDG_STATE_HOME/hoard/recall"
	[ -n "$(recall hermes "$PROMPT" | jq -r .context)" ]
}

@test "client scope also sees the default scope but not other clients" {
	mkdir -p "$V/Agents/globex/x"; printf 'idempotency backoff webhook retries\n' > "$V/Agents/globex/x/y.md"
	out=$(AGENT_SCOPE=acme "$HOOKS/recall.sh" < <(prompt_payload claude "$PROMPT") | jq -r .hookSpecificOutput.additionalContext)
	[[ "$out" == *"billing-secret"* && "$out" == *"billing-retry"* && "$out" != *"globex"* ]]
}

@test "configured default scope replaces personal in cross-scope recall" {
	out=$(HOARD_DEFAULT_SCOPE=acme AGENT_SCOPE=globex "$HOOKS/recall.sh" < <(prompt_payload claude "$PROMPT") | jq -r .hookSpecificOutput.additionalContext)
	[[ "$out" == *"billing-secret"* && "$out" != *"billing-retry"* ]]
}

@test "same doc is not repeated in a session" {
	recall claude "$PROMPT" >/dev/null
	out=$(ctx_of "$(recall claude "$PROMPT")")
	[[ "$out" != *"billing-retry"* ]]
}

@test "silent when disabled, no root, short prompt, or no match" {
	[ -z "$(HOARD_DISABLE=recall "$HOOKS/recall.sh" < <(prompt_payload claude "$PROMPT"))" ]
	[ -z "$(HOARD_VAULT_ROOT=/nope "$HOOKS/recall.sh" < <(prompt_payload claude "$PROMPT"))" ]
	[ -z "$(recall claude "hi")" ]
	[ -z "$(recall claude "zebra quantum marmalade")" ]
}
