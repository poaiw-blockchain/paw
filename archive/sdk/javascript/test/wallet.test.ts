import { PawWallet } from '../src/wallet';

describe('PawWallet', () => {
  describe('generateMnemonic', () => {
    it('should generate a valid 24-word mnemonic', () => {
      const mnemonic = PawWallet.generateMnemonic();
      const words = mnemonic.split(' ');
      expect(words.length).toBe(24);
      expect(PawWallet.validateMnemonic(mnemonic)).toBe(true);
    });
  });

  describe('validateMnemonic', () => {
    it('should validate correct mnemonics', () => {
      const mnemonic = PawWallet.generateMnemonic();
      expect(PawWallet.validateMnemonic(mnemonic)).toBe(true);
    });

    it('should reject invalid mnemonics', () => {
      expect(PawWallet.validateMnemonic('invalid mnemonic phrase')).toBe(false);
      expect(PawWallet.validateMnemonic('')).toBe(false);
    });
  });

  describe('fromMnemonic', () => {
    it('should create wallet from valid mnemonic', async () => {
      const mnemonic = PawWallet.generateMnemonic();
      const wallet = new PawWallet('paw');

      await expect(wallet.fromMnemonic(mnemonic)).resolves.not.toThrow();
    });

    it('should throw error for invalid mnemonic', async () => {
      const wallet = new PawWallet('paw');

      await expect(wallet.fromMnemonic('invalid mnemonic')).rejects.toThrow('Invalid mnemonic phrase');
    });

    it('should create wallet with custom HD path', async () => {
      const mnemonic = PawWallet.generateMnemonic();
      const wallet = new PawWallet('paw');

      await expect(wallet.fromMnemonic(mnemonic, "m/44'/118'/0'/0/0")).resolves.not.toThrow();
    });
  });

  describe('getAccounts', () => {
    it('should return accounts after wallet initialization', async () => {
      const mnemonic = PawWallet.generateMnemonic();
      const wallet = new PawWallet('paw');
      await wallet.fromMnemonic(mnemonic);

      const accounts = await wallet.getAccounts();
      expect(accounts.length).toBeGreaterThan(0);
      expect(accounts[0]).toHaveProperty('address');
      expect(accounts[0]).toHaveProperty('pubkey');
      expect(accounts[0]).toHaveProperty('algo');
    });

    it('should throw error if wallet not initialized', async () => {
      const wallet = new PawWallet('paw');

      await expect(wallet.getAccounts()).rejects.toThrow('Wallet not initialized');
    });
  });

  describe('getAddress', () => {
    it('should return first account address', async () => {
      const mnemonic = PawWallet.generateMnemonic();
      const wallet = new PawWallet('paw');
      await wallet.fromMnemonic(mnemonic);

      const address = await wallet.getAddress();
      expect(address).toBeTruthy();
      expect(address.startsWith('paw')).toBe(true);
    });
  });

  describe('getSigner', () => {
    it('should return signer after wallet initialization', async () => {
      const mnemonic = PawWallet.generateMnemonic();
      const wallet = new PawWallet('paw');
      await wallet.fromMnemonic(mnemonic);

      const signer = wallet.getSigner();
      expect(signer).toBeDefined();
      expect(signer.getAccounts).toBeDefined();
    });

    it('should throw error if wallet not initialized', () => {
      const wallet = new PawWallet('paw');

      expect(() => wallet.getSigner()).toThrow('Wallet not initialized');
    });
  });

  describe('exportMnemonic', () => {
    it('should export the same mnemonic used to create wallet', async () => {
      const mnemonic = PawWallet.generateMnemonic();
      const wallet = new PawWallet('paw');
      await wallet.fromMnemonic(mnemonic);

      const exported = await wallet.exportMnemonic();
      expect(exported).toBe(mnemonic);
    });
  });
});
