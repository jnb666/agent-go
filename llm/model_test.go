package llm

import (
	"context"
	"strings"
	"testing"

	"github.com/jnb666/agent-go/util"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testModel = "Qwen3.5-9B"

func init() {
	log.SetLevel(log.InfoLevel)
}

func TestListModels(t *testing.T) {
	models, err := ListModels(context.Background())
	require.NoError(t, err)
	t.Log(util.Pretty(models))
	// check model used in the tests
	assert.Contains(t, strings.Join(models, ","), testModel)
}

func TestNewModel(t *testing.T) {
	m, err := NewModel(context.Background(), testModel)
	require.NoError(t, err)
	t.Logf("model ID=%q  baseURL=%q server=%q context=%d", m.id, m.baseURL, m.server, m.contextSize)
	assert.Contains(t, m.ID(), testModel)
	assert.Equal(t, "http://deepthought:8080/v1", m.BaseURL())
	assert.Equal(t, "llamacpp", m.Server())
	assert.Equal(t, "reasoning_content", m.reasoning)
}

func TestModelOptions(t *testing.T) {
	m, err := NewModel(context.Background(), testModel)
	require.NoError(t, err)
	m.SetOptions(WithTemperature(0.8), WithTopP(0.95), WithTopK(20), WithPresencePenalty(1.5), WithRepetitionPenalty(1.0))
	opts := util.Pretty(m.Config(), util.Compact)
	t.Log(opts)
	assert.Equal(t, `{presence_penalty: 1.5, reasoning_effort: "medium", repetition_penalty: 1, temperature: 0.8, top_k: 20, top_p: 0.95}`, opts)
}
