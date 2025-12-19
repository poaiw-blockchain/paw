import React, { useEffect, useMemo, useRef, useState } from 'react';
import type { PriceData } from '@paw-chain/wallet-core';
import { DexService, SwapQuote, DenomMetadata, PoolAnalytics } from '../../services/dex';

const SLIPPAGE_PRESETS = [0.1, 0.5, 1, 2];

interface WalletData {
  address: string;
  publicKey?: string;
  type?: string;
}

interface SwapInterfaceProps {
  walletData: WalletData | null;
  service?: DexService;
}

const SwapInterface: React.FC<SwapInterfaceProps> = ({ walletData, service }) => {
  const dexService = useMemo(() => service ?? new DexService(), [service]);
  const isLedger = walletData?.type === 'ledger';
  const [availableTokens, setAvailableTokens] = useState<DenomMetadata[]>([]);
  const [tokenIn, setTokenIn] = useState('');
  const [tokenOut, setTokenOut] = useState('');
  const [amountIn, setAmountIn] = useState('');
  const [slippage, setSlippage] = useState(0.5);
  const [password, setPassword] = useState('');
  const [memo, setMemo] = useState('');
  const [quote, setQuote] = useState<SwapQuote | null>(null);
  const [isLoadingTokens, setIsLoadingTokens] = useState(false);
  const [isQuoting, setIsQuoting] = useState(false);
  const [isSwapping, setIsSwapping] = useState(false);
  const [quoteError, setQuoteError] = useState('');
  const [statusMessage, setStatusMessage] = useState('');
  const [executionError, setExecutionError] = useState('');
  const [oraclePrices, setOraclePrices] = useState<Record<string, PriceData | undefined>>({});
  const [lastTxHash, setLastTxHash] = useState('');
  const [poolAnalytics, setPoolAnalytics] = useState<PoolAnalytics | null>(null);
  const [poolInsightError, setPoolInsightError] = useState('');
  const [showRouteInsights, setShowRouteInsights] = useState(true);
  const quoteRequestRef = useRef(0);

  useEffect(() => {
    loadTokens();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dexService]);

  useEffect(() => {
    if (tokenIn) {
      hydrateOraclePrice(tokenIn);
    }
  }, [tokenIn, dexService]);

  useEffect(() => {
    if (tokenOut) {
      hydrateOraclePrice(tokenOut);
    }
  }, [tokenOut, dexService]);

  useEffect(() => {
    if (!tokenIn || !tokenOut || tokenIn === tokenOut) {
      setPoolAnalytics(null);
      setPoolInsightError('');
      return;
    }

    let cancelled = false;
    setPoolInsightError('');

    dexService
      .getPoolAnalytics(tokenIn, tokenOut)
      .then((analytics) => {
        if (!cancelled) {
          setPoolAnalytics(analytics);
        }
      })
      .catch((error: any) => {
        if (!cancelled) {
          setPoolAnalytics(null);
          setPoolInsightError(error?.message || 'Unable to load route diagnostics');
        }
      });

    return () => {
      cancelled = true;
    };
  }, [tokenIn, tokenOut, dexService]);

  useEffect(() => {
    if (!tokenIn || !tokenOut || !amountIn || tokenIn === tokenOut) {
      setQuote(null);
      return;
    }

    const debounce = setTimeout(() => {
      requestQuote();
    }, 400);

    return () => clearTimeout(debounce);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tokenIn, tokenOut, amountIn, slippage, dexService]);

  const tokenOutOptions = useMemo(() => {
    return availableTokens.filter((token) => token.denom !== tokenIn);
  }, [availableTokens, tokenIn]);

  const expectedUsdValue = useMemo(() => {
    if (!quote) {
      return null;
    }
    const price = oraclePrices[quote.tokenOut.denom]?.price;
    if (!price) {
      return null;
    }
    const usd = parseFloat(price) * parseFloat(quote.expectedAmountOut);
    if (!Number.isFinite(usd)) {
      return null;
    }
    return usd;
  }, [quote, oraclePrices]);

  const oracleSpread = useMemo(() => {
    if (!quote) {
      return null;
    }
    const inPrice = oraclePrices[quote.tokenIn.denom]?.price;
    const outPrice = oraclePrices[quote.tokenOut.denom]?.price;
    if (!inPrice || !outPrice) {
      return null;
    }
    const oracleExecution = parseFloat(outPrice) / parseFloat(inPrice);
    if (!Number.isFinite(oracleExecution) || oracleExecution === 0) {
      return null;
    }
    const execution = parseFloat(quote.executionPrice);
    if (!Number.isFinite(execution) || execution === 0) {
      return null;
    }
    return ((execution - oracleExecution) / oracleExecution) * 100;
  }, [quote, oraclePrices]);

  const quoteFreshness = useMemo(() => {
    if (!quote) {
      return null;
    }
    const delta = Date.now() - quote.updatedAt;
    if (delta < 0) {
      return '<1s';
    }
    if (delta < 60_000) {
      return `${Math.max(1, Math.round(delta / 1000))}s`;
    }
    return `${Math.round(delta / 60_000)}m`;
  }, [quote]);

  const riskLevel = useMemo(() => {
    if (!quote) {
      return 'idle';
    }
    if (quote.priceImpactPercent > 5 || (oracleSpread !== null && Math.abs(oracleSpread) > 5)) {
      return 'high';
    }
    if (quote.priceImpactPercent > 2) {
      return 'medium';
    }
    return 'low';
  }, [quote, oracleSpread]);

  const canSwap = Boolean(
    quote &&
      (!isLedger ? password.length >= 8 : true) &&
      !isSwapping &&
      !isQuoting &&
      amountIn &&
      tokenIn &&
      tokenOut &&
      tokenIn !== tokenOut &&
      !isLedger // disable until ledger swap signing supported
  );

  async function loadTokens() {
    try {
      setIsLoadingTokens(true);
      const tokens = await dexService.getTradableTokens();
      setAvailableTokens(tokens);

      if (tokens.length > 0) {
        setTokenIn(tokens[0].denom);
        const fallback = tokens.find((t) => t.denom !== tokens[0].denom);
        setTokenOut(fallback ? fallback.denom : tokens[0].denom);
      }
    } catch (error: any) {
      setExecutionError(error?.message || 'Failed to load token metadata.');
    } finally {
      setIsLoadingTokens(false);
    }
  }

  async function hydrateOraclePrice(denom: string) {
    try {
      const price = await dexService.getOraclePrice(denom);
      setOraclePrices((prev) => ({
        ...prev,
        [denom]: price,
      }));
    } catch {
      // Ignore oracle failures, UI will simply hide USD projections
    }
  }

  async function requestQuote() {
    const requestId = ++quoteRequestRef.current;
    setIsQuoting(true);
    setQuoteError('');

    try {
      const result = await dexService.quoteSwap({
        tokenIn,
        tokenOut,
        amountIn,
        slippagePercent: slippage,
      });

      if (requestId === quoteRequestRef.current) {
        setQuote(result);
      }
    } catch (error: any) {
      if (requestId === quoteRequestRef.current) {
        setQuote(null);
        setQuoteError(error?.message || 'Failed to build swap quote');
      }
    } finally {
      if (requestId === quoteRequestRef.current) {
        setIsQuoting(false);
      }
    }
  }

  const handleSwapTokens = () => {
    if (!tokenIn || !tokenOut) {
      return;
    }
    setTokenIn(tokenOut);
    setTokenOut(tokenIn);
  };

  const handleSwap = async () => {
    if (!quote || !canSwap) {
      return;
    }
    if (isLedger) {
      setExecutionError('Ledger swaps are disabled until custom DEX signing support is wired. Use software wallet for now.');
      return;
    }

    setIsSwapping(true);
    setExecutionError('');
    setStatusMessage('');

    try {
      const result = await dexService.executeSwap({
        tokenIn,
        tokenOut,
        amountIn,
        slippagePercent: slippage,
        password,
        memo: memo.trim() || undefined,
      });

      setStatusMessage(`Swap broadcast successfully (tx ${result.transactionHash})`);
      setLastTxHash(result.transactionHash);
      setAmountIn('');
      setPassword('');
      setMemo('');
    } catch (error: any) {
      setExecutionError(error?.message || 'Swap failed');
    } finally {
      setIsSwapping(false);
    }
  };

  return (
    <div className="content dex-content">
      <div className="dex-card">
        <div className="dex-card-header">
          <div>
            <h3>Protocol DEX Router</h3>
            <p>Route swaps directly against on-chain liquidity with deterministic slippage controls.</p>
          </div>
          {walletData?.address && (
            <div className="dex-identity">
              <span>Trading as</span>
              <strong>{walletData.address}</strong>
            </div>
          )}
        </div>

        <div className="dex-form-grid">
          <div className="dex-form-section">
            <label className="form-label">You Pay</label>
            <div className="dex-input-row">
              <input
                type="number"
                min="0"
                step="any"
                className="form-input dex-amount-input"
                value={amountIn}
                onChange={(e) => setAmountIn(e.target.value)}
                placeholder="0.0"
                disabled={isLoadingTokens}
              />
              <select
                className="form-input dex-token-select"
                value={tokenIn}
                onChange={(e) => setTokenIn(e.target.value)}
                disabled={isLoadingTokens}
                data-testid="token-in-select"
              >
                {availableTokens.map((token) => (
                  <option key={token.denom} value={token.denom}>
                    {token.symbol} ({token.denom})
                  </option>
                ))}
              </select>
            </div>
          </div>

          <button
            type="button"
            className="btn btn-secondary dex-switch"
            onClick={handleSwapTokens}
            disabled={isLoadingTokens || availableTokens.length < 2}
          >
            ⇅
          </button>

          <div className="dex-form-section">
            <label className="form-label">You Receive</label>
            <div className="dex-input-row">
              <input
                className="form-input dex-amount-input"
                value={quote ? `${quote.expectedAmountOut} ${quote.tokenOut.symbol}` : ''}
                placeholder="Auto calculated"
                readOnly
              />
              <select
                className="form-input dex-token-select"
                value={tokenOut}
                onChange={(e) => setTokenOut(e.target.value)}
                disabled={isLoadingTokens}
                data-testid="token-out-select"
              >
                {tokenOutOptions.map((token) => (
                  <option key={token.denom} value={token.denom}>
                    {token.symbol} ({token.denom})
                  </option>
                ))}
              </select>
            </div>
          </div>
        </div>

        <div className="dex-slippage-row">
          <label className="form-label">Slippage tolerance</label>
          <div className="dex-slippage-controls">
            {SLIPPAGE_PRESETS.map((value) => (
              <button
                type="button"
                key={value}
                className={`dex-pill ${slippage === value ? 'active' : ''}`}
                onClick={() => setSlippage(value)}
              >
                {value}%
              </button>
            ))}
            <input
              type="number"
              min="0.1"
              max="5"
              step="0.1"
              className="form-input dex-slippage-input"
              value={slippage}
              onChange={(e) => setSlippage(parseFloat(e.target.value) || 0.1)}
            />
            <span className="dex-slippage-hint">Allowed range 0.1% - 5%</span>
          </div>
        </div>

        <div className="dex-quote-panel">
          <div className="dex-quote-grid">
            <div>
              <span className="dex-quote-label">Expected Output</span>
              <strong data-testid="expected-output">
                {quote ? `${quote.expectedAmountOut} ${quote.tokenOut.symbol}` : '--'}
              </strong>
              {expectedUsdValue && (
                <small className="dex-quote-subtext">{formatUSD(expectedUsdValue)}</small>
              )}
            </div>
            <div>
              <span className="dex-quote-label">Min Received</span>
              <strong>
                {quote ? `${quote.minAmountOut} ${quote.tokenOut.symbol}` : '--'}
              </strong>
              <small className="dex-quote-subtext">Txn reverts if output falls below</small>
            </div>
            <div>
              <span className="dex-quote-label">Execution Price</span>
              <strong>
                {quote
                  ? `1 ${quote.tokenIn.symbol} ≈ ${quote.executionPrice} ${quote.tokenOut.symbol}`
                  : '--'}
              </strong>
              <small className="dex-quote-subtext">
                {quote ? `Inverse: 1 ${quote.tokenOut.symbol} ≈ ${quote.inverseExecutionPrice} ${quote.tokenIn.symbol}` : ''}
              </small>
            </div>
            <div>
              <span className="dex-quote-label">Price Impact</span>
              <strong className={quote && quote.priceImpactPercent > 2 ? 'text-warning' : ''}>
                {quote ? `${quote.priceImpactPercent.toFixed(2)}%` : '--'}
              </strong>
              {oracleSpread !== null && (
                <small className="dex-quote-subtext">
                  Oracle spread {oracleSpread >= 0 ? '+' : ''}
                  {oracleSpread.toFixed(2)}%
                </small>
              )}
            </div>
          </div>

          {isQuoting && <div className="dex-status">Refreshing quote…</div>}
          {quoteError && <div className="form-error">{quoteError}</div>}
        </div>

        <div className="dex-toggle-row">
          <label className="dex-toggle">
            <input
              type="checkbox"
              checked={showRouteInsights}
              onChange={() => setShowRouteInsights((prev) => !prev)}
            />
            <span>Show route diagnostics</span>
          </label>
          {quoteFreshness && <span className="dex-freshness">Quote age: {quoteFreshness}</span>}
        </div>

        {showRouteInsights && (
          <div className="dex-insights">
            {poolAnalytics ? (
              <>
                <div className="dex-insight-panel">
                  <div className="dex-insight-header">
                    <div>
                      <h4>Pool Route Insights</h4>
                      <span>Pool #{poolAnalytics.poolId}</span>
                    </div>
                    <div className={`dex-risk-badge risk-${riskLevel}`}>
                      {riskLevel === 'high' ? 'High risk' : riskLevel === 'medium' ? 'Elevated risk' : 'Stable'}
                    </div>
                  </div>
                  <div className="dex-insight-grid">
                    {poolAnalytics.reserves.map((reserve) => (
                      <div key={reserve.token.denom}>
                        <span className="dex-quote-label">Reserve {reserve.token.symbol}</span>
                        <strong>{reserve.amount}</strong>
                        {reserve.usdValue !== undefined && (
                          <small className="dex-quote-subtext">{formatUSD(reserve.usdValue)}</small>
                        )}
                      </div>
                    ))}
                    <div>
                      <span className="dex-quote-label">Pool TVL</span>
                      <strong>
                        {poolAnalytics.totalValueLockedUsd
                          ? formatUSD(poolAnalytics.totalValueLockedUsd)
                          : '—'}
                      </strong>
                      <small className="dex-quote-subtext">Oracle derived</small>
                    </div>
                    <div>
                      <span className="dex-quote-label">Depth Balance</span>
                      <strong>{poolAnalytics.depthRatio.toFixed(2)}%</strong>
                      <small className="dex-quote-subtext">
                        100% = perfectly balanced reserves
                      </small>
                    </div>
                  </div>
                </div>
                <div className="dex-insight-panel">
                  <div className="dex-insight-header">
                    <h4>Route Execution</h4>
                    <span>{quote ? `1 ${quote.tokenIn.symbol} ≈ ${poolAnalytics.spotPrice} ${quote.tokenOut.symbol}` : 'Awaiting quote'}</span>
                  </div>
                  <div className="dex-insight-grid">
                    <div>
                      <span className="dex-quote-label">Route</span>
                      <strong>Direct • Single pool</strong>
                      <small className="dex-quote-subtext">No intermediate hops</small>
                    </div>
                    <div>
                      <span className="dex-quote-label">Oracle Spread</span>
                      <strong>
                        {oracleSpread !== null
                          ? `${oracleSpread >= 0 ? '+' : ''}${oracleSpread.toFixed(2)}%`
                          : '—'}
                      </strong>
                      <small className="dex-quote-subtext">Execution vs median oracle</small>
                    </div>
                    <div>
                      <span className="dex-quote-label">Price Impact</span>
                      <strong>{quote ? `${quote.priceImpactPercent.toFixed(2)}%` : '—'}</strong>
                      <small className="dex-quote-subtext">vs pool spot price</small>
                    </div>
                    <div>
                      <span className="dex-quote-label">Route Health</span>
                      <strong>
                        {poolAnalytics.depthRatio > 65 ? 'Deep liquidity' : poolAnalytics.depthRatio > 35 ? 'Moderate depth' : 'Shallow'}
                      </strong>
                      <small className="dex-quote-subtext">
                        Depth ratio {poolAnalytics.depthRatio.toFixed(1)}%
                      </small>
                    </div>
                  </div>
                </div>
              </>
            ) : (
              <div className="dex-insight-panel">
                <div className="dex-insight-header">
                  <h4>Pool Route Insights</h4>
                </div>
                <p className="text-muted">
                  {poolInsightError || 'Select a valid pair to view liquidity diagnostics.'}
                </p>
              </div>
            )}
          </div>
        )}

        <div className="dex-security-tip">
          <h4>Advanced Controls</h4>
          <div className="dex-advanced-grid">
            <div>
              <label className="form-label">Wallet Authentication</label>
              {!isLedger ? (
                <input
                  type="password"
                  className="form-input"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  placeholder="Decrypt keystore to sign"
                />
              ) : (
                <div className="text-muted" style={{ fontSize: '12px' }}>
                  Ledger connected: DEX swaps require custom signing and are temporarily disabled for hardware wallets.
                </div>
              )}
            </div>
            <div>
              <label className="form-label">Memo (optional)</label>
              <input
                className="form-input"
                value={memo}
                onChange={(e) => setMemo(e.target.value)}
                maxLength={128}
                placeholder="Routing tags, analytics, etc."
              />
            </div>
          </div>
        </div>

        {statusMessage && <div className="form-success">{statusMessage}</div>}
        {executionError && <div className="form-error">{executionError}</div>}
        {lastTxHash && (
          <div className="dex-tx-hash">
            Latest tx hash: <code>{lastTxHash}</code>
          </div>
        )}

        <button
          type="button"
          className="btn btn-primary dex-submit"
          onClick={handleSwap}
          disabled={!canSwap}
        >
          {isSwapping ? 'Broadcasting…' : 'Execute Swap'}
        </button>
      </div>
    </div>
  );
};

function formatUSD(value: number): string {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: value >= 1 ? 2 : 6,
  }).format(value);
}

export default SwapInterface;
