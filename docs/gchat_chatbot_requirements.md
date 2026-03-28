# Gas Town Mail GChat Chatbot: Requirements & Design

This document outlines the requirements and design for upgrading the Gas Town GChat bridge to a dedicated chatbot identity.

## 1. Overview
The **Gas Town Mail Chatbot** will serve as a dedicated identity for relaying messages between Google Chat spaces and the Gas Town Mail system. It replaces the current user-centric bridge which relies on ambient user credentials (EUC).

## 2. Infrastructure Requirements (Overseer Tasks)

The following infrastructure must be provisioned before implementation can begin:

1.  **Google Cloud Project (GCP)**:
    *   Create a dedicated consumer GCP project.
    *   Enable **Google Chat API**.
    *   Set **App Status** to "LIVE" in the Configuration tab.
2.  **Bot Identity**:
    *   **Borg Role** (e.g., `gastown-bot`): Recommended for internal Stubby access and LOAS2 auth.
    *   Alternatively, a **Service Account** if using REST API exclusively.
3.  **Bot Configuration**:
    *   **Name**: "Gas Town Mail"
    *   **Avatar**: Dedicated icon (use `go/icons` as a source).
    *   **Space Membership**: The bot identity must be added as a member to the target GChat space (`AAQAcAzTWoU`).

## 3. Data Interaction Matrix

| Data | Corp Service | Operation | Reason |
| :--- | :--- | :--- | :--- |
| Mail Messages | Gas Town Mail | READ / WRITE | Core relay functionality. |
| Beads Issues | Gas Town Beads | READ / WRITE | Support for issue queries/updates via Chat. |
| Chat Messages | Google Chat API | READ / WRITE | Sending and receiving messages in spaces. |

## 4. Compliance & Approvals

Since Gas Town Mail contains "Confidential-Google" data (mail, beads), the following compliance steps are mandatory:

1.  **Corp App Approval Process**: Follow the guide at `go/corpbot-approval-process`.
2.  **Design Review**: Create a formal design doc using the `go/corpbot-design-doc` template.
3.  **Security & Privacy Review**: File a review at `go/securityreview`.
4.  **Stubby Access**: Use **Stubby** for all API calls to ensure proper auditing and access control.

## 5. Dual-Mode Transition Plan

To ensure no service interruption, we will implement a dual-mode phase:

1.  **Tagging**: Bot messages will be tagged with a hidden metadata field (if supported) or a specific prefix to allow loop prevention.
2.  **Loop Prevention**: The adapter MUST ignore messages where `sender.type == "BOT"` to avoid infinite message loops.
3.  **Simultaneous Sync**: For a verification period, the bridge will send messages via BOTH the legacy user identity and the new bot identity.
4.  **Cutover**: Once verified, legacy sending will be disabled.

## 6. Adapter Changes (`gchat_adapter/adapter.sh`)

*   **Authentication**: Support `BOT_IDENTITY` or `BOT_ROLE` environment variables.
*   **Logic**: Conditionally call the `gchat` binary with bot authority when available.
*   **Logging**: Enhanced logging to distinguish between user-sent and bot-sent messages.
