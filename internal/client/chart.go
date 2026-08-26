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

// DeleteChart deletes a chart by ID; treats 404 as success.
func (c *Client) DeleteChart(id int64) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("DELETE", fmt.Sprintf("/api/v1/chart/%d", id), nil, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete chart, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetChartIDByUUID finds a chart ID by UUID. Returns 0, nil if not found.
func (c *Client) GetChartIDByUUID(uuid string) (int64, error) {
	resp, err := c.DoRequest("GET",
		fmt.Sprintf("/api/v1/chart/?q=(filters:!((col:uuid,opr:eq,value:'%s')))", uuid), nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to fetch chart by uuid %q, status code: %d, response: %s", uuid, resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			ID float64 `json:"id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	if len(result.Result) == 0 {
		return 0, nil
	}
	return int64(result.Result[0].ID), nil
}

// GetChartDashboardCount returns the number of dashboards referencing a chart.
func (c *Client) GetChartDashboardCount(chartID int64) (int, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/chart/%d", chartID), nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to get chart %d, status code: %d, response: %s", chartID, resp.StatusCode, string(body))
	}

	var result struct {
		Result struct {
			Dashboards []struct {
				ID int64 `json:"id"`
			} `json:"dashboards"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return len(result.Result.Dashboards), nil
}
