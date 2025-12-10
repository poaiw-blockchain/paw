package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/math"
	storeprefix "cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/paw-chain/paw/x/compute/types"
)

// dispute storage helpers
func (k Keeper) getNextDisputeID(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(NextDisputeIDKey)
	var nextID uint64 = 1
	if bz != nil {
		nextID = binary.BigEndian.Uint64(bz)
	}
	next := make([]byte, 8)
	binary.BigEndian.PutUint64(next, nextID+1)
	store.Set(NextDisputeIDKey, next)
	return nextID, nil
}

func (k Keeper) getNextAppealID(ctx context.Context) (uint64, error) {
	store := k.getStore(ctx)
	bz := store.Get(NextAppealIDKey)
	var nextID uint64 = 1
	if bz != nil {
		nextID = binary.BigEndian.Uint64(bz)
	}
	next := make([]byte, 8)
	binary.BigEndian.PutUint64(next, nextID+1)
	store.Set(NextAppealIDKey, next)
	return nextID, nil
}

// dispute CRUD
func (k Keeper) setDispute(ctx context.Context, dispute types.Dispute) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&dispute)
	if err != nil {
		return err
	}
	store.Set(DisputeKey(dispute.Id), bz)

	// indexes
	store.Set(DisputeByRequestKey(dispute.RequestId, dispute.Id), []byte{})
	store.Set(DisputeByStatusKey(saturateInt64ToUint32(int64(dispute.Status)), dispute.Id), []byte{})
	return nil
}

func (k Keeper) getDispute(ctx context.Context, id uint64) (*types.Dispute, error) {
	store := k.getStore(ctx)
	bz := store.Get(DisputeKey(id))
	if bz == nil {
		return nil, fmt.Errorf("dispute %d not found", id)
	}
	var dispute types.Dispute
	if err := k.cdc.Unmarshal(bz, &dispute); err != nil {
		return nil, err
	}
	return &dispute, nil
}

func (k Keeper) appendEvidence(ctx context.Context, disputeID uint64, evidence types.Evidence) error {
	store := k.getStore(ctx)

	// find next index
	prefix := EvidenceKeyPrefixForDispute(disputeID)
	iter := storetypes.KVStorePrefixIterator(store, prefix)
	defer iter.Close()
	var idx uint64
	for ; iter.Valid(); iter.Next() {
		idx++
	}

	bz, err := k.cdc.Marshal(&evidence)
	if err != nil {
		return err
	}
	store.Set(EvidenceKey(disputeID, idx), bz)
	return nil
}

