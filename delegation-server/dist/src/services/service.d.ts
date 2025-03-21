interface RedeemDelegationRequest {
    request: {
        delegationData: Buffer;
    };
}
interface RedeemDelegationCallback {
    (error: Error | null, response: {
        transactionHash: string;
        success: boolean;
        errorMessage: string;
    }): void;
}
/**
 * Implementation of the DelegationService gRPC service
 */
export declare const delegationService: {
    /**
     * Redeems a delegation by processing the delegation data and executing on-chain transactions
     *
     * @param call - The gRPC call containing the delegation data
     * @param callback - The gRPC callback to return the result
     */
    redeemDelegation(call: RedeemDelegationRequest, callback: RedeemDelegationCallback): Promise<void>;
};
export {};
