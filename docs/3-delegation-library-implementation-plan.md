# Delegation Library Implementation Plan - Part 3

## Executive Summary

This document outlines the plan to consolidate the overlapping delegation logic from the web-app and delegation-server into a shared TypeScript library within the monorepo. This consolidation will eliminate code duplication, ensure consistency across services, and provide a single source of truth for delegation operations while maintaining zero breaking changes.

## Current State Analysis

### Duplicated Code Identified

1. **Delegation Creation & Parsing**
   - Web-app: `/lib/web3/utils/delegation.ts`
   - Server: `/apps/delegation-server/src/utils/delegation-helpers.ts`
   - Overlap: Delegation structure, parsing logic, validation rules

2. **Smart Account Management**
   - Web-app: Multiple hooks and utilities for smart account creation
   - Server: Smart account creation in redemption logic
   - Overlap: Account creation parameters, deployment checks

3. **Network Configuration**
   - Web-app: Dynamic network loading from environment
   - Server: Hardcoded network configurations
   - Overlap: RPC URLs, chain IDs, bundler URLs

4. **Transaction Utilities**
   - Web-app: ERC20 encoding in components
   - Server: ERC20 encoding in redemption service
   - Overlap: ABI definitions, encoding logic

5. **Validation Logic**
   - Web-app: Client-side validation in forms
   - Server: Server-side validation in helpers
   - Overlap: Address validation, delegation structure validation

## Proposed Library Architecture

### Package Structure
```
/libs/typescript/delegation-toolkit/
├── src/
│   ├── index.ts                    # Main exports
│   ├── types/
│   │   ├── delegation.ts           # Core delegation types
│   │   ├── smart-account.ts        # Smart account interfaces
│   │   └── network.ts              # Network configuration types
│   ├── core/
│   │   ├── delegation-factory.ts   # Create delegations
│   │   ├── delegation-parser.ts    # Parse delegations
│   │   └── delegation-validator.ts # Validate delegations
│   ├── smart-accounts/
│   │   ├── account-factory.ts      # Smart account creation
│   │   ├── deployment-checker.ts   # Check deployment status
│   │   └── account-utils.ts        # Common utilities
│   ├── networks/
│   │   ├── network-config.ts       # Network configurations
│   │   ├── chain-utils.ts          # Chain-specific utilities
│   │   └── rpc-manager.ts          # RPC URL management
│   ├── transactions/
│   │   ├── erc20-encoder.ts        # ERC20 transaction encoding
│   │   ├── gas-estimator.ts        # Gas estimation utilities
│   │   └── user-operation.ts       # UserOperation helpers
│   ├── utils/
│   │   ├── address.ts              # Address validation
│   │   ├── crypto.ts               # Crypto utilities
│   │   ├── hex.ts                  # Hex conversion
│   │   └── json.ts                 # JSON with BigInt support
│   ├── abis/
│   │   ├── erc20.ts                # ERC20 ABI
│   │   └── delegation-framework.ts  # Delegation Framework ABI
│   └── constants/
│       ├── addresses.ts            # Contract addresses
│       └── defaults.ts             # Default values
├── tests/
│   ├── unit/                       # Unit tests
│   ├── integration/                # Integration tests
│   └── fixtures/                   # Test fixtures
├── package.json
├── tsconfig.json
├── vitest.config.ts
└── README.md
```

## Implementation Details

### 1. Core Delegation Module

```typescript
// types/delegation.ts
export interface Delegation {
  delegate: `0x${string}`;
  delegator: `0x${string}`;
  authority: `0x${string}`;
  caveats: Caveat[];
  salt: bigint;
  signature: `0x${string}`;
}

export interface Caveat {
  enforcer: `0x${string}`;
  terms: `0x${string}`;
}

// core/delegation-factory.ts
export class DelegationFactory {
  static create(params: {
    from: Address;
    to: Address;
    caveats?: Caveat[];
    salt?: bigint;
  }): Delegation;
  
  static createSalt(): bigint;
}

// core/delegation-parser.ts
export class DelegationParser {
  static parse(data: Uint8Array | Buffer | string): Delegation;
  static serialize(delegation: Delegation): Uint8Array;
  static toHex(delegation: Delegation): `0x${string}`;
}

// core/delegation-validator.ts
export class DelegationValidator {
  static validate(delegation: Partial<Delegation>): ValidationResult;
  static validateAddress(address: string): boolean;
  static validateSignature(delegation: Delegation): Promise<boolean>;
}
```

### 2. Smart Account Module

```typescript
// smart-accounts/account-factory.ts
export class SmartAccountFactory {
  static async create(params: {
    client: PublicClient;
    signer: Account;
    implementation?: Implementation;
    deployParams?: any[];
    deploySalt?: `0x${string}`;
  }): Promise<MetaMaskSmartAccount>;
  
  static getAddress(params: {
    signer: Address;
    implementation: Implementation;
    salt?: `0x${string}`;
  }): Address;
}

// smart-accounts/deployment-checker.ts
export class DeploymentChecker {
  static async isDeployed(
    client: PublicClient,
    address: Address
  ): Promise<boolean>;
  
  static async waitForDeployment(
    client: PublicClient,
    address: Address,
    timeout?: number
  ): Promise<boolean>;
}
```

### 3. Network Configuration Module

```typescript
// networks/network-config.ts
export interface NetworkConfig {
  chainId: number;
  name: string;
  rpcUrl: string;
  bundlerUrl?: string;
  pimlicoApiKey?: string;
  blockExplorer?: string;
  nativeCurrency: {
    name: string;
    symbol: string;
    decimals: number;
  };
}

export class NetworkManager {
  private static configs: Map<number, NetworkConfig>;
  
  static register(config: NetworkConfig): void;
  static get(chainIdOrName: number | string): NetworkConfig;
  static getAll(): NetworkConfig[];
  static getRpcUrl(chainId: number): string;
  static getBundlerUrl(chainId: number): string | undefined;
}
```

