package compute

import (
	"fmt"
	"sort"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v8/modules/core/05-port/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	ibcexported "github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

var (
	_ porttypes.IBCModule = (*IBCModule)(nil)
)

// IBCModule implements the ICS26 interface for the compute module.
// This enables cross-chain compute job distribution and result verification.
type IBCModule struct {
	keeper keeper.Keeper
	cdc    codec.Codec
}

const maxProvidersPerAck = 50

// NewIBCModule creates a new IBCModule given the keeper and codec
func NewIBCModule(keeper keeper.Keeper, cdc codec.Codec) IBCModule {
	return IBCModule{
		keeper: keeper,
		cdc:    cdc,
	}
}

// OnChanOpenInit implements the IBCModule interface
// Validates the channel creation for compute operations
func (im IBCModule) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	// Validate channel ordering - compute requires ordered channels for job sequencing
	if order != channeltypes.ORDERED {
		return "", errorsmod.Wrapf(channeltypes.ErrInvalidChannelOrdering,
			"expected %s channel, got %s", channeltypes.ORDERED, order)
	}

	// Validate version
	if version != types.IBCVersion {
		return "", errorsmod.Wrapf(types.ErrInvalidPacket,
			"expected version %s, got %s", types.IBCVersion, version)
	}

	// Validate port
	if portID != types.PortID {
		return "", errorsmod.Wrapf(porttypes.ErrInvalidPort,
			"expected port %s, got %s", types.PortID, portID)
	}

	if err := im.keeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", errorsmod.Wrap(err, "failed to claim channel capability")
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChannelOpen,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyPortID, portID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyPortID, counterparty.PortId),
			sdk.NewAttribute(types.AttributeKeyCounterpartyChannelID, counterparty.ChannelId),
		),
	)

	return version, nil
}

// OnChanOpenTry implements the IBCModule interface
func (im IBCModule) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	// Validate channel ordering
	if order != channeltypes.ORDERED {
		return "", errorsmod.Wrapf(channeltypes.ErrInvalidChannelOrdering,
			"expected %s channel, got %s", channeltypes.ORDERED, order)
	}

	// Validate version
	if counterpartyVersion != types.IBCVersion {
		return "", errorsmod.Wrapf(types.ErrInvalidPacket,
			"invalid counterparty version: expected %s, got %s", types.IBCVersion, counterpartyVersion)
	}
	if portID != types.PortID {
		return "", errorsmod.Wrapf(porttypes.ErrInvalidPort,
			"expected port %s, got %s", types.PortID, portID)
	}

	if err := im.keeper.ClaimCapability(ctx, chanCap, host.ChannelCapabilityPath(portID, channelID)); err != nil {
		return "", errorsmod.Wrap(err, "failed to claim channel capability")
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChannelOpen,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyPortID, portID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyPortID, counterparty.PortId),
			sdk.NewAttribute(types.AttributeKeyCounterpartyChannelID, counterparty.ChannelId),
		),
	)

	return types.IBCVersion, nil
}

// OnChanOpenAck implements the IBCModule interface
func (im IBCModule) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	// Validate counterparty version
	if counterpartyVersion != types.IBCVersion {
		return errorsmod.Wrapf(types.ErrInvalidPacket,
			"invalid counterparty version: expected %s, got %s", types.IBCVersion, counterpartyVersion)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChannelOpenAck,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyPortID, portID),
			sdk.NewAttribute(types.AttributeKeyCounterpartyChannelID, counterpartyChannelID),
		),
	)

	return nil
}

// OnChanOpenConfirm implements the IBCModule interface
func (im IBCModule) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChannelOpenConfirm,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyPortID, portID),
		),
	)

	return nil
}

// OnChanCloseInit implements the IBCModule interface
func (im IBCModule) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Disallow user-initiated channel closing for compute
	return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "user cannot close channel")
}

// OnChanCloseConfirm implements the IBCModule interface
func (im IBCModule) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	pending := im.keeper.GetPendingOperations(ctx, channelID)
	refunds := 0
	for _, op := range pending {
		if err := im.keeper.RefundOnChannelClose(ctx, op); err != nil {
			ctx.Logger().Error("failed to cleanup compute channel operation",
				"channel", channelID,
				"sequence", op.Sequence,
				"type", op.PacketType,
				"error", err,
			)
			continue
		}
		refunds++
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeChannelClose,
			sdk.NewAttribute(types.AttributeKeyChannelID, channelID),
			sdk.NewAttribute(types.AttributeKeyPortID, portID),
			sdk.NewAttribute(types.AttributeKeyPendingOperations, fmt.Sprintf("%d", refunds)),
		),
	)

	return nil
}