// CreateDispute locks deposit, indexes dispute, and opens evidence period.
func (k Keeper) CreateDispute(ctx context.Context, requester sdk.AccAddress, requestID uint64, reason string, evidenceData []byte, deposit math.Int) (uint64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// validate request exists
	request, err := k.GetRequest(ctx, requestID)
	if err != nil {
		return 0, fmt.Errorf("request %d not found: %w", requestID, err)
	}
	if request.Provider == "" {
		return 0, fmt.Errorf("request %d has no provider assigned", requestID)
	}

	// governance deposit requirement
	govParams, err := k.GetGovernanceParams(ctx)
	if err != nil {
		return 0, err
	}
	minDeposit := govParams.DisputeDeposit
	if deposit.IsNil() || deposit.LT(minDeposit) {
		return 0, fmt.Errorf("deposit too low: got %s, need >= %s", deposit.String(), minDeposit.String())
	}

	// move funds to module escrow
	denom := k.bondDenom(ctx)
	coins := sdk.NewCoins(sdk.NewCoin(denom, deposit))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(sdkCtx, requester, types.ModuleName, coins); err != nil {
		return 0, fmt.Errorf("failed to lock dispute deposit: %w", err)
	}

	disputeID, err := k.getNextDisputeID(ctx)
	if err != nil {
		return 0, err
	}

	now := sdkCtx.BlockTime()
	dispute := types.Dispute{
		Id:             disputeID,
		RequestId:      requestID,
		Requester:      requester.String(),
		Provider:       request.Provider,
		Reason:         reason,
		Status:         types.DISPUTE_STATUS_EVIDENCE_SUBMISSION,
		Deposit:        deposit,
		CreatedAt:      now,
		EvidenceEndsAt: now.Add(secondsToDuration(govParams.EvidencePeriodSeconds)),
		VotingEndsAt:   now.Add(secondsToDuration(govParams.EvidencePeriodSeconds + govParams.VotingPeriodSeconds)),
		Votes:          []types.DisputeVote{},
		Resolution:     types.DISPUTE_RESOLUTION_UNSPECIFIED,
	}

	if err := k.setDispute(ctx, dispute); err != nil {
		return 0, err
	}

	// store initial evidence if provided
	if len(evidenceData) > 0 {
		evidence := types.Evidence{
			Submitter:    requester.String(),
			DisputeId:    disputeID,
			EvidenceType: "initial",
			Data:         evidenceData,
			Description:  reason,
			SubmittedAt:  now,
		}
		if err := k.appendEvidence(ctx, disputeID, evidence); err != nil {
			return 0, err
		}
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_dispute_created",
			sdk.NewAttribute("dispute_id", fmt.Sprintf("%d", disputeID)),
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("provider", request.Provider),
		),
	)

	return disputeID, nil
}

// VoteOnDispute records a validator vote and transitions to voting/tally as appropriate.
func (k Keeper) VoteOnDispute(ctx context.Context, validator sdk.ValAddress, disputeID uint64, vote types.DisputeVoteOption, justification string) error {
	dispute, err := k.getDispute(ctx, disputeID)
	if err != nil {
		return err
	}

	if dispute.Status != types.DISPUTE_STATUS_EVIDENCE_SUBMISSION && dispute.Status != types.DISPUTE_STATUS_VOTING {
		return fmt.Errorf("dispute %d not accepting votes", disputeID)
	}

	// ensure no duplicate vote
	for _, v := range dispute.Votes {
		if v.Validator == validator.String() {
			return fmt.Errorf("validator already voted")
		}
	}

	// determine voting power from staking
	votePower := k.getValidatorPower(ctx, validator)

	dispute.Votes = append(dispute.Votes, types.DisputeVote{
		Validator:     validator.String(),
		Option:        vote,
		Justification: justification,
		VotingPower:   votePower,
		VotedAt:       sdk.UnwrapSDKContext(ctx).BlockTime(),
	})
	dispute.Status = types.DISPUTE_STATUS_VOTING

	return k.setDispute(ctx, *dispute)
}

