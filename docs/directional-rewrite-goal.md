# Directional Rewrite Goal

Goal: Build the Go CLI rewrite until `streamtui` provides deterministic, agent-safe search, stream, subtitle, device, playback, and one-command play workflows while the TUI remains paused.

Success means:
- The Go project skeleton exposes one `streamtui` binary with the v1 command contract from `docs/cli-contract.md`.
- Every JSON response uses the v1 envelope with `ok`, `data`, `error`, and `meta.version`.
- Contract tests cover command parsing, JSON success and error envelopes, semantic exit codes, and non-interactive behavior.
- Provider logic for TMDB, Torrentio, subtitles, devices, and playback stays behind adapters with pure ranking and selection logic covered by unit tests.
- Device resolution uses explicit device, saved default, last successful device, fresh cached devices, then discovery.
- The agent-first command `streamtui play "<query>" --lang <code> --device <name> --json` searches, selects, subtitles, and starts playback without prompts; when finishing without a visible Chromecast check, `streamtui play "<query>" --lang <code> --vlc --json` provides the same workflow through VLC.

Stop when: Phase 6 in `docs/cli-rewrite-plan.md` passes its automated tests and the agent-first VLC workflow is implemented and verified without prompts.

Trace the existing Rust implementation for provider behavior, parsing rules, ranking rules, and external command shapes.
Build the Go CLI in phases that match `docs/cli-rewrite-plan.md`.
Start with Phase 0: create `go.mod`, define the command tree, implement the v1 JSON envelope, map errors to exit codes, and add contract tests.
Keep side effects inside adapters for HTTP, filesystem cache, `webtorrent`, and `catt`.
Use fake HTTP servers and fake subprocess binaries in tests so the suite stays fast and deterministic.
Record each completed phase with the tests that prove it.
Pause at Gate A, Gate B, and Gate C with the exact manual commands to run and the observed result.
