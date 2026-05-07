package application

import (
	"context"
	"fmt"
	pb "github.com/azizAltaleb/goflow/backend/api/v1/go"
	"github.com/azizAltaleb/goflow/backend/libs/model"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	defaultJobLockDuration = 30 * time.Second
	activationPollInterval = 100 * time.Millisecond
)

func (e *Engine) ActivateJobs(ctx context.Context, jobType, worker string, maxJobs int, requestTimeout, lockDuration time.Duration) ([]model.Job, error) {
	jobs, err := e.activateJobsOnce(ctx, jobType, worker, maxJobs, lockDuration)
	if err != nil {
		return nil, err
	}
	if len(jobs) > 0 || requestTimeout <= 0 {
		return jobs, nil
	}

	timeout := time.NewTimer(requestTimeout)
	defer timeout.Stop()
	ticker := time.NewTicker(activationPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout.C:
			return []model.Job{}, nil
		case <-ticker.C:
			jobs, err := e.activateJobsOnce(ctx, jobType, worker, maxJobs, lockDuration)
			if err != nil {
				return nil, err
			}
			if len(jobs) > 0 {
				return jobs, nil
			}
		}
	}
}

func (e *Engine) activateJobsOnce(ctx context.Context, jobType, worker string, maxJobs int, lockDuration time.Duration) ([]model.Job, error) {
	if worker == "" {
		return nil, fmt.Errorf("worker is required")
	}
	if lockDuration <= 0 {
		lockDuration = defaultJobLockDuration
	}

	var activated []model.Job
	err := e.withTx(ctx, func(txEngine *Engine) error {
		jobs, err := txEngine.repo.ListActivatableJobs(ctx, jobType, maxJobs)
		if err != nil {
			return err
		}

		now := time.Now()
		activated = make([]model.Job, 0, len(jobs))
		for _, job := range jobs {
			lockUntil := now.Add(lockDuration)
			job.State = "ACTIVATED"
			job.Worker = worker
			job.LockExpirationTime = &lockUntil
			job.UpdatedAt = now
			if err := txEngine.repo.UpdateJob(ctx, &job); err != nil {
				return err
			}
			activated = append(activated, job)

			// Publish JobActivated
			if err := txEngine.eventPublisher.Publish(ctx, &pb.JobActivated{
				Key:                job.Key,
				Worker:             worker,
				LockExpirationTime: timestamppb.New(*job.LockExpirationTime),
			}, "JobActivated"); err != nil {
				fmt.Printf("failed to publish JobActivated: %v\n", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if activated == nil {
		activated = []model.Job{}
	}

	return activated, nil
}

func (e *Engine) CompleteJob(ctx context.Context, jobKey int64, worker string, variables map[string]any) error {
	return e.withTx(ctx, func(txEngine *Engine) error {
		job, err := txEngine.repo.GetJob(ctx, jobKey)
		if err != nil {
			return err
		}

		if job.State == "COMPLETED" {
			return nil
		}
		if job.State != "ACTIVATED" {
			return fmt.Errorf("job %d is not activated", job.Key)
		}

		now := time.Now()
		if err := ensureActiveLockOwnership(job, worker, now); err != nil {
			return err
		}

		instanceID := strconv.FormatInt(job.ProcessInstanceKey, 10)
		if len(variables) > 0 {
			instance, err := txEngine.GetInstance(ctx, instanceID)
			if err != nil {
				return err
			}
			if instance.Context == nil {
				instance.Context = make(map[string]any)
			}
			for k, v := range variables {
				instance.Context[k] = v
			}
			if err := txEngine.persistVariables(ctx, instanceID, job.ProcessInstanceKey, instance.Context); err != nil {
				return err
			}
		}

		job.State = "COMPLETED"
		job.LockExpirationTime = nil
		job.UpdatedAt = now
		if err := txEngine.repo.UpdateJob(ctx, job); err != nil {
			return err
		}

		// Publish JobCompleted
		if err := txEngine.eventPublisher.Publish(ctx, &pb.JobCompleted{
			Key:       job.Key,
			Worker:    worker,
			Variables: "", // TODO: Serialize variables if needed
			UpdatedAt: timestamppb.New(now),
		}, "JobCompleted"); err != nil {
			fmt.Printf("failed to publish JobCompleted: %v\n", err)
		}

		executionID := strconv.FormatInt(job.ElementInstanceKey, 10)
		return txEngine.completeExecution(ctx, instanceID, executionID)
	})
}

func (e *Engine) FailJob(ctx context.Context, jobKey int64, worker, errorMessage string, retries *int) error {
	return e.withTx(ctx, func(txEngine *Engine) error {
		job, err := txEngine.repo.GetJob(ctx, jobKey)
		if err != nil {
			return err
		}

		if job.State == "COMPLETED" {
			return fmt.Errorf("job %d already completed", job.Key)
		}
		if job.State == "FAILED" {
			return nil
		}

		now := time.Now()
		if job.State == "ACTIVATED" {
			if err := ensureActiveLockOwnership(job, worker, now); err != nil {
				return err
			}
		}

		remainingRetries := job.Retries - 1
		if retries != nil {
			remainingRetries = *retries
		}
		if remainingRetries < 0 {
			remainingRetries = 0
		}

		job.Retries = remainingRetries
		job.Worker = ""
		job.LockExpirationTime = nil
		job.UpdatedAt = now

		if remainingRetries > 0 {
			job.State = "CREATED"
			return txEngine.repo.UpdateJob(ctx, job)
		}

		job.State = "FAILED"
		if err := txEngine.repo.UpdateJob(ctx, job); err != nil {
			return err
		}

		// Publish JobFailed
		if err := txEngine.eventPublisher.Publish(ctx, &pb.JobFailed{
			Key:          job.Key,
			Worker:       worker,
			Retries:      safeInt32(job.Retries),
			ErrorMessage: errorMessage,
			UpdatedAt:    timestamppb.New(now),
		}, "JobFailed"); err != nil {
			fmt.Printf("failed to publish JobFailed: %v\n", err)
		}

		if errorMessage == "" {
			errorMessage = "external worker failure"
		}
		incident := &model.Incident{
			Key:                generateKey(fmt.Sprintf("%d_incident", job.Key)),
			ProcessInstanceKey: job.ProcessInstanceKey,
			ElementInstanceKey: job.ElementInstanceKey,
			JobKey:             job.Key,
			ErrorType:          "JOB_NO_RETRIES",
			ErrorMessage:       errorMessage,
			State:              "CREATED",
			CreatedAt:          time.Now(),
		}
		return txEngine.repo.CreateIncident(ctx, incident)
	})
}

func (e *Engine) ExtendJobLock(ctx context.Context, jobKey int64, worker string, lockDuration time.Duration) error {
	if lockDuration <= 0 {
		return fmt.Errorf("lock duration must be greater than zero")
	}

	return e.withTx(ctx, func(txEngine *Engine) error {
		job, err := txEngine.repo.GetJob(ctx, jobKey)
		if err != nil {
			return err
		}
		if job.State != "ACTIVATED" {
			return fmt.Errorf("job %d is not activated", job.Key)
		}

		now := time.Now()
		if err := ensureActiveLockOwnership(job, worker, now); err != nil {
			return err
		}

		lockUntil := now.Add(lockDuration)
		job.LockExpirationTime = &lockUntil
		job.UpdatedAt = now
		return txEngine.repo.UpdateJob(ctx, job)
	})
}

func ensureActiveLockOwnership(job *model.Job, worker string, now time.Time) error {
	if worker == "" {
		return fmt.Errorf("worker is required")
	}
	if job.Worker != worker {
		return fmt.Errorf("job %d locked by worker %s", job.Key, job.Worker)
	}
	if job.LockExpirationTime == nil {
		return fmt.Errorf("job %d has no active lock", job.Key)
	}
	if now.After(*job.LockExpirationTime) {
		return fmt.Errorf("job %d lock expired", job.Key)
	}

	return nil
}

func (e *Engine) HasProcessedIdempotencyKey(ctx context.Context, key, operation string) (bool, error) {
	key = strings.TrimSpace(key)
	operation = strings.TrimSpace(operation)
	if key == "" || operation == "" {
		return false, nil
	}

	record, err := e.repo.GetIdempotencyRecord(ctx, key, operation)
	if err != nil {
		return false, err
	}
	if e.metrics != nil {
		if record != nil {
			e.metrics.incIdempotencyHit()
		} else {
			e.metrics.incIdempotencyMiss()
		}
	}

	return record != nil, nil
}

func (e *Engine) RecordIdempotencyKey(ctx context.Context, key, operation string) error {
	key = strings.TrimSpace(key)
	operation = strings.TrimSpace(operation)
	if key == "" || operation == "" {
		return nil
	}

	return e.repo.CreateIdempotencyRecord(ctx, &model.IdempotencyRecord{
		Key:       key,
		Operation: operation,
		CreatedAt: time.Now(),
	})
}