// ResolveDispute tallies votes and applies resolution logic; authority-gated.
func (k Keeper) ResolveDispute(ctx context.Context, authority sdk.AccAddress, disputeID uint64) (types.DisputeResolution, error) {
	if authority.String() != k.authority {
		return types.DISPUTE_RESOLUTION_UNSPECIFIED, fmt.Errorf("unauthorized resolution: expected %s", k.authority)
	}

	dispute, err := k.getDispute(ctx, disputeID)
	if err != nil {
		return types.DISPUTE_RESOLUTION_UNSPECIFIED, err
	}

	// power-weighted tally
	weighted := map[types.DisputeVoteOption]math.Int{}
	totalPower := math.ZeroInt()
	for _, v := range dispute.Votes {
		if _, ok := weighted[v.Option]; !ok {
			weighted[v.Option] = math.ZeroInt()
		}
		weighted[v.Option] = weighted[v.Option].Add(v.VotingPower)
		totalPower = totalPower.Add(v.VotingPower)
	}

	// quorum check (best effort, relative to total bonded if available)
	gov, err := k.GetGovernanceParams(ctx)
	if err != nil {
		return types.DISPUTE_RESOLUTION_UNSPECIFIED, err
	}
	if totalPower.IsZero() {
		return types.DISPUTE_RESOLUTION_UNSPECIFIED, fmt.Errorf("no votes submitted")
	}

	resolution := types.DISPUTE_RESOLUTION_TECHNICAL_ISSUE
	maxOpt := k.maxWeightedVote(weighted)
	switch maxOpt {
	case types.DISPUTE_VOTE_PROVIDER_FAULT:
		resolution = types.DISPUTE_RESOLUTION_SLASH_PROVIDER
	case types.DISPUTE_VOTE_REQUESTER_FAULT:
		resolution = types.DISPUTE_RESOLUTION_NO_REFUND
	case types.DISPUTE_VOTE_INSUFFICIENT_EVIDENCE:
		resolution = types.DISPUTE_RESOLUTION_TECHNICAL_ISSUE
	case types.DISPUTE_VOTE_NO_FAULT:
		resolution = types.DISPUTE_RESOLUTION_TECHNICAL_ISSUE
	default:
		resolution = types.DISPUTE_RESOLUTION_TECHNICAL_ISSUE
	}

	// enforce consensus threshold relative to totalPower
	threshold := gov.ConsensusThreshold
	maxPower := weighted[maxOpt]
	if threshold.GT(math.LegacyZeroDec()) {
		if totalPower.IsZero() || math.LegacyNewDecFromInt(maxPower).Quo(math.LegacyNewDecFromInt(totalPower)).LT(threshold) {
			return types.DISPUTE_RESOLUTION_UNSPECIFIED, fmt.Errorf("consensus threshold not met")
		}
	}

	dispute.Resolution = resolution
	dispute.Status = types.DISPUTE_STATUS_RESOLVED
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	dispute.ResolvedAt = &now

	if err := k.setDispute(ctx, *dispute); err != nil {
		return types.DISPUTE_RESOLUTION_UNSPECIFIED, err
	}

	return resolution, nil
}

func (k Keeper) maxWeightedVote(weighted map[types.DisputeVoteOption]math.Int) types.DisputeVoteOption {
	var maxOpt types.DisputeVoteOption
	maxPower := math.ZeroInt()
	for opt, p := range weighted {
		if p.GT(maxPower) {
			maxPower = p
			maxOpt = opt
		}
	}
	return maxOpt
}

// Appeals
func (k Keeper) CreateAppeal(ctx context.Context, provider sdk.AccAddress, slashID uint64, justification string, deposit math.Int) (uint64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	govParams, err := k.GetGovernanceParams(ctx)
	if err != nil {
		return 0, err
	}
	if deposit.IsNil() {
		return 0, fmt.Errorf("appeal deposit required")
	}

	// deposit must respect appeal_deposit_percentage * slash_amount (minimum) and dispute deposit baseline
	minDeposit := govParams.DisputeDeposit
	record, err := k.getSlashRecord(ctx, slashID)
	if err == nil && record.Amount.IsPositive() {
		percent := govParams.AppealDepositPercentage
		percentAmount := percent.MulInt(record.Amount).TruncateInt()
		if percentAmount.GT(minDeposit) {
			minDeposit = percentAmount
		}
	}
	if deposit.LT(minDeposit) {
		return 0, fmt.Errorf("appeal deposit too low: need >= %s", minDeposit.String())
	}

	denom := k.bondDenom(ctx)
	coins := sdk.NewCoins(sdk.NewCoin(denom, deposit))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(sdkCtx, provider, types.ModuleName, coins); err != nil {
		return 0, fmt.Errorf("failed to lock appeal deposit: %w", err)
	}

	appealID, err := k.getNextAppealID(ctx)
	if err != nil {
		return 0, err
	}

	now := sdkCtx.BlockTime()
	appeal := types.Appeal{
		Id:            appealID,
		SlashId:       slashID,
		Provider:      provider.String(),
		Justification: justification,
		Status:        types.APPEAL_STATUS_PENDING,
		Deposit:       deposit,
		CreatedAt:     now,
		VotingEndsAt:  now.Add(secondsToDuration(govParams.VotingPeriodSeconds)),
		Votes:         []types.AppealVote{},
	}

	if err := k.setAppeal(ctx, appeal); err != nil {
		return 0, err
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"compute_appeal_created",
			sdk.NewAttribute("appeal_id", fmt.Sprintf("%d", appealID)),
			sdk.NewAttribute("provider", provider.String()),
		),
	)

	return appealID, nil
}

