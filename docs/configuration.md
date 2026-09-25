# Configuration

fapi works without any settings. This page describes every setting, for when you want to change how fapi behaves.

Settings are separate from your data. Your endpoints, pass-throughs, saved payloads and the on/off switch are data: you change them in the web UI, and fapi saves them to `mocks.json`. Settings control fapi itself, such as its ports and which features are on.

## How settings are chosen

fapi has **built-in defaults**, compiled into the executable.

An **override file** can change some of them. It contains only what it changes; everything else keeps its default. 

fapi uses the first of these that exists:

1. the file given with `fapi serve --config <file>`
2. the file named by the `FAPI_CONFIG` environment variable
3. `config.json` in the data folder
4. none: built-in defaults only

A file named with `--config` or `FAPI_CONFIG` that doesn't exist is an error. A missing `config.json` in the data folder isn't.

Settings are read when fapi starts. After changing them, restart fapi.

### Changing ports in the web UI

The web UI's **Settings** screen changes the port settings (`adminPort` and `mockPorts`) without editing the file by hand. It saves them to the override file fapi started with, or creates `config.json` in the data folder if there was none, and keeps every other setting in the file. Like any settings change, they take effect when fapi restarts; until then the screen shows both the ports in use and the saved ones.

The file must be writable by fapi. In the Docker image, the built-in settings file isn't, so mount your own (see [Docker](#docker)).

At startup, fapi logs which override file it used, or that it used none, and each setting that differs from the defaults:

```
Settings: built-in defaults, overridden by /home/you/.config/fapi/config.json
  features.cors: true → false
```

`GET /api/config` returns the settings in effect.

### Mistakes in the file

fapi refuses to start, naming the setting at fault, if the override file:

- isn't valid JSON (JSON allows no comments and no trailing commas)
- contains a setting that doesn't exist
- has a value of the wrong type, such as `"adminPort": "3100"`
- has a value out of range, such as a port above 65535

## Settings

| Setting | Default | What it does |
|---|---|---|
| `adminPort` | `3100` | Port of the web UI and admin API. |
| `listenAddress` | `"127.0.0.1"` | IPv4 address fapi listens on, for the admin port and every mock port. The default accepts connections from this machine only. `"0.0.0.0"` accepts them from anywhere, which Docker needs. |
| `mockPorts.min`, `mockPorts.max` | `1024`, `65535` | The range of ports endpoints and proxies may use. Narrowing it helps with Docker, where ports have to be published in advance. |
| `mockPorts.default` | `3001` | The fapi port the web UI and terminal UI suggest for a new endpoint or proxy. It must be in the range above. If a file narrows the range without setting it, and 3001 falls outside, the lowest allowed port is used. |
| `requestLog.maxEntries` | `200` | How many requests the request log keeps. The oldest are dropped first. |
| `requestLog.maxBodyBytes` | `4096` | Request bodies longer than this many bytes are cut in the request log, and marked with `…`. Forwarded requests are never cut. |
| `passThrough.host` | `"localhost"` | Host of the real API that unmatched requests are forwarded to, for proxies that don't name their own. A proxy can name its own host (such as `https://api.example.com`) with **The real API is on another host** on the Proxies screen, or the host field in the terminal UI. |

## Feature flags

Every optional feature can be turned off under `features`. Endpoints themselves and the admin API are always on.

| Flag | Default | What it turns on or off |
|---|---|---|
| `webUi` | on | Serving the web UI. The admin API stays available. |
| `passThrough` | on | Pass-through settings and forwarding unmatched requests to the real API. |
| `cors` | on | fapi's answers to browser CORS checks on mock ports, and CORS headers on their responses, so a frontend on another origin can call fapi. |
| `requestLog` | on | Recording requests, the web UI's **Requests** column and the request log's API routes. |
| `requestLogPersistence` | off | Saving the request log to `requests.json` in the data folder, so it survives restarts. |
| `liveUpdates` | on | Streaming new requests to the web UI as they arrive. When off, the web UI checks every 2 seconds instead. |

When a feature is off, its admin API routes answer 404 with `{"message": "<flag> is turned off in the fapi config"}`, and the web UI hides it.

Turning a feature off doesn't delete its saved data. For example, pass-throughs saved in `mocks.json` stay there, unused, until `passThrough` is turned back on.

## The built-in defaults

```json
{
  "adminPort": 3100,
  "listenAddress": "127.0.0.1",
  "mockPorts": { "min": 1024, "max": 65535, "default": 3001 },
  "features": {
    "webUi": true,
    "passThrough": true,
    "cors": true,
    "requestLog": true,
    "requestLogPersistence": false,
    "liveUpdates": true
  },
  "requestLog": { "maxEntries": 200, "maxBodyBytes": 4096 },
  "passThrough": { "host": "localhost" }
}
```

## The data folder

Holds `mocks.json`, the optional `config.json`, the server log `fapi.log` (emptied each time fapi starts) and, with `requestLogPersistence` on, `requests.json`.

It's `fapi` in your user config folder: `~/.config/fapi` on Linux, and `%AppData%\fapi` on Windows and maybe `~/Library/Application Support/fapi` on macOS (this hasnt been tested on macOS). 

Change it with `fapi serve --data-dir <folder>` or the `FAPI_DATA_DIR` environment variable.

`FAPI_CONFIG` and `FAPI_DATA_DIR` are the only environment variables fapi reads.

## Examples

Keep the request log between restarts, and keep more of it:

```json
{
  "features": { "requestLogPersistence": true },
  "requestLog": { "maxEntries": 1000 }
}
```

Move the admin port, and only allow mock ports 4000 to 4099:

```json
{
  "adminPort": 4100,
  "mockPorts": { "min": 4000, "max": 4099 }
}
```

Just the endpoints, with no web UI, CORS or request log:

```json
{
  "features": { "webUi": false, "cors": false, "requestLog": false }
}
```

## Docker

The Docker image sets `FAPI_CONFIG=/etc/fapi/config.json`, a file built into the image ([`packaging/docker-config.json`](../packaging/docker-config.json)):

```json
{
  "listenAddress": "0.0.0.0",
  "passThrough": { "host": "host.docker.internal" }
}
```

- `listenAddress` is `0.0.0.0` because inside a container, requests published by Docker don't arrive from `127.0.0.1`. The container's ports are still only reachable from outside through the ports you publish, so publish them on `127.0.0.1` (as in the README) to keep fapi private to your machine.
- `passThrough.host` is `host.docker.internal`, which is your machine as seen from the container, where your real API runs. On Linux, add `--add-host=host.docker.internal:host-gateway` to `docker run` for that name to work.

To use your own settings, mount a file and point `FAPI_CONFIG` at it.

Setting `mockPorts` to the range you publish means the web UI refuses ports that Docker wouldn't pass on.
