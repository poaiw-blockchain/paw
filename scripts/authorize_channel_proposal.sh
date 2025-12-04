#!/usr/bin/env bash
set -euo pipefail

if ! command -v jq >/dev/null 2>&1; then
  echo "error: jq is required for this script" >&2
  exit 1
fi

if ! command -v pawd >/dev/null 2>&1; then
  echo "error: pawd binary not found in PATH" >&2
  exit 1
fi

if [[ $# -lt 5 ]]; then
  cat <<'EOF' >&2
Usage: authorize_channel_proposal.sh <module> <port-id> <channel-id> <authority-address> <from-key> [extra tx flags...]

Modules supported: dex, oracle, compute
Environment variables:
  DEPOSIT           Proposal deposit (default: 1000000upaw)
  PROPOSAL_FILE     Output file for proposal JSON (default: ./authorize_channel_<module>_<channel>.json)
  SUBMIT_PROPOSAL   If set to 1, the script will immediately submit the proposal using pawd tx gov submit-proposal
  PAW_QUERY_FLAGS   Extra flags appended to all pawd q commands (e.g. "--node tcp://localhost:26657")
EOF
  exit 1
fi

MODULE="$1"
PORT_ID="$2"
CHANNEL_ID="$3"
AUTHORITY="$4"
FROM_KEY="$5"
shift 5 || true

case "$MODULE" in
  dex)
    QUERY_CMD=(pawd q dex params -o json ${PAW_QUERY_FLAGS:-})
    MSG_TYPE="/paw.dex.v1.MsgUpdateParams"
    ;;
  oracle)
    QUERY_CMD=(pawd q oracle params -o json ${PAW_QUERY_FLAGS:-})
    MSG_TYPE="/paw.oracle.v1.MsgUpdateParams"
    ;;
  compute)
    QUERY_CMD=(pawd q compute params -o json ${PAW_QUERY_FLAGS:-})
    MSG_TYPE="/paw.compute.v1.MsgUpdateParams"
    ;;
  *)
    echo "error: unsupported module '$MODULE' (expected dex|oracle|compute)" >&2
    exit 1
    ;;
esac

echo "Fetching current $MODULE params..."
PARAMS_JSON="$("${QUERY_CMD[@]}")"

UPDATED_PARAMS_JSON=$(echo "$PARAMS_JSON" | jq --arg port "$PORT_ID" --arg channel "$CHANNEL_ID" '
  .params as $params
  | $params.authorized_channels as $channels
  | ([$channels[]? | select(.port_id == $port and .channel_id == $channel)] | length) as $exists
  | if $exists > 0
      then .
      else .params.authorized_channels += [{"port_id": $port, "channel_id": $channel}]
    end
')

DEPOSIT=${DEPOSIT:-"1000000upaw"}
PROPOSAL_FILE=${PROPOSAL_FILE:-"./authorize_channel_${MODULE}_${CHANNEL_ID}.json"}
TITLE="Authorize ${MODULE^^} channel ${PORT_ID}/${CHANNEL_ID}"
SUMMARY="Adds ${PORT_ID}/${CHANNEL_ID} to the ${MODULE^^} module's authorized IBC channel allowlist."

echo "Writing proposal payload to ${PROPOSAL_FILE}"
cat >"$PROPOSAL_FILE" <<EOF
{
  "messages": [
    {
      "@type": "${MSG_TYPE}",
      "authority": "${AUTHORITY}",
      "params": $(echo "$UPDATED_PARAMS_JSON" | jq '.params')
    }
  ],
  "metadata": "",
  "deposit": "${DEPOSIT}",
  "title": "${TITLE}",
  "summary": "${SUMMARY}"
}
EOF

echo "Proposal file ready: ${PROPOSAL_FILE}"
echo "Submit with: pawd tx gov submit-proposal ${PROPOSAL_FILE} --from ${FROM_KEY} --yes [flags]"

if [[ "${SUBMIT_PROPOSAL:-0}" == "1" ]]; then
  echo "Submitting proposal..."
  pawd tx gov submit-proposal "${PROPOSAL_FILE}" --from "${FROM_KEY}" --deposit "${DEPOSIT}" "$@"
fi
