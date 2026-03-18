package util

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	Client    = http.Client{Timeout: 30 * time.Second}
	UserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:148.0) Gecko/20100101 Firefox/148.0"
)

type HTTPError struct {
	Code   int
	Status string
}

func (e HTTPError) Error() string {
	return "HTTP error: " + e.Status
}

type Header struct {
	Key   string
	Value string
}

// HTTP get request which returns raw page content
func GetData(ctx context.Context, uri string, headers ...Header) (data []byte, err error) {
	log.Debugf("GET %s", uri)
	req, _ := http.NewRequestWithContext(ctx, "GET", uri, nil)
	for _, h := range headers {
		req.Header.Set(h.Key, h.Value)
	}
	data, _, err = do(req)
	return
}

// HTTP get request for uri with optional headers. Unmarshals JSON response into reply.
func Get(ctx context.Context, uri string, reply any, headers ...Header) error {
	_, err := GetWithHeaders(ctx, uri, reply, headers...)
	return err
}

// HTTP get also returning response headers
func GetWithHeaders(ctx context.Context, uri string, reply any, headers ...Header) (http.Header, error) {
	log.Debugf("GET %s", uri)
	req, _ := http.NewRequestWithContext(ctx, "GET", uri, nil)
	for _, h := range headers {
		req.Header.Set(h.Key, h.Value)
	}
	data, respHeaders, err := do(req)
	if err != nil {
		return respHeaders, err
	}
	err = json.Unmarshal(data, reply)
	return respHeaders, err
}

// HTTP post request for uri with JSON request and optional headers. Unmarshals JSON response into reply.
func Post(ctx context.Context, uri string, request, reply any, headers ...Header) error {
	log.Debugf("POST %s", uri)
	body, err := json.Marshal(request)
	if err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", uri, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for _, h := range headers {
		req.Header.Set(h.Key, h.Value)
	}
	data, _, err := do(req)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, reply)
}

func do(req *http.Request) ([]byte, http.Header, error) {
	resp, err := Client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, nil, HTTPError{Code: resp.StatusCode, Status: resp.Status}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return data, resp.Header, err
}

type ParamObject interface {
	ExtraFields() map[string]any
	SetExtraFields(map[string]any)
}

func SetExtraField(p ParamObject, key string, val any) {
	extra := p.ExtraFields()
	if extra == nil {
		extra = map[string]any{key: val}
	} else {
		extra[key] = val
	}
	p.SetExtraFields(extra)
}