// OnRecvPacket implements the IBCModule interface
// Handles incoming compute packets (job results, provider info, etc.)
func (im IBCModule) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	if !im.keeper.IsAuthorizedChannel(ctx, packet.SourcePort, packet.SourceChannel) {
		err := errorsmod.Wrapf(types.ErrUnauthorizedChannel, "port %s channel %s not authorized", packet.SourcePort, packet.SourceChannel)
		ctx.Logger().Error("unauthorized compute packet source", "port", packet.SourcePort, "channel", packet.SourceChannel)
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"compute_ibc_packet_validation_failed",
				sdk.NewAttribute("port", packet.SourcePort),
				sdk.NewAttribute("channel", packet.SourceChannel),
				sdk.NewAttribute("reason", "unauthorized channel"),
			),
		)
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// Parse packet data
	packetData, err := types.ParsePacketData(packet.Data)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(
			errorsmod.Wrapf(types.ErrInvalidPacket, "failed to parse packet data: %s", err.Error()))
	}

	// Validate packet
	if err := packetData.ValidateBasic(); err != nil {
		return channeltypes.NewErrorAcknowledgement(
			errorsmod.Wrap(types.ErrInvalidPacket, err.Error()))
	}

	packetNonce := im.packetNonce(packetData)
	if packetNonce == 0 {
		return channeltypes.NewErrorAcknowledgement(errorsmod.Wrap(types.ErrInvalidPacket, "packet nonce missing"))
	}

	packetTimestamp := im.packetTimestamp(packetData)
	if packetTimestamp == 0 {
		return channeltypes.NewErrorAcknowledgement(errorsmod.Wrap(types.ErrInvalidPacket, "packet timestamp missing"))
	}

	sender := im.packetSender(packet, packetData)
	if err := im.keeper.ValidateIncomingPacketNonce(ctx, packet.SourceChannel, sender, packetNonce, packetTimestamp); err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// Route packet based on type
	var ack ibcexported.Acknowledgement
	switch packetData.GetType() {
	case types.JobResultType:
		// Handle job result from remote provider
		ack = im.handleJobResult(ctx, packet, packetData, packetNonce)

	case types.DiscoverProvidersType:
		// Handle provider discovery request from remote chain
		ack = im.handleDiscoverProviders(ctx, packet, packetData, packetNonce)

	case types.SubmitJobType:
		// Handle job submission from remote requester
		ack = im.handleSubmitJob(ctx, packet, packetData, packetNonce)

	case types.JobStatusType:
		// Handle job status query
		ack = im.handleJobStatusQuery(ctx, packet, packetData, packetNonce)

	default:
		return channeltypes.NewErrorAcknowledgement(
			errorsmod.Wrapf(types.ErrInvalidPacket, "unknown packet type: %s", packetData.GetType()))
	}

	// Emit receive event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePacketReceive,
			sdk.NewAttribute(types.AttributeKeyPacketType, packetData.GetType()),
			sdk.NewAttribute(types.AttributeKeyChannelID, packet.DestinationChannel),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
		),
	)

	return ack
}

// OnAcknowledgementPacket implements the IBCModule interface
// Handles acknowledgements for sent packets
func (im IBCModule) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	// Security: Limit acknowledgement size to prevent DoS attacks via large payloads
	// Set to 256KB (262144 bytes) - sufficient for legitimate use cases
	// Note: Consider implementing rate limiting at the relayer level for additional protection
	const maxAcknowledgementSize = 256 * 1024 // 256KB guard rail
	if len(acknowledgement) > maxAcknowledgementSize {
		return errorsmod.Wrapf(types.ErrInvalidAck, "ack too large: %d > %d", len(acknowledgement), maxAcknowledgementSize)
	}

	var ack channeltypes.Acknowledgement
	if err := channeltypes.SubModuleCdc.UnmarshalJSON(acknowledgement, &ack); err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrUnknownRequest,
			"cannot unmarshal packet acknowledgement: %v", err)
	}

	// Delegate to keeper's acknowledgement handler
	if err := im.keeper.OnAcknowledgementPacket(ctx, packet, ack); err != nil {
		return err
	}

	// Emit acknowledgement event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePacketAck,
			sdk.NewAttribute(types.AttributeKeyChannelID, packet.SourceChannel),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
			sdk.NewAttribute(types.AttributeKeyAckSuccess, fmt.Sprintf("%t", ack.Success())),
		),
	)

	return nil
}

