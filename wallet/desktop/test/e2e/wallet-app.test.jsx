import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import App from '../../src/App';

const mockWallet = {
  address: 'paw1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqq',
  publicKey: 'pubkey123',
};

const mockGetWallet = jest.fn();
const mockGenerateMnemonic = jest.fn();
const mockCreateWallet = jest.fn();
const mockUnlockWallet = jest.fn();
const mockGetMnemonic = jest.fn();
const mockClearWallet = jest.fn();

jest.mock('../../src/services/keystore', () => ({
  KeystoreService: jest.fn().mockImplementation(() => ({
    getWallet: mockGetWallet,
    generateMnemonic: mockGenerateMnemonic,
    createWallet: mockCreateWallet,
    unlockWallet: mockUnlockWallet,
    getMnemonic: mockGetMnemonic,
    clearWallet: mockClearWallet,
  })),
}));

const mockGetBalance = jest.fn();
const mockGetTransactions = jest.fn();
const mockSendTokens = jest.fn();
const mockGetEndpoint = jest.fn();

jest.mock('../../src/services/api', () => ({
  ApiService: jest.fn().mockImplementation(() => ({
    getBalance: mockGetBalance,
    getTransactions: mockGetTransactions,
    sendTokens: mockSendTokens,
    getEndpoint: mockGetEndpoint,
  })),
}));

const mockGetTradableTokens = jest.fn();
const mockQuoteSwap = jest.fn();
const mockExecuteSwap = jest.fn();
const mockGetOraclePrice = jest.fn();
const mockGetPoolAnalytics = jest.fn();

jest.mock('../../src/services/dex', () => ({
  DexService: jest.fn().mockImplementation(() => ({
    getTradableTokens: mockGetTradableTokens,
    quoteSwap: mockQuoteSwap,
    executeSwap: mockExecuteSwap,
    getOraclePrice: mockGetOraclePrice,
    getPoolAnalytics: mockGetPoolAnalytics,
  })),
}));

const mockGetPortfolio = jest.fn();
const mockDelegate = jest.fn();
const mockUndelegate = jest.fn();
const mockRedelegate = jest.fn();
const mockWithdraw = jest.fn();

jest.mock('../../src/services/staking', () => ({
  StakingService: jest.fn().mockImplementation(() => ({
    getPortfolio: mockGetPortfolio,
    delegate: mockDelegate,
    undelegate: mockUndelegate,
    redelegate: mockRedelegate,
    withdraw: mockWithdraw,
  })),
}));

const renderApp = () => render(<App />);

