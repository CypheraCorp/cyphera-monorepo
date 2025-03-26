// package: delegation
// file: delegation.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as delegation_pb from "./delegation_pb";

interface IDelegationServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    redeemDelegation: IDelegationServiceService_IRedeemDelegation;
}

interface IDelegationServiceService_IRedeemDelegation extends grpc.MethodDefinition<delegation_pb.RedeemDelegationRequest, delegation_pb.RedeemDelegationResponse> {
    path: "/delegation.DelegationService/RedeemDelegation";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<delegation_pb.RedeemDelegationRequest>;
    requestDeserialize: grpc.deserialize<delegation_pb.RedeemDelegationRequest>;
    responseSerialize: grpc.serialize<delegation_pb.RedeemDelegationResponse>;
    responseDeserialize: grpc.deserialize<delegation_pb.RedeemDelegationResponse>;
}

export const DelegationServiceService: IDelegationServiceService;

export interface IDelegationServiceServer extends grpc.UntypedServiceImplementation {
    redeemDelegation: grpc.handleUnaryCall<delegation_pb.RedeemDelegationRequest, delegation_pb.RedeemDelegationResponse>;
}

export interface IDelegationServiceClient {
    redeemDelegation(request: delegation_pb.RedeemDelegationRequest, callback: (error: grpc.ServiceError | null, response: delegation_pb.RedeemDelegationResponse) => void): grpc.ClientUnaryCall;
    redeemDelegation(request: delegation_pb.RedeemDelegationRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: delegation_pb.RedeemDelegationResponse) => void): grpc.ClientUnaryCall;
    redeemDelegation(request: delegation_pb.RedeemDelegationRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: delegation_pb.RedeemDelegationResponse) => void): grpc.ClientUnaryCall;
}

export class DelegationServiceClient extends grpc.Client implements IDelegationServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public redeemDelegation(request: delegation_pb.RedeemDelegationRequest, callback: (error: grpc.ServiceError | null, response: delegation_pb.RedeemDelegationResponse) => void): grpc.ClientUnaryCall;
    public redeemDelegation(request: delegation_pb.RedeemDelegationRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: delegation_pb.RedeemDelegationResponse) => void): grpc.ClientUnaryCall;
    public redeemDelegation(request: delegation_pb.RedeemDelegationRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: delegation_pb.RedeemDelegationResponse) => void): grpc.ClientUnaryCall;
}
