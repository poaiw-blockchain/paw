# PAW 4-Validator Network Diagnosis

## Date: 2025-12-14

## Current Status
The 4-validator network is **stuck at height 0, unable to reach consensus on block 1**.

## What's Working
✅ All 4 nodes are running and connected (3 peers each)
✅ Genesis file has 4 properly configured validators with correct:
- Tokens: 250,000,000,000 upaw each
- Voting power: 250,000 each
- Delegations: Proper self-delegations
- No nil/zero values in critical fields

✅ Peer discovery working - all nodes see each other
✅ Validator keys match between genesis and running nodes
✅ Network topology configured correctly

## The Problem: Validators Voting NIL Instead of Block Hash

### Evidence from Logs
```
DBG precommit step; +2/3 prevoted for nil [height=1] [round=1154]
DBG added vote to prevote prevotes="VoteSet{H:1 R:1154 T:SIGNED_MSG_TYPE_PREVOTE +2/3:<nil>(0.75)}"
Vote{1:1AF24B35096E 1/1154/SIGNED_MSG_TYPE_PREVOTE(Prevote) 000000000000 29852C077712 000000000000}
```

**Key Finding:** Validators receive proposals with valid block hashes, but instead of prevoting for the block, they prevote for `nil` (`000000000000`).

### Why Validators Vote NIL

Validators vote nil when:
1. ❌ **Block fails ABCI validation** (Process

Proposal returns REJECT)
2. ❌ Time issues with the block
3. ❌ Missing or invalid Proof-of-Lock (POL)
4. ❌ **Empty app_hash in genesis** (most likely cause)

### Suspected Root Cause: Empty app_hash

**Genesis file has:**
```json
{
  "app_hash": "",
  "initial_height": "1"
}
```

**Problem:** When genesis contains bonded validators (not in gentxs), Cosmos SDK expects the `app_hash` to match the computed state hash. An empty string may cause the first block to fail validation.

### What Happens
1. Node starts, loads genesis with empty `app_hash`
2. Proposer creates block 1 with state transitions
3. Other validators receive the block
4. **Validators reject the block** (likely in ProcessProposal or block verification)
5. Validators vote `nil` instead of the block hash
6. Consensus stalls - can never get >2/3 prevotes for the actual block

## Comparison with Aura

Need to check if Aura's successful 4-validator genesis has:
- Non-empty app_hash
- Different genesis structure
- Different collect-gentxs implementation

## Next Steps to Fix

1. **Check Aura's working genesis** for app_hash value
2. **Investigate collect-gentxs** - should it compute and set app_hash?
3. **Enable ABCI logging** to see if ProcessProposal is rejecting blocks
4. **Compare SDK versions** between PAW and Aura
5. **Test with debug logs** at ABCI layer to see rejection reason

## Files Modified During This Session

### ✅ Fixed Issues
1. `scripts/devnet/init_node.sh` - Now properly detects and uses 4-validator genesis
2. `scripts/devnet/init_node.sh` - Exports all node IDs before peer configuration (prevents deadlock)
3. `scripts/devnet/init_node.sh` - Configures all nodes as mesh network (each connects to all others)
4. `scripts/devnet/setup-multivalidators.sh` - Already correctly generates 4-validator genesis

### ⚠️ Unresolved
- Empty `app_hash` in genesis causing block rejection
- Need to understand why validators prevote nil

## Logs and Evidence

Node logs show continuous rounds with nil prevotes:
- Round 1154, 1305, 1327, etc.
- All validators connected and communicating
- Proposals being created and broadcast
- But consensus never commits a block

## Commands to Reproduce

```bash
cd /home/hudson/blockchain-projects/paw

# Check current status
curl -s localhost:26657/status | jq '.result.sync_info'

# Check consensus state
curl -s localhost:26657/dump_consensus_state | jq '.result.round_state | {height, round, step}'

# Check validators
curl -s localhost:26657/validators | jq '.result.validators[] | {address, voting_power}'

# Check peers
docker exec paw-node1 pawd tendermint show-node-id --home /root/.paw/node1
```

## Summary

The network is **very close** to working. All infrastructure is correct, but there's a **block validation issue** preventing consensus. The most likely cause is the empty `app_hash` in genesis when validators are already bonded.

**Recommendation:** Compare with Aura's working setup, specifically the `app_hash` field and any differences in how genesis is finalized.
