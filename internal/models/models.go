package models

import "time"

// AudioChunk represents a chunk of audio data
type AudioChunk struct {
	Data       []byte
	Timestamp  time.Time
	Duration   time.Duration
	SampleRate int
}

// TranscriptionResult represents STT output
type TranscriptionResult struct {
	Text       string
	Language   string
	Confidence float64
	Duration   time.Duration
}

// Document represents a parsed document
type Document struct {
	ID          string
	FilePath    string
	Content     string
	Metadata    map[string]string
	Hash        string
	ParsedAt    time.Time
	FileModTime time.Time
}

// Chunk represents a text chunk
type Chunk struct {
	ID         string
	DocumentID string
	Content    string
	Embedding  []float32
	Metadata   map[string]interface{}
	Index      int
}

// RetrievalResult represents a retrieved chunk with score
type RetrievalResult struct {
	Chunk      *Chunk
	Score      float64
	DocumentID string
}

// LLMRequest represents a request to the LLM
type LLMRequest struct {
	Messages    []Message
	MaxTokens   int
	Temperature float64
	Stream      bool
}

// Message represents a chat message
type Message struct {
	Role    string // system, user, assistant
	Content string
}

// LLMResponse represents LLM output
type LLMResponse struct {
	Content      string
	TokensUsed   int
	FinishReason string
	ResponseTime time.Duration
}

// VoiceRequest represents a complete voice interaction
type VoiceRequest struct {
	ID           string
	AudioData    []byte
	Timestamp    time.Time
	WakeWord     string
	ProcessingID string
}

// VoiceResponse represents the complete response
type VoiceResponse struct {
	RequestID      string
	Text           string
	AudioData      []byte
	ProcessingTime time.Duration
	Error          error
}
