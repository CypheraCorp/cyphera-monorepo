// package: delegation
// file: delegation.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";

export class RedeemDelegationRequest extends jspb.Message { 
    getSignature(): Uint8Array | string;
    getSignature_asU8(): Uint8Array;
    getSignature_asB64(): string;
    setSignature(value: Uint8Array | string): RedeemDelegationRequest;
    getMerchantAddress(): string;
    setMerchantAddress(value: string): RedeemDelegationRequest;
    getTokenContractAddress(): string;
    setTokenContractAddress(value: string): RedeemDelegationRequest;
    getTokenAmount(): number;
    setTokenAmount(value: number): RedeemDelegationRequest;
    getTokenDecimals(): number;
    setTokenDecimals(value: number): RedeemDelegationRequest;
    getChainId(): number;
    setChainId(value: number): RedeemDelegationRequest;
    getNetworkName(): string;
    setNetworkName(value: string): RedeemDelegationRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): RedeemDelegationRequest.AsObject;
    static toObject(includeInstance: boolean, msg: RedeemDelegationRequest): RedeemDelegationRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: RedeemDelegationRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): RedeemDelegationRequest;
    static deserializeBinaryFromReader(message: RedeemDelegationRequest, reader: jspb.BinaryReader): RedeemDelegationRequest;
}

export namespace RedeemDelegationRequest {
    export type AsObject = {
        signature: Uint8Array | string,
        merchantAddress: string,
        tokenContractAddress: string,
        tokenAmount: number,
        tokenDecimals: number,
        chainId: number,
        networkName: string,
    }
}

export class RedeemDelegationResponse extends jspb.Message { 
    getTransactionHash(): string;
    setTransactionHash(value: string): RedeemDelegationResponse;
    getSuccess(): boolean;
    setSuccess(value: boolean): RedeemDelegationResponse;
    getErrormessage(): string;
    setErrormessage(value: string): RedeemDelegationResponse;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): RedeemDelegationResponse.AsObject;
    static toObject(includeInstance: boolean, msg: RedeemDelegationResponse): RedeemDelegationResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: RedeemDelegationResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): RedeemDelegationResponse;
    static deserializeBinaryFromReader(message: RedeemDelegationResponse, reader: jspb.BinaryReader): RedeemDelegationResponse;
}

export namespace RedeemDelegationResponse {
    export type AsObject = {
        transactionHash: string,
        success: boolean,
        errormessage: string,
    }
}
