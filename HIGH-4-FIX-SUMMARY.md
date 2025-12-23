# HIGH-4 Fix: Pagination Limits for State Iterations

## Issue
Unbounded iteration in `GetAllPools()` and `IterateRequests()` functions created DoS vulnerability.

## Solution
Added `MaxIterationLimit = 100` constant to both modules:

### DEX Module (`x/dex/keeper/pool.go`)
- `GetAllPools()`: Enforces 100-item limit
- Bonus: Fixed CEI pattern in `CreatePool()` (token transfer before state updates)

### Compute Module (`x/compute/keeper/request.go`)
- `IterateRequests()`: 100-item limit
- `IterateRequestsByRequester()`: 100-item limit
- `IterateRequestsByProvider()`: 100-item limit
- `IterateRequestsByStatus()`: 100-item limit

## Security Impact
✅ Prevents DoS via excessive iteration
✅ Limits memory/gas consumption
✅ 100 items sufficient for UI pagination
✅ Maintains API compatibility

## Commit
cd15793 - "security: add pagination limits to state iterations (HIGH-4)"
