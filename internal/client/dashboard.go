// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Dashboard represents a Superset dashboard.
type Dashboard struct {
	ID             int64  `json:"id"`
	UUID           string `json:"uuid"`
	DashboardTitle string `json:"dashboard_title"`
	Slug           string `json:"slug"`
	CSS            string `json:"css"`
	Published      bool   `json:"published"`
	PositionJSON   string `json:"position_json"`
	JSONMetadata   string `json:"json_metadata"`
	URL            string `json:"url"`
}

// CreateDashboard creates a new dashboard via POST /api/v1/dashboard/.
func (c *Client) CreateDashboard(payload map[string]interface{}) (*Dashboard, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("POST", "/api/v1/dashboard/", payload, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create dashboard, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID     int64     `json:"id"`
		Result Dashboard `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	dash := result.Result
	if dash.ID == 0 {
		dash.ID = result.ID
	}
	return &dash, nil
}

// GetDashboard retrieves a dashboard by numeric ID via GET /api/v1/dashboard/{id}.
func (c *Client) GetDashboard(id int64) (*Dashboard, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/dashboard/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("failed to get dashboard, status code: 404")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get dashboard, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result Dashboard `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// UpdateDashboard updates a dashboard via PUT /api/v1/dashboard/{id}.
func (c *Client) UpdateDashboard(id int64, payload map[string]interface{}) (*Dashboard, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("PUT", fmt.Sprintf("/api/v1/dashboard/%d", id), payload, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("failed to update dashboard, status code: 404")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update dashboard, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result Dashboard `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// DeleteDashboardByID deletes a dashboard by numeric ID via DELETE /api/v1/dashboard/{id}.
// This is named DeleteDashboardByID to avoid collision with the existing DeleteDashboard(id)
// in superset.go which is used by the import resource.
func (c *Client) DeleteDashboardByID(id int64) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("DELETE", fmt.Sprintf("/api/v1/dashboard/%d", id), nil, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("failed to delete dashboard, status code: 404")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete dashboard, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}