// OnTimeoutPacket implements the IBCModule interface
// Handles packet timeout (refunds, cleanup, etc.)
func (im IBCModule) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	// Delegate to keeper's timeout handler
	if err := im.keeper.OnTimeoutPacket(ctx, packet); err != nil {
		return err
	}

	// Emit timeout event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypePacketTimeout,
			sdk.NewAttribute(types.AttributeKeyChannelID, packet.SourceChannel),
			sdk.NewAttribute(types.AttributeKeySequence, fmt.Sprintf("%d", packet.Sequence)),
		),
	)

	return nil
}

// Helper functions for packet handling

func (im IBCModule) handleJobResult(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
	nonce uint64,
) ibcexported.Acknowledgement {
	// Delegate to keeper
	ack, err := im.keeper.OnRecvPacket(ctx, packet, nonce)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}
	return ack
}

func (im IBCModule) handleDiscoverProviders(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
	nonce uint64,
) ibcexported.Acknowledgement {
	req, _ := packetData.(types.DiscoverProvidersPacketData)

	providers := im.discoverActiveProviders(ctx, req.Capabilities, req.MaxPrice)
	if len(providers) == 0 {
		return channeltypes.NewErrorAcknowledgement(errorsmod.Wrap(types.ErrInvalidPacket, "no providers match requested capabilities"))
	}

	total := types.SaturateIntToUint32(len(providers))
	if len(providers) > maxProvidersPerAck {
		providers = providers[:maxProvidersPerAck]
	}

	ackData := types.DiscoverProvidersAcknowledgement{
		Nonce:          nonce,
		Success:        true,
		Providers:      providers,
		TotalProviders: total,
	}

	ackBytes, err := ackData.GetBytes()
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement(ackBytes)
}

func (im IBCModule) handleSubmitJob(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
	nonce uint64,
) ibcexported.Acknowledgement {
	job, _ := packetData.(types.SubmitJobPacketData)

	if err := job.ValidateBasic(); err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	req := job.Requirements
	record := keeper.CrossChainComputeJob{
		JobID:       job.JobID,
		SourceChain: packet.SourceChannel,
		TargetChain: ctx.ChainID(),
		Provider:    job.Provider,
		Requester:   job.Requester,
		JobType:     job.JobType,
		JobData:     job.JobData,
		Requirements: keeper.JobRequirements{
			CPUCores:    req.CPUCores,
			MemoryMB:    req.MemoryMB,
			StorageGB:   req.StorageGB,
			GPURequired: req.GPURequired,
			TEERequired: req.TEERequired,
			MaxDuration: types.SecondsToDuration(req.MaxDuration),
		},
		Status:      "running",
		SubmittedAt: ctx.BlockTime(),
	}
	im.keeper.UpsertCrossChainJob(ctx, &record)

	ackData := types.SubmitJobAcknowledgement{
		Nonce:         nonce,
		Success:       true,
		JobID:         job.JobID,
		Status:        record.Status,
		Progress:      record.Progress,
		EstimatedTime: uint64(req.MaxDuration),
	}
	if ackData.EstimatedTime == 0 && req.MaxDuration == 0 {
		ackData.EstimatedTime = 300
	}

	ackBytes, err := ackData.GetBytes()
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement(ackBytes)
}

func (im IBCModule) handleJobStatusQuery(
	ctx sdk.Context,
	packet channeltypes.Packet,
	packetData types.IBCPacketData,
	nonce uint64,
) ibcexported.Acknowledgement {
	statusReq, _ := packetData.(types.JobStatusPacketData)

	if err := statusReq.ValidateBasic(); err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	job := im.keeper.GetCrossChainJob(ctx, statusReq.JobID)
	if job == nil {
		return channeltypes.NewErrorAcknowledgement(errorsmod.Wrapf(types.ErrInvalidPacket, "job %s not found", statusReq.JobID))
	}

	jobStatus := job.Status
	progress := job.Progress
	if jobStatus == "" {
		jobStatus = "unknown"
	}
	if progress == 0 {
		switch jobStatus {
		case "completed":
			progress = 100
		case "running":
			progress = 70
		case "submitted", "pending":
			progress = 20
		}
	}

	ackData := types.JobStatusAcknowledgement{
		Nonce:    nonce,
		Success:  true,
		JobID:    statusReq.JobID,
		Status:   jobStatus,
		Progress: progress,
	}

	ackBytes, err := ackData.GetBytes()
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	return channeltypes.NewResultAcknowledgement(ackBytes)
}

