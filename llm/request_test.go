package llm

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jnb666/agent-go/util"
	"github.com/openai/openai-go/v3/shared"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testMessages = []Message{
	{Role: "system", Content: "You are a helpful assistant."},
	{Role: "user", Content: "hello"},
	{Role: "assistant",
		Content:   "Hello! How can I help you today?",
		Reasoning: "The user is just saying \"hi\". This is a simple greeting, so I should respond in a friendly and helpful manner",
	},
	{Role: "user", Content: "How many r's are there in Strawberry?"},
}

var toolMessages = []Message{
	{Role: "system", Content: "You are a helpful assistant."},
	{Role: "user", Content: "What's the weather like in London today?"},
	{Role: "assistant",
		Reasoning: "The user is asking about the current weather in London. I need to use the get_current_weather function with the location parameter set to \"London,GB\" (using the ISO 3166 country code for Great Britain).",
		ToolCalls: []ToolCall{{ID: "call_f5fc4884ea3348a9b38d3bf6", Name: "get_current_weather", Arguments: `{"location":"London,GB"}`}},
	},
	{Role: "tool",
		Content:    "Current weather for London,GB: 9°C - mist, feels like 7°C, wind 3.6m/s",
		ToolCallID: "call_f5fc4884ea3348a9b38d3bf6",
	},
}

var toolDef = FunctionDefinition{
	Name: "get_current_weather",
	Description: "Get the current weather in a given location." +
		" Returns conditions with temperatures in Celsius and wind speed in meters/second.",
	Parameters: shared.FunctionParameters{
		"type": "object",
		"properties": map[string]any{
			"location": map[string]any{
				"type":        "string",
				"description": `The city name and ISO 3166 country code, e.g. "London,GB" or "New York,US".`,
			},
		},
		"required": []string{"location"},
	},
}

func TestRequestSimple(t *testing.T) {
	m, err := NewModel(context.Background(), "")
	require.NoError(t, err)
	m.SetOptions(WithTemperature(1.0), WithTopP(0.95), WithTopK(20), WithPresencePenalty(1.5), WithRepetitionPenalty(1.0), WithReasoningEffort("none"))

	req, err := m.newRequest(m.config, testMessages)
	require.NoError(t, err)
	t.Log(util.Pretty(req))

	expect := map[string]any{
		"messages": []map[string]any{
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": "hello"},
			{"role": "assistant", "content": "Hello! How can I help you today?"},
			{"role": "user", "content": "How many r's are there in Strawberry?"},
		},
		"model":              m.id,
		"temperature":        1.0,
		"top_p":              0.95,
		"top_k":              20,
		"presence_penalty":   1.5,
		"repetition_penalty": 1.0,
		"chat_template_kwargs": map[string]any{
			"enable_thinking": false,
		},
	}
	assert.JSONEq(t, toJSON(expect), toJSON(req))
}

func TestRequestWithTools(t *testing.T) {
	m, err := NewModel(context.Background(), "")
	require.NoError(t, err)
	m.SetOptions(WithTemperature(1.0), WithTopP(0.95), WithTopK(20), WithPresencePenalty(1.5), WithRepetitionPenalty(1.0), WithReasoningEffort("medium"), WithTools(toolDef))

	req, err := m.newRequest(m.config, toolMessages)
	require.NoError(t, err)
	t.Log(util.Pretty(req))

	expect := map[string]any{
		"messages": []map[string]any{
			{"role": "system", "content": "You are a helpful assistant."},
			{"role": "user", "content": "What's the weather like in London today?"},
			{"role": "assistant",
				"content":           "",
				"reasoning_content": "The user is asking about the current weather in London. I need to use the get_current_weather function with the location parameter set to \"London,GB\" (using the ISO 3166 country code for Great Britain).",
				"tool_calls": []map[string]any{{
					"id":       "call_f5fc4884ea3348a9b38d3bf6",
					"function": map[string]any{"arguments": "{\"location\":\"London,GB\"}", "name": "get_current_weather"},
					"type":     "function",
				}},
			},
			{"role": "tool", "content": "Current weather for London,GB: 9°C - mist, feels like 7°C, wind 3.6m/s", "tool_call_id": "call_f5fc4884ea3348a9b38d3bf6"},
		},
		"model":                m.id,
		"temperature":          1.0,
		"top_p":                0.95,
		"top_k":                20,
		"presence_penalty":     1.5,
		"repetition_penalty":   1.0,
		"reasoning_effort":     "medium",
		"chat_template_kwargs": map[string]any{"reasoning_effort": "medium"},
		"parallel_tool_calls":  true,
		"tools": []map[string]any{{
			"function": map[string]any{
				"name":        "get_current_weather",
				"description": "Get the current weather in a given location. Returns conditions with temperatures in Celsius and wind speed in meters/second.",
				"parameters": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"location": map[string]any{
							"type":        "string",
							"description": `The city name and ISO 3166 country code, e.g. "London,GB" or "New York,US".`,
						},
					},
					"required": []string{"location"},
				},
			},
			"type": "function",
		}},
	}
	assert.JSONEq(t, toJSON(expect), toJSON(req))
}

func toJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(data)
}
