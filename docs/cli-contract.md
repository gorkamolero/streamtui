# StreamTUI CLI Contract v1

## Output Envelope
All JSON responses use this shape:

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

On failure:

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

## Exit Codes
- `0`: success
- `1`: generic error
- `2`: invalid args
- `3`: network/provider error
- `4`: device not found
- `5`: no streams/subtitles found
- `6`: cast/playback failed

## Non-Interactive Rules
- In JSON mode, commands must never prompt.
- If required data is missing, return structured error + exit code.
- Agent-safe defaults should avoid asking for input where possible.

## Core Commands
- `search <query>`
- `trending`
- `info <id>`
- `streams <imdb_id> [--season --episode]`
- `subtitles <imdb_id> [--lang] [--season --episode]`
- `devices list`
- `devices refresh`
- `devices default set <name>`
- `devices default get`
- `cast <imdb_id> [--device] [--lang] [--subtitle-id] [--vlc]`
- `cast-magnet <magnet> [--device] [--subtitle-file] [--vlc]`
- `status`, `play`, `pause`, `stop`, `seek`, `volume`
- `play "<query>" [--lang] [--device]`

## Device Resolution
- `--device` takes priority.
- Else use saved default.
- Else use last successful cached device.
- Else use cached device list if fresh.
- Else run discovery.

## Agent Workflow Target
A single command should support:

```bash
streamtui play "the batman" --lang es --device "Living Room TV" --json
```

Behavior:
- Picks best search match.
- Selects best stream based on deterministic ranking.
- Picks top Spanish subtitle.
- Starts casting without interactive prompts.

