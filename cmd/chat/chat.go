// Simple interactive chat example using the llm package.
package main

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/chzyer/readline"
	"github.com/jnb666/agent-go/llm"
	"github.com/jnb666/agent-go/util"
	"github.com/openai/openai-go/v3/shared"
	log "github.com/sirupsen/logrus"
)

func main() {
	var nostream, debug bool
	var systemPrompt, reasoning, modelID string
	flag.StringVar(&reasoning, "reasoning", "none", "set reasoning - none, low, medium or high")
	flag.StringVar(&systemPrompt, "system", "You are a helpful assistant.", "set custom system prompt")
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
	model.SetOptions(llm.WithReasoningEffort(shared.ReasoningEffort(reasoning)))
	if !nostream {
		model.SetStreaming(true, printContent, printReasoning)
	}

	var messages []llm.Message
	if systemPrompt != "" {
		messages = append(messages, llm.Message{Role: "system", Content: systemPrompt})
	}

	for {
		question, err := rl.Readline()
		if err != nil {
			break
		}
		messages = append(messages, llm.Message{Role: "user", Content: strings.TrimSpace(question)})

		resp, err := model.Generate(context.Background(), messages)
		if nostream {
			if resp.Message.Reasoning != "" {
				fmt.Println(resp.Message.Reasoning)
			}
			fmt.Println("== response ==")
			fmt.Println(resp.Message.Content)
		}
		if err == nil {
			util.LogDebug("== completion ==\n", resp)
			messages = append(messages, resp.Message)
			log.Info(resp.Stats)
		} else {
			log.Error(err)
			messages = messages[:len(messages)-1]
		}
	}
}

func printReasoning(chunk string, count int, end bool) {
	fmt.Print(chunk)
}

func printContent(chunk string, count int, end bool) {
	if count == 1 {
		fmt.Println("== response ==")
	}
	fmt.Print(chunk)
	if end {
		fmt.Println()
	}
}
