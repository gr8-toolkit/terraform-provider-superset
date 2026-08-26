// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// Chart represents a Superset chart.
type Chart struct {
	ID             int64  `json:"id"`
	UUID           string `json:"uuid"`
	SliceName      string `json:"slice_name"`
	Description    string `json:"description"`
	VizType        string `json:"viz_type"`
	DatasourceID   int64  `json:"datasource_id"`
	DatasourceType string `json:"datasource_type"`
	DatasourceName string `json:"datasource_name_text"`
	Params         string `json:"params"`
	QueryContext   string `json:"query_context"`
	CacheTimeout   *int64 `json:"cache_timeout"`
	URL            string `json:"url"`
}

// CreateChart creates a new chart via POST /api/v1/chart/.
func (c *Client) CreateChart(payload map[string]interface{}) (*Chart, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("POST", "/api/v1/chart/", payload, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create chart, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID     int64 `json:"id"`
		Result Chart `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	chart := result.Result
	if chart.ID == 0 {
		chart.ID = result.ID
	}
	return &chart, nil
}

// GetChart retrieves a chart by ID via GET /api/v1/chart/{id}.
func (c *Client) GetChart(id int64) (*Chart, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/chart/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("failed to get chart, status code: 404")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get chart, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result Chart `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// UpdateChart updates an existing chart via PUT /api/v1/chart/{id}.
func (c *Client) UpdateChart(id int64, payload map[string]interface{}) (*Chart, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("PUT", fmt.Sprintf("/api/v1/chart/%d", id), payload, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("failed to update chart, status code: 404")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update chart, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result Chart `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// DeleteChart deletes a chart by ID via DELETE /api/v1/chart/{id}.
// Note: this method is defined in internal/client/superset.go and shared
// with the chart_import resource. It is not redeclared here.
