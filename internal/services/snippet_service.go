package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/google/uuid"
	"github.com/kingknull/oblivrashell/internal/database"
	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
)

type SnippetService struct {
	BaseService
	ctx     context.Context
	repo    database.SnippetStore
	sshSvc  SessionExecutor
	teamSvc *TeamService
	bus     *eventbus.Bus
	log     *logger.Logger
}

// WHY: SnippetService provides a reusable command engine with dynamic variable substitution.
// This allows users to store complex procedures and adapt them to specific host contexts at runtime.

func (s *SnippetService) Name() string { return "snippet-service" }

// Dependencies returns service dependencies.
func (s *SnippetService) Dependencies() []string {
	return []string{}
}

func NewSnippetService(repo database.SnippetStore, sshSvc SessionExecutor, teamSvc *TeamService, bus *eventbus.Bus, log *logger.Logger) *SnippetService {
	return &SnippetService{
		repo:    repo,
		sshSvc:  sshSvc,
		teamSvc: teamSvc,
		bus:     bus,
		log:     log.WithPrefix("snippets"),
	}
}

func (s *SnippetService) Start(ctx context.Context) error {
	s.ctx = ctx
	return nil
}

func (s *SnippetService) Stop(ctx context.Context) error {
	return nil
}

func (s *SnippetService) List(ctx context.Context) ([]database.Snippet, error) {
	return s.repo.List(ctx)
}

func (s *SnippetService) Get(ctx context.Context, id string) (database.Snippet, error) {
	return s.repo.Get(ctx, id)
}

func (s *SnippetService) Create(ctx context.Context, title, command, description string, tags, variables []string) (database.Snippet, error) {
	if title == "" || command == "" {
		return database.Snippet{}, fmt.Errorf("title and command are required")
	}

	snippet := &database.Snippet{
		ID:          uuid.New().String(),
		Title:       title,
		Command:     command,
		Description: description,
		Tags:        tags,
		Variables:   variables,
	}

	if err := s.repo.Create(ctx, snippet); err != nil {
		s.log.Error("Failed to create snippet: %v", err)
		return database.Snippet{}, err
	}
	s.bus.Publish(eventbus.EventSettingsChanged, "snippets")
	return *snippet, nil
}

func (s *SnippetService) Update(ctx context.Context, id, title, command, description string, tags, variables []string) (database.Snippet, error) {
	snippet, err := s.repo.Get(ctx, id)
	if err != nil {
		return database.Snippet{}, err
	}

	snippet.Title = title
	snippet.Command = command
	snippet.Description = description
	snippet.Tags = tags
	snippet.Variables = variables

	if err := s.repo.Update(ctx, &snippet); err != nil {
		s.log.Error("Failed to update snippet %s: %v", id, err)
		return database.Snippet{}, err
	}
	s.bus.Publish(eventbus.EventSettingsChanged, "snippets")
	return snippet, nil
}

func (s *SnippetService) Delete(ctx context.Context, id string) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		s.log.Error("Failed to delete snippet %s: %v", id, err)
		return err
	}
	s.bus.Publish(eventbus.EventSettingsChanged, "snippets")
	return nil
}

// ShareToTeam saves a copy of the snippet to the team vault
func (s *SnippetService) ShareToTeam(ctx context.Context, snippetID string) error {
	snippet, err := s.repo.Get(ctx, snippetID)
	if err != nil {
		return err
	}

	if s.teamSvc == nil {
		return fmt.Errorf("team service not available")
	}

	// Save snippet as secret in team vault
	data, _ := json.Marshal(snippet)
	_, err = s.teamSvc.AddSecret(snippet.Title, "snippet", string(data))
	if err != nil {
		return fmt.Errorf("save to team vault: %w", err)
	}

	s.log.Info("Shared snippet %s to team vault", snippet.Title)
	return nil
}

