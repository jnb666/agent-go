// Package agents implements an agent loop with tool calling using llm package to communicate with the model.
package agents

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/jnb666/agent-go/llm"
)

// The Tool interface is implemented by any external tools.
type Tool interface {
	Definition() llm.FunctionDefinition
	Call(ctx context.Context, args string) string
}

// An agent drives model generation with a set of registered tools.
type Agent struct {
	Name            string
	Model           *llm.Model
	Memory          *Memory
	PromptTemplate  *template.Template
	PromptArgs      any
	Executor        Executor
	StatsCallback   func(llm.Stats)
	MaxRetries      int
	KeepToolResults int
}

// Create a new agent using the given model.
func New(name string, model *llm.Model) *Agent {
	return &Agent{
		Name:            name,
		Model:           model,
		Memory:          NewMemory(),
		MaxRetries:      2,
		KeepToolResults: 3,
	}
}

func (a *Agent) SetPromptTemplate(promptTemplate string) error {
	var err error
	a.PromptTemplate, err = template.New("prompt").Parse(promptTemplate)
	return err
}

func (a *Agent) WithPromptArguments(args any) *Agent {
	a.PromptArgs = args
	return a
}

func (a *Agent) WithExecutor(exec Executor) *Agent {
	a.Executor = exec
	return a
}

func (a *Agent) String() string {
	var toolNames []string
	for _, tool := range a.Executor.ToolDefinitions() {
		toolNames = append(toolNames, tool.Name)
	}
	return fmt.Sprintf("Agent:%s Model:%s Tools:%s", a.Name, a.Model.ID(), strings.Join(toolNames, ","))
}

// Add the user message to the messages list and run the event loop calling tools as needed until
// either the final content is generated or a fatal error occurs.
func (a *Agent) Run(ctx context.Context, userMessage string) (final llm.Message, err error) {
	a.Memory.Append(llm.Message{Role: "user", Content: userMessage})
	if a.KeepToolResults > 0 {
		a.Memory.compactToolResults(a.KeepToolResults)
	}
	retry := 0
	for {
		prompt, err := execTemplate(a.PromptTemplate, a.PromptArgs)
		if err != nil {
			return llm.Message{}, err
		}
		r, err := a.Model.Generate(ctx, a.Memory.MessageList(prompt), llm.WithTools(a.Executor.ToolDefinitions()...))
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
		retry = 0
		a.Memory.Append(r.Message)
		for _, call := range r.Message.ToolCalls {
			resp := a.Executor.CallTool(ctx, a.Name, r.Message.Reasoning, call)
			a.Memory.Append(llm.Message{Role: "tool", Content: resp, ToolCallID: call.ID})
		}
	}
}

func execTemplate(t *template.Template, args any) (string, error) {
	if t == nil {
		return "", nil
	}
	var buf bytes.Buffer
	err := t.Execute(&buf, args)
	return buf.String(), err
}

// Executor is used to call tools. There are two optional callbacks:
// Before is called before each tool call - if it returns an error the call is skipped.
// After is called with the response from the tool
type Executor struct {
	Tools  []Tool
	Before func(agent, reasoning string, call llm.ToolCall, callNumber int) error
	After  func(id, response string, elapsed time.Duration)
}

func NewExecutor(tools ...Tool) Executor {
	return Executor{Tools: tools}
}

func (t Executor) ToolDefinitions() []llm.FunctionDefinition {
	var defs []llm.FunctionDefinition
	for _, tool := range t.Tools {
		defs = append(defs, tool.Definition())
	}
	return defs
}

func (t Executor) CallTool(ctx context.Context, agent, reasoning string, call llm.ToolCall) string {
	if ctx.Err() != nil {
		return "Error: " + ctx.Err().Error()
	}
	for i, tool := range t.Tools {
		if tool.Definition().Name == call.Name {
			if t.Before != nil {
				err := t.Before(agent, reasoning, call, i)
				if err != nil {
					return err.Error()
				}
			}
			start := time.Now()
			resp := tool.Call(ctx, call.Arguments)
			if t.After != nil {
				t.After(call.ID, resp, time.Since(start))
			}
			return resp
		}
	}
	return fmt.Sprintf("Error: tool function %q is not defined", call.Name)
}
