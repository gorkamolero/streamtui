# StreamTUI

Terminal streaming app and agent-safe media CLI. Search movies and shows, rank streams, find subtitles, resolve playback devices, and start playback from a terminal command.

The original implementation is a Rust TUI/CLI inspired by Stremio and Popcorn Time. The active branch is a Go CLI rewrite focused on deterministic, JSON-first workflows for humans, scripts, and coding agents.

## What It Proves

- Product taste for power-user tooling: fast terminal workflow, keyboard-first interaction, clear command contracts, and practical defaults.
- Agent-first CLI design: every automation path returns a stable v1 JSON envelope with semantic exit codes and no prompts in JSON mode.
- Systems integration: TMDB search, Torrentio stream discovery, Stremio subtitles, WebTorrent playback, VLC fallback, and Chromecast control through `catt`.
- Defensive workflow design: provider adapters, pure ranking logic, fake HTTP/subprocess tests, device cache resolution, and structured errors.
- Willingness to rewrite when the architecture demands it: the Go rewrite keeps the working product idea while making the CLI more deterministic and testable.

## Product Model

```text
query or imdb id
  -> search / metadata provider
  -> stream discovery
  -> deterministic ranking
  -> subtitle search and cache
  -> device resolution
  -> WebTorrent playback
  -> Chromecast or VLC
```

Agent-first target:

```bash
streamtui play "the batman" --lang es --device "Living Room TV" --json
```

When no Chromecast is available, the same workflow can run locally:

```bash
streamtui play "the batman" --lang es --vlc --json
```

## Current Implementations

### Rust App

The Rust implementation contains the original TUI and CLI surface:

- interactive terminal UI
- search, trending, info, stream, subtitle, cast, local playback, and control commands
- Chromecast playback through `webtorrent-cli` and `catt`
- subtitle selection
- integration tests for the existing behavior

### Go CLI Rewrite

The Go rewrite is the current agent-safe direction:

- one `streamtui` binary
- v1 JSON envelope: `ok`, `data`, `error`, `meta.version`
- semantic exit codes
- non-interactive JSON behavior
- command parser built for agent usage
- provider adapters for TMDB, Torrentio, subtitles, devices, playback, and control
- pure ranking and selection logic
- fake HTTP servers and fake subprocess binaries in tests
- device resolution through explicit device, saved default, last-used cache, fresh cache, then discovery
- one-command `play "<query>" --lang <code> --device <name> --json` workflow
- VLC fallback path when Chromecast discovery is unavailable

Rewrite docs:

- [`docs/directional-rewrite-goal.md`](docs/directional-rewrite-goal.md)
- [`docs/cli-contract.md`](docs/cli-contract.md)
- [`docs/cli-rewrite-progress.md`](docs/cli-rewrite-progress.md)
- [`docs/manual-gate-runbook.md`](docs/manual-gate-runbook.md)

## Command Contract

All JSON responses use the v1 envelope:

```json
{
  "ok": true,
  "data": {},
  "error": null,
  "meta": {
    "version": "v1"
  }
}
```

Failures keep the same shape:

```json
{
  "ok": false,
  "data": null,
  "error": {
    "code": "DEVICE_NOT_FOUND",
    "message": "No Chromecast device found"
  },
  "meta": {
    "version": "v1"
  }
}
```

Core commands in the v1 contract:

```text
search <query>
trending
info <id>
streams <imdb_id> [--season --episode]
subtitles <imdb_id> [--lang] [--season --episode]
devices list
devices refresh
devices default set <name>
devices default get
cast <imdb_id> [--device] [--lang] [--subtitle-id] [--vlc]
cast-magnet <magnet> [--device] [--subtitle-file] [--vlc]
status
play
pause
stop
seek
volume
play "<query>" [--lang] [--device] [--vlc]
```

See [`docs/cli-contract.md`](docs/cli-contract.md) for the full contract.

## Example Workflows

Search:

```bash
streamtui search "blade runner" --json
```

Find streams:

```bash
streamtui streams tt1856101 --quality 1080p --sort ranked --json
```

Find subtitles:

```bash
streamtui subtitles tt1856101 --lang es --json
```

Cast by IMDb id:

```bash
streamtui cast tt1856101 --device "Living Room TV" --lang es --json
```

Run the full agent-first workflow:

```bash
streamtui play "blade runner 2049" --lang es --device "Living Room TV" --json
```

Run the same workflow through VLC:

```bash
streamtui play "blade runner 2049" --lang es --vlc --json
```

## Device Resolution

Playback resolves devices in this order:

1. explicit `--device`
2. saved default device
3. last successful cached device
4. fresh cached device list
5. new discovery

This keeps commands useful for automation while still supporting local Chromecast discovery.

## Development

Rust implementation:

```bash
cargo test
cargo build --release
```

Go rewrite:

```bash
gofmt -w cmd internal
go test ./...
go build -o /tmp/streamtui-go-check ./cmd/streamtui
```

The rewrite progress document records the last verification snapshot:

- `go test ./...` passes
- Go binary build passes
- `git diff --check` passes
- `cargo test` passes for the existing Rust implementation
- Gate A passed for real search, streams, and subtitle discovery
- VLC start path passed
- Chromecast gates remain unproven when no Chromecast is discoverable locally

## Architecture

Go rewrite shape:

```text
cmd/streamtui/main.go          CLI entry point
internal/cli                   parser, runner, envelope, exit codes, app orchestration
internal/domain                media, stream, subtitle, device, playback, control models
internal/providers             TMDB, Torrentio, subtitles, webtorrent, catt adapters
internal/config                API keys, provider base URLs, test seams
docs/cli-contract.md           v1 command and JSON contract
docs/cli-rewrite-progress.md   phase evidence and manual gate status
```

Rust implementation shape:

```text
src/main.rs       entry point
src/app.rs        TUI state machine
src/cli.rs        CLI parsing
src/commands.rs   command handlers
src/models.rs     shared data structures
tests/            integration tests
specs/            original design specifications
```

## Dependencies

For full playback behavior:

```bash
npm install -g webtorrent-cli
pip install catt
```

VLC is used for local playback fallback.

## Status

Active rewrite in progress.

Done:

- Go CLI skeleton and command contract
- JSON v1 envelope and semantic exits
- metadata, stream, subtitle, device, playback, control, and agent-first workflows
- deterministic test seams for providers and subprocesses
- VLC fallback for one-command playback

Still pending:

- commit and publish the full Go rewrite branch
- validate real Chromecast Gate B and Gate C when a device is discoverable
- decide whether the public default should be Rust TUI, Go CLI, or a paired release
- add terminal captures and a short demo for the portfolio

## Portfolio Context

StreamTUI is a strong systems/product proof. It is not just a media toy; it shows CLI contract design, real-world provider integration, deterministic testing, and the ability to turn an interactive consumer workflow into an agent-safe command surface.

It fits the portfolio as the practical terminal counterpart to the agentic tools: where `claude-on-discord` exposes coding agents through Discord, StreamTUI exposes media search and playback through deterministic CLI automation.
