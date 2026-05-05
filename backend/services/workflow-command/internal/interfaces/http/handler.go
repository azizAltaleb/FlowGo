package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"workflow-engine/backend/libs/auth"
	"workflow-engine/backend/libs/iam"
	"workflow-engine/backend/libs/logger"
	"workflow-engine/backend/libs/model"
	workerproto "workflow-engine/backend/libs/worker"
	"workflow-engine/backend/services/workflow-command/internal/application"
	"workflow-engine/backend/services/workflow-command/internal/interfaces/http/dto"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type Handler struct {
	engine          *application.Engine
	identityConfig  iam.DeploymentConfig
	identityManager *iam.ZITADELManagementClient
	log             *logger.Logger
}

const idempotencyKeyHeader = "Idempotency-Key"

const maxIdempotencyKeyLength = 128

func NewHandler(e *application.Engine, identityConfig iam.DeploymentConfig) *Handler {
	return &Handler{
		engine:          e,
		identityConfig:  identityConfig,
		identityManager: iam.NewZITADELManagementClient(identityConfig.ZITADELManagement),
		log:             logger.New("command-handler"),
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	readOnly := func(fn http.HandlerFunc) http.Handler {
		return auth.RequireAnyRole(auth.RoleWorkflowsaAdmin, auth.RoleWorkflowsaClient, auth.RoleWorkflowsaViewer)(http.HandlerFunc(fn))
	}
	adminOnly := func(fn http.HandlerFunc) http.Handler {
		return auth.RequireAnyRole(auth.RoleWorkflowsaAdmin)(http.HandlerFunc(fn))
	}
	adminOrClient := func(fn http.HandlerFunc) http.Handler {
		return auth.RequireAnyRole(auth.RoleWorkflowsaAdmin, auth.RoleWorkflowsaClient)(http.HandlerFunc(fn))
	}

	r.HandleFunc("/identity/config", h.getIdentityConfig).Methods("GET")
	if h.identityConfig.Mode == iam.DeploymentModeZITADEL {
		h.registerIdentityManagementRoutes(r)
	}
	r.Handle("/workflows", adminOrClient(h.deployWorkflow)).Methods("POST")
	r.Handle("/workflows", readOnly(h.listWorkflows)).Methods("GET")
	r.Handle("/workflows/{id}", readOnly(h.getWorkflow)).Methods("GET")
	r.Handle("/workflows/{id}", adminOnly(h.deleteWorkflow)).Methods("DELETE")
	r.Handle("/instances", adminOrClient(h.startInstance)).Methods("POST")
	r.Handle("/instances", readOnly(h.listInstances)).Methods("GET")
	r.Handle("/instances/{id}/variables", adminOrClient(h.updateVariables)).Methods("POST")
	r.Handle("/instances/{id}/complete", adminOrClient(h.completeTask)).Methods("POST")
	r.Handle("/instances/{id}", readOnly(h.getInstance)).Methods("GET")
	r.Handle("/instances/{id}", adminOnly(h.deleteInstance)).Methods("DELETE")
	r.Handle("/signals", adminOrClient(h.publishSignal)).Methods("POST")
	r.Handle("/messages", adminOrClient(h.publishMessage)).Methods("POST")
	r.Handle("/jobs/activate", adminOrClient(h.activateJobs)).Methods("POST")
	r.Handle("/jobs/capabilities", readOnly(h.jobsCapabilities)).Methods("GET")
	r.Handle("/jobs/{key}/complete", adminOrClient(h.completeJob)).Methods("POST")
	r.Handle("/jobs/{key}/fail", adminOrClient(h.failJob)).Methods("POST")
	r.Handle("/jobs/{key}/extend-lock", adminOrClient(h.extendJobLock)).Methods("POST")
	r.Handle("/internal/metrics", adminOnly(h.engineMetrics)).Methods("GET")

	r.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)
}

