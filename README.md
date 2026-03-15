# Sentinel, the one that keeps guard.

TUI service dashboard to manage and monitor services operations and live status.

## Demo


## Highlights

- Multi-backend service cards for `docker`, `systemd`, and `k8s` (deployment-based)
- Adaptive, scrollable services grid with per-card focus and state coloring
- Focus navigation across panels with arrows and Vim motions (`hjkl`)
- Panel/content focus model (`enter` to enter panel content, `esc` to exit content focus)
- Service filtering by type and state (`All`, `Running`, `Degraded`, `Stopped`, `Inactive`)
- Theme system with persistent selection (`t` next, `Shift+T` previous)
- Persistent workspace name editing from UI
- Persistent `KUBECONFIG` path editing from UI
- Add service modal with validation and YAML persistence
- Delete service confirmation modal with YAML node-level delete by service `Id`
- Service actions from cards:
  - `space`: start
  - `s`: stop
  - `r`: restart
- Logs preview widget with viewport scrolling for selected service
- Real-time refresh tick loop for runtime metrics/status updates

## Installation

### Recommended setup (Makefile)

```bash
make init
make run
```

For better service-action compatibility (`docker`/`systemd`), it is recommended to run with elevated privileges:

```bash
sudo make run
```

### Build from source (Go)

Requires Go (`go 1.25+` in `go.mod`):

```bash
go build -o bin/sentinel ./cmd/sentinel
./bin/sentinel
```

### Run directly

```bash
go run ./cmd/sentinel/main.go
```

## Prerequisites

<details>
<summary>Click to expand</summary>

- **Go**: required to build/run from source
- **Linux**: current MVP target environment
- **Docker daemon**: required for Docker metrics/actions/logs
- **systemd + journald**: required for systemd metrics/actions/logs
- **Kubernetes cluster + metrics-server**: required for k8s runtime metrics
- **KUBECONFIG access**: required for k8s backend when running outside cluster

</details>

## Quick Start

```bash
make init
sudo make run
```

## Configuration

Sentinel reads services and settings from:

- `internal/config/config.yaml`

Current config model includes:

- `Version`
- `Settings.Polling.Interval`
- `Settings.Workspace.Name`
- `Services[]` with explicit `Id`, `Name`, `Type`, and backend-specific blocks

For `k8s`, MVP uses deployment-driven config:

- `K8s.Context`
- `K8s.Namespace`
- `K8s.Deployment`

## Data Storage

Sentinel currently stores runtime configuration in local files.

| Data | Location |
|---|---|
| Main app config | `internal/config/config.yaml` |
| Theme selection | `${XDG_CONFIG_HOME}/sentinel/theme.json` or `~/.config/sentinel/theme.json` |
| Optional env values (for kubeconfig flow) | `./.env` |

## Keybindings

- `↑↓←→` / `hjkl`: move focus/cursor
- `enter`: enter panel content / apply selection
- `esc`: exit panel content
- `c`: clear filters
- `a`: add service
- `d`: delete selected service (with confirmation)
- `space`: start selected service
- `s`: stop selected service
- `r`: restart selected service
- `t`: next theme
- `Shift+T`: previous theme
- `q` / `ctrl+c`: quit

## Notes

- k8s logs/metrics depend on deployment health and image availability.
- Logs widget is preview-oriented; full external log pane behavior can be extended.

## License

MIT License. See [`LICENSE`](LICENSE).
