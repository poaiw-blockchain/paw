import React, { useEffect, useMemo, useRef, useState } from 'react';
import type { PriceData } from '@paw-chain/wallet-core';
import { DexService, SwapQuote, DenomMetadata } from '../../services/dex';

const SLIPPAGE_PRESETS = [0.1, 0.5, 1, 2];

interface WalletData {
  address: string;
  publicKey?: string;
}

interface SwapInterfaceProps {
  walletData: WalletData | null;
  service?: DexService;
}

const SwapInterface: React.FC<SwapInterfaceProps> = ({ walletData, service }) => {
  const dexService = useMemo(() => service ?? new DexService(), [service]);
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

  const canSwap = Boolean(
    quote &&
      password.length >= 8 &&
      !isSwapping &&
      !isQuoting &&
      amountIn &&
      tokenIn &&
      tokenOut &&
      tokenIn !== tokenOut
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

        <div className="dex-security-tip">
          <h4>Advanced Controls</h4>
          <div className="dex-advanced-grid">
            <div>
              <label className="form-label">Wallet Password</label>
              <input
                type="password"
                className="form-input"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="Decrypt keystore to sign"
              />
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
