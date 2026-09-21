# pixera-mcp

A cross-platform bridge between the **[Pixera Native API](https://help.pixera.one/api)**
(AV Stumpfl) and the **Model Context Protocol (MCP)**, so an AI client can
control a Pixera media server — drive timeline transport, trigger cues, manage
screens and projects, and reach any of Pixera's ~250 API methods.

Written in Go. Single self-contained binary (the admin console is embedded) for
Windows, macOS, and Linux on amd64 and arm64. Designed to run alongside Pixera
and expose MCP to your network, with a small web console to repoint it at a
different Pixera server on the fly.

---

## What it does

The Pixera Native API is **JSON-RPC 2.0 over a TCP socket** (Pixera is the
server; default port **1400**, using the `0xPX`-delimited "TCP(dl)" framing).
`pixera-mcp` connects to it and:

- exposes **curated tools** for timeline transport, cues, screens, project
  load/save and layer/parameter control,
- exposes a generic **`pixera_call`** tool for any other API method,
- serves MCP over **Streamable HTTP on all interfaces** by default,
- runs an **admin console** on a second port to view status (including the live
  API revision) and **repoint** at a different Pixera server without a restart.

> Enable the API in Pixera under **Settings > API** and note the TCP(dl) port.

---

## Install

### Windows (PowerShell) — e.g. on the Pixera server

```powershell
irm https://raw.githubusercontent.com/medcelerate/pixera-mcp/main/scripts/install.ps1 | iex
```

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/medcelerate/pixera-mcp/main/scripts/install.sh | sh
```

Or download a binary from the
[Releases](https://github.com/medcelerate/pixera-mcp/releases) page, or build
from source: `go install github.com/medcelerate/pixera-mcp/cmd/pixera-mcp@latest`.

---

## Quick start

```bash
cp config.example.yaml config.yaml   # set the Pixera host/port
pixera-mcp --config config.yaml
```

Defaults: connects to Pixera at `127.0.0.1:1400`, serves **MCP over HTTP on
`0.0.0.0:8095`**, and opens the **admin console on `0.0.0.0:8096`**.

### Connecting an MCP client

**Remote (Streamable HTTP)** — point Claude (custom connector) or any MCP client
at `http://<host>:8095`.

**Local (stdio)** — set `mcp.transport: stdio` and add to a desktop MCP client:

```json
{ "mcpServers": { "pixera": { "command": "pixera-mcp", "args": ["--config", "C:\\path\\to\\config.yaml"] } } }
```

> Logs go to **stderr** so they never corrupt the stdio MCP stream on stdout.

---

## Configuration

See [`config.example.yaml`](config.example.yaml). Key fields:

| Field | Meaning |
|-------|---------|
| `pixera.host` / `pixera.port` | The Pixera server's Native API (default `127.0.0.1:1400`) |
| `mcp.transport` | `http`, `stdio`, or `both` |
| `mcp.http.addr` | MCP HTTP bind address (default `0.0.0.0:8095`) |
| `web.enabled` / `web.addr` | Admin console (default `0.0.0.0:8096`) |

Overridable via env: `PIXERAMCP_HOST`, `PIXERAMCP_PORT`,
`PIXERAMCP_MCP_TRANSPORT`, `PIXERAMCP_MCP_HTTP_ADDR`, `PIXERAMCP_WEB_ADDR`,
`PIXERAMCP_WEB_ENABLED`, `PIXERAMCP_LOG_LEVEL`, `PIXERAMCP_CONFIG`.

### Admin console

At `web.addr` (default `http://<ip>:8096`) you can see the target, whether
Pixera is reachable, its API revision, and **repoint** at a different host/port.

---

## MCP tools

**Control:** `pixera_status`, `pixera_set_target`, `pixera_get_api_revision`,
`pixera_call` (any method by name + JSON params).

**Timeline:** `pixera_timeline_play` / `pause` / `stop`,
`pixera_timeline_transport_by_index`, `pixera_start_first_timeline` (+ pause/stop),
`pixera_timeline_get_transport_mode`, `pixera_timeline_get_time`,
`pixera_timeline_goto_seconds`, `pixera_timeline_apply_cue_number` /
`apply_cue_name`, `pixera_timeline_set_opacity`.

**Scene:** `pixera_list_screens`, `pixera_save_project` / `save_project_as` /
`load_project`, `pixera_layer_transport`, `pixera_assign_resource_to_layer`,
`pixera_set_param` / `get_param`.

Curated tools use Pixera's name/index-addressed `Compound`, `Session` and
`Screens` methods (no handles needed). For handle-based or less common methods,
use `pixera_call` with the fully-qualified method name and a JSON params object.

All tools carry titles and read-only / write annotations.

## Use it as a connector

The bridge speaks MCP, so it plugs into AI clients as a connector.

**Claude Desktop extension:** build a one-click `.mcpb` and open it in Claude
Desktop:

```bash
sh scripts/build-mcpb.sh   # dist/pixera-mcp-<version>-universal.mcpb
```

One bundle runs on macOS (Intel + Apple Silicon), Windows, and Linux; set the
Pixera **Host/Port** in the extension settings. Pre-built bundles are attached to
each [release](https://github.com/medcelerate/pixera-mcp/releases).

**Remote connector (Claude / ChatGPT):** run with `mcp.transport: http` and point
the client at `http://<host>:8095` (behind TLS + auth if exposed).

## Security

The MCP endpoint and admin console bind to **all interfaces** by default because
the bridge is meant to run on a show network. There is **no built-in
authentication** — run it only on a trusted, isolated production network, and set
the bind addresses to `127.0.0.1` to restrict to the local machine.

## Building from source

```bash
go build ./...   # compile
go test ./...    # run the tests
```

## Privacy Policy

`pixera-mcp` runs on your own infrastructure and talks only to the Pixera server
you configure. It sends no data to the author, Anthropic, OpenAI, or any third
party, and includes no telemetry. See [PRIVACY.md](PRIVACY.md).

## License

GPL-3.0 — see [LICENSE](LICENSE).
