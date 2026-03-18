package agents

import (
	"slices"

	"github.com/google/uuid"
	"github.com/jnb666/agent-go/llm"
	"github.com/jnb666/agent-go/util"
	log "github.com/sirupsen/logrus"
)

// Memory stores the current state associated with an agent.
type Memory struct {
	ID       string        `json:"id"`
	Messages []llm.Message `json:"messages"`
}

func NewMemory() *Memory {
	return &Memory{ID: uuid.Must(uuid.NewV7()).String()}
}

func (m *Memory) NumMessages() int {
	return len(m.Messages)
}

func (m *Memory) Clone() *Memory {
	m2 := *m
	m2.Messages = slices.Clone(m.Messages)
	return &m2
}

func (m *Memory) Append(msg llm.Message) {
	util.LogDebug("== add memory ==\n", msg)
	m.Messages = append(m.Messages, msg)
}

func (m *Memory) Last() (msg llm.Message, ok bool) {
	if len(m.Messages) == 0 {
		return
	}
	return m.Messages[len(m.Messages)-1], true
}

func (m *Memory) WithPrompt(prompt string) (msgs []llm.Message) {
	if prompt != "" {
		log.Debugf("system prompt: %s", prompt)
		msgs = []llm.Message{{Role: "system", Content: prompt}}
	}
	return append(msgs, m.Messages...)
}
