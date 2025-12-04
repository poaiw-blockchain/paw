import React from 'react';
import { render, screen, waitFor, fireEvent, act } from '@testing-library/react';
import SwapInterface from '../../src/components/DEX/SwapInterface';

const mockDexService = {
  getTradableTokens: jest.fn(),
  getOraclePrice: jest.fn(),
  quoteSwap: jest.fn(),
  executeSwap: jest.fn(),
};

jest.mock('../../src/services/dex', () => ({
  DexService: jest.fn().mockImplementation(() => mockDexService),
}));

const wallet = { address: 'paw1qpz7e0mockaddress' };

const defaultQuote = {
  poolId: 1,
  tokenIn: { denom: 'upaw', symbol: 'PAW', display: 'PAW', decimals: 6 },
  tokenOut: { denom: 'uatom', symbol: 'ATOM', display: 'ATOM', decimals: 6 },
  amountIn: '10',
  normalizedAmountIn: '10000000',
  expectedAmountOut: '5.000000',
  minAmountOut: '4.950000',
  minAmountOutBase: '4950000',
  executionPrice: '0.5',
  inverseExecutionPrice: '2',
  priceImpactPercent: 0.25,
  updatedAt: Date.now(),
};

describe('SwapInterface', () => {
  beforeEach(() => {
    jest.useFakeTimers();

    mockDexService.getTradableTokens.mockResolvedValue([
      { denom: 'upaw', symbol: 'PAW', display: 'PAW', decimals: 6 },
      { denom: 'uatom', symbol: 'ATOM', display: 'ATOM', decimals: 6 },
    ]);

    mockDexService.getOraclePrice.mockImplementation(async (denom: string) => ({
      asset: `${denom.toUpperCase()}/USD`,
      price: denom === 'upaw' ? '1.00' : '14.00',
      timestamp: Date.now(),
      sources: 4,
    }));

    mockDexService.quoteSwap.mockResolvedValue(defaultQuote);
    mockDexService.executeSwap.mockResolvedValue({
      code: 0,
      transactionHash: 'MOCK_TX_HASH',
    });
  });

  afterEach(() => {
    jest.runOnlyPendingTimers();
    jest.useRealTimers();
    jest.clearAllMocks();
  });

  it('renders quote after updating amount and tokens', async () => {
    render(<SwapInterface walletData={wallet} />);

    await waitFor(() => expect(mockDexService.getTradableTokens).toHaveBeenCalled());

    const amountInput = screen.getByPlaceholderText('0.0') as HTMLInputElement;
    fireEvent.change(amountInput, { target: { value: '10' } });

    act(() => {
      jest.advanceTimersByTime(500);
    });

    await waitFor(() => expect(mockDexService.quoteSwap).toHaveBeenCalled());
    expect(mockDexService.quoteSwap).toHaveBeenCalledWith({
      tokenIn: 'upaw',
      tokenOut: 'uatom',
      amountIn: '10',
      slippagePercent: 0.5,
    });

    const expectedOutput = await screen.findByTestId('expected-output');
    expect(expectedOutput.textContent).toContain('5.000000 ATOM');
  });

  it('executes swap when password is supplied', async () => {
    render(<SwapInterface walletData={wallet} />);
    await waitFor(() => expect(mockDexService.getTradableTokens).toHaveBeenCalled());

    const amountInput = screen.getByPlaceholderText('0.0');
    fireEvent.change(amountInput, { target: { value: '10' } });

    act(() => {
      jest.advanceTimersByTime(500);
    });
    await waitFor(() => expect(mockDexService.quoteSwap).toHaveBeenCalled());

    const passwordInput = screen.getByPlaceholderText('Decrypt keystore to sign');
    fireEvent.change(passwordInput, { target: { value: 'strong-passphrase' } });

    const submit = screen.getByRole('button', { name: /execute swap/i });
    fireEvent.click(submit);

    await waitFor(() => expect(mockDexService.executeSwap).toHaveBeenCalled());
    expect(mockDexService.executeSwap).toHaveBeenCalledWith({
      tokenIn: 'upaw',
      tokenOut: 'uatom',
      amountIn: '10',
      slippagePercent: 0.5,
      password: 'strong-passphrase',
      memo: undefined,
    });
  });
});
