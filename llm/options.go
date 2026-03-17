package llm

import (
	"slices"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/shared"
)

// Model configuration options.
type Config struct {
	GenerationConfig
	Tools []FunctionDefinition `json:"tools,omitzero"`
}

type GenerationConfig struct {
	Seed                     param.Opt[int64]       `json:"seed,omitzero"`
	Temperature              param.Opt[float64]     `json:"temperature,omitzero"`
	TopP                     param.Opt[float64]     `json:"top_p,omitzero"`
	TopK                     param.Opt[int64]       `json:"top_k,omitzero"`
	PresencePenalty          param.Opt[float64]     `json:"presence_penalty,omitzero"`
	RepetitionPenalty        param.Opt[float64]     `json:"repetition_penalty,omitzero"`
	ReasoningEffort          shared.ReasoningEffort `json:"reasoning_effort,omitzero"`
	DisableParallelToolCalls bool                   `json:"disable_parallel_tool_calls,omitzero"`
}

func (c Config) Clone() Config {
	c.Tools = slices.Clone(c.Tools)
	return c
}

// Tool function definition type. Parameters are defined in jsonschema format as per openapi API.
type FunctionDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// Option to set the parameters used in text generation.
type Option func(c *Config)

func WithSeed(seed int64) Option {
	return func(c *Config) {
		c.Seed = openai.Int(seed)
	}
}

func WithTemperature(temp float64) Option {
	return func(c *Config) {
		c.Temperature = openai.Float(temp)
	}
}

func WithTopP(topP float64) Option {
	return func(c *Config) {
		c.TopP = openai.Float(topP)
	}
}

func WithTopK(topK int) Option {
	return func(c *Config) {
		c.TopK = openai.Int(int64(topK))
	}
}

func WithPresencePenalty(penalty float64) Option {
	return func(c *Config) {
		c.PresencePenalty = openai.Float(penalty)
	}
}

func WithRepetitionPenalty(penalty float64) Option {
	return func(c *Config) {
		c.RepetitionPenalty = openai.Float(penalty)
	}
}

// Standard options are low, medium or high or set to none to disable thinking.
func WithReasoningEffort(effort shared.ReasoningEffort) Option {
	return func(c *Config) {
		c.ReasoningEffort = shared.ReasoningEffort(effort)
	}
}

// List of tool functions which the model can select from.
func WithTools(definitions ...FunctionDefinition) Option {
	return func(c *Config) {
		c.Tools = definitions
	}
}

func DisableParallelToolCalls() Option {
	return func(c *Config) {
		c.DisableParallelToolCalls = true
	}
}
