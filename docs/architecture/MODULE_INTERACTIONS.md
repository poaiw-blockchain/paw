# Module Interaction Diagrams

This document provides sequence diagrams for the primary cross-module flows in PAW blockchain.

## 1. DEX-Oracle Price Flow

Shows how the DEX module queries the Oracle for pricing during swap execution.

```mermaid
sequenceDiagram
    participant User
    participant DEX as DEX Module
    participant Oracle as Oracle Module
    participant Bank as Bank Module

    User->>DEX: MsgSwap(poolID, tokenIn, tokenOut, amountIn, minAmountOut)
    DEX->>DEX: ValidatePoolState(pool)
    DEX->>DEX: CheckPoolCircuitBreaker(poolID)
    DEX->>Oracle: GetPrice(tokenIn)
    Oracle-->>DEX: price, timestamp
    DEX->>DEX: ValidatePriceDeviation(spotPrice, oraclePrice)
    alt Price deviation > threshold
        DEX-->>User: ErrCircuitBreakerTriggered
    else Price within tolerance
        DEX->>DEX: CalculateSwapOutput(amountIn, reserves, fee)
        DEX->>DEX: ValidateSlippage(amountOut, minAmountOut)
        DEX->>Bank: SendCoins(user, module, tokenIn)
        DEX->>Bank: SendCoins(module, user, tokenOut)
        DEX->>DEX: UpdatePoolReserves()
        DEX->>DEX: UpdateTWAP(price0, price1)
        DEX-->>User: SwapResult(amountOut, fee)
    end
```

## 2. Compute Job Flow

Shows the lifecycle of an off-chain compute request from submission to payment.

```mermaid
sequenceDiagram
    participant User as Requester
    participant Compute as Compute Module
    participant Provider as Compute Provider
    participant Bank as Bank Module

    User->>Compute: MsgSubmitRequest(specs, maxPayment)
    Compute->>Compute: ValidateComputeSpecs(specs)
    Compute->>Compute: FindSuitableProvider(specs)
    Compute->>Compute: EstimateCost(provider, specs)
    Compute->>Bank: EscrowPayment(user, maxPayment)
    Compute->>Compute: CreateRequest(status=ASSIGNED)
    Compute-->>User: requestID

    Note over Provider: Provider monitors chain for assignments

    Provider->>Provider: ExecuteComputation(containerImage, command)
    Provider->>Compute: MsgSubmitResult(requestID, outputHash, proof)
    Compute->>Compute: ValidateRequestState()
    Compute->>Compute: ReserveNonce(provider, proof.nonce)
    Compute->>Compute: VerifyProofSignature(proof)
    Compute->>Compute: VerifyResultIntegrity(outputHash)

    alt Verification Passed
        Compute->>Compute: UpdateProviderReputation(+)
        Compute->>Bank: ReleaseEscrow(provider)
        Compute-->>Provider: PaymentReleased
    else Verification Failed
        Compute->>Compute: UpdateProviderReputation(-)
        Compute->>Bank: RefundEscrow(user)
        Compute-->>User: PaymentRefunded
    end
```

## 3. Oracle Aggregation Flow

Shows how validators submit prices and the consensus mechanism aggregates them.

