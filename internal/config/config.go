package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App         AppConfig         `yaml:"app"`
	Audio       AudioConfig       `yaml:"audio"`
	WakeWord    WakeWordConfig    `yaml:"wake_word"`
	STT         STTConfig         `yaml:"stt"`
	TTS         TTSConfig         `yaml:"tts"`
	Context     ContextConfig     `yaml:"context"`
	Retrieval   RetrievalConfig   `yaml:"retrieval"`
	LLM         LLMConfig         `yaml:"llm"`
	Prompts     PromptsConfig     `yaml:"prompts"`
	Performance PerformanceConfig `yaml:"performance"`
	GRPC        GRPCConfig        `yaml:"grpc"`
	Languages   LanguagesConfig   `yaml:"languages"`
}

type AppConfig struct {
	Name     string `yaml:"name"`
	Version  string `yaml:"version"`
	LogLevel string `yaml:"log_level"`
}

type AudioConfig struct {
	SampleRate int    `yaml:"sample_rate"`
	Channels   int    `yaml:"channels"`
	BitDepth   int    `yaml:"bit_depth"`
	BufferSize int    `yaml:"buffer_size"`
	DeviceName string `yaml:"device_name"`
}

type WakeWordConfig struct {
	Enabled       bool   `yaml:"enabled"`
	Word          string `yaml:"word"`
	CaseSensitive bool   `yaml:"case_sensitive"`
	MatchType     string `yaml:"match_type"`
}

type STTConfig struct {
	Provider           string `yaml:"provider"`
	ModelPath          string `yaml:"model_path"`
	Language           string `yaml:"language"`
	Threads            int    `yaml:"threads"`
	Translate          bool   `yaml:"translate"`
	MaxDurationSeconds int    `yaml:"max_duration_seconds"`
}

type TTSConfig struct {
	Provider     string  `yaml:"provider"`
	Language     string  `yaml:"language"`
	Speed        float64 `yaml:"speed"`
	CacheEnabled bool    `yaml:"cache_enabled"`
	CacheDir     string  `yaml:"cache_dir"`
}

type ContextConfig struct {
	Folder               string           `yaml:"folder"`
	WatchIntervalSeconds int              `yaml:"watch_interval_seconds"`
	SupportedExtensions  []string         `yaml:"supported_extensions"`
	Chunking             ChunkingConfig   `yaml:"chunking"`
	Embeddings           EmbeddingsConfig `yaml:"embeddings"`
}

type ChunkingConfig struct {
	Method       string `yaml:"method"`
	ChunkSize    int    `yaml:"chunk_size"`
	ChunkOverlap int    `yaml:"chunk_overlap"`
	MinChunkSize int    `yaml:"min_chunk_size"`
}

type EmbeddingsConfig struct {
	Provider  string `yaml:"provider"`
	Model     string `yaml:"model"`
	Dimension int    `yaml:"dimension"`
	CacheDir  string `yaml:"cache_dir"`
	BatchSize int    `yaml:"batch_size"`
}

type RetrievalConfig struct {
	TopK                int     `yaml:"top_k"`
	SimilarityThreshold float64 `yaml:"similarity_threshold"`
	Rerank              bool    `yaml:"rerank"`
	StorageBackend      string  `yaml:"storage_backend"`
	DBPath              string  `yaml:"db_path"`
}

type LLMConfig struct {
	Provider       string  `yaml:"provider"`
	Model          string  `yaml:"model"`
	APIKey         string  `yaml:"api_key"`
	BaseURL        string  `yaml:"base_url"`
	MaxTokens      int     `yaml:"max_tokens"`
	Temperature    float64 `yaml:"temperature"`
	TimeoutSeconds int     `yaml:"timeout_seconds"`
	Stream         bool    `yaml:"stream"`
}

type PromptsConfig struct {
	System          string `yaml:"system"`
	ContextTemplate string `yaml:"context_template"`
}

type PerformanceConfig struct {
	MaxConcurrentRequests int  `yaml:"max_concurrent_requests"`
	WorkerPoolSize        int  `yaml:"worker_pool_size"`
	RequestTimeoutSeconds int  `yaml:"request_timeout_seconds"`
	EnableMetrics         bool `yaml:"enable_metrics"`
}

type GRPCConfig struct {
	AudioPort             int `yaml:"audio_port"`
	STTPort               int `yaml:"stt_port"`
	TTSPort               int `yaml:"tts_port"`
	RetrieverPort         int `yaml:"retriever_port"`
	LLMPort               int `yaml:"llm_port"`
	OrchestratorPort      int `yaml:"orchestrator_port"`
	MaxReceiveMessageSize int `yaml:"max_receive_message_size"`
	MaxSendMessageSize    int `yaml:"max_send_message_size"`
}

type LanguagesConfig struct {
	Supported  []Language `yaml:"supported"`
	AutoDetect bool       `yaml:"auto_detect"`
	Fallback   string     `yaml:"fallback"`
}

type Language struct {
	Code string `yaml:"code"`
	Name string `yaml:"name"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.WakeWord.Word == "" {
		return fmt.Errorf("wake_word.word cannot be empty")
	}

	if c.Context.Folder == "" {
		return fmt.Errorf("context.folder cannot be empty")
	}

	if c.LLM.APIKey == "" && !strings.Contains(c.LLM.Provider, "local") {
		return fmt.Errorf("llm.api_key must be set for provider: %s", c.LLM.Provider)
	}

	if c.Retrieval.TopK <= 0 {
		return fmt.Errorf("retrieval.top_k must be positive")
	}

	return nil
}

// Singleton instance
var globalConfig *Config

// Get returns the global configuration instance
func Get() *Config {
	return globalConfig
}

// Set sets the global configuration instance
func Set(cfg *Config) {
	globalConfig = cfg
}
