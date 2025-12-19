package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/paw-chain/paw/x/oracle/types"
)

type queryServer struct {
	Keeper
}

const (
	defaultPaginationLimit = 100
	maxPaginationLimit     = 1000
)

// NewQueryServerImpl returns an implementation of the QueryServer interface
func NewQueryServerImpl(keeper Keeper) types.QueryServer {
	return &queryServer{Keeper: keeper}
}

var _ types.QueryServer = queryServer{}

// sanitizePagination enforces sensible defaults and caps for paginated queries.
func sanitizePagination(p *query.PageRequest) *query.PageRequest {
	if p == nil {
		return &query.PageRequest{Limit: defaultPaginationLimit}
	}

	if p.Limit == 0 {
		p.Limit = defaultPaginationLimit
	}

	if p.Limit > maxPaginationLimit {
		p.Limit = maxPaginationLimit
	}

	return p
}

// Price queries the current price for a specific asset
func (qs queryServer) Price(goCtx context.Context, req *types.QueryPriceRequest) (*types.QueryPriceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.Asset == "" {
		return nil, status.Error(codes.InvalidArgument, "asset cannot be empty")
	}

	price, err := qs.GetPrice(goCtx, req.Asset)
	if err != nil {
		return nil, err
	}

	return &types.QueryPriceResponse{Price: &price}, nil
}

// Prices queries all current prices
func (qs queryServer) Prices(goCtx context.Context, req *types.QueryPricesRequest) (*types.QueryPricesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := ctx.KVStore(qs.storeKey)
	priceStore := prefix.NewStore(store, PriceKeyPrefix)

	var prices []types.Price
	pageRes, err := query.Paginate(priceStore, sanitizePagination(req.Pagination), func(key []byte, value []byte) error {
		var price types.Price
		if err := qs.cdc.Unmarshal(value, &price); err != nil {
			return err
		}
		prices = append(prices, price)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryPricesResponse{
		Prices:     prices,
		Pagination: pageRes,
	}, nil
}

// Validator queries oracle validator information
func (qs queryServer) Validator(goCtx context.Context, req *types.QueryValidatorRequest) (*types.QueryValidatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.ValidatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "validator address cannot be empty")
	}

	validatorAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid validator address: %s", err))
	}

	validatorOracle, err := qs.GetValidatorOracle(goCtx, validatorAddr.String())
	if err != nil {
		return nil, err
	}

	return &types.QueryValidatorResponse{Validator: &validatorOracle}, nil
}

// Validators queries all oracle validators
func (qs queryServer) Validators(goCtx context.Context, req *types.QueryValidatorsRequest) (*types.QueryValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := ctx.KVStore(qs.storeKey)
	validatorStore := prefix.NewStore(store, ValidatorOracleKeyPrefix)

	var validators []types.ValidatorOracle
	pageRes, err := query.Paginate(validatorStore, sanitizePagination(req.Pagination), func(key []byte, value []byte) error {
		var validator types.ValidatorOracle
		if err := qs.cdc.Unmarshal(value, &validator); err != nil {
			return err
		}
		validators = append(validators, validator)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryValidatorsResponse{
		Validators: validators,
		Pagination: pageRes,
	}, nil
}

// ValidatorPrice queries a validator's submitted price for an asset
func (qs queryServer) ValidatorPrice(goCtx context.Context, req *types.QueryValidatorPriceRequest) (*types.QueryValidatorPriceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.ValidatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "validator address cannot be empty")
	}

	if req.Asset == "" {
		return nil, status.Error(codes.InvalidArgument, "asset cannot be empty")
	}

	validatorAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid validator address: %s", err))
	}

	validatorPrice, err := qs.GetValidatorPrice(goCtx, validatorAddr, req.Asset)
	if err != nil {
		return nil, err
	}

	return &types.QueryValidatorPriceResponse{ValidatorPrice: &validatorPrice}, nil
}

// Params queries the oracle module parameters
func (qs queryServer) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	params, err := qs.GetParams(goCtx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: params}, nil
}
