# Design Doc: GChat Bridge Systemd Service

## Status
- **Issue**: ga-puh
- **Status**: Draft
- **Author**: nux (gastown polecat)

## Overview
Moving the GChat bridge (`gchat_adapter/adapter.sh`) to a standalone systemd service to improve robustness, responsiveness, and extensibility.

## Goals
1. **Robustness**: Independent of Gas Town rigs/agents.
2. **Responsiveness**: Top-level `gchat subscribe` for inbound messages.
3. **Extensibility**: Enable slash-commands (e.g., `/gt status`, `/gcert`) via the bridge.
4. **Visibility**: Standard logs via `journalctl`.

## Current State
The GChat bridge is a bash script (`gchat_adapter/adapter.sh`) that:
- Polls/Subscribes to GChat space `AAQAcAzTWoU`.
- Syncs inbound GChat messages to Gas Town Mail (as `overseer/`).
- Syncs outbound Gas Town Mail (from `overseer/`) to GChat.
- Runs as a background process or via manual invocation.

## Proposed Design

### 1. Service Unit Definition
A systemd user unit (`gchat-bridge.service`) will be created to manage the bridge process.

```ini
[Unit]
Description=Gas Town GChat Bridge
After=network.target

[Service]
Type=simple
ExecStart={{.TownRoot}}/gchat_adapter/adapter.sh --run
WorkingDirectory={{.TownRoot}}/gchat_adapter
Restart=always
RestartSec=10s
Environment="GT_TOWN_ROOT={{.TownRoot}}"
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=default.target
```

### 2. Integration with `gt` CLI
The `gt` CLI will be updated to support provisioning and managing the GChat bridge service.

- `gt daemon enable-gchat-bridge`: Provisions and enables the systemd service.
- `gt daemon disable-gchat-bridge`: Disables and removes the systemd service.

### 3. Security and Permissions
The service will run as a **systemd user unit** (`systemctl --user`), inheriting the user's environment and Loas2 credentials (`gcert`). This ensures the bridge can authenticate with both GChat (via the `gchat` binary) and Gas Town Mail (via the `gt` binary).

### 4. Interaction with `gt mail`
The bridge continues to act as a relay:
- **Inbound**: Maps GChat messages to `gt mail send overseer/`.
- **Outbound**: Polls `gt mail inbox --identity overseer/` and sends to GChat.

### 5. Extensibility (Slash Commands)
The bridge will be updated to recognize commands prefixed with `/` in GChat.
- `/gt status`: Returns the output of `gt status`.
- `/gcert`: Returns `gcertstatus` information.
- `/bd ready`: Returns a summary of ready issues.

These commands will be executed by the bridge and the output sent back to the GChat thread.

## Implementation Plan

### Phase 1: Service Template
- Add `internal/templates/systemd/gchat-bridge.service` template.
- Update `internal/templates/templates.go` to support provisioning this new service.

### Phase 2: CLI Commands
- Add `gt daemon enable-gchat-bridge` command.
- Add `gt daemon disable-gchat-bridge` command.

### Phase 3: Bridge Logic Updates
- Update `adapter.sh` (or rewrite in Go) to support slash commands.
- Improve error handling and logging for systemd compatibility.

### Phase 4: Verification
- Enable the service and verify inbound/outbound sync.
- Test slash commands in the GChat space.
