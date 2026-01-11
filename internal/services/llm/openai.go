package llm

import (
	"context"
	"fmt"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/shashwatssp/deeprecall/internal/config"
	"github.com/shashwatssp/deeprecall/internal/models"
)

type OpenAIClient struct {
	client *openai.Client
	cfg    *config.Config
}

func NewOpenAIClient(cfg *config.Config) *OpenAIClient {
	client := openai.NewClient(cfg.LLM.APIKey)
	return &OpenAIClient{
		client: client,
		cfg:    cfg,
	}
}

// Generate generates a response from the LLM
func (c *OpenAIClient) Generate(req *models.LLMRequest) (*models.LLMResponse, error) {
	startTime := time.Now()

	// Convert messages
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}

	// Create chat completion
	resp, err := c.client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:       c.cfg.LLM.Model,
			Messages:    messages,
			MaxTokens:   req.MaxTokens,
			Temperature: float32(req.Temperature),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response generated")
	}

	return &models.LLMResponse{
		Content:      resp.Choices[0].Message.Content,
		TokensUsed:   resp.Usage.TotalTokens,
		FinishReason: string(resp.Choices[0].FinishReason),
		ResponseTime: time.Since(startTime),
	}, nil
}