func (k Keeper) setAppeal(ctx context.Context, appeal types.Appeal) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&appeal)
	if err != nil {
		return err
	}
	store.Set(AppealKey(appeal.Id), bz)
	store.Set(AppealByStatusKey(saturateInt64ToUint32(int64(appeal.Status)), appeal.Id), []byte{})
	return nil
}

func (k Keeper) getAppeal(ctx context.Context, id uint64) (*types.Appeal, error) {
	store := k.getStore(ctx)
	bz := store.Get(AppealKey(id))
	if bz == nil {
		return nil, fmt.Errorf("appeal %d not found", id)
	}
	var appeal types.Appeal
	if err := k.cdc.Unmarshal(bz, &appeal); err != nil {
		return nil, err
	}
	return &appeal, nil
}

func (k Keeper) VoteOnAppeal(ctx context.Context, validator sdk.ValAddress, appealID uint64, approve bool, justification string) error {
	appeal, err := k.getAppeal(ctx, appealID)
	if err != nil {
		return err
	}
	if appeal.Status != types.APPEAL_STATUS_PENDING && appeal.Status != types.APPEAL_STATUS_VOTING {
		return fmt.Errorf("appeal %d not accepting votes", appealID)
	}
	for _, v := range appeal.Votes {
		if v.Validator == validator.String() {
			return fmt.Errorf("validator already voted")
		}
	}

	votePower := k.getValidatorPower(ctx, validator)
	appeal.Votes = append(appeal.Votes, types.AppealVote{
		Validator:     validator.String(),
		Approve:       approve,
		Justification: justification,
		VotingPower:   votePower,
		VotedAt:       sdk.UnwrapSDKContext(ctx).BlockTime(),
	})
	appeal.Status = types.APPEAL_STATUS_VOTING
	return k.setAppeal(ctx, *appeal)
}

func (k Keeper) ResolveAppeal(ctx context.Context, authority sdk.AccAddress, appealID uint64) (bool, error) {
	if authority.String() != k.authority {
		return false, fmt.Errorf("unauthorized resolution")
	}

	appeal, err := k.getAppeal(ctx, appealID)
	if err != nil {
		return false, err
	}

	approveCount := 0
	for _, v := range appeal.Votes {
		if v.Approve {
			approveCount++
		}
	}
	approved := approveCount*2 >= len(appeal.Votes) // simple majority
	appeal.Approved = approved
	appeal.Status = types.APPEAL_STATUS_RESOLVED
	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	appeal.ResolvedAt = &now

	if err := k.setAppeal(ctx, *appeal); err != nil {
		return false, err
	}

	return approved, nil
}

// ApplyAppealOutcome adjusts slash record and stake after an appeal.
func (k Keeper) ApplyAppealOutcome(ctx context.Context, appealID uint64, approved bool) error {
	appeal, err := k.getAppeal(ctx, appealID)
	if err != nil {
		return err
	}

	record, err := k.getSlashRecord(ctx, appeal.SlashId)
	if err != nil {
		return err
	}

	// If approved, refund slash amount to provider; otherwise leave slash.
	if approved && record.Amount.IsPositive() {
		providerAddr, err := sdk.AccAddressFromBech32(record.Provider)
		if err != nil {
			return err
		}
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		coins := sdk.NewCoins(sdk.NewCoin("upaw", record.Amount))
		if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, providerAddr, coins); err != nil {
			return fmt.Errorf("failed to refund slash on appeal: %w", err)
		}
		provider, err := k.GetProvider(ctx, providerAddr)
		if err == nil {
			provider.Stake = provider.Stake.Add(record.Amount)
			if err := k.SetProvider(ctx, *provider); err != nil {
				return err
			}
		}
	}

	record.Appealed = true
	record.AppealId = appeal.Id
	return k.setSlashRecord(ctx, *record)
}

