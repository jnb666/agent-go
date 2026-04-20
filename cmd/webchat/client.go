package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jnb666/agent-go/agents"
	"github.com/jnb666/agent-go/llm"
	"github.com/jnb666/agent-go/tools/browser"
	"github.com/jnb666/agent-go/util"
	katex "github.com/jnb666/goldmark-katex"
	log "github.com/sirupsen/logrus"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	htmlRenderer "github.com/yuin/goldmark/renderer/html"
)

const MaxConversations = 30

// Client manages the websockets connection between the server and a web browser UI.
type Client struct {
	agents.Config
	ws            *websocket.Conn
	input         chan Request
	agent         *agents.Agent
	tools         []agents.Tool
	shutdown      func()
	stats         Stats
	start         time.Time
	reasoning     string
	content       string
	updateContent bool
	error         error
}

// Initialise new client and load model
func newClient(ctx context.Context, ws *websocket.Conn) *Client {
	c := &Client{
		ws:    ws,
		input: make(chan Request),
	}
	var server string
	c.Config, server, c.error = initModelConfig(ctx)
	if c.error != nil {
		return c
	}
	err := agents.LoadConfig(server, &c.Config)
	if err != nil {
		log.Warn(err)
	}
	c.tools, c.shutdown, c.error = browser.Tools()
	if c.error != nil {
		return c
	}
	c.updateConfig(ctx)
	c.send(Response{Type: "config", Config: &c.Config})
	return c
}

// Initialise default model config settings
func initModelConfig(ctx context.Context) (cfg agents.Config, server string, err error) {
	cfg = DefaultConfig()
	modelIDs, err := llm.ListModels(ctx)
	if err != nil {
		return cfg, "", err
	}
	for _, id := range modelIDs {
		model, err := llm.NewModel(ctx, id)
		if err != nil {
			return cfg, "", err
		}
		cfg.Models[id] = model.Config().GenerationConfig
		server = model.Server()
	}
	return cfg, server, nil
}

// Update current config - loads model and initialises agent
func (c *Client) updateConfig(ctx context.Context) {
	model, err := llm.NewModel(ctx, c.Model)
	if err != nil {
		c.error = err
		return
	}
	log.Debugf("Connected to %s at %s", model.ID(), model.BaseURL())
	c.Model = model.ID()
	model.SetConfig(llm.Config{GenerationConfig: c.Models[c.Model]})
	model.SetStreaming(true, c.streamContent, c.streamReasoning)

	c.agent = agents.New("chat_agent", model)
	c.agent.SetPromptTemplate(c.SystemPrompt)
	c.agent.PromptArgs = PromptArgs{}
	c.agent.StatsCallback = c.updateStats
	c.agent.Executor.After = c.onToolResponse
	c.agent.Executor.Tools = nil
	for _, tool := range c.tools {
		if slices.ContainsFunc(c.Tools, func(c agents.ToolConfig) bool { return c.Enabled }) {
			c.agent.Executor.Tools = append(c.agent.Executor.Tools, tool)
		}
	}
	log.Info(c.agent)
	c.error = util.SaveJSON(filepath.Join(util.ConfigDir, "config_"+model.Server()+".json"), c.Config)
}

// Load tools and poll websocket for updates
func (c *Client) run(ctx context.Context) {
	if c.error != nil {
		return
	}
	go func() {
		for {
			var req Request
			req.Error = c.ws.ReadJSON(&req)
			if req.Error != nil {
				break
			}
			if req.Type == "ping" {
				c.ws.WriteJSON(Response{Type: "pong"})
			} else {
				c.input <- req
			}
		}
	}()
	for c.error == nil {
		select {
		case <-ctx.Done():
			c.error = ctx.Err()
		case req := <-c.input:
			if req.Error == nil {
				c.handleRequest(ctx, req)
			} else {
				c.error = req.Error
			}
		}
	}
}

// Shutdown tools
func (c *Client) close() {
	if c.shutdown != nil {
		c.shutdown()
	}
}

