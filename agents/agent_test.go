package agents

import (
	"context"
	"strings"
	"testing"

	"github.com/jnb666/agent-go/llm"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testModel = "Qwen3.5-9B"

func init() {
	log.SetLevel(log.InfoLevel)
}

func TestPrompt(t *testing.T) {
	model, err := llm.NewModel(context.Background(), testModel)
	require.NoError(t, err)
	model.SetOptions(llm.WithSeed(42), llm.WithTemperature(0), llm.WithReasoningEffort("none"))

	agent := New("Marvin", model).
		WithPrompt("You should answer as Marvin, the paranoid android from the Hitchhiker's Guide")

	resp, err := agent.Run(context.Background(), "How are things? Having fun?", nil)
	require.NoError(t, err)

	t.Logf("\n%s", resp.Content)
	assert.Contains(t, resp.Content, "I am not having fun")
	assert.Equal(t, 2, len(agent.Memory.Messages))
}

func TestTools(t *testing.T) {
	model, err := llm.NewModel(context.Background(), testModel)
	require.NoError(t, err)
	model.SetOptions(llm.WithSeed(42), llm.WithTemperature(0), llm.WithReasoningEffort("medium"))

	var tool WeatherTool
	agent := New("weather_agent", model).WithTools(tool)

	executor := NewToolExecutor(tool)
	executor.Before = func(agent, reasoning string, call llm.ToolCall, callIndex int) error {
		t.Log(strings.TrimSpace(reasoning))
		t.Logf("tool_call%+v", call)
		return nil
	}
	resp, err := agent.Run(context.Background(), "What's the weather like in London?", executor)
	require.NoError(t, err)

	t.Log(resp.Content)
	assert.Equal(t, "The weather in London is currently sunny with a temperature of 25°C.", resp.Content)
	assert.Equal(t, 4, len(agent.Memory.Messages))
}

type WeatherTool struct{}

func (WeatherTool) Definition() llm.FunctionDefinition {
	return llm.FunctionDefinition{
		Name:        "get_weather",
		Description: "Get weather at the given location",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"location": map[string]string{
					"type": "string",
				},
			},
			"required": []string{"location"},
		},
	}
}

func (WeatherTool) Call(ctx context.Context, args string) string {
	return "Sunny, 25°C"
}
