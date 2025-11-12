# Bootstrap script for PAW controller + compute test node
#
# Creates config directories, writes the genesis placeholder, and emits sample keys so
# the node can be started via `infra/start-test-node.sh`.

set -euo pipefail

BASE_DIR="infra/node"
DATA_DIR="$BASE_DIR/data"

mkdir -p "$DATA_DIR"

cat > "$BASE_DIR/genesis.json" <<'EOF'
{
  "chain_id": "paw-testnet",
  "genesis_time": "2025-12-01T00:00:00Z",
  "initial_balance": 100000000,
  "validators": [
    {
      "name": "validator-1",
      "address": "PAW1VALIDATOR0000000000000000000000000000",
      "vote_power": 50
    },
    {
      "name": "validator-2",
      "address": "PAW1VALIDATOR0000000000000000000000000001",
      "vote_power": 50
    }
  ]
}
EOF

cat > "$BASE_DIR/node.env" <<'EOF'
PAW_CHAIN_ID=paw-testnet
PAW_DATA_DIR=$DATA_DIR
PAW_EMISSION_SCHEDULE=2870,1435,717
PAW_FERNET_SALT=$(python - <<'PY'
import os, base64
print(base64.urlsafe_b64encode(os.urandom(8)).decode())
PY
)
EOF

echo "Bootstrap complete. Run 'infra/start-test-node.sh' to launch the controller node preview."