// Handle incoming request message on websocket
func (c *Client) handleRequest(ctx context.Context, req Request) {
	util.LogDebug("HandleRequest", req)
	switch req.Type {
	case "chat":
		c.reasoning = ""
		c.content = ""
		c.updateContent = false
		newChat := len(c.agent.Memory.Messages) == 0
		c.stats = Stats{}
		c.start = time.Now()
		_, c.error = c.agent.Run(ctx, req.Message.Content)
		if c.error != nil {
			return
		}
		err := util.SaveJSON(filepath.Join(util.ConfigDir, c.agent.Memory.ID+".json"), c.agent.Memory)
		if err != nil {
			log.Errorf("error saving conversation: %v", err)
		} else if newChat {
			c.listChats()
		}
	case "config":
		if req.Config == nil {
			log.Info("get config")
		} else {
			util.LogDebug("update config:", req.Config)
			c.Model = req.Config.Model
			c.SystemPrompt = req.Config.SystemPrompt
			c.Tools = req.Config.Tools
			if values, ok := req.Config.Models[c.Model]; ok {
				c.Models[c.Model] = values
			}
			c.updateConfig(ctx)
		}
		c.send(Response{Type: "config", Config: &c.Config})
	case "list":
		c.listChats()
	case "load":
		c.loadChat(req.ID)
	case "delete":
		c.deleteChat(req.ID)
	default:
		c.error = fmt.Errorf("invalid request type %q", req.Type)
	}
}

// get list of saved conversation ids
func (c *Client) listChats() {
	log.Info("list saved chats")
	entries, err := os.ReadDir(util.ConfigDir)
	if err != nil {
		c.error = err
		return
	}
	var list []Item
	for i := len(entries) - 1; i >= 0 && i >= len(entries)-MaxConversations; i-- {
		e := entries[i]
		if e.Type().IsRegular() && !strings.HasPrefix(e.Name(), "config") && strings.HasSuffix(e.Name(), ".json") {
			var conv agents.Memory
			if err := util.LoadJSON(filepath.Join(util.ConfigDir, e.Name()), &conv); err != nil {
				log.Errorf("Error reading %s: %v", e.Name(), err)
				continue
			}
			if len(conv.Messages) > 0 {
				list = append(list, Item{ID: conv.ID, Summary: conv.Messages[0].Content})
			}
		}
	}
	c.send(Response{Type: "list", List: list, CurrentID: c.agent.Memory.ID})
}

// load conversation with given id, or new conversation if blank
func (c *Client) loadChat(id string) {
	log.Infof("load chat: id=%s", id)
	if id == "" {
		c.agent.Memory = agents.NewMemory()
	} else {
		if err := util.LoadJSON(filepath.Join(util.ConfigDir, id+".json"), c.agent.Memory); err != nil {
			log.Errorf("error loading conversation %s: %v", id, err)
			return
		}
	}
	conv := slices.Clone(c.agent.Memory.Messages)
	for i, msg := range conv {
		if msg.Role == "tool" {
			call := c.agent.Memory.ToolCall(msg.ToolCallID)
			conv[i].Content = toHTML(call.String()+"\n\n"+msg.Content, msg.Role, msg.Compacted != "")
		} else {
			conv[i].Content = toHTML(msg.Content, msg.Role, false)
			conv[i].Reasoning = toHTML(msg.Reasoning, msg.Role, false)
		}
	}
	c.send(Response{Type: "load", Conversation: conv, CurrentID: c.agent.Memory.ID})
}

// delete chat with given id and start a new conversation
func (c *Client) deleteChat(id string) {
	log.Infof("delete conversation: id=%s", id)
	err := os.Remove(filepath.Join(util.ConfigDir, id+".json"))
	if err != nil {
		log.Errorf("error deleting conversation %s: %v", id, err)
		return
	}
	c.listChats()
	c.loadChat("")
}

