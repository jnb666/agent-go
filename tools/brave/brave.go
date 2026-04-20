// Package brave implements a web search tool using the Brave search API - see https://brave.com/search/api/
package brave

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jnb666/agent-go/llm"
	"github.com/jnb666/agent-go/util"
	log "github.com/sirupsen/logrus"
)

var (
	// Default configuration
	Country    = "gb"
	Language   = "en"
	NumResults = 10
	MaxRetries = 3
)

// Tool to search on the web - implements agent.Tool interface
type Search struct {
	nextSearch time.Time
}

func (Search) Definition() llm.FunctionDefinition {
	return llm.FunctionDefinition{
		Name: "web_search",
		Description: "Search for information on the web using brave.com" +
			" Returns a Markdown document with the top 10 most relevant links." +
			" Use the web_fetch tool to retrieve the full contents.",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "Text to search for on the web.",
				},
			},
			"required": []string{"query"},
		},
	}
}

func (t *Search) Call(ctx context.Context, arg string) string {
	apiKey := os.Getenv("BRAVE_API_KEY")
	if apiKey == "" {
		return "Error calling web_search - BRAVE_API_KEY environment variable is not set"
	}

	log.Infof("call web_search(%s)", arg)
	var args struct {
		Query string
	}
	if err := json.Unmarshal([]byte(arg), &args); err != nil {
		return fmt.Sprintf("Error parsing arguments for web_search: %s", err)
	}
	if strings.TrimSpace(args.Query) == "" {
		return "Error calling web_search - query is required"
	}
	resp, err := t.search(ctx, apiKey, args.Query, NumResults)
	if err != nil {
		return "Error calling web_search - " + err.Error()
	}
	if len(resp.Web.Results) == 0 {
		return fmt.Sprintf("Web search for “%s” - no results found\n", args.Query)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## Web search results for “%s”\n\n", args.Query)
	for _, r := range resp.Web.Results {
		fmt.Fprintf(&b, "  * [%s](%s)\n    %s\n\n", r.Title, r.URL, html.UnescapeString(r.Description))
	}
	return b.String()
}

type searchResponse struct {
	Web struct {
		Results []struct {
			Title, URL, Description string
		}
	}
}

type ratelimitHeaders struct {
	Limit     [2]int
	Remaining [2]int
	Reset     [2]int
}

func (t *Search) search(ctx context.Context, apiKey, query string, topn int) (resp searchResponse, err error) {
	if tm := time.Now(); tm.Before(t.nextSearch) {
		wait := t.nextSearch.Sub(tm)
		log.Infof("Brave search rate limit - wait %s", wait.Round(time.Millisecond))
		time.Sleep(wait)
	}
	uri := fmt.Sprintf(
		"https://api.search.brave.com/res/v1/web/search?q=%s&count=%d&country=%s&search_lang=%s&text_decorations=false",
		url.QueryEscape(query), topn, Country, Language,
	)
	var h http.Header
	for range MaxRetries {
		h, err = util.GetWithHeaders(ctx, uri, &resp, util.Header{Key: "X-Subscription-Token", Value: apiKey})
		var status util.HTTPError
		if errors.As(err, &status) && status.Code == 429 {
			log.Warnf("%v - retrying", err)
			time.Sleep(time.Second)
		} else {
			break
		}
	}
	if err != nil {
		return resp, err
	}
	limits := ratelimitHeaders{
		Limit:     parseHeader(h.Get("X-RateLimit-Limit")),
		Remaining: parseHeader(h.Get("X-RateLimit-Remaining")),
		Reset:     parseHeader(h.Get("X-RateLimit-Reset")),
	}
	log.Debugf("Brave search rate limits: %+v", limits)
	if limits.Remaining[0] == 0 {
		t.nextSearch = time.Now().Add(time.Duration(limits.Reset[0]) * time.Second)
	}
	if len(resp.Web.Results) == 0 {
		return resp, fmt.Errorf("no search results returned")
	}
	return resp, nil
}

func parseHeader(h string) (r [2]int) {
	if s1, s2, ok := strings.Cut(h, ","); ok {
		r[0] = atoi(s1)
		r[1] = atoi(s2)
	} else {
		log.Errorf("error parsing header: %s", h)
	}
	return
}

func atoi(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		log.Error(err)
	}
	return n
}
