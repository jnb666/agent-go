// Package agents implements an agent loop with tool calling using llm package to communicate with the model.
package agents

import (
	"context"
	"fmt"
	"strings"

	"github.com/jnb666/agent-go/llm"
)

// The Tool interface is implemented by any external tools.
type Tool interface {
	Definition() llm.FunctionDefinition
	Call(ctx context.Context, args string) string
}

// An Executor is used by the agent to call any external system.
// In case of an error the response should return details of the failure which the agent can process.
type Executor interface {
	CallTool(ctx context.Context, agent, reasoning string, call llm.ToolCall) string
}

// An agent drives model generation with a set of registered tools.
type Agent struct {
	Name          string
	Model         *llm.Model
	Prompt        string
	Memory        Memory
	Tools         []llm.FunctionDefinition
	StatsCallback func(llm.Stats)
	MaxRetries    int
}

// Create a new agent using the given model.
func New(name string, model *llm.Model) *Agent {
	return &Agent{
		Name:       name,
		Model:      model,
		MaxRetries: 2,
	}
}

func (a *Agent) WithPrompt(prompt string) *Agent {
	a.Prompt = prompt
	return a
}

func (a *Agent) WithTools(tools ...Tool) *Agent {
	for _, tool := range tools {
		a.Tools = append(a.Tools, tool.Definition())
	}
	return a
}

func (a *Agent) String() string {
	var toolNames []string
	for _, tool := range a.Tools {
		toolNames = append(toolNames, tool.Name)
	}
	return fmt.Sprintf("Agent:%s Model:%s Tools:%s", a.Name, a.Model.ID(), strings.Join(toolNames, ","))
}

// Add the user message to the messages list and run the event loop calling tools as needed until
// either the final content is generated or a fatal error occurs.
func (a *Agent) Run(ctx context.Context, userMessage string, exec Executor) (final llm.Message, err error) {
	a.Memory.Append(llm.Message{Role: "user", Content: userMessage})
	retry := 0
	for {
		r, err := a.Model.Generate(ctx, a.Memory.WithPrompt(a.Prompt), llm.WithTools(a.Tools...))
		if err != nil {
			return llm.Message{}, err
		}
		if a.StatsCallback != nil {
			a.StatsCallback(r.Stats)
		}
		if len(r.Message.ToolCalls) == 0 {
			if strings.TrimSpace(r.Message.Content) != "" {
				a.Memory.Append(r.Message)
				return r.Message, nil
			} else if retry >= a.MaxRetries {
				r.Message.Content = fmt.Sprintf("Error: failing due to empty response after %d retries", a.MaxRetries)
				a.Memory.Append(r.Message)
				return r.Message, nil
			} else {
				retry++
				continue
			}
		}
		if exec == nil {
			return llm.Message{}, fmt.Errorf("Run: executor not defined")
		}
		retry = 0
		a.Memory.Append(r.Message)
		for _, call := range r.Message.ToolCalls {
			resp := exec.CallTool(ctx, a.Name, r.Message.Reasoning, call)
			if err := ctx.Err(); err != nil {
				resp = err.Error()
			}
			a.Memory.Append(llm.Message{Role: "tool", Content: resp, ToolCallID: call.ID})
		}
	}
}

// ToolExecutor is a simple executor used to call tools. There are two optional callbacks:
// Before is called before each tool call - if it returns an error the call is skipped.
// After is called with the response from the tool
type ToolExecutor struct {
	Tools  []Tool
	Before func(agent, reasoning string, call llm.ToolCall, callNumber int) error
	After  func(id, response string)
}

func NewToolExecutor(tools ...Tool) ToolExecutor {
	return ToolExecutor{Tools: tools}
}

func (t ToolExecutor) CallTool(ctx context.Context, agent, reasoning string, call llm.ToolCall) string {
	for i, tool := range t.Tools {
		if tool.Definition().Name == call.Name {
			if t.Before != nil {
				err := t.Before(agent, reasoning, call, i)
				if err != nil {
					return err.Error()
				}
			}
			resp := tool.Call(ctx, call.Arguments)
			if t.After != nil {
				t.After(call.ID, resp)
			}
			return resp
		}
	}
	return fmt.Sprintf("Error: tool function %q is not defined", call.Name)
}
