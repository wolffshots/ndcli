# ndcli

`ndcli` is a command-line client for the [NoteDiscovery](https://github.com/gamosoft/NoteDiscovery) REST API. NoteDiscovery is a self-hosted Markdown note server.

`ndcli` is one Go binary. It uses the Go standard library only.

## Install

### Homebrew (macOS / Linux)

```sh
brew install wolffshots/tap/ndcli
```

The formula builds from source through [wolffshots/homebrew-tap](https://github.com/wolffshots/homebrew-tap).

### Scoop (Windows)

```powershell
scoop bucket add wolffshots https://github.com/wolffshots/scoop-bucket
scoop install wolffshots/ndcli
```

Scoop installs the Windows x86-64 binary from the latest release through [wolffshots/scoop-bucket](https://github.com/wolffshots/scoop-bucket).

### go install

```sh
go install github.com/wolffshots/ndcli@latest
```

A binary from `go install` prints `ndcli dev` for `ndcli --version`.

### Prebuilt binaries

Download a binary from the [latest release](https://github.com/wolffshots/ndcli/releases/latest). The release has binaries for Linux x86-64, Windows x86-64 and macOS arm64, and a `checksums.txt` file.

## Configuration

`ndcli` reads two environment variables. It has no config file.

| Variable | Use |
|---|---|
| `NOTEDISCOVERY_URL` | Base URL of the server. Default `http://localhost:8000`. |
| `NOTEDISCOVERY_API_KEY` | Optional. If you set it, `ndcli` sends `Authorization: Bearer <key>`. |

These names match the NoteDiscovery MCP server.

```sh
export NOTEDISCOVERY_URL=http://localhost:8000
ndcli health
```

## Commands

| Command | API call |
|---|---|
| `ndcli health` | `GET /health` |
| `ndcli ls [--limit N] [--offset N]` | `GET /api/notes` |
| `ndcli get <path> [--json]` | `GET /api/notes/{path}` |
| `ndcli put <path> [file]` | `POST /api/notes/{path}` |
| `ndcli append <path> [file] [--timestamp]` | `PATCH /api/notes/{path}` |
| `ndcli mv <old> <new>` | `POST /api/notes/move` |
| `ndcli rm <path>` | `DELETE /api/notes/{path}` |
| `ndcli search <query> [--limit N]` | `GET /api/search?q=` |
| `ndcli tags [tag]` | `GET /api/tags` or `GET /api/tags/{tag}` |
| `ndcli mkdir <path>` | `POST /api/folders` |
| `ndcli --version` | None. Prints `ndcli v<version>`. |

- A note path is relative to the vault root, for example `projects/plan.md`. Put quotes around a path that contains a space.
- `put` and `append` read the file argument. If there is no file argument, they read stdin.
- `get` prints the note content only. With `--json`, it prints the full server response, which includes the backlinks.
- All other commands print the server JSON unchanged.
- If the server returns an HTTP error, `ndcli` prints the `detail` field to stderr and exits with code 1.
- A usage error exits with code 2.

### Examples

```sh
ndcli ls --limit 20
ndcli get "projects/road map.md"
printf '# Plan\n' | ndcli put projects/plan.md
ndcli put projects/plan.md plan.md
printf 'New entry\n' | ndcli append journal.md --timestamp
ndcli mv plan.md projects/plan.md
ndcli search "road map" --limit 10
ndcli tags python
ndcli mkdir projects/2025
ndcli rm projects/plan.md
```

## Overwrite risk

NoteDiscovery stores plain `.md` files and has no conflict detection. A write replaces the whole file, and the last write wins.

- `ndcli put` replaces the whole note. It has no conflict guard. Run `ndcli get` first if the note exists.
- Use `ndcli append` to add to a note.
- The web editor saves automatically each second. A note that is open in a browser can overwrite an edit from `ndcli`.

## Not supported

Version 0.1.0 does not support these parts of the API:

- Media
- Sharing
- Templates
- Themes
- Plugins
- HTML export
- ZIP archive

## Agent skill

`skills/notediscovery/SKILL.md` tells an agent how to use `ndcli`. For Claude Code, copy the folder `skills/notediscovery` to `~/.claude/skills/`.

## Development

```sh
go vet ./...
go test ./...
```

## License

[MIT](LICENSE)
