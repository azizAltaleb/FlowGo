package grpc

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	pb "workflow-engine/backend/api/v1/go"
	"workflow-engine/backend/services/workflow-command/internal/application"
)

type Server struct {
	pb.UnimplementedJobWorkerServiceServer
	engine *application.Engine
}

func NewServer(engine *application.Engine) *Server {
	return &Server{
		engine: engine,
	}
}

func (s *Server) ActivateJobs(ctx context.Context, req *pb.ActivateJobsRequest) (*pb.ActivateJobsResponse, error) {
	// Engine methods use simple arguments, not Command structs?
	// Let's check jobs.go signature:
	// ActivateJobs(ctx, jobType, worker, maxJobs, requestTimeout, lockDuration)
	jobs, err := s.engine.ActivateJobs(ctx, req.JobType, req.WorkerName, int(req.MaxJobs), time.Duration(req.TimeoutMs)*time.Millisecond, 20*time.Second) // Default lock duration?
	if err != nil {
		return nil, err
	}

	pbJobs := make([]*pb.Job, 0, len(jobs))
	for _, job := range jobs {
		// Custom Headers - Not available in model.Job yet
		// headers := make(map[string]string)

		var deadline int64
		if job.LockExpirationTime != nil {
			deadline = job.LockExpirationTime.UnixMilli()
		}

		pbJobs = append(pbJobs, &pb.Job{
			Key:                job.Key,
			Id:                 job.ID,
			Type:               job.Type,
			ProcessInstanceKey: job.ProcessInstanceKey,
			// BpmnProcessId:          job.BpmnProcessID, // Missing in Job
			// ProcessDefinitionVersion: int32(job.ProcessDefinitionVersion), // Missing in Job
			ProcessDefinitionKey: job.ProcessDefinitionKey,
			ElementInstanceKey:   job.ElementInstanceKey,
			ElementId:            job.ElementID,
			// CustomHeaders:          headers,
			Worker:   job.Worker,
			Retries:  safeInt32(job.Retries),
			Deadline: deadline,
			// Variables:              string(varsBytes), // Not available in model.Job yet
		})
	}

	return &pb.ActivateJobsResponse{Jobs: pbJobs}, nil
}

func (s *Server) CompleteJob(ctx context.Context, req *pb.CompleteJobRequest) (*pb.CompleteJobResponse, error) {
	var variables map[string]any
	if req.Variables != "" {
		if err := json.Unmarshal([]byte(req.Variables), &variables); err != nil {
			return nil, fmt.Errorf("invalid variables JSON: %w", err)
		}
	}

	err := s.engine.CompleteJob(ctx, req.JobKey, req.WorkerName, variables)
	if err != nil {
		return nil, err
	}

	return &pb.CompleteJobResponse{}, nil
}

func (s *Server) FailJob(ctx context.Context, req *pb.FailJobRequest) (*pb.FailJobResponse, error) {
	retries := int(req.Retries)
	err := s.engine.FailJob(ctx, req.JobKey, req.WorkerName, req.ErrorMessage, &retries)
	if err != nil {
		return nil, err
	}

	return &pb.FailJobResponse{}, nil
}

func safeInt32(value int) int32 {
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	if value < math.MinInt32 {
		return math.MinInt32
	}
	return int32(value)
}
