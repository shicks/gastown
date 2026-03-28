#!/usr/bin/env bash
# GChat <-> Gas Town Mail Adapter v6.0 (Dynamic Catch-up)

set -euo pipefail

GCHAT="/google/bin/releases/gemini-agents-gchat/gchat"
SPACE_ID="AAQAcAzTWoU" # sdh gastown
STATE_DIR="/usr/local/google/home/sdh/gt/gchat_adapter"
STATE_FILE="$STATE_DIR/adapter_state.json"

GT="/usr/local/google/home/sdh/.local/bin/gt"
BD="/usr/local/google/home/sdh/.local/bin/bd"

log() {
  echo "[$(date +'%Y-%m-%dT%H:%M:%S%z')] $*"
}

handle_command() {
  local id="$1"
  local thread_id="$2"
  local text="$3"
  
  if [[ "$text" == "/gt status"* ]]; then
    log "Executing command: /gt status"
    local output=$(unset GT_RIG GT_POLECAT GT_MEMBER && $GT status 2>&1 || echo "Error: status failed")
    $GCHAT send-message --space "$SPACE_ID" --thread "$thread_id" --text "🤖 *GT Status:*
$output"
  elif [[ "$text" == "/gcert"* ]]; then
    log "Executing command: /gcert"
    local output=$(gcertstatus 2>&1 || echo "Error: gcertstatus failed")
    $GCHAT send-message --space "$SPACE_ID" --thread "$thread_id" --text "🤖 *Gcert Status:*
$output"
  elif [[ "$text" == "/bd ready"* ]]; then
    log "Executing command: /bd ready"
    local output=$(unset GT_RIG GT_POLECAT GT_MEMBER && $BD ready --short 2>&1 || echo "Error: bd ready failed")
    $GCHAT send-message --space "$SPACE_ID" --thread "$thread_id" --text "🤖 *BD Ready:*
$output"
  fi
}

check_certs() {
  if ! gcertstatus --format=simple -check_loas2=true -check_ssh=false >/dev/null 2>&1; then
    return 1
  fi
  return 0
}

save_state_key() {
  local key=$1
  local val=$2
  local tmp=$(mktemp)
  jq --arg v "$val" "$key = \$v" "$STATE_FILE" > "$tmp" && mv "$tmp" "$STATE_FILE"
}

# --- GChat -> Mail (Inbound) ---

process_json_message() {
  local msg="$1"
  [ -z "$msg" ] && return

  local name=$(echo "$msg" | jq -r '.name')
  local id=$(basename "$name")
  local text=$(echo "$msg" | jq -r '.text // ""')
  local create_time=$(echo "$msg" | jq -r '.create_time')
  local sender_type=$(echo "$msg" | jq -r '.sender.type')
  local thread_name=$(echo "$msg" | jq -r '.thread.name')
  local thread_id=$(basename "$thread_name")

  if [[ "$text" == *"📬 *Mail from"* ]] || [ "$sender_type" == "BOT" ]; then
    save_state_key ".last_gchat_id" "$id"
    save_state_key ".last_gchat_time" "$create_time"
    return
  fi

  if [[ "$text" == "/"* ]]; then
    handle_command "$id" "$thread_id" "$text"
    save_state_key ".last_gchat_id" "$id"
    save_state_key ".last_gchat_time" "$create_time"
    return
  fi

  log "Syncing inbound GChat message $id"

  local initiator_msg=$( $GCHAT read-thread --json --space "$SPACE_ID" --thread "$thread_id" | jq -c 'sort_by(.create_time) | .[0]' )
  local initiator_text=$( echo "$initiator_msg" | jq -r '.text // ""' )
  
  local target_identity="mayor/"
  if [[ "$initiator_text" =~ ^"📬 *Mail from "([^[:space:]*]+) ]]; then
    target_identity="${BASH_REMATCH[1]}"
  fi

  local mail_id=""
  if [[ "$initiator_text" =~ \*Mail-ID:\*[[:space:]]*([^[:space:]\n]+) ]]; then
    mail_id="${BASH_REMATCH[1]}"
  fi

  local initiator_subject=""
  if [[ "$initiator_text" =~ \*Subject:\*[[:space:]]*([^\n]+) ]]; then
    initiator_subject="${BASH_REMATCH[1]}"
  fi

  local body="GChat-ID: $id
Thread: $thread_name

$text"
  
  local subject="💬 GChat"
  if [ -n "$initiator_subject" ]; then
    local s="${initiator_subject#Re: }"
    subject="Re: $s"
  fi

  local extra_flags=""
  if [ -n "$mail_id" ]; then
    extra_flags="--reply-to $mail_id --type reply"
  fi

  unset GT_RIG GT_POLECAT GT_MEMBER
  GT_ROLE="overseer/" gt mail send "$target_identity" -s "$subject" -m "$body" $extra_flags
  save_state_key ".last_gchat_id" "$id"
  save_state_key ".last_gchat_time" "$create_time"
}

