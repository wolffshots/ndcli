---
name: notediscovery
description: Read, search, create and edit notes on a NoteDiscovery server with the ndcli command-line client. Use when the user names NoteDiscovery, or asks to read, search, create, edit, move or delete a note on that server.
---

# NoteDiscovery (ndcli)

NoteDiscovery is a self-hosted Markdown note server. `ndcli` is a command-line client for its REST API.

## Setup

| Variable | Use |
|---|---|
| `NOTEDISCOVERY_URL` | Base URL of the server. Default `http://localhost:8000`. |
| `NOTEDISCOVERY_API_KEY` | Optional. Set it only if the server has authentication on. |

Run `ndcli health` first. If it fails, stop. Tell the user to check `NOTEDISCOVERY_URL`. Do not print the API key.

## Commands

A note path is relative to the vault root and ends in `.md`. Put quotes around a path that contains a space.

| Command | Example |
|---|---|
| Check the server | `ndcli health` |
| List notes | `ndcli ls --limit 20 --offset 0` |
| Print a note | `ndcli get "projects/road map.md"` |
| Print a note with metadata and backlinks | `ndcli get projects/plan.md --json` |
| Create or replace a note from a file | `ndcli put projects/plan.md plan.md` |
| Create or replace a note from stdin | `printf '# Plan\n' \| ndcli put projects/plan.md` |
| Add to the end of a note | `printf 'New entry\n' \| ndcli append journal.md --timestamp` |
| Move or rename a note | `ndcli mv plan.md projects/plan.md` |
| Delete a note | `ndcli rm projects/plan.md` |
| Search note contents | `ndcli search "road map" --limit 10` |
| List tags | `ndcli tags` |
| List the notes with a tag | `ndcli tags python` |
| Create a folder | `ndcli mkdir projects/2025` |
| Print the version | `ndcli --version` |

`put` and `append` read stdin if there is no file argument.

`get` prints the note content only. All other commands print the server JSON.

If the server returns an error, `ndcli` prints the error to stderr and exits with code 1. Example: `ndcli: HTTP 404: Note not found`.

`search` matches a substring and ignores case. A query of fewer than 2 characters returns no results.

## Rules

1. Run `ndcli get` before `ndcli put` on an existing note. `put` replaces the whole file, and the server has no conflict detection.
2. Use `ndcli append` to add to a note. Do not use `get` and `put` for that.
3. Ask the user before `ndcli rm`. Ask the user before a `put` that removes content from a note.
4. Warn the user that the web editor saves automatically each second. A note that is open in a browser can overwrite an edit from `ndcli`. Tell the user to close the note in the browser before a `put`.

## Not supported

`ndcli` does not support media, sharing, templates, themes, plugins, HTML export or ZIP archive. Use the web interface for these.
