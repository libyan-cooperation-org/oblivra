package osquery

import (
	"encoding/json"
	"fmt"
)

type SSHExecutor interface {
	Exec(sessionID string, cmd string) (string, error)
	ExecWithStdin(sessionID string, cmd string, stdin string) (string, error)
}

type Executor struct {
	ssh SSHExecutor
}

func NewExecutor(ssh SSHExecutor) *Executor {
	return &Executor{ssh: ssh}
}

func (e *Executor) Run(sessionID, query string) ([]map[string]interface{}, error) {
	// Mitigate SSH shell command injection by transmitting the query via stdin
	// directly into osqueryi, bypassing shell metacharacter evaluation in the query itself.
	output, err := e.ssh.ExecWithStdin(sessionID, "osqueryi --json", query)
	if err != nil {
		return nil, fmt.Errorf("SSH exec failed: %w", err)
	}

	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		return nil, fmt.Errorf("failed to parse osquery output: %s", output)
	}

	return results, nil
}