// Governance params storage with defaults
func (k Keeper) GetGovernanceParams(ctx context.Context) (types.GovernanceParams, error) {
	store := k.getStore(ctx)
	bz := store.Get(GovernanceParamsKey)
	if bz == nil {
		return types.DefaultGovernanceParams(), nil
	}
	var params types.GovernanceParams
	if err := k.cdc.Unmarshal(bz, &params); err != nil {
		return types.GovernanceParams{}, err
	}
	return params, nil
}

func (k Keeper) SetGovernanceParams(ctx context.Context, params types.GovernanceParams) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(GovernanceParamsKey, bz)
	return nil
}

// Evidence pagination
func (k Keeper) ListEvidence(ctx context.Context, disputeID uint64, pageReq *query.PageRequest) ([]types.Evidence, *query.PageResponse, error) {
	store := k.getStore(ctx)
	evStore := storeprefix.NewStore(store, EvidenceKeyPrefixForDispute(disputeID))

	var evidence []types.Evidence
	pageRes, err := query.Paginate(evStore, pageReq, func(key []byte, value []byte) error {
		var ev types.Evidence
		if err := k.cdc.Unmarshal(value, &ev); err != nil {
			return err
		}
		evidence = append(evidence, ev)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return evidence, pageRes, nil
}

// SubmitEvidence attaches evidence to a dispute, enforcing size limits and windows.
func (k Keeper) SubmitEvidence(ctx context.Context, submitter sdk.AccAddress, disputeID uint64, evidenceType string, data []byte, description string) error {
	if len(data) == 0 {
		return fmt.Errorf("evidence data cannot be empty")
	}
	gov, err := k.GetGovernanceParams(ctx)
	if err != nil {
		return err
	}
	if gov.MaxEvidenceSize > 0 && uint64(len(data)) > gov.MaxEvidenceSize {
		return fmt.Errorf("evidence size %d exceeds max %d", len(data), gov.MaxEvidenceSize)
	}

	dispute, err := k.getDispute(ctx, disputeID)
	if err != nil {
		return err
	}
	if dispute.Status != types.DISPUTE_STATUS_EVIDENCE_SUBMISSION {
		return fmt.Errorf("dispute not accepting evidence")
	}

	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	if now.After(dispute.EvidenceEndsAt) {
		return fmt.Errorf("evidence window closed")
	}

	ev := types.Evidence{
		DisputeId:    disputeID,
		Submitter:    submitter.String(),
		EvidenceType: evidenceType,
		Data:         data,
		Description:  description,
		SubmittedAt:  now,
	}
	if err := k.appendEvidence(ctx, disputeID, ev); err != nil {
		return err
	}
	return nil
}

// SettleDisputeOutcome routes funds and slashing based on resolution.
func (k Keeper) SettleDisputeOutcome(ctx context.Context, disputeID uint64, resolution types.DisputeResolution) error {
	dispute, err := k.getDispute(ctx, disputeID)
	if err != nil {
		return err
	}
	request, err := k.GetRequest(ctx, dispute.RequestId)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	providerAddr, err := sdk.AccAddressFromBech32(dispute.Provider)
	if err != nil {
		return err
	}
	requesterAddr, err := sdk.AccAddressFromBech32(dispute.Requester)
	if err != nil {
		return err
	}

	// helper to refund deposit
	refundDeposit := func() error {
		if dispute.Deposit.IsPositive() {
			coins := sdk.NewCoins(sdk.NewCoin("upaw", dispute.Deposit))
			if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requesterAddr, coins); err != nil {
				return fmt.Errorf("failed to refund dispute deposit: %w", err)
			}
		}
		return nil
	}

	params, err := k.GetGovernanceParams(ctx)
	if err != nil {
		return err
	}

	escrowAmount := request.EscrowedAmount
	providerRecord, err := k.GetProvider(ctx, providerAddr)
	if err != nil {
		return err
	}
	denom := k.bondDenom(ctx)

	// Check if escrow state exists before attempting operations
	hasEscrow := false
	if !escrowAmount.IsZero() {
		if _, err := k.GetEscrowState(ctx, request.Id); err == nil {
			hasEscrow = true
		}
	}

	switch resolution {
	case types.DISPUTE_RESOLUTION_SLASH_PROVIDER:
		slashAmt := params.SlashPercentage.MulInt(providerRecord.Stake).TruncateInt()
		if slashAmt.IsZero() {
			// minimum meaningful slash: 1 unit if stake positive
			if providerRecord.Stake.IsPositive() {
				slashAmt = math.NewInt(1)
			}
		}
		if slashAmt.GT(providerRecord.Stake) {
			slashAmt = providerRecord.Stake
		}
		if slashAmt.IsPositive() {
			coins := sdk.NewCoins(sdk.NewCoin(denom, slashAmt))
			if err := k.bankKeeper.BurnCoins(sdkCtx, types.ModuleName, coins); err != nil {
				return fmt.Errorf("burn slash coins: %w", err)
			}
			providerRecord.Stake = providerRecord.Stake.Sub(slashAmt)
			if err := k.SetProvider(ctx, *providerRecord); err != nil {
				return err
			}
			if _, err := k.recordSlash(ctx, providerAddr, request.Id, dispute.Id, slashAmt, dispute.Reason); err != nil {
				return err
			}
		}
		if hasEscrow {
			if err := k.RefundEscrow(ctx, request.Id, "provider_fault"); err != nil {
				return fmt.Errorf("failed to refund escrow during dispute settlement: %w", err)
			}
		}
		if err := refundDeposit(); err != nil {
			return err
		}
	case types.DISPUTE_RESOLUTION_NO_REFUND:
		if hasEscrow {
			if err := k.ReleaseEscrow(ctx, request.Id, true); err != nil {
				return fmt.Errorf("failed to release escrow during dispute settlement: %w", err)
			}
		}
		if err := refundDeposit(); err != nil {
			return err
		}
	case types.DISPUTE_RESOLUTION_PARTIAL_REFUND:
		// conservative default: refund escrow; governance can adopt finer-grained policies later
		if hasEscrow {
			if err := k.RefundEscrow(ctx, request.Id, "dispute_partial_refund"); err != nil {
				return fmt.Errorf("failed to refund escrow during dispute settlement: %w", err)
			}
		}
		if err := refundDeposit(); err != nil {
			return err
		}
	default:
		if hasEscrow {
			if err := k.RefundEscrow(ctx, request.Id, "dispute_default_refund"); err != nil {
				return fmt.Errorf("failed to refund escrow during dispute settlement: %w", err)
			}
		}
		if err := refundDeposit(); err != nil {
			return err
		}
	}

	return nil
}

// getValidatorPower returns validator bonded stake; falls back to 1 if unavailable.
func (k Keeper) getValidatorPower(ctx context.Context, val sdk.ValAddress) math.Int {
	if k.stakingKeeper != nil {
		valRec, err := k.stakingKeeper.Validator(ctx, val)
		if err == nil {
			return valRec.GetTokens()
		}
	}
	return math.NewInt(1)
}

func (k Keeper) bondDenom(ctx context.Context) string {
	if k.stakingKeeper != nil {
		if denom, err := k.stakingKeeper.BondDenom(ctx); err == nil && denom != "" {
			return denom
		}
	}
	// default to upaw if staking keeper unavailable
	return "upaw"
}