// Callbacks to handle new events from the agent
func (c *Client) streamContent(chunk string, count int, end bool) {
	c.content += chunk
	if end || strings.Contains(chunk, "\n") {
		var msg Message
		msg.Role = "assistant"
		if msg.Content = toHTML(c.content, msg.Role, false); msg.Content != "" {
			msg.Update = c.updateContent
			msg.End = end
			c.send(Response{Type: "chat", Message: &msg})
			c.updateContent = true
		}
	}
}

func (c *Client) streamReasoning(chunk string, count int, end bool) {
	var msg Message
	msg.Role = "assistant"
	msg.Update = c.reasoning != ""
	c.reasoning += chunk
	if msg.Reasoning = toHTML(c.reasoning, msg.Role, false); msg.Reasoning != "" {
		c.send(Response{Type: "chat", Message: &msg})
		c.updateContent = false
	}
}

func (c *Client) onToolResponse(id, resp string, elapsed time.Duration) {
	c.reasoning = ""
	call := c.agent.Memory.ToolCall(id)
	var msg Message
	msg.Role = "tool"
	msg.Content = toHTML(call.String()+"\n\n"+resp, msg.Role, false)
	c.send(Response{Type: "chat", Message: &msg})
	c.stats.ToolCalls++
	c.updateContent = false
}

func (c *Client) updateStats(s llm.Stats) {
	log.Info(s)
	if ctx := c.Config.Models[c.Config.Model].ContextSize; ctx.Valid() {
		c.stats.ContextUsed = fmt.Sprintf("%.0f%%", 100*float64(s.PromptTokens)/float64(ctx.Value))
	} else {
		c.stats.ContextUsed = fmt.Sprint(s.PromptTokens)
	}
	c.stats.PromptTime = fmt.Sprintf("%.1f s", s.PromptMsec/1000)
	c.stats.TokensGenerated += s.CompletionTokens
	if s.CompletionMsec != 0 {
		c.stats.GenerationSpeed = fmt.Sprintf("%.1f tps", float64(s.CompletionTokens)*1000/s.CompletionMsec)
	}
	c.stats.TotalTime = fmt.Sprintf("%.1f s", time.Since(c.start).Seconds())
	c.send(Response{Type: "stats", Stats: &c.stats})
}

func (c *Client) send(msg Response) {
	c.error = c.ws.WriteJSON(msg)
}

var reLink = regexp.MustCompile(`(?i)(<a href="[^"]+")`)

// Render markdown document to HTML
func renderMarkdown(doc string) (string, error) {
	latexDelims := strings.Contains(doc, "\\[") && strings.Contains(doc, "\\]") ||
		strings.Contains(doc, "\\(") && strings.Contains(doc, "\\)")
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			&katex.Extender{LatexDelimiters: latexDelims, MustContain: '\\'},
			highlighting.NewHighlighting(highlighting.WithStyle("monokai")),
		),
		goldmark.WithRendererOptions(htmlRenderer.WithHardWraps(), htmlRenderer.WithUnsafe()),
	)
	var buf bytes.Buffer
	err := md.Convert([]byte(doc), &buf)
	if err != nil {
		return "", err
	}
	return reLink.ReplaceAllString(buf.String(), `${1} target="_blank"`), nil
}

var urlRegexp = regexp.MustCompile(`(?:http[s]?:\/\/.)?(?:www\.)?[-a-zA-Z0-9@%._\+~#=]{2,256}\.[a-z]{2,6}\b(?:[-a-zA-Z0-9@:%_\+.~#?&\/\/=]*)`)

// utils
func toHTML(content, role string, compacted bool) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	if role == "tool" {
		extra := ""
		if compacted {
			extra = " compacted"
		}
		content = urlRegexp.ReplaceAllStringFunc(content, func(url string) string {
			return fmt.Sprintf(`<a href="%s" target="_blank">%s</a>`, url, url)
		})
		return `<pre><code class="tool-response` + extra + `">` + content + `</code></pre>`
	}
	if role == "assistant" {
		html, err := renderMarkdown(content)
		if err == nil {
			return html
		} else {
			log.Error("error converting markdown:", err)
		}
	}
	return "<p>" + strings.ReplaceAll(content, "\n", "<br>") + "</p>"
}
