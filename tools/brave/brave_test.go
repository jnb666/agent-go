package brave

import (
	"context"
	"regexp"
	"testing"

	"github.com/jnb666/agent-go/util"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var pattern = regexp.MustCompile(`(?m)^  \* \[`)

func TestSearch(t *testing.T) {
	tool := &Search{}
	t.Log(util.Pretty(tool.Definition()))

	logrus.SetLevel(logrus.DebugLevel)
	resp := tool.Call(context.Background(), `{"query": "foo bar"}`)
	t.Logf("\n%s", resp)

	assert.Equal(t, 10, len(pattern.FindAllString(resp, -1)))
}

func TestRatelimit(t *testing.T) {
	tool := &Search{}
	for _, query := range []string{"foo", "bar"} {
		resp := tool.Call(context.Background(), `{"query": "`+query+`"}`)
		t.Logf("\n%s", resp)

		assert.Equal(t, 10, len(pattern.FindAllString(resp, -1)))
	}
}
