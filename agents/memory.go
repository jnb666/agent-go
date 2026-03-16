package agents

import (
	"github.com/jnb666/agent-go/llm"
	"github.com/jnb666/agent-go/util"
)

// Memory stores the current state associated with an agent.
type Memory struct {
	Messages []llm.Message
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
		msgs = []llm.Message{{Role: "system", Content: prompt}}
	}
	return append(msgs, m.Messages...)
}