sync_inbound_catchup() {
  if ! check_certs; then return; fi
  
  local last_id=$(jq -r '.last_gchat_id // ""' "$STATE_FILE")
  local last_time=$(jq -r '.last_gchat_time // ""' "$STATE_FILE")
  
  local hours=24
  if [ -n "$last_time" ] && [ "$last_time" != "null" ] && [ "$last_time" != "" ]; then
    local now_s=$(date +%s)
    local last_s=$(date -d "$last_time" +%s)
    local diff_s=$((now_s - last_s))
    hours=$(( (diff_s / 3600) + 1 ))
    [ $hours -lt 1 ] && hours=1
  fi

  log "Starting catch-up (window: $hours hours)..."
  local all_messages=$( $GCHAT list-messages --space "$SPACE_ID" --hours "$hours" --json )
  
  echo "$all_messages" | jq -c --arg last "$last_id" 'sort_by(.create_time) | . as $list | (map(.name | split("/") | last) | index($last) // -1) as $idx | $list[$idx+1:] | .[]' | while read -r msg; do
    [ -z "$msg" ] && continue
    process_json_message "$msg"
  done
}

# --- Mail -> GChat (Outbound) ---

sync_outbound() {
  if ! check_certs; then return; fi
  
  local last_mail_id=$(jq -r '.last_mail_id // ""' "$STATE_FILE")
  local mail=$(gt mail inbox --identity "overseer/" --json --all | jq 'sort_by(.timestamp)')
  
  local found_last=false
  if [ -z "$last_mail_id" ] || [ "$last_mail_id" == "null" ]; then found_last=true; fi

  if [ "$found_last" = false ] && ! echo "$mail" | jq -e --arg id "$last_mail_id" '.[] | select(.id == $id)' >/dev/null; then
     log "Warning: last_mail_id $last_mail_id lost. Advancing to newest."
     last_mail_id=$(echo "$mail" | jq -r '.[-1].id // ""')
     found_last=true
     save_state_key ".last_mail_id" "$last_mail_id"
  fi

  echo "$mail" | jq -c '.[]' | while read -r m; do
    [ -z "$m" ] && continue
    local id=$(echo "$m" | jq -r '.id')
    
    if [ "$id" = "$last_mail_id" ]; then
      found_last=true
      continue
    fi

    if [ "$found_last" = true ]; then
      local from=$(echo "$m" | jq -r '.from')
      local subject=$(echo "$m" | jq -r '.subject')
      local body=$(echo "$m" | jq -r '.body')

      if [[ "$body" == "GChat-ID: "* ]]; then
        save_state_key ".last_mail_id" "$id"
        continue
      fi

      log "Syncing mail $id to GChat: [$from] $subject"
      local gchat_text="📬 *Mail from $from*
*Subject:* $subject
*Mail-ID:* $id

$body"
      
      env -u AGY_UNATTENDED $GCHAT send-message --space "$SPACE_ID" --text "$gchat_text"
      save_state_key ".last_mail_id" "$id"
    fi
  done
}

# --- Main Logic ---

SELF_PATH=$(realpath "$0")

case "${1:-}" in
  --process-message)
    process_json_message "$(cat)"
    ;;
  --sync-outbound)
    sync_outbound
    ;;
  --sync-inbound-catchup)
    sync_inbound_catchup
    ;;
  --run)
    log "Starting GChat Adapter v6.0 (Dynamic Catch-up)"
    if [[ "$(jq -r '.last_gchat_time // ""' "$STATE_FILE")" == "" ]] || [[ "$(jq -r '.last_gchat_time' "$STATE_FILE")" == "null" ]]; then
      newest_time=$( $GCHAT list-messages --space "$SPACE_ID" --max 1 --json | jq -r '.[0].create_time // ""' )
      [ -n "$newest_time" ] && save_state_key ".last_gchat_time" "$newest_time"
    fi

    "$SELF_PATH" --sync-inbound-catchup || true
    "$SELF_PATH" --sync-outbound || true
    ( while true; do "$SELF_PATH" --sync-outbound || true; sleep 2; done ) &
    ( while true; do sleep 3600; "$SELF_PATH" --sync-inbound-catchup || true; done ) &
    $GCHAT subscribe --space "$SPACE_ID" --interval 2 --command "$SELF_PATH --process-message"
    ;;
  *)
    echo "Usage: $0 {--process-message|--sync-outbound|--sync-inbound-catchup|--run}"
    exit 1
    ;;
esac