beforeEach(() => {
  jest.clearAllMocks();

  mockGetWallet.mockResolvedValue(mockWallet);
  mockGenerateMnemonic.mockResolvedValue(
    'zone visual maze napkin ginger quantum drive window ocean marble jewel paper'
  );
  mockCreateWallet.mockResolvedValue(mockWallet);
  mockUnlockWallet.mockResolvedValue({
    address: mockWallet.address,
    publicKey: mockWallet.publicKey,
    privateKey: 'mock-private-key',
  });
  mockGetMnemonic.mockResolvedValue('zone visual maze ... paper');
  mockClearWallet.mockResolvedValue(undefined);

  mockGetEndpoint.mockResolvedValue('http://localhost:1317');
  mockGetBalance.mockResolvedValue({
    balances: [{ denom: 'upaw', amount: '123000000' }],
  });
  mockGetTransactions.mockResolvedValue([
    {
      txhash: 'ABC1234567890',
      code: 0,
      timestamp: new Date().toISOString(),
      messages: [
        {
          '@type': 'cosmos.bank.v1beta1.MsgSend',
          amount: [{ amount: '1000000', denom: 'upaw' }],
        },
      ],
    },
  ]);
  mockSendTokens.mockResolvedValue({ transactionHash: '0xSEND' });

  mockGetTradableTokens.mockResolvedValue([
    { denom: 'upaw', symbol: 'PAW', display: 'PAW', decimals: 6 },
    { denom: 'uusdc', symbol: 'USDC', display: 'USDC', decimals: 6 },
  ]);

  const now = Date.now();
  mockQuoteSwap.mockResolvedValue({
    poolId: 1,
    tokenIn: { denom: 'upaw', symbol: 'PAW', display: 'PAW', decimals: 6 },
    tokenOut: { denom: 'uusdc', symbol: 'USDC', display: 'USDC', decimals: 6 },
    amountIn: '10',
    normalizedAmountIn: '10000000',
    expectedAmountOut: '9.85',
    minAmountOut: '9.70',
    minAmountOutBase: '9700000',
    executionPrice: '0.985',
    inverseExecutionPrice: '1.0152',
    priceImpactPercent: 1.2,
    updatedAt: now,
  });
  mockExecuteSwap.mockResolvedValue({ transactionHash: '0xSWAP' });
  mockGetOraclePrice.mockResolvedValue({ asset: 'PAW/USD', price: '1.00', timestamp: now, sources: 3 });
  mockGetPoolAnalytics.mockResolvedValue({
    poolId: 1,
    spotPrice: '0.985',
    inverseSpotPrice: '1.015',
    reserves: [
      { token: { denom: 'upaw', symbol: 'PAW', display: 'PAW', decimals: 6 }, amount: '100000', usdValue: 100000 },
      { token: { denom: 'uusdc', symbol: 'USDC', display: 'USDC', decimals: 6 }, amount: '90000', usdValue: 90000 },
    ],
    totalValueLockedUsd: 190000,
    depthRatio: 92,
    updatedAt: now,
  });

  mockGetPortfolio.mockResolvedValue({
    summary: {
      totalDelegatedDisplay: '1,000.000000 PAW',
      totalRewardsDisplay: '12.500000 PAW',
      denom: 'upaw',
      symbol: 'PAW',
      activeValidators: 4,
      averageApr: 18.5,
    },
    validators: [
      {
        validatorAddress: 'pawvaloper1aa',
        moniker: 'Validator One',
        votingPowerPercent: 42.5,
        aprEstimate: 18.5,
        commissionFormatted: '5.00%',
        myDelegationDisplay: '250 PAW',
        jailed: false,
        status: 'BOND_STATUS_BONDED',
      },
    ],
    delegations: [
      {
        validatorAddress: 'pawvaloper1aa',
        validatorMoniker: 'Validator One',
        amountBase: '250000000',
        amountDisplay: '250 PAW',
        rewardsBase: '5000000',
        rewardsDisplay: '5 PAW',
        status: 'BOND_STATUS_BONDED',
        denom: 'upaw',
      },
    ],
    rewards: [
      {
        validatorAddress: 'pawvaloper1aa',
        validatorMoniker: 'Validator One',
        amountBase: '250000000',
        amountDisplay: '250 PAW',
        rewardsBase: '5000000',
        rewardsDisplay: '5 PAW',
        status: 'BOND_STATUS_BONDED',
        denom: 'upaw',
      },
    ],
    updatedAt: now,
  });

  mockDelegate.mockResolvedValue({ transactionHash: '0xDELEGATE' });
  mockUndelegate.mockResolvedValue({ transactionHash: '0xUNDELEGATE' });
  mockRedelegate.mockResolvedValue({ transactionHash: '0xREDELEGATE' });
  mockWithdraw.mockResolvedValue({ transactionHash: '0xWITHDRAW' });
});

