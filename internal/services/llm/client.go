package llm

import (
	"fmt"

	"github.com/shashwatssp/deeprecall/internal/config"
	"github.com/shashwatssp/deeprecall/internal/models"
)

// Client is the interface for LLM providers
type Client interface {
	Generate(req *models.LLMRequest) (*models.LLMResponse, error)
}

// NewClient creates an LLM client based on configuration
func NewClient(cfg *config.Config) (Client, error) {
	switch cfg.LLM.Provider {
	case "openai":
		return NewOpenAIClient(cfg), nil
	case "anthropic":
		return nil, fmt.Errorf("anthropic not implemented yet")
	case "local":
		return nil, fmt.Errorf("local LLM not implemented yet")
	default:
		// Default to OpenAI-compatible
		return NewOpenAIClient(cfg), nil
	}
}
