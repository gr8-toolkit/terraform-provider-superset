// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Global database cache — shared across all Client instances in the same process.
var (
	globalDatabasesCache      []map[string]interface{}
	globalDatabasesCacheTime  time.Time
	globalDatabasesCacheTTL   = 5 * time.Minute
	globalDatabasesCacheMutex sync.RWMutex
)

// ClearGlobalDatabaseCache clears the global database cache.
// Call this in tests that exercise database-related code.
func ClearGlobalDatabaseCache() {
	globalDatabasesCacheMutex.Lock()
	defer globalDatabasesCacheMutex.Unlock()
	globalDatabasesCache = nil
	globalDatabasesCacheTime = time.Time{}
	fmt.Printf("DEBUG ClearGlobalDatabaseCache: Global database cache cleared\n")
}

// Client holds the connection details and auth token for the Superset API.
type Client struct {
	Host     string
	Username string
	Password string
	Provider string
	Token    string
	Cookies  []*http.Cookie
}

// NewClient creates and authenticates a new Superset client.
func NewClient(host, username, password, provider string) (*Client, error) {
	c := &Client{
		Host:     host,
		Username: username,
		Password: password,
		Provider: provider,
	}
	if err := c.authenticate(); err != nil {
		return nil, err
	}
	return c, nil
}

// authenticate obtains a JWT from Superset and stores it on the client.
func (c *Client) authenticate() error {
	url := fmt.Sprintf("%s/api/v1/security/login", c.Host)
	payload := map[string]string{
		"username": c.Username,
		"password": c.Password,
		"provider": c.Provider,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to authenticate with Superset, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return err
	}

	token, ok := result["access_token"].(string)
	if !ok {
		return fmt.Errorf("failed to retrieve access token from response")
	}

	c.Token = token
	c.Cookies = resp.Cookies()
	return nil
}

// DoRequest sends a JSON request with the Bearer token.
func (c *Client) DoRequest(method, endpoint string, payload interface{}) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.Host, endpoint)
	var jsonPayload []byte
	var err error
	if payload != nil {
		jsonPayload, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	return (&http.Client{}).Do(req)
}

// DoRequestWithHeadersAndCookies sends a JSON request with custom headers and cookies.
func (c *Client) DoRequestWithHeadersAndCookies(method, endpoint string, payload interface{}, headers map[string]string, cookies []*http.Cookie) (*http.Response, error) {
	url := fmt.Sprintf("%s%s", c.Host, endpoint)
	var jsonPayload []byte
	var err error
	if payload != nil {
		jsonPayload, err = json.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	return (&http.Client{}).Do(req)
}

// GetCSRFToken fetches a fresh CSRF token and the associated cookies.
func (c *Client) GetCSRFToken() (string, []*http.Cookie, error) {
	resp, err := c.DoRequestWithHeadersAndCookies("GET", "/api/v1/security/csrf_token/", nil,
		map[string]string{"Referer": c.Host}, nil)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("failed to get CSRF token, status code: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", nil, err
	}

	token, ok := result["result"].(string)
	if !ok {
		return "", nil, fmt.Errorf("failed to retrieve CSRF token from response")
	}
	return token, resp.Cookies(), nil
}
