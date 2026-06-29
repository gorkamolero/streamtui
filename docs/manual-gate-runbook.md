# Manual Gate Runbook

This file captures the manual evidence needed to finish `docs/directional-rewrite-goal.md`.

## Gate B - Playback Start Paths

### VLC Path

Status: passed.

Command:

```bash
go run ./cmd/streamtui --json cast-magnet '<WebTorrent Sintel magnet>' --vlc
```

Passing evidence:
- JSON envelope has `ok: true`.
- `data.player` is `VLC`.
- `data.status` is `playing`.

Observed result:
- Passed with WebTorrent's official Sintel test magnet.

### Chromecast Path

Status: blocked until `catt scan` finds a device.

Prerequisites:
- Mac and Chromecast are on the same local network.
- Chromecast is powered on and not isolated by guest Wi-Fi, VPN, or client isolation.
- `catt scan` prints at least one device line.
- If `catt scan` finds no devices, use `dns-sd -B _googlecast._tcp local.` as a lower-level mDNS check; no `_googlecast._tcp` entries means the machine is not seeing Chromecast advertisements.

Commands:

```bash
catt scan
go run ./cmd/streamtui --json devices refresh
go run ./cmd/streamtui --json cast-magnet '<WebTorrent Sintel magnet>' --device '<Chromecast name>'
```

Passing evidence:
- `devices refresh` returns `ok: true` and includes the target device.
- `cast-magnet` returns `ok: true`.
- `data.device` matches the requested Chromecast name.
- Chromecast visibly starts playback.

Current observed blocker:
- `catt scan` returns `Error: No devices found.`
- `go run ./cmd/streamtui --json devices refresh` returns `DEVICE_NOT_FOUND` with message `no Chromecast devices found`.

## Gate C - Agent-First Workflow

Status: blocked until Gate B Chromecast path is possible.

Command:

```bash
go run ./cmd/streamtui play "the batman" --lang es --device "<Chromecast name>" --json
```

Passing evidence:
- JSON envelope has `ok: true`.
- `data.status` is `playing`.
- `data.player` is `Chromecast`.
- `data.device` matches the requested Chromecast name.
- `data.title` is populated.
- `data.subtitle_file` is populated for Spanish subtitles.
- Chromecast visibly starts playback without prompts.

Failure evidence that does not complete the goal:
- `DEVICE_NOT_FOUND` from `devices refresh`.
- `catt scan` prints `Error: No devices found.`
- A JSON `ok: true` response without visible Chromecast playback.
