package scrape

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scrape(t *testing.T, b *Browser, url string) Response {
	t.Log("scraping", url)
	r, err := b.Scrape(context.Background(), url)
	require.NoError(t, err)

	t.Logf("status: %d %s", r.Status, r.StatusText)
	t.Logf("title: %q", r.Title)
	assert.Equal(t, 200, r.Status)

	lines := strings.Split(r.Markdown, "\n")
	for i := 0; i < len(lines) && i < 20; i++ {
		t.Log(lines[i])
	}
	return r
}

func TestScrape(t *testing.T) {
	b, err := New()
	require.NoError(t, err)
	defer b.Shutdown()

	type TestDef struct {
		url      string
		title    string
		redirect string
	}
	tests := []TestDef{
		{"https://ollama.com/", "Ollama", ""},
		{"https://itsabanana.dev/", "It's a banana? No it's a blog.", ""},
		{"https://www.theguardian.com/about", "About us | The Guardian", ""},
		{"https://ollama.com/", "Ollama", ""},
		{"https://www.reuters.com/world/uk/", "UK News | Top Stories from the UK | Reuters", ""},
		{"https://www.reddit.com/r/LocalLLaMA/", "LocalLlama", ""},
		{"https://github.com/jnb666/gpt-go", "GitHub - jnb666/gpt-go: Code to interact with LLM models in go · GitHub", ""},
		{"https://www.yahoo.com/entertainment/", "Yahoo Entertainment", "https://www.yahoo.com/entertainment/?guccounter=1"},
		{"https://www.rottentomatoes.com/m/one_battle_after_another", "One Battle After Another | Rotten Tomatoes", ""},
		{"https://retrocomputing.co.uk/", "Retro Computing Grotto", "https://www.retrocomputing.co.uk/"},
		{"https://en.wikipedia.org/wiki/Liz_Truss", "Liz Truss - Wikipedia", ""},
		{"https://news.google.com/", "Google News", "https://news.google.com/home?hl=en-GB&gl=GB&ceid=GB:en"},
	}
	for _, test := range tests {
		t.Run(getHost(test.url), func(t *testing.T) {
			resp := scrape(t, b, test.url)
			assert.Equal(t, test.title, resp.Title)
			if test.redirect == "" {
				assert.Equal(t, test.url, resp.URL)
			} else {
				assert.Equal(t, test.redirect, resp.URL)
			}
		})
	}
}

func TestInvalidHost(t *testing.T) {
	b, err := New()
	require.NoError(t, err)
	defer b.Shutdown()
	_, err = b.Scrape(context.Background(), "https://itsabanana.de")
	e := new(playwright.Error)
	if errors.As(err, &e) {
		t.Log(e)
		assert.Equal(t, "NS_ERROR_UNKNOWN_HOST", e.Message)
	} else {
		t.Error("expecting playwright error")
	}
}

func TestNotFound(t *testing.T) {
	b, err := New()
	require.NoError(t, err)
	defer b.Shutdown()
	r, err := b.Scrape(context.Background(), "https://itsabanana.dev/notfound")
	require.NoError(t, err)
	t.Logf("status: %d %s", r.Status, r.StatusText)
	t.Logf("title: %q", r.Title)
	t.Logf("content:\n%s\n", r.Markdown)
	assert.Equal(t, 404, r.Status)
}