### 4. Transaction Module

```typescript
// transactions/erc20-encoder.ts
export class ERC20Encoder {
  static transfer(params: {
    to: Address;
    amount: bigint;
  }): `0x${string}`;
  
  static approve(params: {
    spender: Address;
    amount: bigint;
  }): `0x${string}`;
  
  static transferFrom(params: {
    from: Address;
    to: Address;
    amount: bigint;
  }): `0x${string}`;
}

// transactions/user-operation.ts
export class UserOperationBuilder {
  static build(params: {
    account: MetaMaskSmartAccount;
    calls: Call[];
    delegations?: Delegation[];
  }): Promise<UserOperation>;
  
  static estimateGas(
    bundlerClient: BundlerClient,
    userOp: UserOperation
  ): Promise<GasEstimate>;
}
```

## Migration Strategy

### Phase 1: Library Creation (Days 1-3)
1. **Setup Package Structure**
   - Create `/libs/typescript/delegation-toolkit` directory
   - Configure TypeScript, ESLint, and build tools
   - Setup test infrastructure with Vitest

2. **Implement Core Modules**
   - Types and interfaces
   - Core delegation operations
   - Smart account utilities
   - Network configuration

3. **Add Comprehensive Tests**
   - Unit tests for all utilities
   - Integration tests with mock blockchain
   - Test fixtures for common scenarios

### Phase 2: Web-App Migration (Days 4-6)
1. **Install Shared Library**
   ```json
   // apps/web-app/package.json
   {
     "dependencies": {
       "@cyphera/delegation-toolkit": "workspace:*"
     }
   }
   ```

2. **Update Imports**
   - Replace local delegation utilities with library imports
   - Update hooks to use shared types
   - Migrate network configuration to shared module

3. **Refactor Components**
   - Update delegation buttons to use library
   - Ensure all validation uses shared validators
   - Test thoroughly to ensure no breaking changes

### Phase 3: Delegation Server Migration (Days 7-9)
1. **Install Shared Library**
   ```json
   // apps/delegation-server/package.json
   {
     "dependencies": {
       "@cyphera/delegation-toolkit": "workspace:*"
     }
   }
   ```

2. **Update Service Implementation**
   - Replace local helpers with library imports
   - Use shared validation logic
   - Maintain existing gRPC interfaces

3. **Ensure Compatibility**
   - Test delegation redemption flow
   - Verify no changes to external API
   - Performance testing

### Phase 4: Testing & Validation (Days 10-12)
1. **End-to-End Testing**
   - Complete flow from delegation creation to redemption
   - Test all supported networks
   - Verify gas sponsorship works

2. **Security Audit**
   - Review all signing operations
   - Validate address handling
   - Check for any security regressions

3. **Performance Testing**
   - Measure library overhead
   - Optimize hot paths
   - Bundle size analysis

## Testing Strategy

### Unit Tests
```typescript
// Example test for delegation creation
describe('DelegationFactory', () => {
  it('should create a valid delegation', () => {
    const delegation = DelegationFactory.create({
      from: '0x123...',
      to: '0x456...',
      caveats: []
    });
    
    expect(delegation.delegator).toBe('0x123...');
    expect(delegation.delegate).toBe('0x456...');
    expect(delegation.salt).toBeDefined();
  });
  
  it('should generate unique salts', () => {
    const salt1 = DelegationFactory.createSalt();
    const salt2 = DelegationFactory.createSalt();
    expect(salt1).not.toBe(salt2);
  });
});
```

### Integration Tests
```typescript
// Example integration test
describe('Delegation Flow', () => {
  it('should create and parse delegation correctly', async () => {
    const delegation = DelegationFactory.create({...});
    const serialized = DelegationParser.serialize(delegation);
    const parsed = DelegationParser.parse(serialized);
    
    expect(parsed).toEqual(delegation);
  });
});
```

### E2E Tests
- Create delegation in web-app
- Submit to backend
- Redeem via delegation server
- Verify on-chain execution

## Benefits

1. **Code Reusability**: Single source of truth for delegation logic
2. **Consistency**: Same validation and creation logic everywhere
3. **Maintainability**: Updates in one place affect all consumers
4. **Type Safety**: Shared TypeScript types across services
5. **Testing**: Centralized test suite for critical logic
6. **Documentation**: Single place to document delegation concepts

## Risk Mitigation

1. **Zero Breaking Changes**
   - Maintain existing API contracts
   - Gradual migration with fallbacks
   - Comprehensive test coverage

2. **Performance**
   - Lazy loading of heavy modules
   - Tree-shaking support
   - Minimal bundle size impact

3. **Compatibility**
   - Support all existing delegation formats
   - Backward compatible serialization
   - Version management strategy

## Success Criteria

1. **No Breaking Changes**: All existing delegations continue to work
2. **100% Test Coverage**: Critical paths fully tested
3. **Performance Neutral**: No degradation in performance
4. **Reduced Code**: At least 40% reduction in delegation-related code
5. **Developer Experience**: Easier to work with delegations

## Timeline

- **Week 1**: Library development and testing (Days 1-6)
- **Week 2**: Migration and integration testing (Days 7-12)
- **Buffer**: 2 days for unexpected issues

Total estimated time: **2 weeks**

## Next Steps

1. Create the library package structure
2. Begin implementing core modules
3. Set up CI/CD for the library
4. Start migration planning with team
5. Document migration guide for developers