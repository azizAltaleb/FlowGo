// package: flowgo.api.v1
// file: job_worker_service.proto

/* tslint:disable */
/* eslint-disable */

import * as jspb from "google-protobuf";
import * as google_protobuf_timestamp_pb from "google-protobuf/google/protobuf/timestamp_pb";

export class ActivateJobsRequest extends jspb.Message { 
    getWorkerName(): string;
    setWorkerName(value: string): ActivateJobsRequest;
    getJobType(): string;
    setJobType(value: string): ActivateJobsRequest;
    getMaxJobs(): number;
    setMaxJobs(value: number): ActivateJobsRequest;
    getTimeoutMs(): number;
    setTimeoutMs(value: number): ActivateJobsRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ActivateJobsRequest.AsObject;
    static toObject(includeInstance: boolean, msg: ActivateJobsRequest): ActivateJobsRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ActivateJobsRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ActivateJobsRequest;
    static deserializeBinaryFromReader(message: ActivateJobsRequest, reader: jspb.BinaryReader): ActivateJobsRequest;
}

export namespace ActivateJobsRequest {
    export type AsObject = {
        workerName: string,
        jobType: string,
        maxJobs: number,
        timeoutMs: number,
    }
}

export class ActivateJobsResponse extends jspb.Message { 
    clearJobsList(): void;
    getJobsList(): Array<Job>;
    setJobsList(value: Array<Job>): ActivateJobsResponse;
    addJobs(value?: Job, index?: number): Job;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): ActivateJobsResponse.AsObject;
    static toObject(includeInstance: boolean, msg: ActivateJobsResponse): ActivateJobsResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: ActivateJobsResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): ActivateJobsResponse;
    static deserializeBinaryFromReader(message: ActivateJobsResponse, reader: jspb.BinaryReader): ActivateJobsResponse;
}

export namespace ActivateJobsResponse {
    export type AsObject = {
        jobsList: Array<Job.AsObject>,
    }
}

export class Job extends jspb.Message { 
    getKey(): number;
    setKey(value: number): Job;
    getId(): string;
    setId(value: string): Job;
    getType(): string;
    setType(value: string): Job;
    getProcessInstanceKey(): number;
    setProcessInstanceKey(value: number): Job;
    getBpmnProcessId(): string;
    setBpmnProcessId(value: string): Job;
    getProcessDefinitionVersion(): number;
    setProcessDefinitionVersion(value: number): Job;
    getProcessDefinitionKey(): number;
    setProcessDefinitionKey(value: number): Job;
    getElementInstanceKey(): number;
    setElementInstanceKey(value: number): Job;
    getElementId(): string;
    setElementId(value: string): Job;

    getCustomHeadersMap(): jspb.Map<string, string>;
    clearCustomHeadersMap(): void;
    getWorker(): string;
    setWorker(value: string): Job;
    getRetries(): number;
    setRetries(value: number): Job;
    getDeadline(): number;
    setDeadline(value: number): Job;
    getVariables(): string;
    setVariables(value: string): Job;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): Job.AsObject;
    static toObject(includeInstance: boolean, msg: Job): Job.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: Job, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): Job;
    static deserializeBinaryFromReader(message: Job, reader: jspb.BinaryReader): Job;
}

export namespace Job {
    export type AsObject = {
        key: number,
        id: string,
        type: string,
        processInstanceKey: number,
        bpmnProcessId: string,
        processDefinitionVersion: number,
        processDefinitionKey: number,
        elementInstanceKey: number,
        elementId: string,

        customHeadersMap: Array<[string, string]>,
        worker: string,
        retries: number,
        deadline: number,
        variables: string,
    }
}

export class CompleteJobRequest extends jspb.Message { 
    getJobKey(): number;
    setJobKey(value: number): CompleteJobRequest;
    getWorkerName(): string;
    setWorkerName(value: string): CompleteJobRequest;
    getVariables(): string;
    setVariables(value: string): CompleteJobRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CompleteJobRequest.AsObject;
    static toObject(includeInstance: boolean, msg: CompleteJobRequest): CompleteJobRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CompleteJobRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CompleteJobRequest;
    static deserializeBinaryFromReader(message: CompleteJobRequest, reader: jspb.BinaryReader): CompleteJobRequest;
}

export namespace CompleteJobRequest {
    export type AsObject = {
        jobKey: number,
        workerName: string,
        variables: string,
    }
}

export class CompleteJobResponse extends jspb.Message { 

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): CompleteJobResponse.AsObject;
    static toObject(includeInstance: boolean, msg: CompleteJobResponse): CompleteJobResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: CompleteJobResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): CompleteJobResponse;
    static deserializeBinaryFromReader(message: CompleteJobResponse, reader: jspb.BinaryReader): CompleteJobResponse;
}

export namespace CompleteJobResponse {
    export type AsObject = {
    }
}

export class FailJobRequest extends jspb.Message { 
    getJobKey(): number;
    setJobKey(value: number): FailJobRequest;
    getWorkerName(): string;
    setWorkerName(value: string): FailJobRequest;
    getRetries(): number;
    setRetries(value: number): FailJobRequest;
    getErrorMessage(): string;
    setErrorMessage(value: string): FailJobRequest;
    getRetryBackoffMs(): number;
    setRetryBackoffMs(value: number): FailJobRequest;

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): FailJobRequest.AsObject;
    static toObject(includeInstance: boolean, msg: FailJobRequest): FailJobRequest.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: FailJobRequest, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): FailJobRequest;
    static deserializeBinaryFromReader(message: FailJobRequest, reader: jspb.BinaryReader): FailJobRequest;
}

export namespace FailJobRequest {
    export type AsObject = {
        jobKey: number,
        workerName: string,
        retries: number,
        errorMessage: string,
        retryBackoffMs: number,
    }
}

export class FailJobResponse extends jspb.Message { 

    serializeBinary(): Uint8Array;
    toObject(includeInstance?: boolean): FailJobResponse.AsObject;
    static toObject(includeInstance: boolean, msg: FailJobResponse): FailJobResponse.AsObject;
    static extensions: {[key: number]: jspb.ExtensionFieldInfo<jspb.Message>};
    static extensionsBinary: {[key: number]: jspb.ExtensionFieldBinaryInfo<jspb.Message>};
    static serializeBinaryToWriter(message: FailJobResponse, writer: jspb.BinaryWriter): void;
    static deserializeBinary(bytes: Uint8Array): FailJobResponse;
    static deserializeBinaryFromReader(message: FailJobResponse, reader: jspb.BinaryReader): FailJobResponse;
}

export namespace FailJobResponse {
    export type AsObject = {
    }
}
