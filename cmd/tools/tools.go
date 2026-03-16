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
	systemPrompt := fmt.Sprintf("You are a helpful assistant."+
		" You should answer concisely unless the user asks for more details."+
		" The current date is %s.", time.Now().Format("2 January 2006"),
	)
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

	model, err := llm.NewModel(context.Background(), modelID)
	if err != nil {
		log.Fatal(err)
	}
	log.Infof("Connected to %s at %s", model.ID(), model.BaseURL())
	if !nostream {
		model.SetStreaming(true, printContent, printReasoning)
	}

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

	var stats llm.Stats
	agent := agents.New("tool_agent", model).WithTools(tools...).WithPrompt(systemPrompt)
	agent.StatsCallback = func(s llm.Stats) {
		stats = s
	}
	log.Info(agent)

	executor := agents.NewToolExecutor(tools...)
	executor.Before = func(agent, reasoning string, call llm.ToolCall, callIndex int) error {
		if nostream && callIndex == 0 {
			log.Info(strings.TrimSpace(reasoning))
		}
		util.LogDebug("", call)
		return nil
	}
	executor.After = func(id, resp string) {
		log.Info("tool response: ", resp)
	}

	for {
		question, err := rl.Readline()
		if err != nil {
			break
		}
		resp, err := agent.Run(context.Background(), question, executor)
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
