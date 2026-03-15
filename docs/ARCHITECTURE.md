# Architecture

## Overview

Sentinel is a Go terminal application built with Bubble Tea, Docker SDK, k8s.io pkg and go-systemd pkg.
It loads service definitions from YAML, gathers runtime data from backend adapters(`docker`, `systemd`, `k8s`), and renders a panel-based TUI with a service grid, filters, actions, and logs preview.

## Runtime Flow

1. `cmd/sentinel/main.go` creates config/model dependencies and reads `internal/config/config.yaml`.
2. `tui.InitialModel(...)` receives loaded services, initializes theme/workspace/kubeconfig input state, and starts UI model state.
3. `Init()` performs first metrics snapshot per service and starts periodic tick command.
4. On each tick:
   - service runtime data is refreshed (`refreshCard`)
   - logs preview for selected service is refreshed (`refreshLogsPreview`)
5. Bubble Tea renders updated side panels + services grid in alt screen.

## Project Structure

- `cmd/sentinel/main.go`: app bootstrap and Bubble Tea program start.
- `internal/ui/model.go`: main TUI state machine, input handling, rendering, add/delete modals, actions, logs preview.
- `internal/ui/themes/themes.go`: palette definitions and persisted theme selection.
- `internal/config/yaml.go`: YAML schema, load, workspace update, add service, delete service.
- `internal/config/config.yaml`: runtime config and service definitions.
- `internal/backend/docker/docker.go`: Docker runtime metrics, start/stop/restart, logs fetch.
- `internal/backend/systemd/systemd.go`: systemd runtime metrics, start/stop/restart, journal logs fetch.
- `internal/backend/k8s/kubernetes.go`: deployment runtime metrics, start/stop/restart, pod/deployment logs fetch.
- `internal/model/struct.go`: shared runtime view model (`ServiceRuntime`).
- `internal/util/helpers.go`: formatting/helpers used by UI and backends.

## Main Functional Flows

### Service Runtime Update

1. UI tick fires with configured interval.
2. For each configured service, backend-specific metrics/status are collected.
3. Results are stored in `runtimeByID` and card text content is rebuilt.
4. Grid view reflects status/color/state per card.

### Add Service

1. User opens Add Service modal (`a`).
2. User selects type and fills required fields.
3. Validation enforces required fields and unique `Id`.
4. New service is appended in YAML and in-memory state.
5. UI refreshes cards.

### Delete Service

1. User focuses a service card and presses `d`.
2. Confirmation modal asks `Yes/No` for selected target.
3. On `Yes`, service node is removed from YAML `Services` list by `Id`.
4. In-memory `services` and `runtimeByID` are updated, then UI refreshes.

### Service Actions (Start/Stop/Restart)

1. User focuses services content and selects a card.
2. Keybindings dispatch backend-specific action:
   - `space`: start
   - `s`: stop
   - `r`: restart
3. Errors are written to the selected card runtime error field.
4. On success, runtime refresh updates the card state.

### Logs Preview

1. Logs panel is a viewport widget in the side column.
2. Logs are fetched for currently selected/filtered service.
3. Backend mapping:
   - Docker: container logs (tail)
   - systemd: journal unit logs (tail window)
   - k8s: deployment -> active pod -> pod logs (tail)
4. User can focus logs panel and scroll with `j/k` or arrows.

## UI State Model

- Focus areas:
  - Services
  - Workspace
  - Types
  - Filters
  - Kubeconfig
  - Logs
- Content focus:
  - `enter` toggles interaction mode for list/grid/viewport panels.
  - `esc` exits content focus.
- Theme switching:
  - `t` next theme
  - `Shift+T` previous theme
  - selection persisted to user config path.

## Notes and Constraints

- Linux-first MVP.
- Docker and systemd actions may need elevated permissions.
- k8s metrics require metrics-server; image pull failures are surfaced in card errors.
- Current k8s model is deployment-centric (services define deployment, runtime resolves pods).
- Remote multi-host connection is deferred for later phases.
