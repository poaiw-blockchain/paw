package keeper

import (
	"context"
	"fmt"
	"math"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// Cryptoeconomic Analysis Module
// This module provides game-theoretic security guarantees and optimal mechanism design

// GameTheoreticAnalysis contains cryptoeconomic security parameters
type GameTheoreticAnalysis struct {
	// Attack cost in base denomination
	AttackCost sdkmath.Int

	// Expected reward from honest behavior
	HonestReward sdkmath.LegacyDec

	// Expected penalty from dishonest behavior
	DishonestPenalty sdkmath.LegacyDec

	// Profit from successful attack
	AttackProfit sdkmath.LegacyDec

	// Is the system incentive compatible?
	IsIncentiveCompatible bool

	// Security margin (attack cost / attack profit)
	SecurityMargin sdkmath.LegacyDec
}

// ValidatorIncentives models validator behavior incentives
type ValidatorIncentives struct {
	ValidatorAddr     string
	Stake             sdkmath.Int
	ExpectedSlashing  sdkmath.LegacyDec
	ReputationValue   sdkmath.LegacyDec
	TotalExpectedCost sdkmath.LegacyDec
	TotalExpectedGain sdkmath.LegacyDec
	NetExpectedValue  sdkmath.LegacyDec
	ShouldParticipate bool
}

// NashEquilibriumAnalysis analyzes game-theoretic equilibrium
type NashEquilibriumAnalysis struct {
	// Is honest reporting a Nash equilibrium?
	IsNashEquilibrium bool

	// Minimum stake required for security
	MinSecureStake sdkmath.Int

	// Optimal slash fraction
	OptimalSlashFraction sdkmath.LegacyDec

	// Attack success probability
	AttackSuccessProbability sdkmath.LegacyDec

	// Expected value of attack
	AttackExpectedValue sdkmath.LegacyDec
}

// AnalyzeCryptoeconomicSecurity performs comprehensive game-theoretic analysis
func (k Keeper) AnalyzeCryptoeconomicSecurity(ctx context.Context) (GameTheoreticAnalysis, error) {
	// Calculate attack cost (cost to control 33% of validators)
	attackCost, err := k.CalculateAttackCost(ctx)
	if err != nil {
		return GameTheoreticAnalysis{}, err
	}

	// Calculate honest validator rewards (simplified - would include block rewards, fees, etc.)
	params, err := k.GetParams(ctx)
	if err != nil {
		return GameTheoreticAnalysis{}, err
	}

	// Expected reward from honest behavior
	// In practice, this would be calculated from historical validator rewards
	honestReward := sdkmath.LegacyZeroDec()

	// Expected penalty from dishonest behavior (slashing)
	dishonestPenalty := params.SlashFraction

	// Estimated profit from successful oracle manipulation
	// This is asset-dependent; we use a conservative estimate
	attackProfit := sdkmath.LegacyNewDec(1000000) // Placeholder: $1M potential profit

	// Convert attack cost to Dec for comparison
	attackCostDec := sdkmath.LegacyNewDecFromInt(attackCost)

	// Security margin: how much more expensive is attack than potential profit?
	securityMargin := sdkmath.LegacyZeroDec()
	if attackProfit.GT(sdkmath.LegacyZeroDec()) {
		securityMargin = attackCostDec.Quo(attackProfit)
	}

	// System is incentive compatible if attack cost >> attack profit
	// AND dishonest penalty > honest reward loss
	isIncentiveCompatible := securityMargin.GT(sdkmath.LegacyNewDec(10)) && // 10x safety margin
		dishonestPenalty.GT(honestReward)

	return GameTheoreticAnalysis{
		AttackCost:            attackCost,
		HonestReward:          honestReward,
		DishonestPenalty:      dishonestPenalty,
		AttackProfit:          attackProfit,
		IsIncentiveCompatible: isIncentiveCompatible,
		SecurityMargin:        securityMargin,
	}, nil
}

// CalculateOptimalSlashFraction determines the optimal slashing penalty
// Using mechanism design theory and principal-agent models
func (k Keeper) CalculateOptimalSlashFraction(ctx context.Context) (sdkmath.LegacyDec, error) {
	// Get current validator set
	bondedVals, err := k.GetBondedValidators(ctx)
	if err != nil {
		return sdkmath.LegacyZeroDec(), err
	}

	if len(bondedVals) == 0 {
		return sdkmath.LegacyZeroDec(), fmt.Errorf("no bonded validators")
	}

	// Calculate average validator stake
	totalStake := sdkmath.ZeroInt()

	for _, val := range bondedVals {
		totalStake = totalStake.Add(val.GetTokens())
	}

	avgStake := sdkmath.LegacyNewDecFromInt(totalStake).QuoInt64(int64(len(bondedVals)))

	// Optimal slash fraction should be high enough to deter attacks
	// but low enough to not discourage participation

	// Formula: slash_fraction = (attack_profit / validator_stake) * safety_factor
	// where safety_factor ensures attack is unprofitable

	attackProfit := sdkmath.LegacyNewDec(1000000) // Conservative estimate
	safetyFactor := sdkmath.LegacyNewDec(2)       // 2x safety margin

	optimalSlash := sdkmath.LegacyZeroDec()
	if avgStake.GT(sdkmath.LegacyZeroDec()) {
		optimalSlash = attackProfit.Quo(avgStake).Mul(safetyFactor)
	}

	// Cap optimal slash between reasonable bounds
	minSlash := sdkmath.LegacyMustNewDecFromStr("0.0001") // 0.01%
	maxSlash := sdkmath.LegacyMustNewDecFromStr("0.01")   // 1%

	if optimalSlash.LT(minSlash) {
		optimalSlash = minSlash
	}
	if optimalSlash.GT(maxSlash) {
		optimalSlash = maxSlash
	}

	return optimalSlash, nil
}

// AnalyzeValidatorIncentives analyzes incentive structure for a validator
func (k Keeper) AnalyzeValidatorIncentives(ctx context.Context, validatorAddr string) (ValidatorIncentives, error) {
	valAddr, err := sdk.ValAddressFromBech32(validatorAddr)
	if err != nil {
		return ValidatorIncentives{}, err
	}

	// Get validator
	validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
	if err != nil {
		return ValidatorIncentives{}, err
	}

	stake := validator.GetTokens()

	// Get validator's reputation score
	reputationScore, _ := k.GetValidatorOutlierReputation(ctx, validatorAddr, "")

	// Calculate expected slashing risk
	params, err := k.GetParams(ctx)
	if err != nil {
		return ValidatorIncentives{}, err
	}

	// Expected slashing = slash_fraction * probability_of_slash
	// Probability increases with lower reputation
	probabilityOfSlash := sdkmath.LegacyOneDec().Sub(reputationScore) // Inverse of reputation
	expectedSlashing := params.SlashFraction.Mul(probabilityOfSlash)

	// Reputation value (intangible but real)
	reputationValue := reputationScore.Mul(sdkmath.LegacyNewDec(100000)) // Scaled value

	// Total expected cost of dishonest behavior
	slashingCost := expectedSlashing.MulInt(stake)
	totalExpectedCost := slashingCost.Add(reputationValue)

	// Total expected gain from honest participation (oracle rewards, block rewards, etc.)
	// Simplified: assume fixed percentage return on stake
	honestReturn := sdkmath.LegacyMustNewDecFromStr("0.10") // 10% annual return
	totalExpectedGain := honestReturn.MulInt(stake)

	// Net expected value of participation
	netExpectedValue := totalExpectedGain.Sub(totalExpectedCost)

	// Should validator participate? (if NEV > 0)
	shouldParticipate := netExpectedValue.GT(sdkmath.LegacyZeroDec())

	return ValidatorIncentives{
		ValidatorAddr:     validatorAddr,
		Stake:             stake,
		ExpectedSlashing:  expectedSlashing,
		ReputationValue:   reputationValue,
		TotalExpectedCost: totalExpectedCost,
		TotalExpectedGain: totalExpectedGain,
		NetExpectedValue:  netExpectedValue,
		ShouldParticipate: shouldParticipate,
	}, nil
}

// CalculateNashEquilibrium determines if honest behavior is Nash equilibrium
func (k Keeper) CalculateNashEquilibrium(ctx context.Context) (NashEquilibriumAnalysis, error) {
	// Get cryptoeconomic parameters
	analysis, err := k.AnalyzeCryptoeconomicSecurity(ctx)
	if err != nil {
		return NashEquilibriumAnalysis{}, err
	}

	// Nash equilibrium exists if no validator can profitably deviate
	// Condition: Expected(honest) >= Expected(dishonest) for all validators

	// Calculate minimum stake required for security
	// This is the stake needed to make attack cost > attack profit
	minSecureStake := analysis.AttackProfit.TruncateInt()

	// Calculate optimal slash fraction
	optimalSlash, err := k.CalculateOptimalSlashFraction(ctx)
	if err != nil {
		return NashEquilibriumAnalysis{}, err
	}

	// Attack success probability (assuming Byzantine tolerance)
	// With 33% Byzantine tolerance, attack needs >33% colluding validators
	// Probability decreases exponentially with number of independent validators
	bondedVals, err := k.GetBondedValidators(ctx)
	if err != nil {
		return NashEquilibriumAnalysis{}, err
	}

	numValidators := float64(len(bondedVals))

	// Simplified model: P(attack success) = e^(-k*n) where n = validators, k = security factor
	securityFactor := 0.3
	attackSuccessProbability := math.Exp(-securityFactor * numValidators)
	attackSuccessProbabilityDec := sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%.18f", attackSuccessProbability))

	// Expected value of attack = attack_profit * P(success) - attack_cost * P(failure)
	expectedProfit := analysis.AttackProfit.Mul(attackSuccessProbabilityDec)
	expectedCost := sdkmath.LegacyNewDecFromInt(analysis.AttackCost).Mul(
		sdkmath.LegacyOneDec().Sub(attackSuccessProbabilityDec))
	attackExpectedValue := expectedProfit.Sub(expectedCost)

	// System is in Nash equilibrium if attack has negative expected value
	isNashEquilibrium := attackExpectedValue.LT(sdkmath.LegacyZeroDec())

	return NashEquilibriumAnalysis{
		IsNashEquilibrium:        isNashEquilibrium,
		MinSecureStake:           minSecureStake,
		OptimalSlashFraction:     optimalSlash,
		AttackSuccessProbability: attackSuccessProbabilityDec,
		AttackExpectedValue:      attackExpectedValue,
	}, nil
}

// CalculateSchelling Point finds natural coordination point for honest prices
// Based on Schelling point / focal point theory
func (k Keeper) CalculateSchellingPoint(ctx context.Context, asset string) (sdkmath.LegacyDec, error) {
	// Get recent price history
	params, err := k.GetParams(ctx)
	if err != nil {
		return sdkmath.LegacyZeroDec(), err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	minHeight := sdkCtx.BlockHeight() - int64(params.TwapLookbackWindow)

	snapshots := []types.PriceSnapshot{}
	err = k.IteratePriceSnapshots(ctx, asset, func(snapshot types.PriceSnapshot) bool {
		if snapshot.BlockHeight >= minHeight {
			snapshots = append(snapshots, snapshot)
		}
		return false
	})

	if err != nil || len(snapshots) == 0 {
		return sdkmath.LegacyZeroDec(), fmt.Errorf("insufficient price history for Schelling point")
	}

	// Extract prices for calculation
	prices := make([]sdkmath.LegacyDec, len(snapshots))
	for i, snapshot := range snapshots {
		prices[i] = snapshot.Price
	}

	// Schelling point is the natural focal point rational actors converge to
	// In price oracles, this is typically the median of recent honest prices
	schellingPoint := k.calculateMedian(prices)

	return schellingPoint, nil
}

// CalculateInformationCost estimates cost of obtaining accurate price information
func (k Keeper) CalculateInformationCost(ctx context.Context) sdkmath.LegacyDec {
	// Information cost includes:
	// 1. Infrastructure cost (APIs, data feeds)
	// 2. Operational cost (monitoring, maintenance)
	// 3. Opportunity cost (validator time)

	// Simplified model: fixed cost per validator per period
	costPerValidator := sdkmath.LegacyNewDec(100) // $100 equivalent per period

	bondedVals, err := k.GetBondedValidators(ctx)
	if err != nil {
		return sdkmath.LegacyZeroDec()
	}

	totalInfoCost := costPerValidator.MulInt64(int64(len(bondedVals)))

	return totalInfoCost
}

// CalculateMechanismEfficiency evaluates oracle mechanism efficiency
func (k Keeper) CalculateMechanismEfficiency(ctx context.Context) (sdkmath.LegacyDec, error) {
	// Efficiency = (Security Provided) / (Total Cost)
	// Where Security Provided = Attack Cost
	// And Total Cost = Information Cost + Slashing Risk + Infrastructure

	attackCost, err := k.CalculateAttackCost(ctx)
	if err != nil {
		return sdkmath.LegacyZeroDec(), err
	}

	infoCost := k.CalculateInformationCost(ctx)

	// Total cost (simplified)
	totalCost := infoCost

	if totalCost.IsZero() {
		return sdkmath.LegacyZeroDec(), nil
	}

	// Efficiency ratio
	efficiency := sdkmath.LegacyNewDecFromInt(attackCost).Quo(totalCost)

	return efficiency, nil
}

// AnalyzeCollusionResistance evaluates resistance to validator collusion
func (k Keeper) AnalyzeCollusionResistance(ctx context.Context) (sdkmath.LegacyDec, error) {
	bondedVals, err := k.GetBondedValidators(ctx)
	if err != nil {
		return sdkmath.LegacyZeroDec(), err
	}

	n := len(bondedVals)
	if n == 0 {
		return sdkmath.LegacyZeroDec(), fmt.Errorf("no validators")
	}

	// Collusion resistance increases with:
	// 1. Number of independent validators (harder to coordinate)
	// 2. Stake distribution (reduces benefit of collusion)
	// 3. Geographic/organizational diversity (harder to collude)

	// Calculate Herfindahl-Hirschman Index (HHI) for stake concentration
	// HHI = sum(stake_share^2)
	// Lower HHI = better distribution = higher collusion resistance

	totalPower := int64(0)
	powerReduction := k.stakingKeeper.PowerReduction(ctx)
	powers := []int64{}

	for _, val := range bondedVals {
		power := val.GetConsensusPower(powerReduction)
		totalPower += power
		powers = append(powers, power)
	}

	if totalPower == 0 {
		return sdkmath.LegacyZeroDec(), nil
	}

	hhi := sdkmath.LegacyZeroDec()
	for _, power := range powers {
		share := sdkmath.LegacyNewDec(power).Quo(sdkmath.LegacyNewDec(totalPower))
		hhi = hhi.Add(share.Mul(share))
	}

	// Convert HHI to resistance score (1 - normalized HHI)
	// Perfect distribution (n equal validators) has HHI = 1/n
	perfectHHI := sdkmath.LegacyOneDec().QuoInt64(int64(n))

	// Normalize: resistance = 1 - (HHI - perfectHHI) / (1 - perfectHHI)
	denominator := sdkmath.LegacyOneDec().Sub(perfectHHI)
	if denominator.IsZero() {
		return sdkmath.LegacyOneDec(), nil
	}

	resistance := sdkmath.LegacyOneDec().Sub(
		hhi.Sub(perfectHHI).Quo(denominator),
	)

	// Clamp to [0, 1]
	if resistance.LT(sdkmath.LegacyZeroDec()) {
		resistance = sdkmath.LegacyZeroDec()
	}
	if resistance.GT(sdkmath.LegacyOneDec()) {
		resistance = sdkmath.LegacyOneDec()
	}

	return resistance, nil
}

// CalculateSystemSecurityScore provides overall cryptoeconomic security score
func (k Keeper) CalculateSystemSecurityScore(ctx context.Context) (sdkmath.LegacyDec, error) {
	// Composite security score based on multiple factors

	// Factor 1: Nash equilibrium analysis
	nashAnalysis, err := k.CalculateNashEquilibrium(ctx)
	if err != nil {
		return sdkmath.LegacyZeroDec(), err
	}

	nashScore := sdkmath.LegacyZeroDec()
	if nashAnalysis.IsNashEquilibrium {
		nashScore = sdkmath.LegacyOneDec()
	}

	// Factor 2: Collusion resistance
	collusionResistance, err := k.AnalyzeCollusionResistance(ctx)
	if err != nil {
		return sdkmath.LegacyZeroDec(), err
	}

	// Factor 3: Security margin (attack cost / attack profit)
	cryptoAnalysis, err := k.AnalyzeCryptoeconomicSecurity(ctx)
	if err != nil {
		return sdkmath.LegacyZeroDec(), err
	}

	// Normalize security margin to [0, 1] using sigmoid
	// score = 1 / (1 + e^(-k*(margin - threshold)))
	marginFloat, _ := cryptoAnalysis.SecurityMargin.Float64()
	threshold := 10.0 // Margin of 10x is good
	k_param := 0.2
	securityMarginScore := 1.0 / (1.0 + math.Exp(-k_param*(marginFloat-threshold)))
	securityMarginScoreDec := sdkmath.LegacyMustNewDecFromStr(fmt.Sprintf("%.18f", securityMarginScore))

	// Weighted composite score
	weights := map[string]sdkmath.LegacyDec{
		"nash":      sdkmath.LegacyMustNewDecFromStr("0.4"),
		"collusion": sdkmath.LegacyMustNewDecFromStr("0.3"),
		"margin":    sdkmath.LegacyMustNewDecFromStr("0.3"),
	}

	compositeScore := nashScore.Mul(weights["nash"]).
		Add(collusionResistance.Mul(weights["collusion"])).
		Add(securityMarginScoreDec.Mul(weights["margin"]))

	return compositeScore, nil
}

// ValidateCryptoeconomicSecurity ensures system maintains security properties
func (k Keeper) ValidateCryptoeconomicSecurity(ctx context.Context) error {
	score, err := k.CalculateSystemSecurityScore(ctx)
	if err != nil {
		return err
	}

	// Require minimum security score
	minScore := sdkmath.LegacyMustNewDecFromStr("0.7") // 70% minimum

	if score.LT(minScore) {
		return fmt.Errorf("cryptoeconomic security score too low: %s < %s (SECURITY RISK)",
			score.String(), minScore.String())
	}

	return nil
}
