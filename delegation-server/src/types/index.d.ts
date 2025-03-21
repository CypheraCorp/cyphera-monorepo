// Type declarations for packages without type definitions

declare module '@metamask-private/delegator-core-viem' {
  import { type PublicClient, type Address, type Hex } from 'viem';
  import { type PrivateKeyAccount } from 'viem/accounts';

  export type DelegationStruct = {
    delegate: Address;
    delegator: Address;
    authority: Address;
    caveats: any[];
    salt: string;
    signature: Hex;
  };

  export type ExecutionStruct = {
    target: Address;
    value: bigint;
    callData: Hex;
  };

  export type Call = {
    to: Address;
    data: Hex;
    value?: bigint;
  };

  export enum Implementation {
    Hybrid = 'hybrid'
  }

  export const SINGLE_DEFAULT_MODE: string;

  export const DelegationFramework: {
    encode: {
      redeemDelegations: (
        delegationChains: DelegationStruct[][],
        modes: string[],
        executions: ExecutionStruct[][]
      ) => Hex;
    };
  };

  export interface SmartAccountInterface {
    address: Address;
    encodeCallData: (calls: Call[]) => Hex;
  }

  export function toMetaMaskSmartAccount(params: {
    client: PublicClient;
    implementation: Implementation;
    deployParams: [Address, any[], any[], any[]];
    deploySalt: Hex;
    signatory: { account: PrivateKeyAccount };
  }): Promise<SmartAccountInterface>;
}

declare module 'viem/account-abstraction' {
  import { type Chain, type Transport, type Account, type PublicClient, type Address } from 'viem';
  
  export interface PaymasterClient {
    sponsorUserOperation: (args: any) => Promise<any>;
  }

  export interface BundlerClient {
    sendUserOperation: (params: {
      account: Account;
      userOperation: any;
      entryPoint: Address;
    }) => Promise<string>;
  }

  export function createBundlerClient(params: {
    chain: Chain;
    transport: Transport;
    paymaster?: PaymasterClient;
  }): BundlerClient;

  export function createPaymasterClient(params: {
    transport: Transport;
  }): PaymasterClient;
}

declare module 'permissionless/clients/bundler' {
  import { type Chain, type Transport, type Account, type PublicClient, type Address } from 'viem';
  
  export interface PaymasterClient {
    sponsorUserOperation: (args: any) => Promise<any>;
  }

  export interface BundlerClient {
    sendUserOperation: (params: {
      account: Account;
      userOperation: any;
      entryPoint: Address;
    }) => Promise<string>;
  }

  export function createBundlerClient(params: {
    chain: Chain;
    transport: Transport;
    paymaster?: PaymasterClient;
  }): BundlerClient;

  export function createPaymasterClient(params: {
    transport: Transport;
  }): PaymasterClient;
} 