func (im IBCModule) discoverActiveProviders(ctx sdk.Context, requestedCaps []string, maxPrice math.LegacyDec) []types.ProviderInfo {
	providerInfos := []types.ProviderInfo{}
	capSet := map[string]struct{}{}
	for _, c := range requestedCaps {
		capSet[c] = struct{}{}
	}

	_ = im.keeper.IterateActiveProviders(ctx, func(provider types.Provider) (bool, error) {
		caps := deriveCapabilities(provider.AvailableSpecs)
		if len(capSet) > 0 && !hasAllCaps(caps, capSet) {
			return false, nil
		}

		price := provider.Pricing.CpuPricePerMcoreHour
		// Prefer GPU price when GPUs are available and requested
		if containsCap(capSet, "gpu") && provider.AvailableSpecs.GpuCount > 0 && !provider.Pricing.GpuPricePerHour.IsZero() {
			price = provider.Pricing.GpuPricePerHour
		}

		if maxPrice.IsPositive() && price.GT(maxPrice) {
			return false, nil
		}

		providerInfos = append(providerInfos, types.ProviderInfo{
			ProviderID:   provider.Moniker,
			Address:      provider.Address,
			Capabilities: caps,
			PricePerUnit: price,
			Reputation:   math.LegacyNewDec(int64(provider.Reputation)),
		})
		return false, nil
	})

	// Sort by reputation desc, then price asc for deterministic pagination.
	sort.Slice(providerInfos, func(i, j int) bool {
		if !providerInfos[i].Reputation.Equal(providerInfos[j].Reputation) {
			return providerInfos[i].Reputation.GT(providerInfos[j].Reputation)
		}
		return providerInfos[i].PricePerUnit.LT(providerInfos[j].PricePerUnit)
	})

	return providerInfos
}

func deriveCapabilities(spec types.ComputeSpec) []string {
	caps := []string{"cpu"}
	if spec.GpuCount > 0 {
		caps = append(caps, "gpu")
	}
	if spec.StorageGb > 0 {
		caps = append(caps, "storage")
	}
	return caps
}

func hasAllCaps(caps []string, required map[string]struct{}) bool {
	if len(required) == 0 {
		return true
	}
	// Allow unknown requested caps like "tee" to pass through.
	allowMissing := map[string]struct{}{"tee": {}}

	has := map[string]struct{}{}
	for _, c := range caps {
		has[c] = struct{}{}
	}
	for req := range required {
		if _, ok := allowMissing[req]; ok {
			continue
		}
		if _, ok := has[req]; !ok {
			return false
		}
	}
	return true
}

func containsCap(req map[string]struct{}, cap string) bool {
	if len(req) == 0 {
		return false
	}
	_, ok := req[cap]
	return ok
}

func (im IBCModule) packetNonce(packetData types.IBCPacketData) uint64 {
	switch req := packetData.(type) {
	case types.DiscoverProvidersPacketData:
		return req.Nonce
	case types.SubmitJobPacketData:
		return req.Nonce
	case types.JobResultPacketData:
		return req.Nonce
	case types.JobStatusPacketData:
		return req.Nonce
	case types.ReleaseEscrowPacketData:
		return req.Nonce
	default:
		return 0
	}
}

func (im IBCModule) packetSender(packet channeltypes.Packet, packetData types.IBCPacketData) string {
	switch req := packetData.(type) {
	case types.DiscoverProvidersPacketData:
		if req.Requester != "" {
			return req.Requester
		}
	case types.SubmitJobPacketData:
		if req.Requester != "" {
			return req.Requester
		}
	case types.JobResultPacketData:
		if req.Provider != "" {
			return req.Provider
		}
	case types.JobStatusPacketData:
		if req.Requester != "" {
			return req.Requester
		}
	case types.ReleaseEscrowPacketData:
		if req.Provider != "" {
			return req.Provider
		}
	}
	if packet.SourcePort != "" {
		return packet.SourcePort
	}
	return packet.SourceChannel
}

func (im IBCModule) packetTimestamp(packetData types.IBCPacketData) int64 {
	switch req := packetData.(type) {
	case types.DiscoverProvidersPacketData:
		return req.Timestamp
	case types.SubmitJobPacketData:
		return req.Timestamp
	case types.JobResultPacketData:
		return req.Timestamp
	case types.JobStatusPacketData:
		return req.Timestamp
	case types.ReleaseEscrowPacketData:
		return req.Timestamp
	default:
		return 0
	}
}
