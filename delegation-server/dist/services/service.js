"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.delegationService = void 0;
const blockchain_1 = require("./blockchain");
const utils_1 = require("../utils/utils");
/**
 * Implementation of the DelegationService
 */
exports.delegationService = {
    /**
     * Redeems a delegation sent from the golang backend
     */
    async redeemDelegation(call, callback) {
        console.log("Received delegation redemption request");
        try {
            // Parse the delegation from the request
            const delegationData = call.request.delegationData;
            const delegationString = delegationData.toString('utf-8');
            const delegation = JSON.parse(delegationString);
            console.log("Parsed delegation:", (0, utils_1.customJSONStringify)(delegation));
            // Redeem the delegation
            const transactionHash = await (0, blockchain_1.redeemDelegation)(delegation);
            // Return success response
            callback(null, {
                transactionHash,
                success: true,
                errorMessage: ''
            });
        }
        catch (error) {
            console.error("Error redeeming delegation:", error);
            // Return error response
            callback(null, {
                transactionHash: '',
                success: false,
                errorMessage: error instanceof Error ? error.message : 'Unknown error'
            });
        }
    }
};
