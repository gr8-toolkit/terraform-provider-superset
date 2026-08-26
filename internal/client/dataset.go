// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// DatasetRequest is the payload for creating a dataset.
type DatasetRequest struct {
	TableName string `json:"table_name"`
	Database  int64  `json:"database"`
	Schema    string `json:"schema,omitempty"`
	SQL       string `json:"sql,omitempty"`
}

// DatasetUpdateRequest is the payload for updating a dataset.
// The database field cannot be changed after creation.
type DatasetUpdateRequest struct {
	TableName string `json:"table_name"`
	Schema    string `json:"schema,omitempty"`
	SQL       string `json:"sql,omitempty"`
}

// GetAllDatasets returns all datasets.
func (c *Client) GetAllDatasets() ([]map[string]interface{}, error) {
	resp, err := c.DoRequest("GET", "/api/v1/dataset/?q=(page_size:5000)", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch datasets from Superset, status code: %d", resp.StatusCode)
	}

	var result struct {
		Result []map[string]interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Result, nil
}

// CreateDataset creates a new dataset.
func (c *Client) CreateDataset(dataset DatasetRequest) (*map[string]interface{}, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	fmt.Printf("DEBUG CreateDataset: Sending request to /api/v1/dataset/ with payload: %+v\n", dataset)

	resp, err := c.DoRequestWithHeadersAndCookies("POST", "/api/v1/dataset/", dataset, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create dataset, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetDataset retrieves a dataset by ID.
func (c *Client) GetDataset(id int64) (*map[string]interface{}, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/dataset/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("dataset with ID %d not found", id)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch dataset, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result map[string]interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// UpdateDataset updates an existing dataset (database cannot be changed).
func (c *Client) UpdateDataset(id int64, tableName, schema, sql string) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	payload := DatasetUpdateRequest{TableName: tableName, Schema: schema, SQL: sql}
	fmt.Printf("DEBUG UpdateDataset: Sending UPDATE request to /api/v1/dataset/%d with payload: %+v\n", id, payload)

	resp, err := c.DoRequestWithHeadersAndCookies("PUT", fmt.Sprintf("/api/v1/dataset/%d", id), payload, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update dataset, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

// DeleteDataset deletes a dataset; treats 404 as success.
func (c *Client) DeleteDataset(id int64) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("DELETE", fmt.Sprintf("/api/v1/dataset/%d", id), nil, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete dataset, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetDatasetIDByUUID finds a dataset ID by UUID. Returns 0, nil if not found.
func (c *Client) GetDatasetIDByUUID(uuid string) (int64, error) {
	resp, err := c.DoRequest("GET",
		fmt.Sprintf("/api/v1/dataset/?q=(filters:!((col:uuid,opr:eq,value:'%s')))", uuid), nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to fetch dataset by uuid %q, status code: %d, response: %s", uuid, resp.StatusCode, string(body))
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

// GetDatasetChartCount returns the number of charts backed by a dataset.
func (c *Client) GetDatasetChartCount(datasetID int64) (int, error) {
	resp, err := c.DoRequest("GET",
		fmt.Sprintf("/api/v1/chart/?q=(filters:!((col:datasource_id,opr:eq,value:%d)))", datasetID), nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to get charts for dataset %d, status code: %d, response: %s", datasetID, resp.StatusCode, string(body))
	}

	var result struct {
		Count int `json:"count"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	return result.Count, nil
}
