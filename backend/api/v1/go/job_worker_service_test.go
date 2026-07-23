package workflowapi

import (
	"bytes"
	"context"
	"net"
	"sync/atomic"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

const canonicalJobWorkerServiceName = "artificialflow.api.v1.JobWorkerService"

type compatibilityTestServer struct {
	UnimplementedJobWorkerServiceServer
	calls atomic.Int32
}

func (s *compatibilityTestServer) ActivateJobs(_ context.Context, req *ActivateJobsRequest) (*ActivateJobsResponse, error) {
	s.calls.Add(1)
	return &ActivateJobsResponse{
		Jobs: []*Job{{Type: req.GetJobType()}},
	}, nil
}

func startJobWorkerTestServer(t *testing.T) (*grpc.ClientConn, *compatibilityTestServer, *atomic.Int32) {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	interceptorCalls := &atomic.Int32{}
	interceptor := func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		interceptorCalls.Add(1)
		if md, ok := metadata.FromIncomingContext(ctx); !ok || len(md.Get("authorization")) == 0 {
			return nil, status.Error(codes.Unauthenticated, "authorization metadata required")
		}
		return handler(ctx, req)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(interceptor))
	service := &compatibilityTestServer{}
	RegisterJobWorkerServiceServer(grpcServer, service)

	go func() {
		_ = grpcServer.Serve(listener)
	}()

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		grpcServer.Stop()
		_ = listener.Close()
		t.Fatalf("create gRPC client: %v", err)
	}

	t.Cleanup(func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = listener.Close()
	})

	return conn, service, interceptorCalls
}

func TestCanonicalJobWorkerServicePath(t *testing.T) {
	conn, service, interceptorCalls := startJobWorkerTestServer(t)

	method := "/" + canonicalJobWorkerServiceName + "/ActivateJobs"
	request := &ActivateJobsRequest{JobType: "payment"}

	err := conn.Invoke(context.Background(), method, request, new(ActivateJobsResponse))
	if status.Code(err) != codes.Unauthenticated {
		t.Fatalf("expected auth interceptor rejection, got %v: %v", status.Code(err), err)
	}

	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer token")
	response := new(ActivateJobsResponse)
	if err := conn.Invoke(ctx, method, request, response); err != nil {
		t.Fatalf("invoke %s: %v", method, err)
	}
	if len(response.GetJobs()) != 1 || response.GetJobs()[0].GetType() != "payment" {
		t.Fatalf("unexpected response from %s: %v", method, response)
	}

	if got := interceptorCalls.Load(); got != 2 {
		t.Fatalf("expected interceptor on both calls, got %d", got)
	}
	if got := service.calls.Load(); got != 1 {
		t.Fatalf("expected one authenticated call, got %d", got)
	}
}

func TestProtobufNamespaceAndWireEncoding(t *testing.T) {
	const goPackage = "github.com/artificialflow/artificialflow/backend/api/v1/go;workflowapi"
	for name, descriptor := range map[string]protoreflect.FileDescriptor{
		"events":             File_backend_api_proto_events_proto,
		"job worker service": File_backend_api_proto_job_worker_service_proto,
	} {
		if got := string(descriptor.Package()); got != "artificialflow.api.v1" {
			t.Errorf("%s protobuf package = %q", name, got)
		}
		options, ok := descriptor.Options().(*descriptorpb.FileOptions)
		if !ok {
			t.Fatalf("%s options have type %T", name, descriptor.Options())
		}
		if got := options.GetGoPackage(); got != goPackage {
			t.Errorf("%s go_package = %q", name, got)
		}
	}

	var wire []byte
	wire = protowire.AppendTag(wire, 1, protowire.BytesType)
	wire = protowire.AppendString(wire, "worker-1")
	wire = protowire.AppendTag(wire, 2, protowire.BytesType)
	wire = protowire.AppendString(wire, "payment")
	wire = protowire.AppendTag(wire, 3, protowire.VarintType)
	wire = protowire.AppendVarint(wire, 7)
	wire = protowire.AppendTag(wire, 4, protowire.VarintType)
	wire = protowire.AppendVarint(wire, 500)

	request := &ActivateJobsRequest{
		WorkerName: "worker-1",
		JobType:    "payment",
		MaxJobs:    7,
		TimeoutMs:  500,
	}
	canonicalWire, err := proto.Marshal(request)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	if !bytes.Equal(canonicalWire, wire) {
		t.Fatalf("wire encoding changed: got %x, want %x", canonicalWire, wire)
	}

	decoded := new(ActivateJobsRequest)
	if err := proto.Unmarshal(wire, decoded); err != nil {
		t.Fatalf("unmarshal wire payload: %v", err)
	}
	if !proto.Equal(decoded, request) {
		t.Fatalf("wire payload decoded to %v, want %v", decoded, request)
	}
}
