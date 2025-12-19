package integration_test

import (
	"bytes"
	"reflect"
	"testing"

	proto "github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	_ "github.com/paw-chain/paw/x/compute/types"
	_ "github.com/paw-chain/paw/x/dex/types"
	_ "github.com/paw-chain/paw/x/oracle/types"
)

var (
	computeTxProtoTypes = []string{
		"paw.compute.v1.MsgRegisterProvider",
		"paw.compute.v1.MsgRegisterProviderResponse",
		"paw.compute.v1.MsgUpdateProvider",
		"paw.compute.v1.MsgUpdateProviderResponse",
		"paw.compute.v1.MsgDeactivateProvider",
		"paw.compute.v1.MsgDeactivateProviderResponse",
		"paw.compute.v1.MsgSubmitRequest",
		"paw.compute.v1.MsgSubmitRequestResponse",
		"paw.compute.v1.MsgCancelRequest",
		"paw.compute.v1.MsgCancelRequestResponse",
		"paw.compute.v1.MsgSubmitResult",
		"paw.compute.v1.MsgSubmitResultResponse",
		"paw.compute.v1.MsgUpdateParams",
		"paw.compute.v1.MsgUpdateParamsResponse",
		"paw.compute.v1.MsgCreateDispute",
		"paw.compute.v1.MsgCreateDisputeResponse",
		"paw.compute.v1.MsgVoteOnDispute",
		"paw.compute.v1.MsgVoteOnDisputeResponse",
		"paw.compute.v1.MsgResolveDispute",
		"paw.compute.v1.MsgResolveDisputeResponse",
		"paw.compute.v1.MsgSubmitEvidence",
		"paw.compute.v1.MsgSubmitEvidenceResponse",
		"paw.compute.v1.MsgAppealSlashing",
		"paw.compute.v1.MsgAppealSlashingResponse",
		"paw.compute.v1.MsgVoteOnAppeal",
		"paw.compute.v1.MsgVoteOnAppealResponse",
		"paw.compute.v1.MsgResolveAppeal",
		"paw.compute.v1.MsgResolveAppealResponse",
		"paw.compute.v1.MsgUpdateGovernanceParams",
		"paw.compute.v1.MsgUpdateGovernanceParamsResponse",
	}
	computeStateProtoTypes = []string{
		"paw.compute.v1.ComputeSpec",
		"paw.compute.v1.Provider",
		"paw.compute.v1.Pricing",
		"paw.compute.v1.Request",
		"paw.compute.v1.Result",
		"paw.compute.v1.EscrowState",
		"paw.compute.v1.Params",
		"paw.compute.v1.AuthorizedChannel",
		"paw.compute.v1.Dispute",
		"paw.compute.v1.DisputeVote",
		"paw.compute.v1.Evidence",
		"paw.compute.v1.SlashRecord",
		"paw.compute.v1.Appeal",
		"paw.compute.v1.AppealVote",
		"paw.compute.v1.GovernanceParams",
		"paw.compute.v1.PerformanceRecord",
		"paw.compute.v1.ProviderReputation",
		"paw.compute.v1.ProviderLoadTracker",
		"paw.compute.v1.RateLimitBucket",
		"paw.compute.v1.ResourceQuota",
		"paw.compute.v1.GenesisState",
	}
	computeQueryProtoTypes = []string{
		"paw.compute.v1.QueryParamsRequest",
		"paw.compute.v1.QueryParamsResponse",
		"paw.compute.v1.QueryProviderRequest",
		"paw.compute.v1.QueryProviderResponse",
		"paw.compute.v1.QueryProvidersRequest",
		"paw.compute.v1.QueryProvidersResponse",
		"paw.compute.v1.QueryActiveProvidersRequest",
		"paw.compute.v1.QueryActiveProvidersResponse",
		"paw.compute.v1.QueryRequestRequest",
		"paw.compute.v1.QueryRequestResponse",
		"paw.compute.v1.QueryRequestsRequest",
		"paw.compute.v1.QueryRequestsResponse",
		"paw.compute.v1.QueryRequestsByRequesterRequest",
		"paw.compute.v1.QueryRequestsByRequesterResponse",
		"paw.compute.v1.QueryRequestsByProviderRequest",
		"paw.compute.v1.QueryRequestsByProviderResponse",
		"paw.compute.v1.QueryRequestsByStatusRequest",
		"paw.compute.v1.QueryRequestsByStatusResponse",
		"paw.compute.v1.QueryResultRequest",
		"paw.compute.v1.QueryResultResponse",
		"paw.compute.v1.QueryEstimateCostRequest",
		"paw.compute.v1.QueryEstimateCostResponse",
		"paw.compute.v1.QueryDisputeRequest",
		"paw.compute.v1.QueryDisputeResponse",
		"paw.compute.v1.QueryDisputesRequest",
		"paw.compute.v1.QueryDisputesResponse",
		"paw.compute.v1.QueryDisputesByRequestRequest",
		"paw.compute.v1.QueryDisputesByRequestResponse",
		"paw.compute.v1.QueryDisputesByStatusRequest",
		"paw.compute.v1.QueryDisputesByStatusResponse",
		"paw.compute.v1.QueryEvidenceRequest",
		"paw.compute.v1.QueryEvidenceResponse",
		"paw.compute.v1.QuerySlashRecordRequest",
		"paw.compute.v1.QuerySlashRecordResponse",
		"paw.compute.v1.QuerySlashRecordsRequest",
		"paw.compute.v1.QuerySlashRecordsResponse",
		"paw.compute.v1.QuerySlashRecordsByProviderRequest",
		"paw.compute.v1.QuerySlashRecordsByProviderResponse",
		"paw.compute.v1.QueryAppealRequest",
		"paw.compute.v1.QueryAppealResponse",
		"paw.compute.v1.QueryAppealsRequest",
		"paw.compute.v1.QueryAppealsResponse",
		"paw.compute.v1.QueryAppealsByStatusRequest",
		"paw.compute.v1.QueryAppealsByStatusResponse",
		"paw.compute.v1.QueryGovernanceParamsRequest",
		"paw.compute.v1.QueryGovernanceParamsResponse",
	}
	computeZKProtoTypes = []string{
		"paw.compute.v1.ZKProof",
		"paw.compute.v1.VerifyingKey",
		"paw.compute.v1.CircuitParams",
		"paw.compute.v1.ZKMetrics",
		"paw.compute.v1.ProofGenerationMetadata",
	}

	dexProtoTypes = []string{
		"paw.dex.v1.MsgCreatePool",
		"paw.dex.v1.MsgCreatePoolResponse",
		"paw.dex.v1.MsgAddLiquidity",
		"paw.dex.v1.MsgAddLiquidityResponse",
		"paw.dex.v1.MsgRemoveLiquidity",
		"paw.dex.v1.MsgRemoveLiquidityResponse",
		"paw.dex.v1.MsgSwap",
		"paw.dex.v1.MsgSwapResponse",
		"paw.dex.v1.Params",
		"paw.dex.v1.AuthorizedChannel",
		"paw.dex.v1.Pool",
		"paw.dex.v1.PoolTWAP",
		"paw.dex.v1.LimitOrder",
		"paw.dex.v1.CircuitBreakerStateExport",
		"paw.dex.v1.LiquidityPositionExport",
		"paw.dex.v1.GenesisState",
		"paw.dex.v1.QueryParamsRequest",
		"paw.dex.v1.QueryParamsResponse",
		"paw.dex.v1.QueryPoolRequest",
		"paw.dex.v1.QueryPoolResponse",
		"paw.dex.v1.QueryPoolsRequest",
		"paw.dex.v1.QueryPoolsResponse",
		"paw.dex.v1.QueryPoolByTokensRequest",
		"paw.dex.v1.QueryPoolByTokensResponse",
		"paw.dex.v1.QueryLiquidityRequest",
		"paw.dex.v1.QueryLiquidityResponse",
		"paw.dex.v1.QuerySimulateSwapRequest",
		"paw.dex.v1.QuerySimulateSwapResponse",
		"paw.dex.v1.QueryLimitOrderRequest",
		"paw.dex.v1.QueryLimitOrderResponse",
		"paw.dex.v1.QueryLimitOrdersRequest",
		"paw.dex.v1.QueryLimitOrdersResponse",
		"paw.dex.v1.QueryLimitOrdersByOwnerRequest",
		"paw.dex.v1.QueryLimitOrdersByOwnerResponse",
		"paw.dex.v1.QueryLimitOrdersByPoolRequest",
		"paw.dex.v1.QueryLimitOrdersByPoolResponse",
		"paw.dex.v1.QueryOrderBookRequest",
		"paw.dex.v1.QueryOrderBookResponse",
	}

	oracleProtoTypes = []string{
		"paw.oracle.v1.MsgSubmitPrice",
		"paw.oracle.v1.MsgSubmitPriceResponse",
		"paw.oracle.v1.MsgDelegateFeedConsent",
		"paw.oracle.v1.MsgDelegateFeedConsentResponse",
		"paw.oracle.v1.MsgUpdateParams",
		"paw.oracle.v1.MsgUpdateParamsResponse",
		"paw.oracle.v1.Params",
		"paw.oracle.v1.AuthorizedChannel",
		"paw.oracle.v1.Price",
		"paw.oracle.v1.ValidatorPrice",
		"paw.oracle.v1.ValidatorOracle",
		"paw.oracle.v1.PriceSnapshot",
		"paw.oracle.v1.GenesisState",
		"paw.oracle.v1.CircuitBreakerState",
		"paw.oracle.v1.QueryPriceRequest",
		"paw.oracle.v1.QueryPriceResponse",
		"paw.oracle.v1.QueryPricesRequest",
		"paw.oracle.v1.QueryPricesResponse",
		"paw.oracle.v1.QueryValidatorRequest",
		"paw.oracle.v1.QueryValidatorResponse",
		"paw.oracle.v1.QueryValidatorsRequest",
		"paw.oracle.v1.QueryValidatorsResponse",
		"paw.oracle.v1.QueryValidatorPriceRequest",
		"paw.oracle.v1.QueryValidatorPriceResponse",
		"paw.oracle.v1.QueryParamsRequest",
		"paw.oracle.v1.QueryParamsResponse",
	}
)

