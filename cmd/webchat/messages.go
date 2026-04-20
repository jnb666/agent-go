package main

import (
	"time"

	"github.com/jnb666/agent-go/agents"
	"github.com/jnb666/agent-go/llm"
)

func DefaultConfig() agents.Config {
	return agents.Config{
		Models:       map[string]llm.GenerationConfig{},
		SystemPrompt: `You are a helpful assistant. The current date is {{ .Time.Format "Monday 2 January 2006" }}.`,
		Tools: []agents.ToolConfig{
			{Name: "web_search", Enabled: true},
			{Name: "web_fetch", Enabled: true},
			{Name: "web_find", Enabled: true},
		},
	}
}

type PromptArgs struct{}

func (PromptArgs) Time() time.Time {
	return time.Now()
}

// Message sent from web UI to server
type Request struct {
	Type    string         `json:"type"`             // chat | list | load | delete | config | ping
	Message llm.Message    `json:"message,omitzero"` // if action = chat
	ID      string         `json:"id,omitzero"`      // if action = load, delete
	Config  *agents.Config `json:"config,omitzero"`  // if action = config
	Error   error          `json:"-"`
}

// Message sent back from server to web UI
type Response struct {
	Type         string           `json:"type"`                  // chat | list | load | config | stats | pong
	Message      *Message         `json:"message,omitzero"`      // if action = add -> multiple updates are streamed
	Conversation []agents.Message `json:"conversation,omitzero"` // if action = load
	List         []Item           `json:"list,omitzero"`         // if action = list
	CurrentID    string           `json:"current_id,omitzero"`   // if action = load or list
	Config       *agents.Config   `json:"config,omitzero"`       // if action = config
	Stats        *Stats           `json:"stats,omitzero"`        // if action = stats
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

type Stats struct {
	ContextUsed     string `json:"context_used"`
	PromptTime      string `json:"prompt_time"`
	TokensGenerated int    `json:"tokens_generated"`
	GenerationSpeed string `json:"generation_speed"`
	ToolCalls       int    `json:"tool_calls"`
	TotalTime       string `json:"total_time"`
}
