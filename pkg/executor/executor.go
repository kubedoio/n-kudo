package executor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/n-kudo/n-kudo-edge/pkg/state"
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
	result := PlanResult{ExecutionID: plan.ExecutionID, Results: make([]ActionResult, 0, len(plan.Actions))}

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
	startedAt := time.Now().UTC()
	log := func(level, msg string) {
		if e.Logs != nil {
			e.Logs.Write(parent, LogEntry{ExecutionID: executionID, ActionID: action.ActionID, Level: level, Message: msg})
		}
	}

	if cached, found, err := e.Store.GetActionRecord(action.ActionID); err == nil && found {
		log("INFO", "action reused from idempotency cache")
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
	default:
		err = fmt.Errorf("unknown action type: %s", action.Type)
	}

	res.FinishedAt = time.Now().UTC()
	if err != nil {
		res.OK = false
		res.ErrorCode = "ACTION_FAILED"
		res.Message = err.Error()
		log("ERROR", "action failed: "+err.Error())
	} else {
		res.OK = true
		res.Message = "ok"
		log("INFO", "action completed")
	}

	record := state.ActionRecord{
		ActionID:    action.ActionID,
		ExecutionID: executionID,
		OK:          res.OK,
		ErrorCode:   res.ErrorCode,
		Message:     res.Message,
	}
	if putErr := e.Store.PutActionRecord(record); putErr != nil {
		log("WARN", "failed to store action result: "+putErr.Error())
	}
	return res
}
