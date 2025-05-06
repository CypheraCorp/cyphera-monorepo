# Product Plan: Multi-Network & Multi-Token Support for Delegation Server

**Date:** May 6th, 2024

**Goal:** Refactor the `delegation-server` to support delegation redemption for various ERC-20 tokens (and potentially native tokens) across multiple EVM-compatible networks, removing the hardcoded dependency on Ethereum Sepolia and USDC.

## 1. Current State Analysis

*   **Network:** The server currently hardcodes the **Ethereum Sepolia** network (`viem/chains/sepolia`) for all operations, including client initialization (public, bundler, paymaster).
*   **Token:** The server assumes a token with **6 decimals** (likely USDC) when encoding the `transfer` calldata. The `token_contract_address` is passed via gRPC, but its properties (like decimals) are not dynamically checked.
*   **Configuration:** Network details like RPC URL (`RPC_URL`) and Bundler URL (`BUNDLER_URL`) are configurable via environment variables but are applied globally to the hardcoded Sepolia chain.
*   **gRPC Interface (`delegation.proto`):** The `RedeemDelegationRequest` message accepts `token_contract_address` but lacks a field to specify the target network (`chainId`).

## 2. Proposed Changes

### 2.1. gRPC Interface (`delegation.proto`)

*   **Add `chain_id` field:** Introduce a `uint32 chain_id` field to the `RedeemDelegationRequest` message. This will allow the client (API server) to specify the target network for each redemption request.
    ```protobuf
    // Request message containing delegation data to be redeemed
    message RedeemDelegationRequest {
      // ... existing fields ...
      string price = 4;
      // The EVM chain ID for the transaction
      uint32 chain_id = 5; 
    }
    ```
*   **Regenerate Proto Code:** Run `make proto-build-all` to update Go and Node.js gRPC code.

### 2.2. Configuration Management (`src/config/`, `src/config.ts`, Environment Variables)

*   **Network Configuration:** Define a structured way to manage RPC URLs, Bundler URLs, and potentially Paymaster URLs per `chainId`. Options:
    *   **Environment Variables:** Use patterned environment variables (e.g., `RPC_URL_1=...`, `RPC_URL_11155111=...`, `BUNDLER_URL_11155111=...`). Requires parsing logic in the config loader.
    *   **JSON Config File:** A `networks.json` file mapping `chainId` to network details. Requires loading this file.
    *   **Recommendation:** Start with environment variables for simplicity, but consider a config file for better organization as the number of supported networks grows.
*   **Config Loading:** Update `src/config.ts` (or a dedicated network config module) to parse and provide network-specific details based on a given `chainId`.

### 2.3. Service Layer (`src/services/`)

*   **`service.ts` (gRPC Handler):**
    *   Extract `chain_id`, `token_contract_address`, `signature`, `merchant_address`, and `price` from the `call.request`.
    *   Pass the `chain_id` down to the `redeemDelegation` implementation.
*   **`redeem-delegation.ts` (Core Logic):**
    *   **Dynamic Chain Object:** Replace the static `const chain = sepolia;` with logic to dynamically obtain the correct `viem` chain object based on the provided `chain_id`. This might involve a helper function or map (e.g., `import * as chains from 'viem/chains'; function getChain(chainId) { return chains[chainId] || chains.sepolia; }`). Handle unknown/unsupported `chainId`s gracefully.
    *   **Dynamic Client Initialization:** Modify the creation of `publicClient`, `bundlerClient`, `paymasterClient`, and `pimlicoClient` to:
        *   Accept the dynamic `chain` object.
        *   Use the network-specific RPC/Bundler/Paymaster URLs fetched from the configuration based on the `chain_id`.
    *   **Dynamic Token Decimals:**
        *   Remove the hardcoded `6` in `parseUnits`.
        *   Use the dynamically created `publicClient` to call the `decimals()` function on the `token_contract_address` for the specified `chain_id`.
        *   Use the fetched decimals in `parseUnits`.
        *   Add error handling if the `decimals()` call fails (e.g., invalid token address, non-ERC20 contract).
    *   **Native Token Support (Optional/Future):** For native token transfers (ETH, MATIC, etc.), the `executions` array would need to be structured differently (setting the `value` field instead of `callData` to a token contract). This requires further design based on how native token delegations are structured. The current `erc20Abi` approach won't work directly.

### 2.4. Testing

*   **Unit Tests:** Update existing tests and add new ones to cover:
    *   Different `chainId` inputs.
    *   Different `token_contract_address` inputs (with varying decimals).
    *   Handling of unsupported networks/tokens.
    *   Correct client initialization for different networks.
*   **Integration Tests (`test-integration`):** Enhance integration tests (or add new ones) to potentially mock responses for different networks and tokens, ensuring the end-to-end flow works with the dynamic parameters.

## 3. Implementation Steps

1.  **Modify `delegation.proto`:** Add the `chain_id` field.
2.  **Regenerate Proto Code:** Run `make proto-build-all`. Update the API server's client code (`internal/grpc/delegation/client.go`) to send the `chain_id`.
3.  **Refactor Configuration:** Choose a method (env vars or file) and implement logic to load/provide network details per `chainId`. Update `.env.example` or add `networks.json.example`.
4.  **Update `service.ts`:** Extract `chain_id` and pass it to `redeemDelegation`.
5.  **Refactor `redeem-delegation.ts`:**
    *   Implement dynamic chain object selection.
    *   Update client creation logic for dynamic chains/URLs.
    *   Implement dynamic decimal fetching.
    *   Add error handling for unsupported chains/tokens and failed decimal lookups.
6.  **Update/Add Tests:** Implement unit and integration tests for the new multi-network/token functionality.
7.  **Documentation:** Update `README.md` regarding new configuration requirements (environment variables/config file) and supported networks/tokens.

## 4. Considerations

*   **Paymaster/Bundler Availability:** Ensure the chosen Paymaster/Bundler service (Pimlico) supports the target networks. URLs might differ per network.
*   **Error Handling:** Provide clear error messages to the client (API server) if an unsupported `chainId` or `token_contract_address` is provided, or if fetching decimals fails.
*   **Native Tokens:** Explicitly decide if native token support is required in this phase and adjust the plan accordingly. It adds complexity to the `executions` logic.
*   **Contract Addresses:** Ensure any other potentially hardcoded addresses (e.g., DelegationFramework contract itself if it differs per network) are also handled dynamically. (Further code search might be needed).
*   **Security:** Ensure proper validation of `chainId` and `token_contract_address` inputs. 