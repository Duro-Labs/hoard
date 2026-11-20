# HOARD — design

**H**ooks · **O**bsidian · **A**rchive · **R**ecall · **D**rafts

HOARD gives coding agents (Claude Code, Codex, Hermes) a shared Obsidian vault as long-term memory: a fixed place to write notes, recall of related notes before work starts, mirroring of notes written to temporary directories, and a reviewed bulk-tidy flow. Nothing in it blocks an agent: hooks only add context or copy files.

## Pillars

- **Obsidian:** the vault is an Obsidian vault. Agent-authored notes are ordinary Markdown (or HTML) files with an `agent:` frontmatter field, readable and linkable in Obsidian.
- **Archive (layout and scopes):** agents write to `Agents/<scope>/<subject>/<deliverable>.md`. Subject grouping comes first; other categorization is a subfolder of the subject. Each scope (for example `personal` or a client slug) is its own folder; agents do not write into another scope's folder.
- **Recall:** before work starts, a hook surfaces related vault notes so the agent reads them first.
- **Drafts:** text written to be pasted elsewhere and then discarded lives in `<subject>/Drafts/<slug>/`, not at the scope root.
- **Hooks:** the mechanism for all of the above, plus mirroring files agents write to temporary directories into the vault.

Agents never depend on a vault path: they search (`rg`, honouring `<vault>/.ignore`) rather than hardcode, because notes move. Facts an agent must rely on belong in agent config, not the vault.

## Vault layout (stable contract)

```
<HOARD_VAULT_ROOT>/
  .ignore                       # gitignore syntax; paths agents skip when searching
  Agents/<scope>/
    <subject>/                  # kebab-case topic or repo-issue slug
      <deliverable>.md|.html
      Drafts/<slug>/
    _Inbox/                     # mirrored temp files, not yet filed
      <YYYY-MM-DD>-<name>
      scratch/<YYYY-MM-DD>-<session>/<path>
    _Archive/<subject>/         # retired subjects, same shape
  <anything else>/              # the user's own notes
```

