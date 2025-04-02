"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.redeemDelegation = void 0;
/**
 * Mock blockchain service for testing purposes
 * This file provides mock implementations of the blockchain service methods
 * to enable testing without actual blockchain interactions.
 */
const utils_1 = require("../utils/utils");
const delegation_helpers_1 = require("../utils/delegation-helpers");
/**
 * Mock implementation of the redeemDelegation function
 * @param delegationData The serialized delegation data (signature)
 * @param merchantAddress The address of the merchant
 * @param tokenContractAddress The address of the token contract
 * @param price The price of the token
 * @returns A mock transaction hash
 */
const redeemDelegation = async (delegationData, merchantAddress, tokenContractAddress, price) => {
    try {
        // Validate inputs first to avoid undefined errors
        if (!delegationData || delegationData.length === 0) {
            throw new Error("Delegation data is required");
        }
        if (!merchantAddress) {
            throw new Error("Merchant address is required");
        }
        if (!tokenContractAddress) {
            throw new Error("Token contract address is required");
        }
        if (!price) {
            throw new Error("Price is required");
        }
        // Parse and validate the delegation - this is real code that will actually check
        // the delegation format, so our test is still meaningful
        const delegation = (0, delegation_helpers_1.parseDelegation)(delegationData);
        (0, delegation_helpers_1.validateDelegation)(delegation);
        utils_1.logger.info("[MOCK] Redeeming delegation...");
        utils_1.logger.debug("[MOCK] Delegation details:", {
            delegate: delegation.delegate,
            delegator: delegation.delegator,
            merchantAddress,
            tokenContractAddress,
            price
        });
        // Simulate processing time to make the test more realistic
        await new Promise(resolve => setTimeout(resolve, 1000));
        // Generate a mock transaction hash
        const mockTxHash = '0x' + [...Array(64)].map(() => Math.floor(Math.random() * 16).toString(16)).join('');
        utils_1.logger.info("[MOCK] Transaction confirmed:", mockTxHash);
        return mockTxHash;
    }
    catch (error) {
        utils_1.logger.error("[MOCK] Error redeeming delegation:", error);
        throw error;
    }
};
exports.redeemDelegation = redeemDelegation;
