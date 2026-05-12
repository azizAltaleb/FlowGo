// package: flowgo.api.v1
// file: job_worker_service.proto

/* tslint:disable */
/* eslint-disable */

import * as grpc from "@grpc/grpc-js";
import * as job_worker_service_pb from "./job_worker_service_pb";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

interface IJobWorkerServiceService extends grpc.ServiceDefinition<grpc.UntypedServiceImplementation> {
    activateJobs: IJobWorkerServiceService_IActivateJobs;
    completeJob: IJobWorkerServiceService_ICompleteJob;
    failJob: IJobWorkerServiceService_IFailJob;
}

interface IJobWorkerServiceService_IActivateJobs extends grpc.MethodDefinition<job_worker_service_pb.ActivateJobsRequest, job_worker_service_pb.ActivateJobsResponse> {
    path: "/flowgo.api.v1.JobWorkerService/ActivateJobs";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<job_worker_service_pb.ActivateJobsRequest>;
    requestDeserialize: grpc.deserialize<job_worker_service_pb.ActivateJobsRequest>;
    responseSerialize: grpc.serialize<job_worker_service_pb.ActivateJobsResponse>;
    responseDeserialize: grpc.deserialize<job_worker_service_pb.ActivateJobsResponse>;
}
interface IJobWorkerServiceService_ICompleteJob extends grpc.MethodDefinition<job_worker_service_pb.CompleteJobRequest, job_worker_service_pb.CompleteJobResponse> {
    path: "/flowgo.api.v1.JobWorkerService/CompleteJob";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<job_worker_service_pb.CompleteJobRequest>;
    requestDeserialize: grpc.deserialize<job_worker_service_pb.CompleteJobRequest>;
    responseSerialize: grpc.serialize<job_worker_service_pb.CompleteJobResponse>;
    responseDeserialize: grpc.deserialize<job_worker_service_pb.CompleteJobResponse>;
}
interface IJobWorkerServiceService_IFailJob extends grpc.MethodDefinition<job_worker_service_pb.FailJobRequest, job_worker_service_pb.FailJobResponse> {
    path: "/flowgo.api.v1.JobWorkerService/FailJob";
    requestStream: false;
    responseStream: false;
    requestSerialize: grpc.serialize<job_worker_service_pb.FailJobRequest>;
    requestDeserialize: grpc.deserialize<job_worker_service_pb.FailJobRequest>;
    responseSerialize: grpc.serialize<job_worker_service_pb.FailJobResponse>;
    responseDeserialize: grpc.deserialize<job_worker_service_pb.FailJobResponse>;
}

export const JobWorkerServiceService: IJobWorkerServiceService;

export interface IJobWorkerServiceServer extends grpc.UntypedServiceImplementation {
    activateJobs: grpc.handleUnaryCall<job_worker_service_pb.ActivateJobsRequest, job_worker_service_pb.ActivateJobsResponse>;
    completeJob: grpc.handleUnaryCall<job_worker_service_pb.CompleteJobRequest, job_worker_service_pb.CompleteJobResponse>;
    failJob: grpc.handleUnaryCall<job_worker_service_pb.FailJobRequest, job_worker_service_pb.FailJobResponse>;
}

export interface IJobWorkerServiceClient {
    activateJobs(request: job_worker_service_pb.ActivateJobsRequest, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.ActivateJobsResponse) => void): grpc.ClientUnaryCall;
    activateJobs(request: job_worker_service_pb.ActivateJobsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.ActivateJobsResponse) => void): grpc.ClientUnaryCall;
    activateJobs(request: job_worker_service_pb.ActivateJobsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.ActivateJobsResponse) => void): grpc.ClientUnaryCall;
    completeJob(request: job_worker_service_pb.CompleteJobRequest, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.CompleteJobResponse) => void): grpc.ClientUnaryCall;
    completeJob(request: job_worker_service_pb.CompleteJobRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.CompleteJobResponse) => void): grpc.ClientUnaryCall;
    completeJob(request: job_worker_service_pb.CompleteJobRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.CompleteJobResponse) => void): grpc.ClientUnaryCall;
    failJob(request: job_worker_service_pb.FailJobRequest, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.FailJobResponse) => void): grpc.ClientUnaryCall;
    failJob(request: job_worker_service_pb.FailJobRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.FailJobResponse) => void): grpc.ClientUnaryCall;
    failJob(request: job_worker_service_pb.FailJobRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.FailJobResponse) => void): grpc.ClientUnaryCall;
}

export class JobWorkerServiceClient extends grpc.Client implements IJobWorkerServiceClient {
    constructor(address: string, credentials: grpc.ChannelCredentials, options?: Partial<grpc.ClientOptions>);
    public activateJobs(request: job_worker_service_pb.ActivateJobsRequest, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.ActivateJobsResponse) => void): grpc.ClientUnaryCall;
    public activateJobs(request: job_worker_service_pb.ActivateJobsRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.ActivateJobsResponse) => void): grpc.ClientUnaryCall;
    public activateJobs(request: job_worker_service_pb.ActivateJobsRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.ActivateJobsResponse) => void): grpc.ClientUnaryCall;
    public completeJob(request: job_worker_service_pb.CompleteJobRequest, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.CompleteJobResponse) => void): grpc.ClientUnaryCall;
    public completeJob(request: job_worker_service_pb.CompleteJobRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.CompleteJobResponse) => void): grpc.ClientUnaryCall;
    public completeJob(request: job_worker_service_pb.CompleteJobRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.CompleteJobResponse) => void): grpc.ClientUnaryCall;
    public failJob(request: job_worker_service_pb.FailJobRequest, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.FailJobResponse) => void): grpc.ClientUnaryCall;
    public failJob(request: job_worker_service_pb.FailJobRequest, metadata: grpc.Metadata, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.FailJobResponse) => void): grpc.ClientUnaryCall;
    public failJob(request: job_worker_service_pb.FailJobRequest, metadata: grpc.Metadata, options: Partial<grpc.CallOptions>, callback: (error: grpc.ServiceError | null, response: job_worker_service_pb.FailJobResponse) => void): grpc.ClientUnaryCall;
}
