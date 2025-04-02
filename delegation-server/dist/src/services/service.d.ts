import { ServerUnaryCall, sendUnaryData } from '@grpc/grpc-js';
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
    redeemDelegation(call: ServerUnaryCall<any, any>, callback: sendUnaryData<any>): Promise<void>;
};
