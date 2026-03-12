package services

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	sshpkg "github.com/kingknull/oblivrashell/internal/ssh"
	"github.com/kingknull/oblivrashell/internal/vault"
)

type MultiExecService struct {
	BaseService
	hosts          database.HostStore
	vault          vault.Provider
	bus            *eventbus.Bus
	log            *logger.Logger
	jobs           map[string]*MultiExecJob
	mu             *sync.RWMutex
	maxConcurrency int
}

func (s *MultiExecService) Name() string { return "multiexec-service" }

// Dependencies returns service dependencies
func (s *MultiExecService) Dependencies() []string {
	return []string{"vault", "eventbus"}
}

func (s *MultiExecService) Start(ctx context.Context) error {
	return nil
}

func (s *MultiExecService) Stop(ctx context.Context) error {
	return nil
}

// DestructiveCheckResult holds the result of a safety analysis
type DestructiveCheckResult struct {
	IsDestructive bool     `json:"is_destructive"`
	Threats       []string `json:"threats"`
	Severity      string   `json:"severity"` // "low", "medium", "high", "critical"
}

func NewMultiExecService(
	h database.HostStore,
	v vault.Provider,
	bus *eventbus.Bus,
	log *logger.Logger,
) *MultiExecService {
	return &MultiExecService{
		hosts:          h,
		vault:          v,
		bus:            bus,
		log:            log.WithPrefix("multiexec"),
		jobs:           make(map[string]*MultiExecJob),
		mu:             &sync.RWMutex{},
		maxConcurrency: 5, // Default safe limit for batch runs
	}
}

// SetMaxConcurrency allows adjusting the batch size
func (s *MultiExecService) SetMaxConcurrency(n int) {
	if n < 1 {
		n = 1
	}
	if n > 50 {
		n = 50 // Upper safety bound
	}
	s.maxConcurrency = n
}

// CheckSafety analyzes a command for potentially destructive behavior
func (s *MultiExecService) CheckSafety(command string) DestructiveCheckResult {
	res := DestructiveCheckResult{
		IsDestructive: false,
		Threats:       make([]string, 0),
		Severity:      "low",
	}

	dangerousPatterns := map[string]string{
		"rm -rf /":      "Attempting to delete root directory",
		"rm -rf *":      "Attempting to delete all files in current directory",
		"mkfs":          "Filesystem creation detected",
		"dd if=":        "Low-level disk writing detected",
		"iptables -F":   "Flushing all firewall rules",
		"> /dev/sd":     "Direct disk overwrite detected",
		":(){ :|:& };:": "Fork bomb detected",
		"reboot":        "Remote reboot command detected",
		"shutdown":      "Remote shutdown command detected",
		"mv /*":         "Mass file move from root detected",
	}

	for pattern, threat := range dangerousPatterns {
		// Simple string contains for now, could be regex in future
		if containsIgnoreCase(command, pattern) {
			res.IsDestructive = true
			res.Threats = append(res.Threats, threat)
			res.Severity = "critical"
		}
	}

	if len(res.Threats) > 0 {
		s.log.Warn("Destructive command detected: %q - Threats: %v", command, res.Threats)
	}

	return res
}

func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// Execute runs a command on multiple hosts concurrently
func (s *MultiExecService) Execute(command string, hostIDs []string, timeoutSeconds int) (string, error) {
	if len(hostIDs) == 0 {
		return "", fmt.Errorf("no hosts specified")
	}

	if command == "" {
		return "", fmt.Errorf("empty command")
	}

	jobID := uuid.New().String()
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	// Initialize job
	results := make([]MultiExecResult, len(hostIDs))
	for i, hostID := range hostIDs {
		host, err := s.hosts.GetByID(context.Background(), hostID)
		if err != nil {
			results[i] = MultiExecResult{
				HostID:    hostID,
				HostLabel: "Unknown",
				Status:    "error",
				Error:     "host not found",
			}
			continue
		}
		results[i] = MultiExecResult{
			HostID:    hostID,
			HostLabel: host.Label,
			Hostname:  host.Hostname,
			Status:    "pending",
		}
	}

	job := &MultiExecJob{
		ID:        jobID,
		Command:   command,
		HostIDs:   hostIDs,
		Results:   results,
		StartedAt: time.Now().Format(time.RFC3339),
		Status:    "running",
	}

	s.mu.Lock()
	s.jobs[jobID] = job
	s.pruneJobs() // cap memory usage to maxJobHistory entries
	s.mu.Unlock()

	s.log.Info("Starting multi-exec job %s: command=%q hosts=%d", jobID, command, len(hostIDs))

	// Execute concurrently
	go s.executeAll(job, command, timeout)

	return jobID, nil
}

