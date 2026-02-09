package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kubedoio/n-kudo/internal/edge/logger"
	"github.com/kubedoio/n-kudo/internal/edge/metrics"
	"github.com/kubedoio/n-kudo/internal/edge/state"
)

type Executor struct {
	Store    *state.Store
	Provider MicroVMProvider
	Logs     LogSink
}

func (e *Executor) ExecutePlan(ctx context.Context, plan Plan) (PlanResult, error) {
	if strings.TrimSpace(plan.ExecutionID) == "" {
		return PlanResult{}, errors.New("execution_id required")
	}
	result := PlanResult{PlanID: plan.PlanID, ExecutionID: plan.ExecutionID, Results: make([]ActionResult, 0, len(plan.Actions))}

	for _, action := range plan.Actions {
		r := e.executeAction(ctx, plan.ExecutionID, action)
		result.Results = append(result.Results, r)
		if !r.OK {
			return result, fmt.Errorf("action %s failed: %s", action.ActionID, r.Message)
		}
	}
	return result, nil
}

func (e *Executor) executeAction(parent context.Context, executionID string, action Action) ActionResult {
	start := time.Now()
	startedAt := time.Now().UTC()

	// Log start
	logger.WithComponent("executor").WithFields(map[string]interface{}{
		"action_id":    action.ActionID,
		"action_type":  action.Type,
		"execution_id": executionID,
	}).Info("Starting action execution")

	log := func(level, msg string) {
		if e.Logs != nil {
			e.Logs.Write(parent, LogEntry{ExecutionID: executionID, ActionID: action.ActionID, Level: level, Message: msg})
		}
	}

	if cached, found, err := e.Store.GetActionRecord(action.ActionID); err == nil && found {
		log("INFO", "action reused from idempotency cache")
		logger.WithComponent("executor").WithFields(map[string]interface{}{
			"action_id":   action.ActionID,
			"action_type": action.Type,
		}).Info("action reused from idempotency cache")
		return ActionResult{
			ExecutionID: executionID,
			ActionID:    action.ActionID,
			OK:          cached.OK,
			ErrorCode:   cached.ErrorCode,
			Message:     cached.Message,
			StartedAt:   startedAt,
			FinishedAt:  time.Now().UTC(),
		}
	}

	ctx := parent
	if action.TimeoutSecond > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(parent, time.Duration(action.TimeoutSecond)*time.Second)
		defer cancel()
	}

	res := ActionResult{
		ExecutionID: executionID,
		ActionID:    action.ActionID,
		StartedAt:   startedAt,
		FinishedAt:  time.Now().UTC(),
	}

	var err error
	var cmdResult *CommandResult
	switch action.Type {
	case ActionMicroVMCreate:
		var params MicroVMParams
		err = json.Unmarshal(action.Params, &params)
		if err == nil {
			err = e.Provider.Create(ctx, params)
		}
	case ActionMicroVMStart:
		var params MicroVMParams
		err = json.Unmarshal(action.Params, &params)
		if err == nil {
			err = e.Provider.Start(ctx, params.VMID)
		}
	case ActionMicroVMStop:
		var params MicroVMParams
		err = json.Unmarshal(action.Params, &params)
		if err == nil {
			err = e.Provider.Stop(ctx, params.VMID)
		}
	case ActionMicroVMDelete:
		var params MicroVMParams
		err = json.Unmarshal(action.Params, &params)
		if err == nil {
			err = e.Provider.Delete(ctx, params.VMID)
		}
	case ActionMicroVMPause:
		err = e.executePause(ctx, action)
	case ActionMicroVMResume:
		err = e.executeResume(ctx, action)
	case ActionMicroVMSnapshot:
		err = e.executeSnapshot(ctx, action)
	case ActionCommandExecute:
		cmdResult, err = e.executeCommand(ctx, action)
	default:
		err = fmt.Errorf("unknown action type: %s", action.Type)
	}

	res.FinishedAt = time.Now().UTC()
	duration := time.Since(start)

	// Record metrics
	status := "success"
	if err != nil {
		res.OK = false
		res.ErrorCode = "ACTION_FAILED"
		res.Message = err.Error()
		status = "failure"
		log("ERROR", "action failed: "+err.Error())
	} else {
		res.OK = true
		if cmdResult != nil {
			res.Message = fmt.Sprintf("Command exited with code %d", cmdResult.ExitCode)
			if cmdResult.Stdout != "" {
				log("INFO", "stdout: "+cmdResult.Stdout)
			}
			if cmdResult.Stderr != "" {
				log("WARN", "stderr: "+cmdResult.Stderr)
			}
		} else {
			res.Message = "ok"
		}
		log("INFO", "action completed")
	}

	// Update Prometheus metrics
	metrics.ActionDuration.WithLabelValues(string(action.Type)).Observe(duration.Seconds())
	metrics.ActionsExecuted.WithLabelValues(string(action.Type), status).Inc()

	// Log completion
	logger.WithComponent("executor").WithFields(map[string]interface{}{
		"action_id":    action.ActionID,
		"action_type":  action.Type,
		"execution_id": executionID,
		"duration_ms":  duration.Milliseconds(),
		"status":       status,
	}).Info("Action execution completed")

	record := state.ActionRecord{
		ActionID:    action.ActionID,
		ExecutionID: executionID,
		OK:          res.OK,
		ErrorCode:   res.ErrorCode,
		Message:     res.Message,
	}
	if putErr := e.Store.PutActionRecord(record); putErr != nil {
		log("WARN", "failed to store action result: "+putErr.Error())
		logger.WithComponent("executor").WithFields(map[string]interface{}{
			"action_id": action.ActionID,
			"error":     putErr.Error(),
		}).Warn("failed to store action result")
	}
	return res
}
