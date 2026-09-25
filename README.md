# fapi

An API proxy for testing how a frontend handles API errors, without changing any production code.

fapi sits between your frontend and your real API. It forwards every request to the real API, except for the endpoints you choose to override: those get the response and payloadyou set. Everything else keeps working, so only the endpoint you're testing changes.

It's one executable with a web UI and a terminal UI, and everything runs and stays on your own machine.

## Features

- Forward requests to your real API, except for overridden endpoints.
- Override endpoints with custom responses and payloads.
- Work with multiple frontend and backend setups simultaneously.
- Save and reuse custom payloads for endpoints.
- Control which endpoints and proxies are active.

## Install

| System | How |
|---|---|
| Windows | Download the `.zip`, unzip it and put `fapi.exe` on your `PATH`. The first time, SmartScreen warns that it's unsigned: click **More info**, then **Run anyway**. |
| Debian, Ubuntu | `sudo apt install ./fapi_*.*.*_linux_amd64.deb`. This also installs and starts the `fapi` service. |
| Fedora, RHEL | `sudo dnf install ./fapi_*.*.*_linux_amd64.rpm`. This also installs and starts the `fapi` service. |
| Docker | See [Docker](#docker). |
| Anything else | [Build it from source](#building-from-source). |

Releases come in three editions, which differ only in their user interfaces:

| Edition | Web UI | Terminal UI |
|---|---|---|
| `fapi` | ✓ | ✓ |
| `fapi-web` | ✓ | |
| `fapi-cli` | | ✓ |

## Run

With the `.deb` or `.rpm`, fapi already runs as a service, so just open the web UI or run `fapi`.

```bash
fapi serve     # run fapi; the web UI is at http://127.0.0.1:3100
fapi           # open the terminal UI (it can also start fapi for you)
```

## Use

1. **Add a proxy**: a fapi port (such as `3001`) and the port/host your real API runs on. Requests to fapi's port are forwarded to your API. If the API isn't on your machine, tick **The real API is on another host** and enter its host, such as `https://api.example.com` (port `443`).
2. **Point your frontend** at `http://127.0.0.1:3001` instead of your API.
3. **Override an endpoint**: choose the port, method, path (such as `/api/users/:id`) and the status and JSON body to return. Matching requests get that response instead of reaching your API.
4. **Watch the requests**: each endpoint shows the requests it answered, with their payloads, live.

Each endpoint and proxy has an **On** switch. An endpoint that's off lets its requests through to the proxy; a proxy that's off stops forwarding, so its port's other requests get a 404. Both are kept, ready to switch back on.

On the **Payloads** screen you can save responses you use often (a name, a status and a JSON body), then choose one when overriding an endpoint instead of typing the JSON again. Changing or deleting a payload doesn't change endpoints already added from it.

The **Endpoints on** switch, at the foot of the sidebar, turns every override off at once, so all requests reach your real API again.

How paths match:

- A segment starting with `:` matches anything: `/api/users/:id` matches `/api/users/42`.
- A trailing slash makes a different path: `/api/users/` doesn't match `/api/users`.
- If several endpoints match, the first one added wins.

A few more things to know:

- **You don't need a real API.** A port with overrides but no real API answers the overrides and returns 404 for everything else.
- **Frontends on another origin work.** fapi answers browser CORS checks itself.
- **recommended but not required to use `127.0.0.1`, not `localhost`.** 

Your endpoints and payloads are saved in `mocks.json`, in fapi's data folder: `~/.config/fapi` on Linux, `%AppData%\fapi` on Windows, or `/var/lib/fapi` for the service. To change ports, turn features off or keep the request log across restarts, see [docs/configuration.md](docs/configuration.md).

## The Linux service

The `.deb` and `.rpm` install fapi to `/opt/fapi` and run it as the `fapi` systemd service:

```bash
systemctl status fapi
sudo systemctl restart fapi
journalctl -u fapi
sudo apt remove fapi
```

Uninstalling keeps your endpoints in `/var/lib/fapi`. To upgrade, install a newer package the same way.

## Docker

Docker only supports the `fapi-web` edition. 
While fapi runs in docker support is limited at the moment due to the many limitations of containerized networking and port management.

```bash
docker run --rm \
  -p 127.0.0.1:3100:3100 \
  -p 127.0.0.1:3001-3010:3001-3010 \
  -v fapi-data:/data \
  --add-host=host.docker.internal:host-gateway \
  ghcr.io/mjq7/fapi:latest
```

For your own settings, see [docs/configuration.md](docs/configuration.md#docker).

## Building from source

You need [Go](https://go.dev/dl/) (version in `go.mod`), and [Node.js](https://nodejs.org) 22 or later for the web UI.

The easiest way is the build script, which starts from scratch each time (it reinstalls the web UI's packages) and puts every edition in `bin/`:

```bash
scripts/build.sh # specify the edition if needed, e.g., fapi-cli, fapi-web
```

It builds for the system it runs on, so run it in Git Bash on Windows for `.exe` files. To build by hand instead:

```bash
cd web && npm install && npm run build && cd ..
go build ./cmd/fapi                   # fapi: both UIs
go build -tags notui ./cmd/fapi       # fapi-web: no terminal UI
go build -tags nowebui ./cmd/fapi     # fapi-cli: no web UI, and no Node.js needed
```

To run it as a service on another Linux system with systemd, do what the packages do:

```bash
sudo install -D -m 755 fapi /opt/fapi/fapi
sudo ln -sf /opt/fapi/fapi /usr/local/bin/fapi
sudo install -m 644 packaging/fapi.service /etc/systemd/system/fapi.service
sudo systemctl daemon-reload && sudo systemctl enable --now fapi
```

To uninstall, run `sudo systemctl disable --now fapi`, then delete `/opt/fapi`, `/usr/local/bin/fapi` and the service file.

## License

[MIT](LICENSE)
