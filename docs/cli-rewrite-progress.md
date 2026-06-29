# CLI Rewrite Progress

## Current Verification Snapshot

Automated verification on 2026-05-30:
- `gofmt -w cmd internal` completed.
- `go test ./...` passes.
- `go build -o /tmp/streamtui-go-check ./cmd/streamtui` passes.
- `git diff --check` passes.
- `cargo test` passes for the existing Rust implementation.
- `go run ./cmd/streamtui play --json` returns a v1 `DEVICE_NOT_FOUND` envelope when the saved session points at a missing Chromecast.

External tooling installed for manual gates:
- `go version go1.26.3 darwin/arm64`
- `webtorrent 6.0.0 (2.8.5)`
- `catt v0.13.1`
- VLC is installed at `/Applications/VLC.app`.

Manual gate status:
- Gate A passed for real TMDB search, Torrentio stream discovery, and Stremio subtitle discovery.
- Gate B VLC start path passed with WebTorrent's official Sintel test magnet.
- Agent-first VLC path is available for the one-command workflow when Chromecast discovery is unavailable.
- Gate B Chromecast path remains unproven because `catt scan` reports `Error: No devices found.`
- Gate C remains unproven because the required end-to-end Chromecast device is not discoverable on the local network.

## Phase 0 - Contract and Skeleton

Status: implemented and automated tests pass.

Evidence added:
- `go.mod`
- `cmd/streamtui/main.go`
- `internal/cli` parser, exit-code, JSON envelope, and run helpers
- Go contract tests for command parsing, v1 success envelope, v1 error envelope, semantic exit code mapping, and non-interactive invalid-argument handling
- Parser accepts trailing `--json/-j` and `--quiet/-q` on commands as harmless global-style flags, matching agent command habits while preserving positional values such as `volume -10`
- Parser joins unquoted multi-word queries for `search` and agent-first `play`, while ID-based commands reject accidental extra positionals
- Parser joins unquoted device words for `devices default set`
- Zero-argument commands such as `trending`, `pause`, and `devices refresh` reject accidental extra positionals with structured `INVALID_ARGS`
- The ambiguous `play` command keeps both supported forms agent-safe: `play --json` resumes playback, while `play "<query>" --json` starts the agent-first search workflow.

Available verification:
- `gofmt -w cmd internal` passes.
- `go test ./...` passes.
- `git diff --check` passes.
- `cargo test` passes for the existing Rust implementation.

## Phase 1 - Metadata and Stream Discovery

Status: implemented and automated tests pass.

Evidence added:
- `internal/domain` media and stream models
- Deterministic stream quality, seed, size, magnet, filtering, and ranking logic
- `streams --sort ranked|seeds|quality|size` applies deterministic ordering and rejects unknown sort values with `INVALID_ARGS`
- `internal/providers` TMDB adapter for search, trending, movie detail, and TV detail
- `internal/providers` Torrentio adapter for movie and episode stream discovery
- Go tests using `httptest` for TMDB and Torrentio behavior
- `internal/cli.App` provider interfaces and command execution for `search`, `trending`, `info`, and `streams`
- Go command-level tests with fake providers for filtering, limiting, ranking, explicit sort modes, indexing, and provider error mapping

Available verification:
- `gofmt -w cmd internal` passes.
- `go test ./...` passes.
- Gate A real commands for search and streams pass.
- `git diff --check` passes.
- `cargo test` passes for the existing Rust implementation.

## Phase 2 - Subtitle Discovery

Status: implemented and automated tests pass.

Evidence added:
- `internal/domain` subtitle model, language normalization, language names, trust scoring, and SRT-to-WebVTT conversion
- `internal/providers` Stremio subtitle adapter for movie and episode searches
- Subtitle download and WebVTT cache helper
- `internal/cli.App` provider interface and command execution for `subtitles`
- Go tests using `httptest` for subtitle search, language filtering, episode paths, invalid JSON, download conversion, and cache reuse
- Go command-level tests with fake providers for trusted filtering, trust sorting, limiting, and TV season/episode validation

Available verification:
- `gofmt -w cmd internal` passes.
- `go test ./...` passes.
- Gate A real subtitle discovery passes.
- `git diff --check` passes.
- `cargo test` passes for the existing Rust implementation.

## Phase 3 - Device and Cache Model

Status: implemented and automated tests pass.

