// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DashboardEmbedded holds the embedded dashboard configuration.
type DashboardEmbedded struct {
	UUID           string   `json:"uuid"`
	AllowedDomains []string `json:"allowed_domains"`
}

// GetDashboardEmbedded retrieves the embedded configuration for a dashboard.
// Returns nil, nil if the dashboard has no embedding configured.
func (c *Client) GetDashboardEmbedded(dashboardID int64) (*DashboardEmbedded, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/dashboard/%d/embedded", dashboardID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get embedded dashboard, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result DashboardEmbedded `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// CreateDashboardEmbedded creates or replaces the embedded configuration for a dashboard.
func (c *Client) CreateDashboardEmbedded(dashboardID int64, allowedDomains []string) (*DashboardEmbedded, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("POST",
		fmt.Sprintf("/api/v1/dashboard/%d/embedded", dashboardID),
		map[string]interface{}{"allowed_domains": allowedDomains}, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create embedded dashboard, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result DashboardEmbedded `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// DeleteDashboardEmbedded removes the embedded configuration from a dashboard.
func (c *Client) DeleteDashboardEmbedded(dashboardID int64) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("DELETE",
		fmt.Sprintf("/api/v1/dashboard/%d/embedded", dashboardID), nil, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete embedded dashboard, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}
