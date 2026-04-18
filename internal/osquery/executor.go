package osquery

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

type SSHExecutor interface {
	Exec(sessionID string, cmd string) (string, error)
}

type Executor struct {
	ssh SSHExecutor
}

func NewExecutor(ssh SSHExecutor) *Executor {
	return &Executor{ssh: ssh}
}

func (e *Executor) Run(sessionID, query string) ([]map[string]interface{}, error) {
	// Mitigate SSH shell command injection by transmitting the query as base64
	// and decoding it directly into osqueryi, bypassing shell metacharacter evaluation.
	encodedQuery := base64.StdEncoding.EncodeToString([]byte(query))
	cmd := fmt.Sprintf("echo %s | base64 -d | osqueryi --json", encodedQuery)

	// Create a temporary channel or mechanism to capture output from this specific command execution.
	// We'll rely on the SSH service's exec capability. If SSHService.Exec doesn't exist, we must add it.
	// Wait, does SSHService have an Exec method? Let me check how snippets work.

	output, err := e.ssh.Exec(sessionID, cmd)
	if err != nil {
		return nil, fmt.Errorf("SSH exec failed: %w", err)
	}

	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		return nil, fmt.Errorf("failed to parse osquery output: %s", output)
	}

	return results, nil
}