Evidence added:
- `internal/domain` cast device model and `catt scan` parser
- Device cache model with `updated_at`, `last_seen`, `last_used`, default device, TTL freshness, and resolution order helpers
- File-backed device cache store at the user cache path
- `catt scan` device discovery adapter
- `internal/cli.App` provider/store interfaces and command execution for `devices list`, `devices refresh`, `devices default set`, and `devices default get`
- Go pure tests for scan parsing, TTL freshness, cache update behavior, and device resolution order
- Go command-level tests with fake device providers/stores for cache reuse, forced refresh, and default set/get
- Go runner-level test covers the `devices refresh --json` `DEVICE_NOT_FOUND` envelope and `no Chromecast devices found` message
- `cast` and `cast-magnet` use cache-based resolution and mark `last_used` after playback starts
- `catt scan` no-device output maps to `DEVICE_NOT_FOUND` instead of `NETWORK_ERROR`
- Explicit `--device` values pass through directly and do not require cache/discovery before playback starts

Available verification:
- `gofmt -w cmd internal` passes.
- `go test ./...` passes.
- `go run ./cmd/streamtui --json devices refresh` returns `DEVICE_NOT_FOUND` with message `no Chromecast devices found` when `catt scan` reports no local devices.
- `git diff --check` passes.
- `cargo test` passes for the existing Rust implementation.

## Phase 4 - Playback Adapters

Status: implemented and automated tests pass, VLC start path verified, pending Gate B Chromecast validation.

Evidence added:
- `internal/domain` playback request/start models and magnet validation
- `internal/providers` webtorrent playback adapter
- Pure webtorrent argument builders for VLC and Chromecast paths
- `cast-magnet --vlc` command execution
- `cast-magnet -d <device>` command execution with device resolution/cache updates
- `cast <imdb_id>` orchestration from stream discovery to ranked stream selection to optional subtitle download to playback start
- Cast stream selection treats `--quality` as a preferred quality, choosing an exact match ahead of higher-quality streams while preserving deterministic tie-breaks
- Go provider tests for webtorrent argument builders
- Go command-level tests with fake playback adapter for VLC, Chromecast, device last-used updates, stream selection, subtitle cache path use, and playback response shape
- Go command-level tests prove preferred-quality cast selection chooses the requested quality ahead of higher-quality alternatives
- Go subprocess test with a fake `webtorrent` binary proves the playback adapter invokes the expected command shape
- Go subprocess test proves immediate `webtorrent` failures are reported instead of returning false `ok: true` playback starts
- Go command-level test proves explicit `--device` skips discovery and passes through to playback

Available verification:
- `gofmt -w cmd internal` passes.
- `go test ./...` passes.
- `git diff --check` passes.
- `cargo test` passes for the existing Rust implementation.
- `webtorrent --version` returns `6.0.0 (2.8.5)`.
- `go run ./cmd/streamtui --json cast-magnet <WebTorrent Sintel magnet> --vlc` returns `ok: true` with `player: "VLC"` after the playback adapter's immediate-exit grace period.

Missing verification:
- Gate B Chromecast start-path validation still needs a reachable Chromecast.

## Phase 5 - Control Commands

Status: implemented and automated tests pass.

Evidence added:
- `internal/domain` playback status, control result, seek/volume command parsing, `catt status` parsing, and session metadata JSON helpers
- `internal/providers` `catt` control adapter for `status`, `play`, `pause`, `stop`, `seek`, and `volume`
- File-backed session metadata store at the user cache path
- `internal/cli.App` command execution for `status`, playback resume `play`, `pause`, `stop`, `seek`, and `volume`
- Control device resolution through explicit/global device, saved session device, then cached/default discovery
- Cast and cast-magnet paths save session metadata after playback starts
- Status fills missing title from saved session metadata when `catt status` omits it
- `catt` control errors for missing devices map to `DEVICE_NOT_FOUND` envelopes instead of leaking raw subprocess failures.
- Go command-level tests with fake control/session stores for session device lookup, global device lookup, relative seek, relative volume, and session persistence
- Go subprocess tests with a fake `catt` binary prove the control adapter invokes and parses the expected command shape and maps missing-device output to the shared not-found error

Available verification:
- `gofmt -w cmd internal` passes.
- `go test ./...` passes.
- `git diff --check` passes.
- `cargo test` passes for the existing Rust implementation.
- `catt --version` returns `catt v0.13.1, Zaniest Zapper.`

