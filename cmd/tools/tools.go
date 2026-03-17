// Interactive agent example using the weather or browser tools.
package main

import (
	"context"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/jnb666/agent-go/agents"
	"github.com/jnb666/agent-go/llm"
	"github.com/jnb666/agent-go/tools/browser"
	"github.com/jnb666/agent-go/tools/weather"
	"github.com/jnb666/agent-go/util"
	log "github.com/sirupsen/logrus"
)

func main() {
	var nostream, debug, useBrowser bool
	var modelID string
	systemPrompt := `You are a helpful assistant. The current date is {{ .Time.Format "Monday 2 January 2006" }}.`
	flag.StringVar(&systemPrompt, "system", systemPrompt, "set custom system prompt")
	flag.BoolVar(&useBrowser, "browser", false, "use browser tools - default is weather tools")
	flag.BoolVar(&nostream, "nostream", false, "disable streaming")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.StringVar(&modelID, "model", "", "model name - defaults to first listed if not set")
	flag.Parse()
	if debug {
		log.SetLevel(log.DebugLevel)
	}
	rl, err := readline.New("> ")
	if err != nil {
		log.Fatal(err)
	}
	defer rl.Close()

	var tools []agents.Tool
	if useBrowser {
		var shutdown func()
		tools, shutdown, err = browser.Tools()
		if err != nil {
			log.Fatal(err)
		}
		defer shutdown()
	} else {
		tools = weather.Tools
	}

	agent, err := initAgent(modelID, nostream, tools, systemPrompt)
	if err != nil {
		log.Fatal(err)
	}
	log.Info(agent)

	var stats llm.Stats
	agent.StatsCallback = func(s llm.Stats) { stats = s }

	for {
		question, err := rl.Readline()
		if err != nil {
			break
		}
		resp, err := agent.Run(context.Background(), question)
		if nostream {
			if resp.Reasoning != "" {
				fmt.Println(resp.Reasoning)
			}
			fmt.Println("== response ==")
			fmt.Println(resp.Content)
		}
		log.Info(stats)
		if err != nil {
			log.Error(err)
		}
	}
}

type PromptArgs struct{}

func (PromptArgs) Time() time.Time {
	return time.Now()
}

func initAgent(modelID string, nostream bool, tools []agents.Tool, systemPrompt string) (*agents.Agent, error) {
	model, err := llm.NewModel(context.Background(), modelID)
	if err != nil {
		return nil, err
	}
	log.Infof("Connected to %s at %s", model.ID(), model.BaseURL())
	if !nostream {
		model.SetStreaming(true, printContent, printReasoning)
	}

	executor := agents.NewExecutor(tools...)
	executor.Before = func(agent, reasoning string, call llm.ToolCall, callIndex int) error {
		if nostream && callIndex == 0 {
			log.Info(strings.TrimSpace(reasoning))
		}
		util.LogDebug("", call)
		return nil
	}
	executor.After = func(id, resp string, elapsed time.Duration) {
		log.Info("tool response: ", resp)
	}

	agent := agents.New("tool_agent", model).WithExecutor(executor)
	err = agent.SetPromptTemplate(systemPrompt)
	agent.PromptArgs = PromptArgs{}
	return agent, err
}

func printReasoning(chunk string, count int, end bool) {
	fmt.Print(chunk)
}

func printContent(chunk string, count int, end bool) {
	if count == 1 {
		fmt.Println("\n== content ==")
	}
	fmt.Print(chunk)
	if end {
		fmt.Println()
	}
}
