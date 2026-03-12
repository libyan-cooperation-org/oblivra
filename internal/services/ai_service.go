package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kingknull/oblivrashell/internal/eventbus"
	"github.com/kingknull/oblivrashell/internal/logger"
	"github.com/kingknull/oblivrashell/internal/vault"
)

type AIModel string

const (
	ModelOllamaLlama3 AIModel = "llama3"
	ModelGPT4o        AIModel = "gpt-4o"
)

type AIService struct {
	BaseService
	ctx    context.Context
	bus    *eventbus.Bus
	log    *logger.Logger
	client *http.Client
	history []Message
}

type Message struct {
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

func (s *AIService) Name() string { return "ai-service" }

// Dependencies returns service dependencies
func (s *AIService) Dependencies() []string {
	return []string{"eventbus"}
}

func NewAIService(v vault.Provider, bus *eventbus.Bus, log *logger.Logger) *AIService {
	return &AIService{
		bus: bus,
		log: log.WithPrefix("ai-svc"),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *AIService) Start(ctx context.Context) error {
	s.ctx = ctx
	s.history = []Message{
		{Role: "system", Content: "You are the Sovereign Terminal AI assistant. Help the user with command generation, security analysis, and automation.", Timestamp: time.Now().Format(time.RFC3339)},
	}
	s.log.Info("AI service started")
	return nil
}

func (s *AIService) Stop(ctx context.Context) error {
	return nil
}

func (s *AIService) GetChatHistory() ([]Message, error) {
	return s.history, nil
}

func (s *AIService) SendMessage(content string) (string, error) {
	s.history = append(s.history, Message{Role: "user", Content: content, Timestamp: time.Now().Format(time.RFC3339)})
	
	resp, err := s.GenerateCommand(content) // Using GenerateCommand as a proxy for simple Q&A for now
	if err != nil {
		return "", err
	}
	
	s.history = append(s.history, Message{Role: "assistant", Content: resp.Text, Timestamp: time.Now().Format(time.RFC3339)})
	return resp.Text, nil
}

// ExplainError asks the AI to explain a terminal error
func (s *AIService) ExplainError(errorOutput string) (*AIResponse, error) {
	s.log.Info("Explaining error: %s...", errorOutput[:min(len(errorOutput), 20)])

	prompt := fmt.Sprintf("Explain the following terminal error and suggest a fix:\n\n%s", errorOutput)
	return s.callOllama(prompt)
}

// GenerateCommand asks the AI to generate a shell command from natural language
func (s *AIService) GenerateCommand(description string) (*AIResponse, error) {
	s.log.Info("Generating command for: %s", description)

	prompt := fmt.Sprintf("Generate a single shell command for: %s. Return ONLY the command.", description)
	return s.callOllama(prompt)
}

func (s *AIService) callOllama(prompt string) (*AIResponse, error) {
	url := "http://localhost:11434/api/generate"

	payload := map[string]interface{}{
		"model":  string(ModelOllamaLlama3),
		"prompt": prompt,
		"stream": false,
	}

	jsonPayload, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(s.ctx, "POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w (ensure ollama is running)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned error status: %d", resp.StatusCode)
	}

	var ollamaResp struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, err
	}

	return &AIResponse{Text: ollamaResp.Response}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
