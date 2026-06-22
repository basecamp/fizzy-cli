package harness

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// Execute runs the CLI binary with the given arguments and returns the result.
func Execute(binaryPath string, args []string, env map[string]string) *Result {
	cmd := exec.Command(binaryPath, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Set up environment
	baseEnv := os.Environ()
	if len(env) > 0 {
		overrides := make(map[string]struct{}, len(env))
		for k := range env {
			overrides[k] = struct{}{}
		}
		filtered := baseEnv[:0]
		for _, entry := range baseEnv {
			key, _, _ := strings.Cut(entry, "=")
			if _, ok := overrides[key]; ok {
				continue
			}
			filtered = append(filtered, entry)
		}
		baseEnv = filtered
	}
	cmd.Env = append([]string(nil), baseEnv...)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	err := cmd.Run()

	result := &Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				result.ExitCode = status.ExitStatus()
			}
		} else {
			// Command failed to start
			result.ExitCode = -1
			result.Stderr = err.Error()
		}
	}

	// Try to parse JSON response
	if result.Stdout != "" {
		var resp Response
		if err := json.Unmarshal([]byte(result.Stdout), &resp); err != nil {
			result.ParseError = err
		} else {
			result.Response = &resp
		}
	}

	return result
}