func (s *MultiExecService) prepareSSHConfig(host *database.Host, password []byte, privateKey []byte, passphrase []byte, connectTimeout time.Duration) sshpkg.ConnectionConfig {
	cfg := sshpkg.DefaultConfig()
	cfg.Host = host.Hostname
	cfg.Port = host.Port
	cfg.Username = host.Username
	cfg.ConnectTimeout = connectTimeout

	if len(password) > 0 {
		cfg.AuthMethod = sshpkg.AuthPassword
		cfg.Password = password
	} else if len(privateKey) > 0 {
		cfg.AuthMethod = sshpkg.AuthPublicKey
		cfg.PrivateKey = privateKey
		cfg.Passphrase = passphrase
	} else {
		// Fallback to default keys if no specific credentials provided
		defaultKeyData, _ := sshpkg.LoadDefaultKeys()
		cfg.AuthMethod = sshpkg.AuthPublicKey
		cfg.PrivateKey = defaultKeyData
	}
	return cfg
}

func (s *MultiExecService) executeAll(job *MultiExecJob, command string, timeout time.Duration) {
	var wg sync.WaitGroup
	// Execute with concurrency limiting
	sem := make(chan struct{}, s.maxConcurrency)
	for i, hostID := range job.HostIDs {
		if job.Results[i].Status == "error" {
			continue // Skip hosts that failed to resolve
		}

		wg.Add(1)
		sem <- struct{}{} // Acquire worker slot
		go func(index int, hID string) {
			defer wg.Done()
			defer func() { <-sem }() // Release worker slot
			s.executeOnHost(job, index, hID, command, timeout)
		}(i, hostID)
	}

	wg.Wait()

	// Determine final status
	nowStr := time.Now().Format(time.RFC3339)
	job.mu.Lock()
	job.EndedAt = &nowStr
	job.mu.Unlock()

	hasError := false
	hasSuccess := false
	for _, r := range job.Results {
		if r.Status == "success" {
			hasSuccess = true
		}
		if r.Status == "error" || r.Status == "timeout" {
			hasError = true
		}
	}

	job.mu.Lock()
	if hasError && hasSuccess {
		job.Status = "partial"
	} else if hasError {
		job.Status = "failed"
	} else {
		job.Status = "completed"
	}
	job.mu.Unlock()

	s.bus.Publish("multiexec.completed", job)
	s.log.Info("Multi-exec job %s completed: %s", job.ID, job.Status)
}