func TestModuleProtoEncodingPrimitives(t *testing.T) {
	t.Parallel()

	computeProtoTypes := append([]string{}, computeTxProtoTypes...)
	computeProtoTypes = append(computeProtoTypes, computeStateProtoTypes...)
	computeProtoTypes = append(computeProtoTypes, computeQueryProtoTypes...)
	computeProtoTypes = append(computeProtoTypes, computeZKProtoTypes...)

	modules := map[string][]string{
		"compute": computeProtoTypes,
		"dex":     dexProtoTypes,
		"oracle":  oracleProtoTypes,
	}

	for module, typeNames := range modules {
		module := module
		typeNames := typeNames
		t.Run(module, func(t *testing.T) {
			for _, typeName := range typeNames {
				typeName := typeName
				t.Run(typeName, func(t *testing.T) {
					rt := proto.MessageType(typeName)
					require.NotNil(t, rt, "proto type %s missing registration", typeName)

					msg := reflect.New(rt.Elem()).Interface().(proto.Message)
					bz, err := proto.Marshal(msg)
					require.NoError(t, err, "marshal %s", typeName)

					decoded := reflect.New(rt.Elem()).Interface().(proto.Message)
					require.NoError(t, proto.Unmarshal(bz, decoded), "unmarshal %s", typeName)

					reMarshaled, err := proto.Marshal(decoded)
					require.NoError(t, err, "re-marshal %s", typeName)
					require.True(t, bytes.Equal(bz, reMarshaled), "round-trip mismatch for %s", typeName)
				})
			}
		})
	}
}