test('guides a new user through wallet creation before showing dashboard', async () => {
  mockGetWallet.mockResolvedValueOnce(null).mockResolvedValue(mockWallet);
  const user = userEvent.setup();

  renderApp();

  expect(await screen.findByText(/Welcome to PAW Wallet/i)).toBeInTheDocument();
  await user.type(screen.getByPlaceholderText(/Enter password/i), 'supersafepass');
  await user.type(screen.getByPlaceholderText(/Confirm password/i), 'supersafepass');
  await user.click(screen.getByRole('button', { name: /Generate Mnemonic/i }));

  expect(mockGenerateMnemonic).toHaveBeenCalled();
  expect(await screen.findByText(/Backup Your Mnemonic/i)).toBeInTheDocument();

  await user.click(screen.getByRole('checkbox', { name: /written down/i }));
  await user.click(screen.getByRole('button', { name: /Create Wallet/i }));

  expect(mockCreateWallet).toHaveBeenCalledWith(expect.any(String), 'supersafepass');
  await waitFor(() => expect(screen.getByText(/Balance/i)).toBeInTheDocument());
});

test('navigates wallet flows and sends a transaction successfully', async () => {
  const user = userEvent.setup();
  renderApp();

  expect(await screen.findByText(/Balance/i)).toBeInTheDocument();

  await user.click(screen.getByText('Send'));
  await user.type(screen.getByLabelText(/Recipient Address/i), 'paw1recipientaddress000000000');
  await user.type(screen.getByLabelText(/Amount \(PAW\)/i), '1.5');
  await user.type(screen.getByLabelText(/Password/i), 'walletpass');
  await user.click(screen.getByRole('button', { name: /Preview Transaction/i }));

  expect(await screen.findByText(/Confirm Transaction/i)).toBeInTheDocument();
  await user.click(screen.getByRole('button', { name: /Confirm & Send/i }));

  await waitFor(() => expect(mockSendTokens).toHaveBeenCalled());
  expect(mockSendTokens).toHaveBeenCalledWith(
    mockWallet.address,
    'paw1recipientaddress000000000',
    1500000,
    'upaw',
    '',
    'mock-private-key'
  );
  expect(await screen.findByText(/Transaction successful/i)).toBeInTheDocument();

  await user.click(screen.getByText('Receive'));
  await user.click(screen.getByRole('button', { name: /Copy Address/i }));
  expect(navigator.clipboard.writeText).toHaveBeenCalledWith(mockWallet.address);

  await user.click(screen.getByText('History'));
  expect(await screen.findByText('1.000000 UPAW')).toBeInTheDocument();
});

test('executes swap and staking actions with diagnostics available', async () => {
  const user = userEvent.setup();
  renderApp();

  await screen.findByText(/Balance/i);

  await user.click(screen.getByText('DEX'));
  await screen.findByText(/Protocol DEX Router/);
  const amountInput = screen.getByPlaceholderText('0.0');
  await user.clear(amountInput);
  await user.type(amountInput, '10');
  await user.type(screen.getByPlaceholderText(/Decrypt keystore/i), 'dexpass123');

  await waitFor(() => expect(mockQuoteSwap).toHaveBeenCalled());
  const executeButton = await screen.findByRole('button', { name: /Execute Swap/i });
  await user.click(executeButton);

  await waitFor(() => expect(mockExecuteSwap).toHaveBeenCalled());
  expect(await screen.findByText(/Swap broadcast successfully/i)).toBeInTheDocument();

  await user.click(screen.getByText('Staking'));
  expect(await screen.findByText(/1,000\.000000 PAW/)).toBeInTheDocument();

  await user.selectOptions(screen.getByLabelText(/Action/i), 'delegate');
  await user.selectOptions(screen.getByLabelText(/^Validator$/i), 'pawvaloper1aa');
  await user.type(screen.getByLabelText(/Amount \(PAW\)/i), '5');
  await user.type(screen.getByLabelText(/Wallet Password/i), 'stakingpass');
  await user.click(screen.getByRole('button', { name: /Execute/i }));

  await waitFor(() => expect(mockDelegate).toHaveBeenCalledWith({
    validatorAddress: 'pawvaloper1aa',
    amount: '5',
    password: 'stakingpass',
    memo: undefined,
  }));
  expect(await screen.findByText(/Transaction broadcast successfully/i)).toBeInTheDocument();
});
