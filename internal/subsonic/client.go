package subsonic

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	user    string
	pass    string
	salt    string
	token   string
	client  string
	apiVer  string
	http    *http.Client
}

func NewClient(serverURL, user, pass string) *Client {
	salt := fmt.Sprintf("%d", time.Now().UnixNano())
	hash := md5.Sum([]byte(pass + salt))
	token := hex.EncodeToString(hash[:])
	base := strings.TrimRight(serverURL, "/")
	if !strings.HasSuffix(base, "/rest") {
		base += "/rest"
	}
	return &Client{
		baseURL: base,
		user:    user,
		pass:    pass,
		salt:    salt,
		token:   token,
		client:  "minitone",
		apiVer:  "1.16.1",
		http:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) get(endpoint string, params map[string]string) (map[string]any, error) {
	return c.getContext(context.Background(), endpoint, params)
}

// getContext is get with cancellation, so a superseded search stops hitting
// the server immediately instead of waiting for the client timeout.
func (c *Client) getContext(ctx context.Context, endpoint string, params map[string]string) (map[string]any, error) {
	u, _ := url.Parse(c.baseURL + "/" + endpoint + ".view")
	q := u.Query()
	q.Set("u", c.user)
	q.Set("t", c.token)
	q.Set("s", c.salt)
	q.Set("v", c.apiVer)
	q.Set("c", c.client)
	q.Set("f", "json")
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("bad request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var root map[string]any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}
	sr, _ := root["subsonic-response"].(map[string]any)
	if sr == nil {
		return nil, fmt.Errorf("invalid response")
	}
	if status, _ := sr["status"].(string); status != "ok" {
		errMsg, _ := sr["error"].(map[string]any)
		if errMsg != nil {
			msg, _ := errMsg["message"].(string)
			return nil, fmt.Errorf("API error: %s", msg)
		}
		return nil, fmt.Errorf("API error: unknown")
	}
	return sr, nil
}

func (c *Client) Ping() error {
	_, err := c.get("ping", nil)
	return err
}

// PingContext is Ping with a caller-supplied deadline.
func (c *Client) PingContext(ctx context.Context) error {
	_, err := c.getContext(ctx, "ping", nil)
	return err
}

// GetContext is the cancellable variant of get, used by searches.
func (c *Client) GetContext(ctx context.Context, endpoint string, params map[string]string) (map[string]any, error) {
	return c.getContext(ctx, endpoint, params)
}

func (c *Client) StreamURL(id string) string {
	u, _ := url.Parse(c.baseURL + "/stream.view")
	q := u.Query()
	q.Set("u", c.user)
	q.Set("t", c.token)
	q.Set("s", c.salt)
	q.Set("v", c.apiVer)
	q.Set("c", c.client)
	q.Set("id", id)
	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Client) Scrobble(id string, submission bool) error {
	_, err := c.get("scrobble", map[string]string{
		"id":         id,
		"submission": strconv.FormatBool(submission),
	})
	return err
}