## Phase 6 - Agent-First Command

Status: implemented and automated tests pass, pending Gate C Chromecast validation.

Evidence added:
- `play "<query>" --lang <code> --device <name>` now executes through `internal/cli.App`
- `play "<query>" --lang <code> --vlc` executes the same agent-first search, stream, and subtitle workflow through VLC without resolving a Chromecast device.
- Query playback searches TMDB, chooses the first playable movie result, resolves a missing IMDb id through movie detail, selects the best ranked stream, picks the top matching subtitle, resolves the target device, starts Chromecast playback, marks the device used, and saves session metadata
- Query playback prefers an exact movie-title match before falling back to the first playable movie, so the agent-first command follows the contract's "best search match" behavior without selecting TV results.
- Parser accepts the documented trailing `--json` form for the agent-first play command
- Parser accepts `--vlc` on the agent-first play command.
- Go command-level tests cover exact-title search result preference, fallback movie selection, detail IMDb lookup, deterministic stream selection, Spanish subtitle selection, Chromecast device targeting, playback start response, and session persistence
- Go command-level tests cover the VLC agent-first path and prove it skips device resolution.
- Go runner-level test covers `play "the batman" --lang es --device "Living Room TV" --json` through the v1 JSON envelope
- Go default-app end-to-end test covers the real CLI entry point with fake TMDB, Torrentio, subtitle HTTP services, subtitle download/cache, and fake `webtorrent`
- Go config fallback mirrors the Rust TMDB key-pool behavior when `TMDB_API_KEY` is not set
- `STREAMTUI_TMDB_BASE_URL`, `STREAMTUI_TORRENTIO_BASE_URL`, and `STREAMTUI_SUBTITLES_BASE_URL` provide deterministic test seams for the default app without changing production defaults

Available verification:
- `gofmt -w cmd internal` passes.
- `go test ./...` passes.
- `go build -o /tmp/streamtui-go-check ./cmd/streamtui` passes.
- `git diff --check` passes.
- `cargo test` passes for the existing Rust implementation.
- Gate A real metadata/stream/subtitle checks pass.

Missing verification:
- Gate B real Chromecast start-path validation remains unrun.
- Gate C one-command agent workflow validation remains unrun because `catt scan` reports no Chromecast devices on the local network.

Next actions:
- Re-run Gate B and Gate C when a Chromecast is discoverable on the local network.

## Manual Gate Commands

Detailed pass/fail evidence requirements now live in `docs/manual-gate-runbook.md`.

Gate A observed commands:

```bash
go run ./cmd/streamtui --json search "the batman"
go run ./cmd/streamtui --json streams tt1877830 --limit 3
go run ./cmd/streamtui --json subtitles tt1877830 --lang es --limit 3
```

Observed result: all three commands returned `ok: true` v1 JSON envelopes with real data.

Gate B commands to run with a reachable Chromecast and a safe test magnet:

```bash
go run ./cmd/streamtui --json devices refresh
go run ./cmd/streamtui --json cast-magnet "<WebTorrent Sintel magnet>" --vlc
go run ./cmd/streamtui --json cast-magnet "<safe-test-magnet>" --device "<Chromecast name>"
```

Observed result in this environment:
- `go run ./cmd/streamtui --json cast-magnet <WebTorrent Sintel magnet> --vlc` returned `ok: true` with `player: "VLC"`.
- `go run ./cmd/streamtui --json devices refresh` returns `DEVICE_NOT_FOUND` with message `no Chromecast devices found` because `catt scan` reports `Error: No devices found.`
- `go run ./cmd/streamtui play --json` returns `DEVICE_NOT_FOUND` with message `no Chromecast devices found` when the saved session device is not reachable.

Gate C command to run with a reachable Chromecast:

```bash
go run ./cmd/streamtui play "the batman" --lang es --device "<Chromecast name>" --json
```

Observed result in this environment: not run to completion because no Chromecast is discoverable on the local network.

Agent-first VLC finish command when Chromecast discovery is unavailable:

```bash
go run ./cmd/streamtui play "the batman" --lang es --vlc --json
```

Observed automated result: the real CLI entry point is covered by `TestRunAgentFirstPlayWithDefaultAppAndFakeServices`, and the app-level VLC branch is covered by `TestAgentFirstPlaySupportsVLCWithoutDeviceResolution`.
