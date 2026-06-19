package cider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

func NewFromEnv() *Client {
	base := strings.TrimRight(os.Getenv("CIDER_API_BASE"), "/")
	if base == "" {
		base = "http://localhost:10767"
	}
	return New(base, strings.TrimSpace(os.Getenv("CIDER_API_TOKEN")))
}

func New(baseURL, token string) *Client {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "http://localhost:10767"
	}
	return &Client{
		baseURL: base,
		token:   strings.TrimSpace(token),
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) doGET(path string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, ErrNotRunning
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if err := c.checkStatus(resp.StatusCode, body); err != nil {
		return nil, err
	}
	return body, nil
}

func (c *Client) doPOST(path string, payload any) error {
	var body []byte
	if payload != nil {
		var err error
		body, err = json.Marshal(payload)
		if err != nil {
			return err
		}
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.setHeaders(req)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return ErrNotRunning
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return c.checkStatus(resp.StatusCode, respBody)
}

func (c *Client) runV3(apiPath string) (map[string]any, error) {
	payload := map[string]string{"path": apiPath}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/api/v1/amapi/run-v3", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, ErrNotRunning
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if err := c.checkStatus(resp.StatusCode, respBody); err != nil {
		return nil, err
	}

	var root map[string]any
	if err := json.Unmarshal(respBody, &root); err != nil {
		return nil, fmt.Errorf("invalid response: %w", err)
	}
	return root, nil
}

func (c *Client) setHeaders(req *http.Request) {
	if c.token != "" {
		req.Header.Set("apitoken", c.token)
	}
	req.Header.Set("Accept", "application/json")
}

func (c *Client) checkStatus(code int, body []byte) error {
	switch {
	case code == http.StatusForbidden:
		return ErrUnauthorized
	case code == http.StatusNotFound:
		return ErrNotFound
	case code >= 300:
		return fmt.Errorf("HTTP %d: %s", code, strings.TrimSpace(string(body)))
	}
	return nil
}

func extractString(m map[string]any, path ...string) string {
	var cur any = m
	for _, p := range path {
		switch node := cur.(type) {
		case map[string]any:
			cur = node[p]
		default:
			return ""
		}
	}
	s, _ := cur.(string)
	return s
}

func extractInt64(m map[string]any, path ...string) int64 {
	var cur any = m
	for _, p := range path {
		switch node := cur.(type) {
		case map[string]any:
			cur = node[p]
		default:
			return 0
		}
	}
	switch n := cur.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	}
	return 0
}

func ciderStorefront() string {
	sf := strings.TrimSpace(os.Getenv("CIDER_STOREFRONT"))
	if sf == "" {
		sf = "us"
	}
	return sf
}