func (s *MultiExecService) executeOnHost(
	job *MultiExecJob,
	index int,
	hostID string,
	command string,
	timeout time.Duration,
) {
	// Update status
	job.mu.Lock()
	job.Results[index].Status = "running"
	job.mu.Unlock()

	s.bus.Publish("multiexec.progress", map[string]interface{}{
		"job_id": job.ID,
		"index":  index,
		"status": "running",
	})

	start := time.Now()

	host, err := s.hosts.GetByID(context.Background(), hostID)
	if err != nil {
		job.mu.Lock()
		job.Results[index].Status = "error"
		job.Results[index].Error = err.Error()
		job.mu.Unlock()
		return
	}

	var password []byte
	var privateKey []byte
	var passphrase []byte

	// Get credentials
	if host.CredentialID != "" && s.vault != nil && s.vault.IsUnlocked() {
		switch host.AuthMethod {
		case "password":
			password, _ = s.vault.GetPassword(host.CredentialID)
		case "key":
			var passphraseStr string
			privateKey, passphraseStr, _ = s.vault.GetPrivateKey(host.CredentialID)
			passphrase = []byte(passphraseStr)
		}
	} else if host.HasPassword {
		// HasPassword flag is set but no linked credential — credential lookup failed.
		// Fail safe: do NOT fall back to a plaintext field (host.Password is always empty in the DTO).
		job.mu.Lock()
		job.Results[index].Status = "error"
		job.Results[index].Error = "credential required but vault is locked or credential not found"
		job.Results[index].Duration = time.Since(start).String()
		job.mu.Unlock()
		return
	}

	cfg := s.prepareSSHConfig(host, password, privateKey, passphrase, timeout)

	// Shred plaintext credential copies from RAM immediately after handoff
	if password != nil {
		vault.ZeroSlice(password)
	}
	if privateKey != nil {
		vault.ZeroSlice(privateKey)
	}
	if passphrase != nil {
		vault.ZeroSlice(passphrase)
	}

	// Create ephemeral client
	client := sshpkg.NewClient(cfg)

	if err := client.Connect(); err != nil {
		job.mu.Lock()
		job.Results[index].Status = "error"
		job.Results[index].Error = fmt.Sprintf("connect: %v", err)
		job.Results[index].Duration = time.Since(start).String()
		job.mu.Unlock()
		return
	}
	defer client.Close()

	// Execute command
	output, err := client.ExecuteCommand(command)
	duration := time.Since(start)

	job.mu.Lock()
	job.Results[index].Output = string(output)
	job.Results[index].Duration = duration.String()

	if err != nil {
		job.Results[index].Status = "error"
		job.Results[index].Error = err.Error()
		job.Results[index].ExitCode = 1
	} else {
		job.Results[index].Status = "success"
		job.Results[index].ExitCode = 0
	}
	job.mu.Unlock()

	s.bus.Publish("multiexec.progress", map[string]interface{}{
		"job_id": job.ID,
		"index":  index,
		"status": job.Results[index].Status,
		"output": job.Results[index].Output,
	})
}

// maxJobHistory is the maximum number of completed jobs to retain in memory.
const maxJobHistory = 100

// pruneJobs removes the oldest completed jobs when the cap is exceeded.
// Must be called with s.mu held for writing.
func (s *MultiExecService) pruneJobs() {
	if len(s.jobs) <= maxJobHistory {
		return
	}
	// Collect all jobs sorted by start time ascending; drop the oldest.
	type entry struct {
		id  string
		ts  string
	}
	entries := make([]entry, 0, len(s.jobs))
	for id, j := range s.jobs {
		entries = append(entries, entry{id, j.StartedAt})
	}
	sort.Slice(entries, func(i, k int) bool { return entries[i].ts < entries[k].ts })
	for i := 0; i < len(entries)-maxJobHistory; i++ {
		delete(s.jobs, entries[i].id)
	}
}

// GetJob returns a job by ID
func (s *MultiExecService) GetJob(jobID string) (*MultiExecJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job %s not found", jobID)
	}
	return job, nil
}

// GetRecentJobs returns recent execution jobs
func (s *MultiExecService) GetRecentJobs(limit int) []*MultiExecJob {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]*MultiExecJob, 0)
	for _, job := range s.jobs {
		jobs = append(jobs, job)
	}

	// Sort by started_at desc
	for i := 0; i < len(jobs)-1; i++ {
		for j := i + 1; j < len(jobs); j++ {
			ts_i, _ := time.Parse(time.RFC3339, jobs[i].StartedAt)
			ts_j, _ := time.Parse(time.RFC3339, jobs[j].StartedAt)
			if ts_j.After(ts_i) {
				jobs[i], jobs[j] = jobs[j], jobs[i]
			}
		}
	}

	if len(jobs) > limit {
		jobs = jobs[:limit]
	}

	return jobs
}
