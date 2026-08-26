// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// RowLevelSecurity represents a Superset row-level security rule.
type RowLevelSecurity struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Tables      []int64 `json:"tables"`
	Clause      string  `json:"clause"`
	RoleIDs     []int64 `json:"roles"`
	GroupKey    string  `json:"group_key"`
	FilterType  string  `json:"filter_type"`
	Description string  `json:"description"`
}

func buildRLSPayload(name string, tables []int64, clause string, roleIDs []int64, groupKey, filterType, description string) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"tables":      tables,
		"clause":      clause,
		"roles":       roleIDs,
		"group_key":   groupKey,
		"filter_type": filterType,
		"description": description,
	}
}

// CreateRowLevelSecurity creates a new RLS rule.
func (c *Client) CreateRowLevelSecurity(name string, tables []int64, clause string, roleIDs []int64, groupKey, filterType, description string) (int64, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return 0, err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("POST", "/api/v1/rowlevelsecurity/",
		buildRLSPayload(name, tables, clause, roleIDs, groupKey, filterType, description), headers, cookies)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to create RLS rule, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	id, ok := result["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("failed to retrieve RLS rule ID from response")
	}
	return int64(id), nil
}

// GetRowLevelSecurity retrieves an RLS rule by ID.
func (c *Client) GetRowLevelSecurity(id int64) (*RowLevelSecurity, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/rowlevelsecurity/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch RLS rule, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result struct {
			ID     int64  `json:"id"`
			Name   string `json:"name"`
			Tables []struct {
				ID int64 `json:"id"`
			} `json:"tables"`
			Clause      string `json:"clause"`
			GroupKey    string `json:"group_key"`
			FilterType  string `json:"filter_type"`
			Description string `json:"description"`
			Roles       []struct {
				ID int64 `json:"id"`
			} `json:"roles"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	r := result.Result
	roleIDs := make([]int64, len(r.Roles))
	for i, role := range r.Roles {
		roleIDs[i] = role.ID
	}
	tableIDs := make([]int64, len(r.Tables))
	for i, t := range r.Tables {
		tableIDs[i] = t.ID
	}

	return &RowLevelSecurity{
		ID:          r.ID,
		Name:        r.Name,
		Tables:      tableIDs,
		Clause:      r.Clause,
		GroupKey:    r.GroupKey,
		FilterType:  r.FilterType,
		Description: r.Description,
		RoleIDs:     roleIDs,
	}, nil
}

// UpdateRowLevelSecurity updates an existing RLS rule.
func (c *Client) UpdateRowLevelSecurity(id int64, name string, tables []int64, clause string, roleIDs []int64, groupKey, filterType, description string) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("PUT", fmt.Sprintf("/api/v1/rowlevelsecurity/%d", id),
		buildRLSPayload(name, tables, clause, roleIDs, groupKey, filterType, description), headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update RLS rule, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

// DeleteRowLevelSecurity deletes an RLS rule by ID.
func (c *Client) DeleteRowLevelSecurity(id int64) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("DELETE", fmt.Sprintf("/api/v1/rowlevelsecurity/%d", id), nil, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete RLS rule, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}
