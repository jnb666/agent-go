package browser

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/jnb666/agent-go/scrape"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrowserOpen(t *testing.T) {
	browser, err := scrape.New()
	require.NoError(t, err)
	defer browser.Shutdown()

	resp := Open{browser: browser}.Call(context.Background(), marshal(map[string]any{
		"url": "https://itsabanana.dev/posts/local_llm_hosting-part1/",
	}))
	t.Logf("response:\n%s", resp)

	expect := `## Local LLM models: Part 1 - getting started - It's a banana? No it's a blog.
(https://itsabanana.dev/posts/local_llm_hosting-part1/)
**viewing lines [1 - 25] of 86**
`
	assert.Contains(t, resp, expect)
}

func TestBrowserScroll(t *testing.T) {
	browser, err := scrape.New()
	require.NoError(t, err)
	defer browser.Shutdown()

	resp := Open{browser: browser}.Call(context.Background(), marshal(map[string]any{
		"url":  "https://itsabanana.dev/posts/local_llm_hosting-part1/",
		"line": 50,
	}))
	t.Logf("response:\n%s", resp)

	expect := `## Local LLM models: Part 1 - getting started - It's a banana? No it's a blog.
(https://itsabanana.dev/posts/local_llm_hosting-part1/)
**viewing lines [50 - 74] of 86**
`
	assert.Contains(t, resp, expect)
}

func TestBrowserFind(t *testing.T) {
	browser, err := scrape.New()
	require.NoError(t, err)
	defer browser.Shutdown()

	resp := Open{browser: browser}.Call(context.Background(), marshal(map[string]any{
		"url": "https://itsabanana.dev/posts/local_llm_hosting-part3/",
	}))

	resp = Find{browser: browser}.Call(context.Background(), marshal(map[string]any{
		"pattern": "OWM_API_KEY",
	}))
	t.Logf("response:\n%s", resp)

	expect := `## Local LLM models: Part 3 - calling tool functions - It's a banana? No it's a blog.
(https://itsabanana.dev/posts/local_llm_hosting-part3/) find results for “OWM_API_KEY”
**viewing lines [138 - 162] of 319**
`
	assert.Contains(t, resp, expect)
}

func marshal(args any) string {
	data, _ := json.Marshal(args)
	return string(data)
}
