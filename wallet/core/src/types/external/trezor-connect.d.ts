declare module '@trezor/connect-web' {
  export interface Features {
    model: string;
    major_version: number;
    minor_version: number;
    patch_version: number;
    device_id?: string | null;
  }

  export type Success<TPayload> = {
    success: true;
    payload: TPayload;
  };

  export type Unsuccessful = {
    success: false;
    payload: {
      error: string;
      code?: string;
    };
  };

  export interface CosmosAddressParams {
    path: string;
    showOnTrezor?: boolean;
  }

  export interface CosmosAddressBundle {
    bundle: CosmosAddressParams[];
  }

  export interface CosmosAddressPayload {
    address: string;
    path: string;
    serializedPath?: number[];
  }

  export interface PublicKeyPayload {
    publicKey: string;
  }

  export interface CosmosSignTransactionParams {
    path: string;
    transaction: any;
  }

  export interface EthereumSignMessageParams {
    path: string;
    message: string;
    hex?: boolean;
  }

  export interface EthereumSignMessageResult {
    signature: string;
  }

  export interface CosmosSignResult {
    signature: string;
  }

  export const DEVICE_EVENT: string;
  export const UI_EVENT: string;

  const TrezorConnect: {
    init(options: { manifest: { email: string; appUrl: string }; lazyLoad?: boolean; debug?: boolean }): Promise<void>;
    dispose(): void;
    on(event: string, handler: (event: any) => void): void;
    getFeatures(): Promise<Success<Features> | Unsuccessful>;
    getPublicKey(params: { path: string; coin?: string; showOnTrezor?: boolean }): Promise<Success<PublicKeyPayload> | Unsuccessful>;
    cosmosGetAddress(
      params: CosmosAddressParams | CosmosAddressBundle
    ): Promise<
      | Success<CosmosAddressPayload>
      | Success<Array<{ success: boolean; payload: CosmosAddressPayload }>>
      | Unsuccessful
    >;
    cosmosSignTransaction(params: CosmosSignTransactionParams): Promise<Success<CosmosSignResult> | Unsuccessful>;
    ethereumSignMessage(params: EthereumSignMessageParams): Promise<Success<EthereumSignMessageResult> | Unsuccessful>;
  };

  export default TrezorConnect;
}