// ExtractVariables finds all {{var}} instances in a command string.
// This is used by the frontend to dynamically generate input fields for the user.
func (s *SnippetService) ExtractVariables(command string) []string {
	uniqueVars := make(map[string]bool)
	var result []string

	addVar := func(v string) {
		if v == "" || v == "if" || v == "else" || v == "end" || v == "range" || v == "with" || v == "eq" || v == "ne" {
			return
		}
		if !uniqueVars[v] {
			uniqueVars[v] = true
			result = append(result, v)
		}
	}

	// 1. Legacy simple variables: {{VarName}}
	legacyPattern := regexp.MustCompile(`\{\{([A-Za-z0-9_]+)\}\}`)
	for _, match := range legacyPattern.FindAllStringSubmatch(command, -1) {
		if len(match) > 1 {
			addVar(match[1])
		}
	}

	// 2. Go Template variables: .VarName inside {{ ... }}
	blockPattern := regexp.MustCompile(`\{\{(.*?)\}\}`)
	dotVarPattern := regexp.MustCompile(`\.([A-Za-z0-9_]+)`)
	for _, block := range blockPattern.FindAllStringSubmatch(command, -1) {
		if len(block) > 1 {
			content := block[1]
			for _, dotMatch := range dotVarPattern.FindAllStringSubmatch(content, -1) {
				if len(dotMatch) > 1 {
					addVar(dotMatch[1])
				}
			}
		}
	}

	return result
}

// ExecuteSnippet runs a snippet on an active SSH session after replacing variables.
// It optionally applies sudo prefixing for known privileged commands.
func (s *SnippetService) ExecuteSnippet(ctx context.Context, snippetID string, sessionID string, variables map[string]string, autoSudo bool) error {
	snippet, err := s.repo.Get(ctx, snippetID)
	if err != nil {
		return fmt.Errorf("load snippet: %w", err)
	}

	commandString := s.resolveVariables(snippet.Command, variables)

	if autoSudo && s.shouldApplySudo(commandString) {
		commandString = "sudo " + commandString
	}

	// Ensure the command is terminated with a newline to trigger execution in the remote shell.
	if !strings.HasSuffix(commandString, "\n") {
		commandString += "\n"
	}

	if err := s.sshSvc.SendInput(sessionID, commandString); err != nil {
		return fmt.Errorf("execute on session %s: %w", sessionID, err)
	}

	go s.repo.IncrementUseCount(ctx, snippetID)
	return nil
}

func (s *SnippetService) resolveVariables(command string, variables map[string]string) string {
	res := command

	// First pass: Legacy direct replacement for {{VarName}}
	for k, v := range variables {
		res = strings.ReplaceAll(res, fmt.Sprintf("{{%s}}", k), v)
	}

	// Second pass: Process as Go text/template for conditional logic (If-Then-Else)
	tmpl, err := template.New("playbook").Parse(res)
	if err == nil {
		var buf bytes.Buffer
		// Execute template with variables map as root context
		if err := tmpl.Execute(&buf, variables); err == nil {
			return buf.String()
		} else {
			s.log.Error("Template execution failed: %v", err)
		}
	} else {
		// Log parse error but return what we have (best effort)
		s.log.Error("Template parsing failed: %v", err)
	}

	return res
}

func (s *SnippetService) shouldApplySudo(command string) bool {
	if strings.HasPrefix(command, "sudo ") {
		return false
	}
	privilegedPrefixes := []string{"apt", "systemctl", "docker", "yum", "dnf", "ufw"}
	for _, p := range privilegedPrefixes {
		if strings.HasPrefix(command, p) {
			return true
		}
	}
	return strings.Contains(command, "mkdir /") || strings.Contains(command, "rm -rf /")
}

// ExportJSON returns all snippets as a JSON byte array
func (s *SnippetService) ExportJSON(ctx context.Context) ([]byte, error) {
	snippets, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(snippets, "", "  ")
}

// ImportJSON adds snippets from a JSON byte array
func (s *SnippetService) ImportJSON(ctx context.Context, data []byte) error {
	var snippets []database.Snippet
	if err := json.Unmarshal(data, &snippets); err != nil {
		return err
	}

	for _, snippet := range snippets {
		snippet.ID = uuid.New().String() // Assign new IDs to avoid collisions during import
		if err := s.repo.Create(ctx, &snippet); err != nil {
			s.log.Error("Failed to import snippet %s: %v", snippet.Title, err)
		}
	}
	s.bus.Publish(eventbus.EventSettingsChanged, "snippets")
	return nil
}
