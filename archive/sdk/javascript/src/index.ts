/**
 * PAW Blockchain SDK
 *
 * Official TypeScript/JavaScript SDK for the PAW blockchain (paw-mvp-1 testnet)
 *
 * @example
 * ```typescript
 * import { PawClient, PawWallet, PAW_TESTNET_CONFIG } from '@paw-chain/sdk';
 *
 * const client = new PawClient(PAW_TESTNET_CONFIG);
 * await client.connect();
 *
 * // Query validators
 * const validators = await client.staking.getValidators();
 * ```
 *
 * @packageDocumentation
 */

// Main exports
export { PawClient } from './client';
export { PawWallet } from './wallet';
export { TxBuilder } from './tx';

// Enhanced wallet exports
export {
  PawWalletEnhanced,
  KeystoreManager,
  AddressBook,
  type HDPath,
  type KeystoreOptions,
  type SerializedKeystore,
  type AddressBookEntry
} from './wallet-enhanced';

// Hardware wallet exports
export {
  HardwareWallet,
  LedgerWallet,
  TrezorWallet,
  HardwareWalletType,
  createHardwareWallet,
  isHardwareWalletSupported,
  getSupportedHardwareWallets,
  type HardwareWalletOptions
} from './hardware-wallet';

// Module exports
export { BankModule } from './modules/bank';
export { DexModule } from './modules/dex';
export { StakingModule } from './modules/staking';
export { GovernanceModule } from './modules/governance';

// Type exports
export * from './types';

// Re-export commonly used CosmJS types
export type { Coin } from '@cosmjs/stargate';
export type { OfflineDirectSigner } from '@cosmjs/proto-signing';
