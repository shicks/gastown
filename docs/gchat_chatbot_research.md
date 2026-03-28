# Research: Upgrading GChat Bridge to Dedicated Chatbot (ga-49r)

## Current State Analysis

The GChat adapter (`gchat_adapter/adapter.sh`) currently functions as a relay between GChat and Gas Town Mail.

### Key Components:
- **Binary**: Uses `/google/bin/releases/gemini-agents-gchat/gchat`.
- **Authentication**: Ambient user credentials (EUC) via `gcert`/Loas2.
- **Workflow**:
    - **Inbound**: Polling/Subscribing to GChat space messages and sending as `overseer/` via `gt mail send`.
    - **Outbound**: Reading `overseer/` mail and sending to GChat via `gchat send-message`.
- **Identification**: Messages in GChat appear as sent by the user running the script, with a "📬 *Mail from...*" prefix in the text.

## Requirements for a Proper Chatbot

To upgrade to a dedicated chatbot identity, the following infrastructure and configuration are required:

### 1. Google Cloud Project (GCP)
- A consumer GCP project must be created (e.g., via Pantheon).
- **Google Chat API** must be enabled.
- **App Status** must be set to "LIVE" in the Configuration tab.

### 2. Identity & Authentication Models
There are two primary ways to authenticate the chatbot in the Google environment:

#### Option A: Borg Role (Recommended for Stubby/Internal)
- **Identity**: A dedicated Borg role (e.g., `gastown-bot`).
- **Auth**: Uses LOAS2 credentials automatically when running on Borg.
- **Pros**: Matches current `gchat` binary's Stubby implementation; easier access to corporate data; no credential rotation needed.
- **Cons**: Requires Borg role setup and `RpcSp` allowlisting.

#### Option B: Service Account (Recommended for REST/Cloud)
- **Identity**: A GCP Service Account.
- **Auth**: JSON key file or Workload Identity.
- **Pros**: Standard for Cloud-based apps; works with REST API.
- **Cons**: `gchat` binary currently lacks explicit support for JSON keys (would require code changes or a different client); credential rotation management.

### 3. Bot Configuration
- **Bot Name**: e.g., "Gas Town Mail".
- **Avatar**: A dedicated icon (can use placeholders from `go/icons`).
- **Authorization**: The bot (Borg role or Service Account) must be added as a member to the target GChat space (`AAQAcAzTWoU`).

### 4. Compliance (Corp App Approval)
- Since the bot accesses Gas Town (which contains corp data/services), it must follow the **Corp App Approval Process** (`go/corpbot-approval-process`).
- This involves:
    - Designing the app (Design Doc).
    - Security and Privacy review (`go/securityreview`).
    - Final Chat Platform review.

## Dual-Mode Transition Plan

To ensure a smooth transition, we will implement a "dual-mode" where the adapter can send via both the legacy user-centric method and the new chatbot method.

### Phase 1: Infrastructure Setup
- [ ] Overseer creates GCP project and service account/Borg role.
- [ ] Register the identity in the Google Chat API configuration.
- [ ] Add the Bot to the GChat space.

### Phase 2: Adapter Updates
- Update `adapter.sh` to support a `BOT_IDENTITY` or `BOT_CREDENTIALS` environment variable.
- Implement a switch to send via the Bot identity.

### Phase 3: Verification (Dual-Mode)
- Enable both methods simultaneously.
- **Loop Prevention**: Tag bot messages to allow the adapter to filter them out during inbound sync. The current script already filters `sender.type == "BOT"`.
- Verify bot messages appear with the correct name and avatar.

### Phase 4: Full Cutover
- Disable legacy user-centric outbound sync.
- Transition inbound sync to use the Bot's authority if appropriate.

## Proposed Changes to `gchat_adapter/adapter.sh`

### Variable Additions
```bash
# Toggle for dual-mode
USE_BOT=true
# Identity to use (if running as a different role)
# BOT_ROLE="gastown-bot"
```

### Outbound Sync Logic (Modified)
```bash
# In sync_outbound()
if [ "$USE_BOT" = true ]; then
  log "Syncing mail $id to GChat via BOT identity"
  # If running under a different role, we might use 'runas' or similar,
  # or ensure the cron job/service runs as the bot role.
  # env -u AGY_UNATTENDED $GCHAT send-message --space "$SPACE_ID" --text "$gchat_text"
fi

# Keep legacy for dual-mode
log "Syncing mail $id to GChat via USER identity"
env -u AGY_UNATTENDED $GCHAT send-message --space "$SPACE_ID" --text "$gchat_text"
```

## Security & Compliance Notes
- **No Impersonation**: The bot will send messages as itself, with the "Mail from [User]" info in the message body.
- **Access Control**: The identity should have the minimum necessary permissions.
- **Data Protection**: Ensure any stored credentials or keys are handled according to Google security standards.
