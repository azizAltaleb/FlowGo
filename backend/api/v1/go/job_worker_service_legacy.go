package workflowapi

import "google.golang.org/grpc"

// LegacyJobWorkerServiceName is the pre-migration service name kept temporarily
// so existing clients can continue to use the same protobuf wire contract.
const LegacyJobWorkerServiceName = "flowgo.api.v1.JobWorkerService"

// RegisterLegacyJobWorkerServiceServer registers the legacy service name with
// the generated handlers and server implementation used by the canonical API.
func RegisterLegacyJobWorkerServiceServer(s grpc.ServiceRegistrar, srv JobWorkerServiceServer) {
	legacyServiceDesc := JobWorkerService_ServiceDesc
	legacyServiceDesc.ServiceName = LegacyJobWorkerServiceName
	s.RegisterService(&legacyServiceDesc, srv)
}
