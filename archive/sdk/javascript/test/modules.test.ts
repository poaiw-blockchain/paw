import { DexModule } from '../src/modules/dex';
import { PawClient } from '../src/client';

jest.mock('../src/client');

describe('DexModule', () => {
  let mockClient: jest.Mocked<PawClient>;
  let dexModule: DexModule;

  beforeEach(() => {
    mockClient = {
      getConfig: jest.fn().mockReturnValue({
        rpcEndpoint: 'http://localhost:26657',
        restEndpoint: 'http://localhost:1317',
        chainId: 'paw-mvp-1'
      })
    } as any;

    dexModule = new DexModule(mockClient);
  });

  describe('calculateSwapOutput', () => {
    it('should calculate correct swap output', () => {
      const amountIn = '1000000';
      const reserveIn = '10000000';
      const reserveOut = '20000000';
      const swapFee = '0.003';

      const output = dexModule.calculateSwapOutput(amountIn, reserveIn, reserveOut, swapFee);

      // Expected: (1000000 * 0.997 * 20000000) / (10000000 + 1000000 * 0.997)
      expect(BigInt(output)).toBeGreaterThan(0n);
      expect(BigInt(output)).toBeLessThan(BigInt(amountIn) * 2n);
    });

    it('should return 0 for zero input', () => {
      const output = dexModule.calculateSwapOutput('0', '10000000', '20000000');
      expect(output).toBe('0');
    });

    it('should handle large numbers', () => {
      const amountIn = '1000000000000';
      const reserveIn = '10000000000000';
      const reserveOut = '20000000000000';

      const output = dexModule.calculateSwapOutput(amountIn, reserveIn, reserveOut);
      expect(BigInt(output)).toBeGreaterThan(0n);
    });
  });

  describe('calculatePriceImpact', () => {
    it('should calculate price impact percentage', () => {
      const amountIn = '1000000';
      const reserveIn = '10000000';
      const reserveOut = '20000000';

      const impact = dexModule.calculatePriceImpact(amountIn, reserveIn, reserveOut);

      expect(impact).toBeGreaterThanOrEqual(0);
      expect(impact).toBeLessThan(100);
    });

    it('should show higher impact for larger trades', () => {
      const reserveIn = '10000000';
      const reserveOut = '20000000';

      const smallImpact = dexModule.calculatePriceImpact('100000', reserveIn, reserveOut);
      const largeImpact = dexModule.calculatePriceImpact('1000000', reserveIn, reserveOut);

      expect(largeImpact).toBeGreaterThan(smallImpact);
    });
  });

  describe('calculateShares', () => {
    it('should calculate shares for first liquidity provider', () => {
      const shares = dexModule.calculateShares('1000000', '2000000', '0', '0', '0');

      const expected = BigInt(1000000) * BigInt(2000000);
      expect(shares).toBe(expected.toString());
    });

    it('should calculate shares proportionally for existing pool', () => {
      const amountA = '1000000';
      const amountB = '2000000';
      const reserveA = '10000000';
      const reserveB = '20000000';
      const totalShares = '100000000';

      const shares = dexModule.calculateShares(amountA, amountB, reserveA, reserveB, totalShares);

      const expected = (BigInt(amountA) * BigInt(totalShares)) / BigInt(reserveA);
      expect(shares).toBe(expected.toString());
    });
  });
});

describe('BankModule', () => {
  describe('formatBalance', () => {
    it('should format balance with default decimals', () => {
      const { BankModule } = require('../src/modules/bank');
      const mockClient = {} as any;
      const bank = new BankModule(mockClient);

      const formatted = bank.formatBalance({ denom: 'upaw', amount: '1000000' });
      expect(formatted).toBe('1.000000 PAW');
    });

    it('should format balance with custom decimals', () => {
      const { BankModule } = require('../src/modules/bank');
      const mockClient = {} as any;
      const bank = new BankModule(mockClient);

      const formatted = bank.formatBalance({ denom: 'upaw', amount: '150' }, 2);
      expect(formatted).toBe('1.50 PAW');
    });
  });
});

describe('GovernanceModule', () => {
  describe('getVoteOptionName', () => {
    it('should return correct vote option names', () => {
      const { GovernanceModule } = require('../src/modules/governance');
      const { VoteOption } = require('../src/types');
      const mockClient = {} as any;
      const gov = new GovernanceModule(mockClient);

      expect(gov.getVoteOptionName(VoteOption.YES)).toBe('Yes');
      expect(gov.getVoteOptionName(VoteOption.NO)).toBe('No');
      expect(gov.getVoteOptionName(VoteOption.ABSTAIN)).toBe('Abstain');
      expect(gov.getVoteOptionName(VoteOption.NO_WITH_VETO)).toBe('No with Veto');
      expect(gov.getVoteOptionName(VoteOption.UNSPECIFIED)).toBe('Unspecified');
    });
  });
});
