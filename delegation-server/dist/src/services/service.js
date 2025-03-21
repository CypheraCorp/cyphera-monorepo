"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || (function () {
    var ownKeys = function(o) {
        ownKeys = Object.getOwnPropertyNames || function (o) {
            var ar = [];
            for (var k in o) if (Object.prototype.hasOwnProperty.call(o, k)) ar[ar.length] = k;
            return ar;
        };
        return ownKeys(o);
    };
    return function (mod) {
        if (mod && mod.__esModule) return mod;
        var result = {};
        if (mod != null) for (var k = ownKeys(mod), i = 0; i < k.length; i++) if (k[i] !== "default") __createBinding(result, mod, k[i]);
        __setModuleDefault(result, mod);
        return result;
    };
})();
Object.defineProperty(exports, "__esModule", { value: true });
exports.delegationService = void 0;
const utils_1 = require("../utils/utils");
// Conditionally import real or mock blockchain service based on MOCK_MODE
let redeemDelegation;
if (process.env.MOCK_MODE === 'true') {
    utils_1.logger.info('Running in MOCK MODE - using mock blockchain service');
    Promise.resolve().then(() => __importStar(require('./mock-blockchain'))).then(module => {
        redeemDelegation = module.redeemDelegation;
    });
}
else {
    Promise.resolve().then(() => __importStar(require('./blockchain'))).then(module => {
        redeemDelegation = module.redeemDelegation;
    });
}
/**
 * Implementation of the DelegationService gRPC service
 */
exports.delegationService = {
    /**
     * Redeems a delegation by processing the delegation data and executing on-chain transactions
     *
     * @param call - The gRPC call containing the delegation data
     * @param callback - The gRPC callback to return the result
     */
    async redeemDelegation(call, callback) {
        const startTime = Date.now();
        utils_1.logger.info("Received delegation redemption request");
        try {
            // Wait for the dynamic import to complete
            if (!redeemDelegation) {
                // Import is still in progress, wait for it to complete
                if (process.env.MOCK_MODE === 'true') {
                    const module = await Promise.resolve().then(() => __importStar(require('./mock-blockchain')));
                    redeemDelegation = module.redeemDelegation;
                }
                else {
                    const module = await Promise.resolve().then(() => __importStar(require('./blockchain')));
                    redeemDelegation = module.redeemDelegation;
                }
            }
            // Extract the delegation data from the request
            const delegationData = call.request.delegationData;
            if (!delegationData || delegationData.length === 0) {
                throw new Error("Delegation data is empty or invalid");
            }
            utils_1.logger.debug(`Received delegation data of size: ${delegationData.length} bytes`);
            // Redeem the delegation using the blockchain service
            const transactionHash = await redeemDelegation(new Uint8Array(delegationData));
            const elapsedTime = (Date.now() - startTime) / 1000;
            utils_1.logger.info(`Delegation redeemed successfully in ${elapsedTime.toFixed(2)}s, transaction hash: ${transactionHash}`);
            // Return success response
            callback(null, {
                transactionHash,
                success: true,
                errorMessage: ''
            });
        }
        catch (error) {
            const elapsedTime = (Date.now() - startTime) / 1000;
            utils_1.logger.error(`Error redeeming delegation after ${elapsedTime.toFixed(2)}s:`, error);
            // Return error response
            callback(null, {
                transactionHash: '',
                success: false,
                errorMessage: error instanceof Error ? error.message : 'Unknown error'
            });
        }
    }
};
