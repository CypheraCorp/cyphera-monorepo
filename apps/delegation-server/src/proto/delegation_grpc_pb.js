// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var delegation_pb = require('./delegation_pb.js');

function serialize_delegation_RedeemDelegationRequest(arg) {
  if (!(arg instanceof delegation_pb.RedeemDelegationRequest)) {
    throw new Error('Expected argument of type delegation.RedeemDelegationRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_delegation_RedeemDelegationRequest(buffer_arg) {
  return delegation_pb.RedeemDelegationRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_delegation_RedeemDelegationResponse(arg) {
  if (!(arg instanceof delegation_pb.RedeemDelegationResponse)) {
    throw new Error('Expected argument of type delegation.RedeemDelegationResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_delegation_RedeemDelegationResponse(buffer_arg) {
  return delegation_pb.RedeemDelegationResponse.deserializeBinary(new Uint8Array(buffer_arg));
}


var DelegationServiceService = exports.DelegationServiceService = {
  // Redeems a delegation
redeemDelegation: {
    path: '/delegation.DelegationService/RedeemDelegation',
    requestStream: false,
    responseStream: false,
    requestType: delegation_pb.RedeemDelegationRequest,
    responseType: delegation_pb.RedeemDelegationResponse,
    requestSerialize: serialize_delegation_RedeemDelegationRequest,
    requestDeserialize: deserialize_delegation_RedeemDelegationRequest,
    responseSerialize: serialize_delegation_RedeemDelegationResponse,
    responseDeserialize: deserialize_delegation_RedeemDelegationResponse,
  },
};

exports.DelegationServiceClient = grpc.makeGenericClientConstructor(DelegationServiceService, 'DelegationService');
