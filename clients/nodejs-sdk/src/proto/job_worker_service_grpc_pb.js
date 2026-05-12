// GENERATED CODE -- DO NOT EDIT!

'use strict';
var grpc = require('@grpc/grpc-js');
var job_worker_service_pb = require('./job_worker_service_pb.js');
var google_protobuf_timestamp_pb = require('google-protobuf/google/protobuf/timestamp_pb.js');

function serialize_flowgo_api_v1_ActivateJobsRequest(arg) {
  if (!(arg instanceof job_worker_service_pb.ActivateJobsRequest)) {
    throw new Error('Expected argument of type flowgo.api.v1.ActivateJobsRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_flowgo_api_v1_ActivateJobsRequest(buffer_arg) {
  return job_worker_service_pb.ActivateJobsRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_flowgo_api_v1_ActivateJobsResponse(arg) {
  if (!(arg instanceof job_worker_service_pb.ActivateJobsResponse)) {
    throw new Error('Expected argument of type flowgo.api.v1.ActivateJobsResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_flowgo_api_v1_ActivateJobsResponse(buffer_arg) {
  return job_worker_service_pb.ActivateJobsResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_flowgo_api_v1_CompleteJobRequest(arg) {
  if (!(arg instanceof job_worker_service_pb.CompleteJobRequest)) {
    throw new Error('Expected argument of type flowgo.api.v1.CompleteJobRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_flowgo_api_v1_CompleteJobRequest(buffer_arg) {
  return job_worker_service_pb.CompleteJobRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_flowgo_api_v1_CompleteJobResponse(arg) {
  if (!(arg instanceof job_worker_service_pb.CompleteJobResponse)) {
    throw new Error('Expected argument of type flowgo.api.v1.CompleteJobResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_flowgo_api_v1_CompleteJobResponse(buffer_arg) {
  return job_worker_service_pb.CompleteJobResponse.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_flowgo_api_v1_FailJobRequest(arg) {
  if (!(arg instanceof job_worker_service_pb.FailJobRequest)) {
    throw new Error('Expected argument of type flowgo.api.v1.FailJobRequest');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_flowgo_api_v1_FailJobRequest(buffer_arg) {
  return job_worker_service_pb.FailJobRequest.deserializeBinary(new Uint8Array(buffer_arg));
}

function serialize_flowgo_api_v1_FailJobResponse(arg) {
  if (!(arg instanceof job_worker_service_pb.FailJobResponse)) {
    throw new Error('Expected argument of type flowgo.api.v1.FailJobResponse');
  }
  return Buffer.from(arg.serializeBinary());
}

function deserialize_flowgo_api_v1_FailJobResponse(buffer_arg) {
  return job_worker_service_pb.FailJobResponse.deserializeBinary(new Uint8Array(buffer_arg));
}


var JobWorkerServiceService = exports.JobWorkerServiceService = {
  // ActivateJobs polls for jobs available for a specific worker/type.
// It returns a stream of jobs to allow for immediate pushing if long-polling.
// For MVP, request/response is fine, but stream is future-proof.
activateJobs: {
    path: '/flowgo.api.v1.JobWorkerService/ActivateJobs',
    requestStream: false,
    responseStream: false,
    requestType: job_worker_service_pb.ActivateJobsRequest,
    responseType: job_worker_service_pb.ActivateJobsResponse,
    requestSerialize: serialize_flowgo_api_v1_ActivateJobsRequest,
    requestDeserialize: deserialize_flowgo_api_v1_ActivateJobsRequest,
    responseSerialize: serialize_flowgo_api_v1_ActivateJobsResponse,
    responseDeserialize: deserialize_flowgo_api_v1_ActivateJobsResponse,
  },
  // CompleteJob marks a job as completed with variables.
completeJob: {
    path: '/flowgo.api.v1.JobWorkerService/CompleteJob',
    requestStream: false,
    responseStream: false,
    requestType: job_worker_service_pb.CompleteJobRequest,
    responseType: job_worker_service_pb.CompleteJobResponse,
    requestSerialize: serialize_flowgo_api_v1_CompleteJobRequest,
    requestDeserialize: deserialize_flowgo_api_v1_CompleteJobRequest,
    responseSerialize: serialize_flowgo_api_v1_CompleteJobResponse,
    responseDeserialize: deserialize_flowgo_api_v1_CompleteJobResponse,
  },
  // FailJob marks a job as failed with retries.
failJob: {
    path: '/flowgo.api.v1.JobWorkerService/FailJob',
    requestStream: false,
    responseStream: false,
    requestType: job_worker_service_pb.FailJobRequest,
    responseType: job_worker_service_pb.FailJobResponse,
    requestSerialize: serialize_flowgo_api_v1_FailJobRequest,
    requestDeserialize: deserialize_flowgo_api_v1_FailJobRequest,
    responseSerialize: serialize_flowgo_api_v1_FailJobResponse,
    responseDeserialize: deserialize_flowgo_api_v1_FailJobResponse,
  },
};

exports.JobWorkerServiceClient = grpc.makeGenericClientConstructor(JobWorkerServiceService, 'JobWorkerService');
