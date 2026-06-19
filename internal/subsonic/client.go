package subsonic

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	baseURL  string
	user     string
	pass     string
	salt     string
	token    string
	client   string
	apiVer  string
	http     *http.Client
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
		salt:   salt,
		token:  token,
		client: "minitone",
		apiVer: "1.16.1",
		http:   &http.Client{Timeout: 15 * time.Second},
	}
}

func (c *Client) NewFromEnv() *Client {
	server := os.Getenv("NAVIDROME_URL")
	user := os.Getenv("NAVIDROME_USER")
	pass := os.Getenv("NAVIDROME_PASS")
	if server == "" || user == "" || pass == "" {
		return nil
	}
	return NewClient(server, user, pass)
}

func (c *Client) get(endpoint string, params map[string]string) (map[string]any, error) {
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

	resp, err := c.http.Get(u.String())
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

// Ping tests the connection
func (c *Client) Ping() error {
	_, err := c.get("ping", nil)
	return err
}

// Scrobble sends a now-playing notification
func (c *Client) Scrobble(id string, submission bool) error {
	_, err := c.get("scrobble", map[string]string{
		"id": id,
		"submission": strconv.FormatBool(submission),
	})
	return err
}

// StreamURL returns the full stream URL for a song ID
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
