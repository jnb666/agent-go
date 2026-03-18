package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/jnb666/agent-go/agents"
	"github.com/jnb666/agent-go/llm"
	log "github.com/sirupsen/logrus"
)

var ConfigDir = getConfigDir()

func DefaultConfig() Config {
	return Config{
		Models:       map[string]llm.GenerationConfig{},
		SystemPrompt: `You are a helpful assistant. The current date is {{ .Time.Format "Monday 2 January 2006" }}.`,
		Tools:        []ToolConfig{{"web_search", true}, {"browser_open", true}, {"browser_find", true}},
	}
}

type PromptArgs struct{}

func (PromptArgs) Time() time.Time {
	return time.Now()
}

// Message sent from web UI to server
type Request struct {
	Type    string      `json:"type"`             // chat | list | load | delete | config | ping
	Message llm.Message `json:"message,omitzero"` // if action = chat
	ID      string      `json:"id,omitzero"`      // if action = load, delete
	Config  *Config     `json:"config,omitzero"`  // if action = config
	Error   error       `json:"-"`
}

// Message sent back from server to web UI
type Response struct {
	Type         string         `json:"type"`                  // chat | list | load | config | stats | pong
	Message      *Message       `json:"message,omitzero"`      // if action = add -> multiple updates are streamed
	Conversation *agents.Memory `json:"conversation,omitzero"` // if action = load
	List         []Item         `json:"list,omitzero"`         // if action = list
	CurrentID    string         `json:"current_id,omitzero"`   // if action = list
	Config       *Config        `json:"config,omitzero"`       // if action = config
	Stats        *Stats         `json:"stats,omitzero"`        // if action = stats
}

type Message struct {
	llm.Message
	Update bool `json:"update"`
	End    bool `json:"end"`
}

type Item struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

type Config struct {
	Model        string                          `json:"model"`
	Models       map[string]llm.GenerationConfig `json:"models"`
	SystemPrompt string                          `json:"system_prompt"`
	Tools        []ToolConfig                    `json:"tools"`
}

type ToolConfig struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type Stats struct {
	ContextSize           int     `json:"context_size"`
	TokensGenerated       int     `json:"tokens_generated"`
	PromptSpeed           float64 `json:"prompt_speed"`
	GenerationSpeed       float64 `json:"generation_speed"`
	GenerationElapsedMsec float64 `json:"generation_time"`
	ToolCalls             int     `json:"tool_calls"`
	ToolCallElapsedMsec   float64 `json:"tool_time"`
}

func loadJSON(file string, v any) error {
	filename := filepath.Join(ConfigDir, file)
	log.Debugf("Load JSON from %s", filename)
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func saveJSON(file string, v any) error {
	filename := filepath.Join(ConfigDir, file)
	log.Debugf("Save JSON to %s", filename)
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func getConfigDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	dir := filepath.Join(base, "agent-go")
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		panic(err)
	}
	return dir
}