Consumers (for example project-appraisal's opt-in `--vault` connector) may rely on:

1. **Root:** `HOARD_VAULT_ROOT`, from the environment or `$XDG_CONFIG_HOME/hoard/config.env` (`$HOME/.config/hoard/config.env` when `XDG_CONFIG_HOME` is unset, on every OS). The file holds `KEY=value` lines, values optionally double-quoted; an environment variable of the same name wins.
2. **Subject of a path under the root:**
   - `Agents/<scope>/<subject>/…` → `Agents/<scope>/<subject>`;
   - `Agents/<scope>/_Archive/<subject>/…` → `Agents/<scope>/<subject>` (archiving does not change identity);
   - `Agents/<scope>/_Inbox/…` → no subject (unfiled; consumers fall back to their non-vault grouping);
   - anything else → its first three directory levels.

A change to this contract is a breaking change and is called out in the changelog.

## Components

```
hoard/
  hooks/
    _lib.sh          payload parsing for Claude/Codex/Hermes, config loading, path normalization
    recall.sh        prompt-submit hook
    after-write.sh   post-write hook: mirror + nudges
  skills/tidy/       SKILL.md, scan.sh, apply.sh
  rules/HOARD.md     the rules agents read (~60 lines)
  cmd/hoard/         Go: install / uninstall / doctor
  test/              bats (hooks × runtimes), go test (installer)
```

Hooks are bash; the installer is Go.

### Configuration

`$XDG_CONFIG_HOME/hoard/config.env` (fallback `$HOME/.config`), written by the installer and parsed (never sourced) by the hooks:

| Key | Default | Meaning |
|---|---|---|
| `HOARD_VAULT_ROOT` | (required) | vault root |
| `HOARD_DEFAULT_SCOPE` | `personal` | scope when `AGENT_SCOPE` is unset; also the shared scope recall searches from other scopes |
| `HOARD_MIRROR_EXCLUDE` | empty | colon-separated globs the mirror skips |
| `HOARD_DISABLE` | empty | comma list of `recall`, `mirror`, `nudge` |

Environment variables of the same name override the file. The active scope is `$AGENT_SCOPE`, else `HOARD_DEFAULT_SCOPE`; it must match `^[a-z0-9][a-z0-9-]*$` or the default is used.

State lives in `${XDG_STATE_HOME:-~/.local/state}/hoard/`; files older than 7 days are pruned.

### `recall.sh` — prompt submit

Ported from an existing deterministic recall hook, without any external relevance-scoring service.

- Extracts terms from the prompt and the cwd's git branch: issue numbers (3–5 digits, not years), quoted phrases, identifiers (dotted, snake, CamelCase), and words of 4+ letters minus a stop list; at most 12 terms.
- Searches `*.md` and `*.html` with `rg` under: the active scope's `Agents/<scope>/`, `Agents/<HOARD_DEFAULT_SCOPE>/` when the active scope differs, and every top-level vault folder except `Agents/`. Other scopes are never searched. `_Archive` and `_Inbox/scratch` are skipped. `.ignore` applies.
- Scores files (issue match 5, subject-name match 2, phrase/identifier hit 2, word hit 1; keep ≥ 3), groups by subject, prefers durable files over drafts, emits at most 3 subjects × 3 files with title, age and matched terms.
- Per-session state (path + mtime) suppresses repeats.
- Output is the runtime's prompt-time context injection: Claude Code and Codex `UserPromptSubmit` (`hookSpecificOutput.additionalContext`), Hermes `pre_llm_call` (`{"context": …}`, prompt read from `extra.user_message`).

### `after-write.sh` — post write

Wired on Claude `PostToolUse` (Write, Edit), Codex `PostToolUse` (`apply_patch`), Hermes `post_tool_call` (`write_file`, `patch`). Never blocks, never modifies the written file. Hermes ignores `post_tool_call` output, so there the mirror still copies but the agent gets no notes; `hoard doctor` says so. For each written path:

1. **Mirror.** A `.md` or `.html` file under `/tmp` or `$TMPDIR`, not matching `HOARD_MIRROR_EXCLUDE`, is copied into `Agents/<scope>/_Inbox/`:
   - inside an agent scratchpad under the temp root (Claude Code's `<tmp>/claude-<uid>/<project>/<session>/scratchpad/`; Codex and Hermes keep scratch outside the temp root, so it is not mirrored) → `_Inbox/scratch/<YYYY-MM-DD>-<session-short>/<path under the scratch root>`, silently;
   - otherwise → `_Inbox/<YYYY-MM-DD>-<name>`, and the agent is told where the copy is and to move it into its subject.
   
   A per-session `src → dest` map makes later edits overwrite the same copy. A collision with a file the mirror did not create gets `-2`, `-3`, … suffixes.
2. **Draft nudge.** A draft-shaped `.md`/`.html` name written directly in `Agents/<scope>/` adds context: drafts belong in `<subject>/Drafts/<slug>/`. Draft-shaped means a name starting `pr-`, `issue-`, `polished-`, `msg-`, `prompt-`, `draft-` (or the word followed by a digit), or ending `-comment`, `-comments`, `-reply`, `-replies`, `-description`, `-body`, `-adjudication`, `-verdict`, `-closing-comment`.
3. **Scope nudge.** A write into `Agents/<other-scope>/` adds context naming that scope and the active one.

Shell redirections (`cat > /tmp/x.md`) are invisible to post-write hooks; `HOARD.md` states the mirror rule so agents cover that case themselves.

### Error handling

Every hook exits 0 with no output when a dependency is missing, the vault root does not exist, the payload is malformed, or its feature is disabled. Problems surface through `hoard doctor`, not through hook failures. Hermes treats any non-JSON stdout as a block, so allow paths print nothing.

### Tidy skill

Ported from an existing tidy skill; paths come from config. One scope per run:

1. `scan.sh` inventories `Agents/<scope>/` outside `_Archive` (type, path, file count, newest date).
2. The agent groups rows by subject and presents one table (subject, members, newest, suggested `group`/`archive`/`skip`), flagging destinations reached through symlinks.
3. The user approves or edits; ambiguous replies get one clarifying question.
4. The agent writes a `src<TAB>dest` move plan; `apply.sh` executes it, refusing to overwrite, logging every move to the state dir for manual undo.
5. Empty directories are pruned (never under `_Archive`), and a rescan verifies.

`_Inbox/` and `_Inbox/scratch/<session>/` appear as ordinary rows; scratch sessions older than 30 days are suggested for `archive`. Tidy is the bulk tool, not the only permitted way to move notes: agents may move individual notes into place directly. Moves never rename files, so Obsidian wiki-links survive.

### Rules (`rules/HOARD.md`)

Covers the pillars, the layout, scope separation, search-don't-hardcode, the mirror, and tidy. Claude Code gets one `@<data>/rules/HOARD.md` import line in `CLAUDE.md`. Codex `AGENTS.md` and Hermes `SOUL.md` have no import syntax, so the installer pastes the content between `<!-- hoard:begin -->` / `<!-- hoard:end -->` markers and refreshes it on reinstall.

### Installer (`hoard`, Go)

- `hoard install --agent claude|codex|hermes [--vault <path>]`: writes `config.env` (`--vault` is required when no config exists), copies hooks, skill and rules to `~/.local/share/hoard/`, merges hook entries into the agent's config (`~/.claude/settings.json`, `~/.codex/hooks.json`, Hermes hook config), installs the tidy skill as `hoard-tidy` where the agent loads skills, and adds the rules (see Rules). Codex and Hermes ask the user to trust new hooks on first run; install prints how. Backs up every file it edits; running twice changes nothing.
- `hoard uninstall --agent …`: removes exactly what install added.
- `hoard doctor`: checks dependencies, config, vault root, and per-agent wiring, including whether recall can inject context in that runtime.

## Dependencies

Hooks at runtime:

- bash ≥ 4
- jq
- ripgrep
- git
- coreutils/findutils, used through portable fallbacks so stock macOS works (GNU `stat -c` vs BSD `stat -f`, `date`, `find`)
- POSIX grep, sed

Build and development: Go ≥ 1.23 (release binaries remove this for users), bats-core, ShellCheck.

## Testing

- bats per hook × runtime, using Claude, Codex and Hermes payload fixtures against a temporary vault of synthetic notes: recall ranking, scope exclusion, `.ignore`, repeat suppression; mirror destinations, re-edit overwrite, collisions, scratch vs loose; both nudges; every fail-open path (no jq, no rg, no root, malformed payload) exits 0 silently.
- `go test` for the installer: merge into existing settings preserves other entries, second install is a no-op, uninstall restores the original, backups are written.
- CI: GitHub Actions on ubuntu and macos, ShellCheck at warning severity.

## Release

- Public repo `Duro-Labs/hoard`, MIT license.
- Before the first push, a scripted check fails on personal names, emails, home paths, or client identifiers anywhere in the tree; all fixtures are synthetic.
- goreleaser builds `hoard` for linux and darwin, amd64 and arm64; `go install` also works.
- README: what HOARD is, install, dependencies, configuration, the layout contract.

## Out of scope for v1

- External relevance scoring for recall (a pluggable gate can come later).
- Windows.
- Agents other than Claude Code, Codex and Hermes.
- Blocking guards of any kind.
- Migrating the author's existing personal setup onto HOARD (a separate follow-up once v1 is tagged).
