package llm

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/jnb666/agent-go/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateNoReasoning(t *testing.T) {
	if testing.Short() {
		return
	}
	m, err := NewModel(context.Background(), testModel)
	require.NoError(t, err)
	t.Log(m.ID())
	m.SetOptions(WithSeed(42), WithTemperature(0), WithReasoningEffort("none"))

	for _, name := range []string{"sync", "streaming"} {
		t.Run(name, func(t *testing.T) {
			streamer := testStreamer{T: t}
			m.SetStreaming(name == "streaming", streamer.streamContent, nil)

			res, err := m.Generate(context.Background(), testMessages)
			require.NoError(t, err)
			streamer.flush()
			t.Log(util.Pretty(res))

			answer := "There are **3** \"r\"s in the word **Strawberry**.\n\nHere is the breakdown:\nS - **t** - **r** (1) - a - w - b - e - **r** (2) - **r** (3) - y"
			assert.Equal(t, "stop", res.FinishReason)
			assert.Equal(t, "assistant", res.Message.Role)
			assert.Equal(t, answer, res.Message.Content)
			assert.Equal(t, 52, res.Stats.PromptTokens)
			assert.Equal(t, 60, res.Stats.CompletionTokens)
			if name == "streaming" {
				assert.Equal(t, res.Message.Content, streamer.content)
			}
		})
	}
}

func TestGenerateWithReasoning(t *testing.T) {
	if testing.Short() {
		return
	}
	m, err := NewModel(context.Background(), testModel)
	require.NoError(t, err)
	t.Log(m.ID())
	m.SetOptions(WithSeed(42), WithTemperature(0), WithReasoningEffort("medium"))

	for _, name := range []string{"sync", "streaming"} {
		t.Run(name, func(t *testing.T) {
			streamer := testStreamer{T: t}
			m.SetStreaming(name == "streaming", streamer.streamContent, streamer.streamReasoning)

			res, err := m.Generate(context.Background(), testMessages)
			require.NoError(t, err)
			streamer.flush()
			t.Log(util.Pretty(res))

			assert.Equal(t, "stop", res.FinishReason)
			assert.Equal(t, "assistant", res.Message.Role)
			assert.Greater(t, len(res.Message.Content), 100)
			assert.Greater(t, len(res.Message.Reasoning), 500)
			assert.Equal(t, 50, res.Stats.PromptTokens)
			assert.Greater(t, res.Stats.CompletionTokens, 300)
			if name == "streaming" {
				assert.Equal(t, res.Message.Content, streamer.content)
				assert.Equal(t, res.Message.Reasoning, streamer.reasoning)
			}
		})
	}
}

func TestGenerateWithCancel(t *testing.T) {
	if testing.Short() {
		return
	}
	m, err := NewModel(context.Background(), testModel)
	require.NoError(t, err)
	t.Log(m.ID())
	m.SetOptions(WithSeed(42), WithTemperature(0), WithReasoningEffort("medium"))

	for i, name := range []string{"sync", "streaming"} {
		t.Run(name, func(t *testing.T) {
			streamer := testStreamer{T: t}
			m.SetStreaming(name == "streaming", streamer.streamContent, streamer.streamReasoning)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			res, err := m.Generate(ctx, testMessages)
			t.Log(err)
			assert.ErrorIs(t, err, context.DeadlineExceeded)

			streamer.flush()
			t.Log(util.Pretty(res))

			assert.Equal(t, "deadline_exceeded", res.FinishReason)
			assert.Equal(t, "", res.Message.Content)
			if name == "streaming" {
				assert.Greater(t, len(res.Message.Reasoning), 50)
			}
			assert.Equal(t, 200, int(res.Stats.TotalMsec/10))
		})
		if i == 0 {
			time.Sleep(4 * time.Second)
		}
	}
}

func TestGenerateWithTools(t *testing.T) {
	if testing.Short() {
		return
	}
	m, err := NewModel(context.Background(), testModel)
	require.NoError(t, err)
	t.Log(m.ID())
	m.SetOptions(WithSeed(42), WithTemperature(0), WithReasoningEffort("medium"), WithTools(toolDef))

	for _, name := range []string{"sync", "streaming"} {
		t.Run(name, func(t *testing.T) {
			streamer := testStreamer{T: t}
			m.SetStreaming(name == "streaming", streamer.streamContent, streamer.streamReasoning)

			res, err := m.Generate(context.Background(), toolMessages[:2])
			require.NoError(t, err)
			streamer.flush()
			t.Log(util.Pretty(res))

			assert.Equal(t, "tool_calls", res.FinishReason)
			assert.Equal(t, "assistant", res.Message.Role)
			assert.Equal(t, "", res.Message.Content)
			assert.Greater(t, len(res.Message.Reasoning), 100)
			assert.Equal(t, 334, res.Stats.PromptTokens)
			assert.Greater(t, res.Stats.CompletionTokens, 100)
			require.Equal(t, 1, len(res.Message.ToolCalls))
			assert.NotEqual(t, "", res.Message.ToolCalls[0].ID)
			assert.Equal(t, "get_current_weather", res.Message.ToolCalls[0].Name)
			assert.Equal(t, "{\"location\":\"London,GB\"}", res.Message.ToolCalls[0].Arguments)
			if name == "streaming" {
				assert.Equal(t, res.Message.Content, streamer.content)
				assert.Equal(t, res.Message.Reasoning, streamer.reasoning)
			}
		})
	}
}

type testStreamer struct {
	*testing.T
	line      string
	content   string
	reasoning string
	mode      string
}

func (t *testStreamer) streamReasoning(s string, count int, end bool) {
	if count == 1 {
		t.mode = "reasoning"
	}
	t.reasoning += s
	t.line += s
	if strings.ContainsRune(t.line, '\n') {
		printLines(t.T, "reasoning", t.line)
		t.line = ""
	}
}

func (t *testStreamer) streamContent(s string, count int, end bool) {
	if count == 1 {
		t.flush()
		t.mode = "content"
	}
	t.content += s
	t.line += s
	if strings.ContainsRune(t.line, '\n') {
		printLines(t.T, "content", t.line)
		t.line = ""
	}
}

func (t *testStreamer) flush() {
	if t.line != "" {
		t.Logf("%s: %s", t.mode, t.line)
		t.line = ""
	}
}

func printLines(t *testing.T, title, s string) {
	s = strings.Trim(s, "\n")
	for _, line := range strings.Split(s, "\n") {
		t.Logf("%s: %s", title, line)
	}
}
