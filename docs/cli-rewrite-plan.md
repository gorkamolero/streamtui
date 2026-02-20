# CLI Rewrite Plan (Go, TUI Paused)

## Goals
- CLI finds torrents.
- CLI finds subtitles.
- CLI casts to VLC.
- CLI casts to local Chromecast devices.
- Agent workflows stay simple: one command can do search + subtitle + cast.
- Device selection is cached to avoid repeated prompts.

## Guiding Architecture
- Keep one binary and one process model (modular monolith).
- Separate pure decision logic from side effects.
- Keep all command outputs deterministic and machine-readable.
- Keep external tooling (`webtorrent`, `catt`) behind adapters.

## Phases

### Phase 0 - Contract and skeleton
- Define CLI command contract and JSON schema.
- Create Go project structure and baseline command parser.
- Add contract tests for output format and exit codes.

### Phase 1 - Metadata and stream discovery
- Implement `search`, `trending`, `info` via TMDB provider.
- Implement `streams` via Torrentio provider.
- Keep sorting and ranking deterministic.

### Phase 2 - Subtitle discovery
- Implement `subtitles` command.
- Normalize language handling (`es`, `spa`, mixed input).
- Add subtitle cache.

### Phase 3 - Device and cache model
- Implement `devices list`, `devices refresh`, `devices default set/get`.
- Persist device cache with TTL and `last_used` metadata.
- Device resolution order:
  1. `--device`
  2. stored default
  3. last successful device
  4. cached devices
  5. forced scan

### Phase 4 - Playback adapters
- Implement `cast-magnet --vlc`.
- Implement `cast-magnet -d <device>`.
- Implement `cast` orchestration (content id -> stream -> cast).

### Phase 5 - Control commands
- Implement `status`, `play`, `pause`, `seek`, `volume`, `stop`.
- Add session metadata cache to support better control reporting.

### Phase 6 - Agent-first command
- Implement `play "<query>" --lang <code> --device <name>`.
- Keep behavior non-interactive by default.

## Test Strategy Per Phase
- Unit tests: pure logic and ranking behavior.
- Adapter tests: HTTP and subprocess mocks.
- Contract tests: JSON shape snapshots.
- E2E tests: fake `webtorrent` and fake `catt` binaries.

## Manual Test Gates (where to pause)
- Gate A after Phase 2: verify search/streams/subtitles against real data.
- Gate B after Phase 4: verify VLC playback and Chromecast start path.
- Gate C after Phase 6: verify one-command agent workflow end-to-end.

