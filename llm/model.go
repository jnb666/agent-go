// Package llm wraps the openAI SDK to provide a simpler interface to call the chat completions REST API.
//
// It supports either synchronous or streaming calls with optional reasoning tokens and tool calling.
package llm

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/jnb666/agent-go/util"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// LLM model identifier and connection information.
type Model struct {
	client          openai.Client
	config          Config
	streaming       bool
	streamContent   CallbackFunc
	streamReasoning CallbackFunc
	id              string
	alias           string
	baseURL         string
	server          string
	reasoning       string
}

// Callback called from streaming generation. An empty call with end=true is sent after the last chunk.
type CallbackFunc func(chunk string, count int, end bool)

// List available model ids. Connects to server given by OPENAI_BASE_URL and OPENAI_API_KEY environment variables unless overriden by request options.
func ListModels(ctx context.Context, options ...option.RequestOption) ([]string, error) {
	client := openai.NewClient(options...)
	models, err := client.Models.List(ctx)
	if err != nil {
		return nil, err
	}
	util.LogDebug("== models ==\n", models)
	var ids []string
	for _, model := range models.Data {
		ids = append(ids, model.ID)
	}
	return ids, nil
}

// Connects to server given by OPENAI_BASE_URL and OPENAI_API_KEY environment variables unless overriden by request options.
// If modelID is empty then uses the first model returned by ListModels
func NewModel(ctx context.Context, modelID string, options ...option.RequestOption) (*Model, error) {
	m := new(Model)
	m.client = openai.NewClient(options...)
	extra := option.WithMiddleware(func(req *http.Request, nxt option.MiddlewareNext) (*http.Response, error) {
		m.baseURL = strings.TrimSuffix(req.URL.String(), "/models")
		return nxt(req)
	})
	models, err := m.client.Models.List(ctx, extra)
	if err != nil {
		return nil, err
	}
	util.LogDebug("== models ==\n", models)
	for _, model := range models.Data {
		if modelID == "" || strings.Contains(model.ID, modelID) {
			m.id = model.ID
			m.server = model.OwnedBy
			if m.server == "llamacpp" {
				m.reasoning = "reasoning_content"
			} else {
				m.reasoning = "reasoning"
			}
			return m, nil
		}
	}
	return nil, fmt.Errorf("model %s not found on server", modelID)
}

// Model ID string returned from /models endpoint
func (m *Model) ID() string {
	return m.id
}

// Base connection URL from OPENAI_BASE_URL or set in the request options
func (m *Model) BaseURL() string {
	return m.baseURL
}

// Server name - e.g. llamacpp, vllm etc. - from /models response
func (m *Model) Server() string {
	return m.server
}

// Enable or disable streaming option and set callback functions if not nil
func (m *Model) SetStreaming(enabled bool, contentCallback, reasoningCallback CallbackFunc) {
	m.streaming = enabled
	m.streamContent = contentCallback
	m.streamReasoning = reasoningCallback
}

// Get configuration set using SetOptions.
func (m *Model) Options() Config {
	return m.config
}

// Set model options which are used by default in the Generate method.
// If called multiple times the latest version of each option is used.
func (m *Model) SetOptions(opts ...Option) {
	for _, opt := range opts {
		opt(&m.config)
	}
}
