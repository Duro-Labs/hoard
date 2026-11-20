# HOARD

**H**ooks · **O**bsidian · **A**rchive · **R**ecall · **D**rafts

HOARD gives Claude Code, Codex, and Hermes a shared Obsidian vault as long-term memory. It gives agents a fixed place for notes, recalls related notes at prompt time, mirrors notes written to temporary directories, and provides a reviewed bulk-tidy flow. Hooks only add context or copy files; they never block an agent.

## Install

Install with `go install github.com/Duro-Labs/hoard/cmd/hoard@latest` or use a release binary. Point the first agent at an existing vault, then add other agents using the saved configuration:

```sh
hoard install --agent claude --vault ~/Notes
hoard install --agent codex
hoard install --agent hermes
hoard doctor --agent claude
```

Codex and Hermes ask you to trust new hooks on first use. Run `hoard doctor --agent <agent>` for each agent you use; running `hoard doctor` without an agent checks all three.

## Dependencies

Hooks need bash 4 or newer, `jq`, `rg` (ripgrep), `git`, standard `find`, `stat`, and `date` utilities, and POSIX `grep` and `sed`. Building needs Go 1.23 or newer. Development tests use bats-core and ShellCheck.

## Configuration

The installer writes `${XDG_CONFIG_HOME:-$HOME/.config}/hoard/config.env` on every OS. Hooks parse `KEY=value` lines without sourcing the file; values may be double-quoted. Environment variables of the same name win.

| Key | Default | Meaning |
|---|---|---|
| `HOARD_VAULT_ROOT` | required | Vault root. Pass `--vault` on first install. |
| `HOARD_DEFAULT_SCOPE` | `personal` | Scope when `AGENT_SCOPE` is unset; also the shared scope recall searches from another scope. |
| `HOARD_MIRROR_EXCLUDE` | empty | Colon-separated globs the mirror skips. |
| `HOARD_DISABLE` | empty | Comma list of `recall`, `mirror`, `nudge`. |

`HOARD_TEMP_ROOTS` is an advanced, environment-only colon-separated list of temporary roots; its default is `/tmp:$TMPDIR`. State lives in `${XDG_STATE_HOME:-$HOME/.local/state}/hoard/`, and installed assets live in `${XDG_DATA_HOME:-$HOME/.local/share}/hoard/`.

## Hooks

Recall extracts terms from a prompt and its git branch, searches the active and default scopes plus non-Agent vault folders, and surfaces up to three matching subjects. It honors the vault's `.ignore` and suppresses repeated files per session.

After a file-write tool writes a Markdown or HTML note under a temporary root, the mirror copies it to `Agents/<scope>/_Inbox/`. Later edits update that copy. Claude Code scratchpad notes under the temporary root go into `_Inbox/scratch/`. Shell redirections are invisible to post-write hooks.

The post-write hook also nudges draft-shaped filenames placed at the scope root and writes into another scope. Hermes ignores post-write hook output, so its copies still happen but these notes do not reach the agent.

| | Claude Code | Codex | Hermes |
|---|---|---|---|
| Recall | `UserPromptSubmit` | `UserPromptSubmit` | `pre_llm_call` |
| Mirror | `PostToolUse` Write/Edit | `PostToolUse` apply_patch | `post_tool_call` write_file/patch |
| Post-write notes to agent | yes | yes | no (output ignored) |

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

## Uninstall

Run `hoard uninstall --agent claude|codex|hermes` for each installed agent. After uninstalling all agents, the installed data at `${XDG_DATA_HOME:-$HOME/.local/share}/hoard/` can be deleted.

## License

MIT.
