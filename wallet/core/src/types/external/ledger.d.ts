declare module '@ledgerhq/hw-app-cosmos' {
  export interface LedgerAppVersion {
    major: number;
    minor: number;
    patch: number;
  }

  export interface LedgerAddressResponse {
    address: string;
    publicKey: string;
    path: string;
  }

  export interface LedgerSignResponse {
    signature: string;
    publicKey?: string;
  }

  export default class CosmosApp {
    constructor(transport: any);
    getVersion(): Promise<LedgerAppVersion>;
    getAddress(path: string, prefix?: string, display?: boolean): Promise<LedgerAddressResponse>;
    sign(path: string, message: string): Promise<LedgerSignResponse>;
  }
}