// @Summary Deploy a workflow
// @Description Deploy a new workflow definition using BPMN 2.0 XML
// @Tags workflows
// @Accept xml
// @Produce json
// @Param file body string true "BPMN XML File"
// @Success 200 {object} dto.WorkflowDefinitionResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /workflows [post]
func (h *Handler) deployWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)
	start := time.Now()

	h.log.Info(ctx, "deploying workflow", nil)

	body, readErr := io.ReadAll(r.Body)
	if readErr != nil {
		h.log.Error(ctx, "failed to read request body", map[string]any{"error": readErr.Error()})
		http.Error(w, readErr.Error(), http.StatusBadRequest)
		return
	}

	h.log.Debug(ctx, "received BPMN payload", map[string]any{"size": len(body)})

	wf, err := h.engine.DeployWorkflowFromBPMN(ctx, body)
	if err != nil {
		h.log.Error(ctx, "workflow deployment failed", map[string]any{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.log.Info(ctx, "workflow deployed", map[string]any{
		"workflow_id":   wf.ID,
		"workflow_name": wf.Name,
		"version":       wf.Version,
		"duration_ms":   time.Since(start).Milliseconds(),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.ToWorkflowResponse(wf))
}

// @Summary List workflows
// @Description Get a list of all deployed workflow definitions
// @Tags workflows
// @Produce json
// @Success 200 {array} dto.WorkflowDefinitionResponse
// @Failure 500 {string} string "Internal Server Error"
// @Router /workflows [get]
func (h *Handler) listWorkflows(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)

	wfs, err := h.engine.ListWorkflows(ctx)
	if err != nil {
		h.log.Error(ctx, "failed to list workflows", map[string]any{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.log.Debug(ctx, "listed workflows", map[string]any{"count": len(wfs)})

	responseWfs := make([]dto.WorkflowDefinitionResponse, len(wfs))
	for i, wf := range wfs {
		responseWfs[i] = dto.ToWorkflowResponse(wf)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseWfs)
}

// @Summary Get workflow definition
// @Description Get a specific workflow definition by ID or BPMN Process ID
// @Tags workflows
// @Produce json
// @Param id path string true "Workflow ID"
// @Success 200 {object} dto.WorkflowDefinitionResponse
// @Failure 404 {string} string "Not Found"
// @Router /workflows/{id} [get]
func (h *Handler) getWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)
	vars := mux.Vars(r)
	id := vars["id"]

	h.log.Debug(ctx, "getting workflow", map[string]any{"workflow_id": id})

	wf, err := h.engine.GetWorkflow(ctx, id)
	if err != nil {
		h.log.Warn(ctx, "workflow not found", map[string]any{"workflow_id": id, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.ToWorkflowResponse(wf))
}

// @Summary Delete workflow
// @Description Delete a workflow definition
// @Tags workflows
// @Param id path string true "Workflow ID"
// @Success 204 {string} string "No Content"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /workflows/{id} [delete]
func (h *Handler) deleteWorkflow(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)
	vars := mux.Vars(r)
	id := vars["id"]

	h.log.Info(ctx, "deleting workflow", map[string]any{"workflow_id": id})

	if err := h.engine.DeleteWorkflow(ctx, id); err != nil {
		h.log.Error(ctx, "failed to delete workflow", map[string]any{"workflow_id": id, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.log.Info(ctx, "workflow deleted", map[string]any{"workflow_id": id})
	w.WriteHeader(http.StatusNoContent)
}

// @Summary Start a workflow instance
// @Description Start a new instance of a workflow
// @Tags instances
// @Accept json
// @Produce json
// @Param request body dto.StartInstanceRequest true "Start Instance Request"
// @Success 200 {object} dto.WorkflowInstanceResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances [post]
func (h *Handler) startInstance(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)
	start := time.Now()

	var req dto.StartInstanceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error(ctx, "failed to decode start instance request", map[string]any{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.log.Info(ctx, "starting workflow instance", map[string]any{
		"workflow_id":  req.WorkflowID,
		"context_keys": len(req.Context),
	})

	instance, err := h.engine.StartInstance(ctx, req.WorkflowID, req.Context)
	if err != nil {
		h.log.Error(ctx, "failed to start instance", map[string]any{
			"workflow_id": req.WorkflowID,
			"error":       err.Error(),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.log.Info(ctx, "workflow instance started", map[string]any{
		"instance_id": instance.ID,
		"workflow_id": req.WorkflowID,
		"status":      instance.Status,
		"duration_ms": time.Since(start).Milliseconds(),
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.ToWorkflowInstanceResponse(instance))
}

// @Summary List active workflow instances
// @Description Get a list of all currently active workflow instances
// @Tags instances
// @Produce json
// @Success 200 {array} dto.WorkflowInstanceResponse
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances [get]
func (h *Handler) listInstances(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)

	instances, err := h.engine.ListActiveInstances(ctx)
	if err != nil {
		h.log.Error(ctx, "failed to list instances", map[string]any{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if instances == nil {
		instances = []*model.WorkflowInstance{}
	}

	h.log.Debug(ctx, "listed instances", map[string]any{"count": len(instances)})

	responseInstances := make([]dto.WorkflowInstanceResponse, len(instances))
	for i, instance := range instances {
		responseInstances[i] = dto.ToWorkflowInstanceResponse(instance)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseInstances)
}

// @Summary Update instance variables
// @Description Update variables (parameters) for a running workflow instance
// @Tags instances
// @Accept json
// @Produce json
// @Param id path string true "Instance ID"
// @Param request body dto.UpdateVariablesRequest true "Variables to update"
// @Success 200 {string} string "Variables updated"
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{id}/variables [post]
func (h *Handler) updateVariables(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)
	vars := mux.Vars(r)
	id := vars["id"]

	var req dto.UpdateVariablesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error(ctx, "failed to decode update variables request", map[string]any{
			"instance_id": id,
			"error":       err.Error(),
		})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Extract variable names for logging (not values for security)
	varNames := make([]string, 0, len(req.Variables))
	for k := range req.Variables {
		varNames = append(varNames, k)
	}

	h.log.Info(ctx, "updating instance variables", map[string]any{
		"instance_id":    id,
		"variable_names": varNames,
	})

	if err := h.engine.UpdateInstanceVariables(ctx, id, req.Variables); err != nil {
		h.log.Error(ctx, "failed to update variables", map[string]any{
			"instance_id": id,
			"error":       err.Error(),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.log.Info(ctx, "variables updated", map[string]any{
		"instance_id":    id,
		"variable_count": len(req.Variables),
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Variables updated"))
}

// @Summary Complete a task
// @Description Complete the current task for a workflow instance. Optionally specify step_id for parallel tasks.
// @Tags instances
// @Param id path string true "Instance ID"
// @Param request body dto.CompleteTaskRequest false "Complete Task Request"
// @Success 200 {string} string "Task completed"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{id}/complete [post]
func (h *Handler) completeTask(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)
	start := time.Now()
	vars := mux.Vars(r)
	id := vars["id"]

	var req dto.CompleteTaskRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	h.log.Info(ctx, "completing task", map[string]any{
		"instance_id": id,
		"step_id":     req.StepID,
	})

	if err := h.engine.CompleteTask(ctx, id, req.StepID); err != nil {
		h.log.Error(ctx, "failed to complete task", map[string]any{
			"instance_id": id,
			"step_id":     req.StepID,
			"error":       err.Error(),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.log.Info(ctx, "task completed", map[string]any{
		"instance_id": id,
		"step_id":     req.StepID,
		"duration_ms": time.Since(start).Milliseconds(),
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Task completed"))
}

// @Summary Activate jobs
// @Description Activate available jobs for an external worker. Supports long-poll timeout and lock duration.
// @Tags jobs
// @Accept json
// @Produce json
// @Param request body dto.ActivateJobsRequest false "Activate Jobs Request"
// @Success 200 {object} dto.ActivateJobsResponse
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /jobs/activate [post]
func (h *Handler) activateJobs(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)
	if err := validateWorkerProtocolVersion(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setWorkerProtocolHeaders(w)

	var req dto.ActivateJobsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		h.log.Error(ctx, "failed to decode activate jobs request", map[string]any{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	requestTimeout := time.Duration(req.TimeoutMs) * time.Millisecond
	lockDuration := time.Duration(req.LockDurationMs) * time.Millisecond

	jobs, err := h.engine.ActivateJobs(ctx, req.Type, req.Worker, req.MaxJobs, requestTimeout, lockDuration)
	if err != nil {
		h.log.Error(ctx, "failed to activate jobs", map[string]any{
			"job_type": req.Type,
			"worker":   req.Worker,
			"error":    err.Error(),
		})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	responseJobs := dto.ToJobResponses(jobs)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.ActivateJobsResponse{Jobs: responseJobs})
}

// @Summary Get worker API capabilities
// @Description Returns protocol version and supported worker capabilities.
// @Tags jobs
// @Produce json
// @Success 200 {object} dto.WorkerCapabilitiesResponse
// @Router /jobs/capabilities [get]
func (h *Handler) jobsCapabilities(w http.ResponseWriter, r *http.Request) {
	setWorkerProtocolHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(dto.WorkerCapabilitiesResponse{
		ProtocolVersion: workerproto.WorkerProtocolVersion,
		Capabilities: []string{
			"activate",
			"complete",
			"fail",
			"extend-lock",
		},
	})
}

// @Summary Complete a job
// @Description Complete an activated external worker job and optionally persist result variables.
// @Tags jobs
// @Accept json
// @Produce plain
// @Param key path string true "Job Key"
// @Param request body dto.CompleteJobRequest false "Complete Job Request"
// @Success 200 {string} string "Job completed"
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /jobs/{key}/complete [post]
func (h *Handler) completeJob(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)
	if err := validateWorkerProtocolVersion(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setWorkerProtocolHeaders(w)

	vars := mux.Vars(r)
	jobKey, err := strconv.ParseInt(vars["key"], 10, 64)
	if err != nil {
		http.Error(w, "invalid job key", http.StatusBadRequest)
		return
	}
	idempotencyKey, err := readIdempotencyKey(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	idempotencyOperation := fmt.Sprintf("jobs.complete:%d", jobKey)
	if idempotencyKey != "" {
		alreadyProcessed, err := h.engine.HasProcessedIdempotencyKey(ctx, idempotencyKey, idempotencyOperation)
		if err != nil {
			h.log.Error(ctx, "failed to check idempotency key", map[string]any{"error": err.Error(), "operation": idempotencyOperation})
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if alreadyProcessed {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Job completed"))
			return
		}
	}

	var req dto.CompleteJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		h.log.Error(ctx, "failed to decode complete job request", map[string]any{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.engine.CompleteJob(ctx, jobKey, req.Worker, req.Variables); err != nil {
		h.log.Error(ctx, "failed to complete job", map[string]any{"job_key": jobKey, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if idempotencyKey != "" {
		if err := h.engine.RecordIdempotencyKey(ctx, idempotencyKey, idempotencyOperation); err != nil {
			h.log.Warn(ctx, "failed to store idempotency key", map[string]any{"error": err.Error(), "operation": idempotencyOperation})
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Job completed"))
}

// @Summary Fail a job
// @Description Report external worker job failure and optionally override remaining retries.
// @Tags jobs
// @Accept json
// @Produce plain
// @Param key path string true "Job Key"
// @Param request body dto.FailJobRequest false "Fail Job Request"
// @Success 200 {string} string "Job failed"
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /jobs/{key}/fail [post]
func (h *Handler) failJob(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)
	if err := validateWorkerProtocolVersion(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setWorkerProtocolHeaders(w)

	vars := mux.Vars(r)
	jobKey, err := strconv.ParseInt(vars["key"], 10, 64)
	if err != nil {
		http.Error(w, "invalid job key", http.StatusBadRequest)
		return
	}
	idempotencyKey, err := readIdempotencyKey(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	idempotencyOperation := fmt.Sprintf("jobs.fail:%d", jobKey)
	if idempotencyKey != "" {
		alreadyProcessed, err := h.engine.HasProcessedIdempotencyKey(ctx, idempotencyKey, idempotencyOperation)
		if err != nil {
			h.log.Error(ctx, "failed to check idempotency key", map[string]any{"error": err.Error(), "operation": idempotencyOperation})
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if alreadyProcessed {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Job failed"))
			return
		}
	}

	var req dto.FailJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		h.log.Error(ctx, "failed to decode fail job request", map[string]any{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.engine.FailJob(ctx, jobKey, req.Worker, req.ErrorMessage, req.Retries); err != nil {
		h.log.Error(ctx, "failed to fail job", map[string]any{"job_key": jobKey, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if idempotencyKey != "" {
		if err := h.engine.RecordIdempotencyKey(ctx, idempotencyKey, idempotencyOperation); err != nil {
			h.log.Warn(ctx, "failed to store idempotency key", map[string]any{"error": err.Error(), "operation": idempotencyOperation})
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Job failed"))
}

// @Summary Extend job lock
// @Description Extend the lock for an activated job owned by the requesting worker.
// @Tags jobs
// @Accept json
// @Produce plain
// @Param key path string true "Job Key"
// @Param request body dto.ExtendJobLockRequest false "Extend Job Lock Request"
// @Success 200 {string} string "Job lock extended"
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /jobs/{key}/extend-lock [post]
func (h *Handler) extendJobLock(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)
	if err := validateWorkerProtocolVersion(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setWorkerProtocolHeaders(w)

	vars := mux.Vars(r)
	jobKey, err := strconv.ParseInt(vars["key"], 10, 64)
	if err != nil {
		http.Error(w, "invalid job key", http.StatusBadRequest)
		return
	}
	idempotencyKey, err := readIdempotencyKey(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	idempotencyOperation := fmt.Sprintf("jobs.extend-lock:%d", jobKey)
	if idempotencyKey != "" {
		alreadyProcessed, err := h.engine.HasProcessedIdempotencyKey(ctx, idempotencyKey, idempotencyOperation)
		if err != nil {
			h.log.Error(ctx, "failed to check idempotency key", map[string]any{"error": err.Error(), "operation": idempotencyOperation})
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if alreadyProcessed {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("Job lock extended"))
			return
		}
	}

	var req dto.ExtendJobLockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		h.log.Error(ctx, "failed to decode extend lock request", map[string]any{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	lockDuration := time.Duration(req.LockDurationMs) * time.Millisecond
	if err := h.engine.ExtendJobLock(ctx, jobKey, req.Worker, lockDuration); err != nil {
		h.log.Error(ctx, "failed to extend job lock", map[string]any{"job_key": jobKey, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if idempotencyKey != "" {
		if err := h.engine.RecordIdempotencyKey(ctx, idempotencyKey, idempotencyOperation); err != nil {
			h.log.Warn(ctx, "failed to store idempotency key", map[string]any{"error": err.Error(), "operation": idempotencyOperation})
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Job lock extended"))
}

func setWorkerProtocolHeaders(w http.ResponseWriter) {
	w.Header().Set(workerproto.HeaderEngineProtocolVersion, workerproto.WorkerProtocolVersion)
}

func validateWorkerProtocolVersion(r *http.Request) error {
	version := strings.TrimSpace(r.Header.Get(workerproto.HeaderWorkerProtocolVersion))
	if version == "" || version == workerproto.WorkerProtocolVersion {
		return nil
	}
	return fmt.Errorf("unsupported worker protocol version %q (supported: %s)", version, workerproto.WorkerProtocolVersion)
}

func readIdempotencyKey(r *http.Request) (string, error) {
	key := strings.TrimSpace(r.Header.Get(idempotencyKeyHeader))
	if key == "" {
		return "", nil
	}
	if len(key) > maxIdempotencyKeyLength {
		return "", fmt.Errorf("%s exceeds %d characters", idempotencyKeyHeader, maxIdempotencyKeyLength)
	}
	return key, nil
}

// Simple helper to expose reading state for verification
// @Summary Get workflow instance
// @Description Get the details of a workflow instance
// @Tags instances
// @Param id path string true "Instance ID"
// @Success 200 {object} dto.WorkflowInstanceResponse
// @Failure 404 {string} string "Not Found"
// @Router /instances/{id} [get]
func (h *Handler) getInstance(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)
	vars := mux.Vars(r)
	id := vars["id"]

	h.log.Debug(ctx, "getting instance", map[string]any{"instance_id": id})

	instance, err := h.engine.GetInstance(ctx, id)
	if err != nil {
		h.log.Warn(ctx, "instance not found", map[string]any{"instance_id": id, "error": err.Error()})
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.ToWorkflowInstanceResponse(instance))
}

func (h *Handler) engineMetrics(w http.ResponseWriter, r *http.Request) {
	snapshot := h.engine.MetricsSnapshot()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dto.EngineMetricsResponse{
		OutboxPending:        snapshot.OutboxPending,
		OutboxPublishSuccess: snapshot.OutboxPublishSuccess,
		OutboxPublishFailure: snapshot.OutboxPublishFailure,
		OutboxPublishLagSec:  snapshot.OutboxPublishLagSec,
		OutboxMaxAttempts:    h.engine.OutboxMaxAttempts(),
		IdempotencyHit:       snapshot.IdempotencyHit,
		IdempotencyMiss:      snapshot.IdempotencyMiss,
	})
}

// @Summary Delete workflow instance
// @Description Delete a workflow instance
// @Tags instances
// @Param id path string true "Instance ID"
// @Success 204 {string} string "No Content"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal Server Error"
// @Router /instances/{id} [delete]
func (h *Handler) deleteInstance(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)
	start := time.Now()
	vars := mux.Vars(r)
	id := vars["id"]

	h.log.Info(ctx, "deleting instance", map[string]any{"instance_id": id})

	if err := h.engine.DeleteInstance(ctx, id); err != nil {
		h.log.Error(ctx, "failed to delete instance", map[string]any{
			"instance_id": id,
			"error":       err.Error(),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.log.Info(ctx, "instance deleted", map[string]any{
		"instance_id": id,
		"duration_ms": time.Since(start).Milliseconds(),
	})

	w.WriteHeader(http.StatusNoContent)
}

// @Description Publish a signal to trigger intermediate catch events
// @Tags signals
// @Accept json
// @Produce json
// @Param request body dto.PublishSignalRequest true "Publish Signal Request"
// @Success 200 {string} string "Signal published"
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /signals [post]
func (h *Handler) publishSignal(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)

	var req dto.PublishSignalRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error(ctx, "failed to decode signal request", map[string]any{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.log.Info(ctx, "publishing signal", map[string]any{
		"signal_name": req.SignalName,
	})

	if err := h.engine.PublishSignal(ctx, req.SignalName, req.Payload); err != nil {
		h.log.Error(ctx, "failed to publish signal", map[string]any{
			"signal_name": req.SignalName,
			"error":       err.Error(),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.log.Info(ctx, "signal published", map[string]any{"signal_name": req.SignalName})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Signal published"))
}

// @Summary Publish a message
// @Description Publish a message to trigger intermediate catch events
// @Tags messages
// @Accept json
// @Produce json
// @Param request body dto.PublishMessageRequest true "Publish Message Request"
// @Success 200 {string} string "Message published"
// @Failure 400 {string} string "Bad Request"
// @Failure 500 {string} string "Internal Server Error"
// @Router /messages [post]
func (h *Handler) publishMessage(w http.ResponseWriter, r *http.Request) {
	ctx := logger.ContextFromRequest(r)

	var req dto.PublishMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Error(ctx, "failed to decode message request", map[string]any{"error": err.Error()})
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.log.Info(ctx, "publishing message", map[string]any{
		"message_name":    req.MessageName,
		"correlation_key": req.CorrelationKey,
	})

	if err := h.engine.PublishMessage(ctx, req.MessageName, req.CorrelationKey, req.Payload); err != nil {
		h.log.Error(ctx, "failed to publish message", map[string]any{
			"message_name":    req.MessageName,
			"correlation_key": req.CorrelationKey,
			"error":           err.Error(),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.log.Info(ctx, "message published", map[string]any{
		"message_name":    req.MessageName,
		"correlation_key": req.CorrelationKey,
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Message published"))
}