```mermaid
sequenceDiagram
    participant V1 as Validator 1
    participant V2 as Validator 2
    participant V3 as Validator 3
    participant Oracle as Oracle Module
    participant Staking as Staking Module

    Note over Oracle: BeginBlocker - Check vote period start
    Oracle->>Staking: GetBondedValidators()
    Staking-->>Oracle: validators[]
    Oracle->>Oracle: SnapshotVotingPowers()

    par Validator Price Submissions
        V1->>Oracle: MsgSubmitPrice(asset, price1, signature)
        V2->>Oracle: MsgSubmitPrice(asset, price2, signature)
        V3->>Oracle: MsgSubmitPrice(asset, price3, signature)
    end

    Oracle->>Oracle: ValidatePriceSignatures()
    Oracle->>Oracle: GetSnapshotVotingPower(validator)

    Note over Oracle: EndBlocker - Aggregate prices
    Oracle->>Oracle: AggregateAssetPrice(asset)
    Oracle->>Oracle: DetectAndFilterOutliers(prices)
    Oracle->>Oracle: CalculateMAD(prices, median)
    Oracle->>Oracle: CalculateIQR(prices)

    alt Outlier Detected
        Oracle->>Oracle: ClassifyOutlierSeverity()
        Oracle->>Oracle: HandleOutlierSlashing(validator)
    end

    Oracle->>Oracle: CalculateWeightedMedian(validPrices)
    Oracle->>Oracle: SetPrice(asset, aggregatedPrice)
    Oracle->>Oracle: SetPriceSnapshot(asset, price, height)
    Oracle->>Oracle: CheckMissedVotes(asset)
```

## 4. IBC Cross-Chain Swap Flow

Shows how cross-chain swaps are handled via IBC packet transmission.

```mermaid
sequenceDiagram
    participant User as User (Chain A)
    participant DEXA as DEX Module (Chain A)
    participant IBC as IBC Core
    participant DEXB as DEX Module (Chain B)
    participant Relayer

    User->>DEXA: MsgInitiateIBCSwap(channel, tokenIn, tokenOut, amount)
    DEXA->>DEXA: ValidateChannel(channelID)
    DEXA->>DEXA: EscrowTokens(user, tokenIn, amount)
    DEXA->>DEXA: GeneratePacketNonce()
    DEXA->>IBC: SendPacket(CrossChainSwapPacket)
    IBC-->>DEXA: packetSequence

    Relayer->>IBC: RelayPacket(packet)
    IBC->>DEXB: OnRecvPacket(packet)
    DEXB->>DEXB: ValidatePacketNonce(nonce)
    DEXB->>DEXB: ValidatePacketTimestamp()
    DEXB->>DEXB: ExecuteSwapFromIBC(poolID, tokenIn, tokenOut)

    alt Swap Successful
        DEXB->>DEXB: MintOrTransferTokens(recipient)
        DEXB-->>IBC: ResultAcknowledgement(success, amountOut)
    else Swap Failed
        DEXB-->>IBC: ErrorAcknowledgement(reason)
    end

    Relayer->>IBC: RelayAck(acknowledgement)
    IBC->>DEXA: OnAcknowledgementPacket(packet, ack)

    alt ACK Success
        DEXA->>DEXA: CleanupPendingOperation()
        DEXA-->>User: SwapComplete(amountOut)
    else ACK Error
        DEXA->>DEXA: RefundEscrowedTokens(user)
        DEXA-->>User: SwapRefunded(reason)
    end

    Note over DEXA,DEXB: Timeout Handling
    alt Packet Timeout
        IBC->>DEXA: OnTimeoutPacket(packet)
        DEXA->>DEXA: RefundOnTimeout(user, amount)
        DEXA-->>User: TimeoutRefund(amount)
    end
```

## Module Dependency Summary

| Source Module | Target Module | Interaction Type |
|---------------|---------------|------------------|
| DEX | Oracle | Price queries for circuit breaker |
| DEX | Bank | Token transfers, escrow |
| Oracle | Staking | Validator voting power |
| Compute | Bank | Payment escrow/release |
| All | IBC Core | Cross-chain packet handling |

## Event Flow

All modules emit events for indexing and monitoring:

- **DEX**: `dex_swap`, `dex_pool_created`, `dex_liquidity_added`
- **Oracle**: `oracle_price_aggregated`, `oracle_outlier`, `oracle_fallback`
- **Compute**: `request_submitted`, `result_submitted`, `request_completed`
- **IBC**: `ibc_packet_receive`, `ibc_packet_ack`, `ibc_packet_timeout`
