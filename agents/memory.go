package agents

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/jnb666/agent-go/llm"
	"github.com/jnb666/agent-go/util"
	log "github.com/sirupsen/logrus"
)

const (
	minContentLength = 100
)

// Memory stores the current state associated with an agent.
type Memory struct {
	ID       string    `json:"id"`
	Messages []Message `json:"messages"`
}

type Message struct {
	llm.Message
	Compacted string `json:"compacted,omitzero"`
	Deleted   bool   `json:"deleted,omitzero"`
}

func NewMemory() *Memory {
	return &Memory{ID: uuid.Must(uuid.NewV7()).String()}
}

func (m *Memory) MessageList(prompt string) []llm.Message {
	var msgs []llm.Message
	if prompt != "" {
		log.Debugf("system prompt: %s", prompt)
		msgs = append(msgs, llm.Message{Role: "system", Content: prompt})
	}
	for _, msg := range m.Messages {
		if msg.Deleted {
			continue
		}
		if msg.Compacted != "" {
			msg.Content = msg.Compacted
		}
		msgs = append(msgs, msg.Message)
	}
	return msgs
}

func (m *Memory) Append(msg llm.Message) {
	util.LogDebug("== add memory ==\n", msg)
	m.Messages = append(m.Messages, Message{Message: msg})
}

func (m *Memory) compactToolResults(keep int) {
	turns := 0
	for i := len(m.Messages) - 1; i >= 0; i-- {
		msg := m.Messages[i]
		if msg.Role == "user" {
			turns++
		}
		if turns > keep && msg.Role == "tool" && msg.Compacted == "" && len(msg.Content) > minContentLength {
			name := m.ToolCall(msg.ToolCallID)
			log.Infof("message %d: compact %s tool call", i, name)
			m.Messages[i].Compacted = fmt.Sprintf("## Tool result compacted - call %s to regenerate", name)
		}
	}
}

func (m *Memory) ToolCall(id string) llm.ToolCall {
	for _, msg := range m.Messages {
		for _, call := range msg.ToolCalls {
			if call.ID == id {
				return call
			}
		}
	}
	return llm.ToolCall{Name: "unknown"}
}
