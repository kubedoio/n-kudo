package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"
)

type CommandResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

func (e *Executor) executeCommand(ctx context.Context, action Action) (*CommandResult, error) {
	var params CommandParams
	if err := json.Unmarshal(action.Params, &params); err != nil {
		return nil, fmt.Errorf("unmarshal command params: %w", err)
	}

	if params.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Set timeout (default 30 seconds)
	timeout := time.Duration(params.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, params.Command, params.Args...)
	if params.Dir != "" {
		cmd.Dir = params.Dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return &CommandResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}
