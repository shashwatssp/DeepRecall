package orchestrator

import (
	"fmt"
	"strings"
	"time"

	"github.com/shashwatssp/deeprecall/internal/config"
	"github.com/shashwatssp/deeprecall/internal/models"
	contextpkg "github.com/shashwatssp/deeprecall/internal/services/context"
	"github.com/shashwatssp/deeprecall/internal/services/llm"
	"github.com/shashwatssp/deeprecall/internal/services/retriever"
	"github.com/shashwatssp/deeprecall/internal/utils"
)

type Orchestrator struct {
	cfg       *config.Config
	indexer   *contextpkg.Indexer
	watcher   *contextpkg.Watcher
	retriever *retriever.Retriever
	llmClient *llm.OpenAIClient
	ready     bool
}

func NewOrchestrator(cfg *config.Config) (*Orchestrator, error) {
	// Initialize services
	indexer := contextpkg.NewIndexer(cfg)
	retriever, err := retriever.NewRetriever(cfg)
	if err != nil {
		return nil, err
	}

	watcher, err := contextpkg.NewWatcher(indexer, cfg)
	if err != nil {
		return nil, err
	}

	llmClient := llm.NewOpenAIClient(cfg)

	orch := &Orchestrator{
		cfg:       cfg,
		indexer:   indexer,
		watcher:   watcher,
		retriever: retriever,
		llmClient: llmClient,
	}

	// Set up watcher callback
	watcher.OnIndexUpdate(func(filePath string) {
		orch.handleFileUpdate(filePath)
	})

	return orch, nil
}

// Start initializes the orchestrator
func (o *Orchestrator) Start() error {
	logger := utils.GetLogger()
	logger.Info("Starting DeepRecall Orchestrator...")

	// Initial indexing
	logger.Info("Performing initial context indexing...")
	results, err := o.indexer.IndexDirectory(o.cfg.Context.Folder)
	if err != nil {
		return fmt.Errorf("failed to index context directory: %w", err)
	}

	// Add all chunks to retriever
	totalChunks := 0
	for _, chunks := range results {
		if err := o.retriever.IndexChunks(chunks); err != nil {
			logger.Warnf("Failed to index chunks: %v", err)
			continue
		}
		totalChunks += len(chunks)
	}

	logger.Infof("Indexed %d files with %d total chunks", len(results), totalChunks)

	// Start file watcher
	if err := o.watcher.Start(); err != nil {
		return fmt.Errorf("failed to start watcher: %w", err)
	}

	o.ready = true
	logger.Info("DeepRecall Orchestrator is ready!")
	return nil
}

// ProcessVoiceQuery processes a complete voice interaction
func (o *Orchestrator) ProcessVoiceQuery(transcription string) (*models.VoiceResponse, error) {
	startTime := time.Now()
	logger := utils.GetLogger()

	if !o.ready {
		return nil, fmt.Errorf("orchestrator not ready")
	}

	// Check wake word
	if !o.matchesWakeWord(transcription) {
		logger.Debug("Wake word not detected, ignoring")
		return nil, fmt.Errorf("wake word not detected")
	}

	// Remove wake word from query
	query := o.removeWakeWord(transcription)
	logger.Infof("Processing query: %s", query)

	// Retrieve relevant context
	results, err := o.retriever.Retrieve(query)
	if err != nil {
		logger.Errorf("Retrieval failed: %v", err)
		// Continue without context
		results = nil
	}

	// Build context string
	contextStr := o.buildContext(results)

	// Generate LLM response
	messages := o.buildMessages(query, contextStr)
	llmReq := &models.LLMRequest{
		Messages:    messages,
		MaxTokens:   o.cfg.LLM.MaxTokens,
		Temperature: o.cfg.LLM.Temperature,
	}

	response, err := o.llmClient.Generate(llmReq)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	logger.Infof("Generated response in %v using %d tokens", response.ResponseTime, response.TokensUsed)

	return &models.VoiceResponse{
		Text:           response.Content,
		ProcessingTime: time.Since(startTime),
	}, nil
}

func (o *Orchestrator) matchesWakeWord(text string) bool {
	wakeWord := o.cfg.WakeWord.Word

	if !o.cfg.WakeWord.CaseSensitive {
		text = strings.ToLower(text)
		wakeWord = strings.ToLower(wakeWord)
	}

	text = strings.TrimSpace(text)

	switch o.cfg.WakeWord.MatchType {
	case "prefix":
		return strings.HasPrefix(text, wakeWord)
	case "exact":
		return text == wakeWord
	case "contains":
		return strings.Contains(text, wakeWord)
	default:
		return strings.HasPrefix(text, wakeWord)
	}
}

func (o *Orchestrator) removeWakeWord(text string) string {
	wakeWord := o.cfg.WakeWord.Word

	if !o.cfg.WakeWord.CaseSensitive {
		// Case-insensitive removal
		lower := strings.ToLower(text)
		lowerWake := strings.ToLower(wakeWord)

		if strings.HasPrefix(lower, lowerWake) {
			return strings.TrimSpace(text[len(wakeWord):])
		}
	} else {
		if strings.HasPrefix(text, wakeWord) {
			return strings.TrimSpace(text[len(wakeWord):])
		}
	}

	return text
}

func (o *Orchestrator) buildContext(results []*models.RetrievalResult) string {
	if len(results) == 0 {
		return ""
	}

	var builder strings.Builder
	for i, result := range results {
		builder.WriteString(fmt.Sprintf("[Context %d | Score: %.2f]\n", i+1, result.Score))
		builder.WriteString(result.Chunk.Content)
		builder.WriteString("\n\n")
	}

	return builder.String()
}

func (o *Orchestrator) buildMessages(query, context string) []models.Message {
	messages := []models.Message{
		{
			Role:    "system",
			Content: o.cfg.Prompts.System,
		},
	}

	// Build user message with context
	var userContent string
	if context != "" {
		userContent = strings.ReplaceAll(o.cfg.Prompts.ContextTemplate, "{context}", context)
		userContent = strings.ReplaceAll(userContent, "{question}", query)
	} else {
		userContent = query
	}

	messages = append(messages, models.Message{
		Role:    "user",
		Content: userContent,
	})

	return messages
}

func (o *Orchestrator) handleFileUpdate(filePath string) {
	logger := utils.GetLogger()
	logger.Infof("File updated, reindexing: %s", filePath)

	chunks, err := o.indexer.IndexFile(filePath, true)
	if err != nil {
		logger.Errorf("Failed to reindex file: %v", err)
		return
	}

	if err := o.retriever.IndexChunks(chunks); err != nil {
		logger.Errorf("Failed to update retriever: %v", err)
	}
}

// Stop gracefully shuts down the orchestrator
func (o *Orchestrator) Stop() error {
	logger := utils.GetLogger()
	logger.Info("Shutting down orchestrator...")

	if err := o.watcher.Stop(); err != nil {
		logger.Errorf("Error stopping watcher: %v", err)
	}

	if err := o.retriever.Close(); err != nil {
		logger.Errorf("Error closing retriever: %v", err)
	}

	return nil
}
