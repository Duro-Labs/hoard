# HOARD — the agent vault

The user keeps an Obsidian vault at `$HOARD_VAULT_ROOT` (see `${XDG_CONFIG_HOME:-$HOME/.config}/hoard/config.env`). It is shared memory between agents and the user.

## Where to write

- Notes you author go in `Agents/<scope>/<subject>/<deliverable>.md` (or `.html`), with frontmatter `agent: <your agent name>`. `<scope>` is `$AGENT_SCOPE`, else the configured default scope.
- `<subject>` is a kebab-case topic or repo-issue slug (`api-245`, `billing-retry`). Look for an existing subject folder before creating one.
- Text written to be pasted somewhere else and then discarded goes in `Agents/<scope>/<subject>/Drafts/<slug>/`.
- Retired subjects live in `Agents/<scope>/_Archive/<subject>/`.
- Never write agent notes in the vault root, the scope root, or the current working directory (unless the file belongs in the repo).
- Never write into another scope's `Agents/<other-scope>/` unless the user asks.

## Temporary files

Notes you write under `/tmp` or `$TMPDIR` are copied into `Agents/<scope>/_Inbox/` automatically when you use a file-write tool. A note written by a shell redirect is not copied: write it with your file tool, or put it in the vault yourself. When told a file was mirrored, move it into its subject folder once you know the topic.

## Finding notes

- Search; never hardcode a vault path, because notes move. Use `rg` from the vault root; it honours `<vault>/.ignore`.
- Facts you must rely on later belong in your agent config, not only in the vault.
- When related notes are surfaced at the start of a task, read the ones that bear on it before acting.

## Reorganizing

You may move single notes into the right subject folder. For bulk reorganization use the `hoard-tidy` skill, which moves nothing until the user approves its table. Never rename notes while moving them (Obsidian links depend on names).
