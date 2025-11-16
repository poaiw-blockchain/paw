package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		Tasks:      []ComputeTask{},
		NextTaskId: 1,
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Validate params
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	// Track seen task IDs to detect duplicates
	seenTaskIds := make(map[uint64]bool)
	maxTaskId := uint64(0)

	// Valid task statuses
	validStatuses := map[string]bool{
		"pending":  true,
		"verified": true,
		"failed":   true,
		"expired":  true,
	}

	// Validate each compute task
	for i, task := range gs.Tasks {
		// Validate task ID
		if task.Id == 0 {
			return fmt.Errorf("task %d: id cannot be zero", i)
		}

		// Check for duplicate task IDs
		if seenTaskIds[task.Id] {
			return fmt.Errorf("task %d: duplicate task id %d", i, task.Id)
		}
		seenTaskIds[task.Id] = true

		// Track max task ID
		if task.Id > maxTaskId {
			maxTaskId = task.Id
		}

		// Validate requester address
		if task.Requester == "" {
			return fmt.Errorf("task %d (id=%d): requester cannot be empty", i, task.Id)
		}
		if _, err := sdk.AccAddressFromBech32(task.Requester); err != nil {
			return fmt.Errorf("task %d (id=%d): invalid requester address %s: %w", i, task.Id, task.Requester, err)
		}

		// Validate task type
		if task.TaskType == "" {
			return fmt.Errorf("task %d (id=%d): task_type cannot be empty", i, task.Id)
		}

		// Validate reward is positive
		if task.Reward.IsNil() || !task.Reward.IsPositive() {
			return fmt.Errorf("task %d (id=%d): reward must be positive", i, task.Id)
		}

		// Validate status
		if task.Status == "" {
			return fmt.Errorf("task %d (id=%d): status cannot be empty", i, task.Id)
		}
		if !validStatuses[task.Status] {
			return fmt.Errorf("task %d (id=%d): invalid status %s (must be pending, verified, failed, or expired)", i, task.Id, task.Status)
		}

		// Validate timestamps
		if task.CreatedAt < 0 {
			return fmt.Errorf("task %d (id=%d): created_at cannot be negative", i, task.Id)
		}
		if task.VerifiedAt < 0 {
			return fmt.Errorf("task %d (id=%d): verified_at cannot be negative", i, task.Id)
		}
		if task.VerifiedAt > 0 && task.VerifiedAt < task.CreatedAt {
			return fmt.Errorf("task %d (id=%d): verified_at (%d) cannot be before created_at (%d)", i, task.Id, task.VerifiedAt, task.CreatedAt)
		}

		// If task is verified, verified_at should be set
		if task.Status == "verified" && task.VerifiedAt == 0 {
			return fmt.Errorf("task %d (id=%d): verified task must have verified_at timestamp", i, task.Id)
		}
	}

	// Validate NextTaskId
	if gs.NextTaskId == 0 {
		return fmt.Errorf("next_task_id cannot be zero")
	}
	if gs.NextTaskId <= maxTaskId {
		return fmt.Errorf("next_task_id (%d) must be greater than the highest task id (%d)", gs.NextTaskId, maxTaskId)
	}

	return nil
}
