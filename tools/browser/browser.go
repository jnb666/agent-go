// Package browser implements a web page scraper tool using the scrape package.
package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/jnb666/agent-go/agents"
	"github.com/jnb666/agent-go/llm"
	"github.com/jnb666/agent-go/scrape"
	"github.com/jnb666/agent-go/tools/brave"
	log "github.com/sirupsen/logrus"
)

// Maximum number of lines to return per request
var MaxLines = 50

// Get all web search, open and find tools
func Tools(opts ...scrape.Option) (tools []agents.Tool, shutdown func(), err error) {
	browser, err := scrape.New(opts...)
	if err != nil {
		return nil, nil, err
	}
	tools = []agents.Tool{&brave.Search{}, Open{browser}, Find{browser}}
	shutdown = func() {
		browser.Shutdown()
	}
	return
}

// Tool to retrieve a web page and return the text content in Markdown format.
type Open struct {
	*scrape.Browser
}

func (Open) Definition() llm.FunctionDefinition {
	return llm.FunctionDefinition{
		Name:        "web_fetch",
		Description: "Fetches the contents of a web page and converts the text to Markdown.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "url of the web page to retrieve. If blank then the latest page is used.",
				},
				"startline": map[string]any{
					"type":        "number",
					"description": "Line number in the page to position the viewport. Defaults to the start if not provided.",
				},
				"maxlines": map[string]any{
					"type":        "number",
					"description": "Maximum number of lines to return. Defaults to " + strconv.Itoa(MaxLines) + ".",
				},
			},
		},
	}
}

func (t Open) Call(ctx context.Context, arg string) string {
	if t.Browser == nil {
		return "Error calling web_fetch - web browser not initialised"
	}
	log.Infof("call web_fetch(%s)", arg)
	var args struct {
		URL       string
		Startline float64
		Maxlines  float64
	}
	if err := json.Unmarshal([]byte(arg), &args); err != nil {
		return fmt.Sprintf("Error parsing arguments for web_fetch: %s", err)
	}
	if args.URL == "" {
		args.URL = t.LastURL()
	}
	if args.URL == "" {
		return "Error calling web_fetch - no page to load"
	}
	r, err := t.Scrape(ctx, args.URL)
	if err != nil {
		return fmt.Sprintf("Error calling web_fetch with url=%q - %s", args.URL, err)
	}
	startLine := 1
	if args.Startline > 0 {
		startLine = int(args.Startline)
		t.SetLine(startLine)
	}
	maxLines := MaxLines
	if args.Maxlines > 0 {
		maxLines = int(args.Maxlines)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n", r.Title)
	fmt.Fprintf(&b, "(%s)\n", r.URL)
	lines := strings.Split(strings.TrimSpace(r.Markdown), "\n")
	formatDocument(&b, lines, startLine, maxLines)
	return b.String()
}

// Tool to search for text in a page previously loaded by the open tool.
type Find struct {
	*scrape.Browser
}

func (Find) Definition() llm.FunctionDefinition {
	return llm.FunctionDefinition{
		Name: "web_find",
		Description: "Finds exact matches for `pattern` in the latest page which was loaded by the web_fetch tool." +
			" If a match is found then returns the page scrolled to the first line containing that string." +
			" Repeat the same web_find call to scroll to the next match.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{
					"type":        "string",
					"description": "Text to search for in the page.",
				},
				"url": map[string]any{
					"type":        "string",
					"description": "url of the web page to retrieve. If blank then the latest page is used.",
				},
				"maxlines": map[string]any{
					"type":        "number",
					"description": "Maximum number of lines to return. Defaults to " + strconv.Itoa(MaxLines) + ".",
				},
			},
			"required": []string{"pattern"},
		},
	}
}

func (t Find) Call(ctx context.Context, arg string) string {
	if t.Browser == nil {
		return "Error calling web_find - web browser not initialised"
	}
	log.Infof("call web_find(%s)", arg)
	var args struct {
		URL      string
		Pattern  string
		Maxlines float64
	}
	if err := json.Unmarshal([]byte(arg), &args); err != nil {
		return fmt.Sprintf("Error parsing arguments for web_find: %s", err)
	}
	if args.URL == "" {
		args.URL = t.LastURL()
	}
	if args.URL == "" {
		return "Error calling web_find - no page to search"
	}
	pattern := strings.TrimSpace(args.Pattern)
	if pattern == "" {
		return "Error calling web_find - pattern is required"
	}
	r, err := t.Scrape(ctx, args.URL)
	if err != nil {
		return fmt.Sprintf("Error calling web_find with url=%q - %s", args.URL, err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n", r.Title)
	fmt.Fprintf(&b, "(%s) ", r.URL)
	lines := strings.Split(strings.TrimSpace(r.Markdown), "\n")
	startLine := findPattern(pattern, lines, t.Line())
	if startLine >= 0 {
		maxLines := MaxLines
		if args.Maxlines > 0 {
			maxLines = int(args.Maxlines)
		}
		fmt.Fprintf(&b, "find results for “%s”\n", pattern)
		formatDocument(&b, lines, startLine+1, maxLines)
		t.SetLine(startLine + 1)
	} else {
		fmt.Fprintf(&b, "pattern “%s” not found in page\n", args.Pattern)
		t.SetLine(0)
	}
	return b.String()
}

func findPattern(pattern string, lines []string, startLine int) int {
	pattern = strings.ToLower(pattern)
	for i := startLine; i < len(lines); i++ {
		if strings.Contains(strings.ToLower(lines[i]), pattern) {
			return i
		}
	}
	return -1
}

func formatDocument(w io.Writer, lines []string, startLine, maxLines int) {
	startLine = max(min(startLine, len(lines)-maxLines), 1)
	endLine := min(startLine+maxLines-1, len(lines))
	fmt.Fprintf(w, "**viewing lines [%d - %d] of %d**\n\n", startLine, endLine, len(lines))

	for i := startLine - 1; i < endLine; i++ {
		fmt.Fprintf(w, "L%d: %s\n", i+1, lines[i])
	}
}
