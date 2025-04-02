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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.delegationService = void 0;
const utils_1 = require("../utils/utils");
const config_1 = __importDefault(require("../config"));
// Conditionally import real or mock blockchain service based on MOCK_MODE
let redeemDelegation;
utils_1.logger.info('===== SERVICE.TS INITIALIZATION =====');
utils_1.logger.info(`MOCK_MODE from environment: "${process.env.MOCK_MODE || 'not set'}"`);
utils_1.logger.info(`MOCK_MODE from config: ${config_1.default.mockMode}`);
if (config_1.default.mockMode) {
    utils_1.logger.info('SERVICE.TS: Running in MOCK MODE - using mock blockchain service');
    Promise.resolve().then(() => __importStar(require('./mock-redeem-delegation'))).then(module => {
        redeemDelegation = module.redeemDelegation;
        utils_1.logger.info('SERVICE.TS: Successfully loaded MOCK blockchain service');
    }).catch(error => {
        utils_1.logger.error('SERVICE.TS: Failed to load mock blockchain service:', error);
    });
}
else {
    utils_1.logger.info('SERVICE.TS: Running in REAL MODE - using real blockchain service');
    Promise.resolve().then(() => __importStar(require('./redeem-delegation'))).then(module => {
        redeemDelegation = module.redeemDelegation;
        utils_1.logger.info('SERVICE.TS: Successfully loaded REAL blockchain service');
    }).catch(error => {
        utils_1.logger.error('SERVICE.TS: Failed to load real blockchain service:', error);
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
        try {
            utils_1.logger.info('Received RedeemDelegation request');
            // Check if the implementation was loaded
            if (!redeemDelegation) {
                utils_1.logger.error('Blockchain service implementation not loaded yet');
                callback(null, {
                    transaction_hash: "",
                    transactionHash: "",
                    success: false,
                    error_message: "Service not ready yet, try again later",
                    errorMessage: "Service not ready yet, try again later"
                });
                return;
            }
            // Extract request parameters
            const signature = call.request.signature;
            const merchantAddress = call.request.merchant_address || call.request.merchantAddress;
            const tokenContractAddress = call.request.token_contract_address || call.request.tokenContractAddress;
            const price = call.request.price;
            utils_1.logger.info('Request parameters:', {
                signatureLength: signature ? signature.length : 0,
                merchantAddress,
                tokenContractAddress,
                price
            });
            // Call the implementation
            const transactionHash = await redeemDelegation(signature, merchantAddress, tokenContractAddress, price);
            utils_1.logger.info(`Redemption successful, transaction hash: ${transactionHash}`);
            // Send success response with both snake_case and camelCase fields for compatibility
            callback(null, {
                transaction_hash: transactionHash,
                transactionHash: transactionHash,
                success: true,
                error_message: "",
                errorMessage: ""
            });
        }
        catch (error) {
            // Handle errors
            const errorMessage = error instanceof Error ? error.message : String(error);
            utils_1.logger.error('Error in redeemDelegation:', errorMessage);
            // Send error response with both snake_case and camelCase fields for compatibility
            callback(null, {
                transaction_hash: "",
                transactionHash: "",
                success: false,
                error_message: errorMessage,
                errorMessage: errorMessage
            });
        }
    }
};
