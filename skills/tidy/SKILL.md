---
name: hoard-tidy
description: Tidy one vault scope (Agents/<scope>/): scan everything outside _Archive, group files and folders by subject, present a decision table, then execute the user-approved group/archive moves. Use for "tidy the vault", "organize the agents folder", "file the inbox", or "archive vault subjects". User-invoked; never run proactively.
---

# hoard-tidy

Tidy one scope at a time under `<vault>/Agents/<scope>/`. `<vault>` is `HOARD_VAULT_ROOT` from hoard's config. Agents may move single notes into place directly; this skill is for bulk reorganization. Nothing moves until the user approves a table of proposed changes.

## Target layout

```
<vault>/Agents/<scope>/
  <subject>/
    <deliverable>.md
    Drafts/<slug>/
  _Inbox/
    <YYYY-MM-DD>-<name>
    scratch/<YYYY-MM-DD>-<session>/
  _Archive/<subject>/
```

## Workflow

1. Run `<this skill's directory>/scripts/scan.sh` and capture its output and exit status. Do not group against an incomplete inventory. Each TSV row is `type<TAB>relpath<TAB>filecount<TAB>newest(YYYY-MM-DD)`.
2. Group rows by subject, using names first and reading ambiguous files only as needed. `_Inbox` items are grouped like other rows. Suggest `archive` for `_Inbox/scratch/<session>` folders older than 30 days.
3. Present one table with Subject, Members, Newest, and Suggested (`group`, `archive`, or `skip`). Flag any destination whose parent chain crosses a symlink. Suggest `skip` for already grouped subjects. Wait for the user's approval or edits; ask one clarifying question if the reply is ambiguous.
4. Write an approved `src<TAB>dest` plan with clean paths relative to the scope root. Group files into `<subject>/`, drafts into `<subject>/Drafts/<slug>/`, and archive by moving `<subject>` to `_Archive/<subject>` after grouping. Never plan a destination that already exists; for merges, plan each child separately. Do not use paths with leading `./`, `//`, `.` or `..` components, or absolute paths.
5. Check final destination parent chains for symlinks again. Run `<this skill's directory>/scripts/apply.sh moves.tsv`, with the same scope root used for scanning. Exit 2 means setup failed before any row; exit 1 means at least one row failed. Report `moved:` and `FAIL` lines, including a failure if a row was moved but its log append failed. Do not improvise collision renames.
6. Remove empty directories outside `_Archive`; never prune inside `_Archive`. Rescan and report the number moved, archived, and failed.

## Notes

Moves keep filenames, preserving Obsidian wiki-links. The audit log is `$XDG_STATE_HOME/hoard/tidy/moves.log` (default `~/.local/state/hoard/tidy/moves.log`), with timestamp, source, and destination columns. Undo manually in reverse order. Folder names may contain spaces; the plan is tab separated. A source beginning `#` is treated as a comment by `apply.sh`, so rename such a file manually before planning its move.
