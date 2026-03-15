# Development

## Requirements

- Go 1.25+
- Linux (current MVP target)
- Docker CLI + Docker daemon (for Docker metrics/actions/logs)
- `systemctl` + journald/systemd (for systemd metrics/actions/logs)
- Kubernetes cluster + metrics-server (for k8s metrics)
- `kubectl` (recommended for cluster debugging)

## Quick Setup

```bash
make init
sudo make run
```

## Common Commands

```bash
make run      # run app
make build    # build ./bin/sentinel
make fmt      # go fmt ./...
make test     # go test ./...
make clean    # remove ./bin
make deps     # check optional runtime dependencies
```

## Configuration Workflow

- Main config file: `internal/config/config.yaml`
- Main config loader/writer: `internal/config/yaml.go`
- Workspace name is persisted by updating only the related YAML node.
- Service add/delete flows update only `Services` nodes in YAML.
- Theme selection is persisted in:
  - `${XDG_CONFIG_HOME}/sentinel/theme.json`
  - or `~/.config/sentinel/theme.json`

## Service Definitions (Current)

- Service key fields:
  - `Id`
  - `Name`
  - `Type`
- Type-specific config:
  - Docker: `Docker.Container`
  - systemd: `Systemd.Unit`
  - k8s: `K8s.Context`, `K8s.Namespace`, `K8s.Deployment`
- Polling interval:
  - `Settings.Polling.Interval`

## Runtime and Refresh Notes

- UI refresh uses `tea.Tick(...)` with interval from config.
- Tick updates call:
  - runtime metrics refresh (`refreshCard`)
  - logs preview refresh (`refreshLogsPreview`)
- Tick is re-armed on every `TickMsg` in `Update`.

## Logs and Debugging

- Logs preview panel is fed by:
  - Docker: `GetLogsFromContainer`
  - systemd: `GetUnitLogs`
  - k8s: `GetLogsFromDeployment`
- Docker and systemd log readers use near-tail behavior to limit payload.
- For k8s runtime/debugging, useful commands:

```bash
kubectl -n <namespace> get pods -l app=<deployment>
kubectl -n <namespace> describe pod <pod-name>
kubectl -n <namespace> logs <pod-name> --tail=200
```

## Known Platform Scope

- Primary support target today is Linux.
- k8s metrics depend on metrics-server availability.
- service actions may require elevated privileges (often run via `sudo make run`).
- Remote multi-host connection is intentionally deferred for post-MVP.
