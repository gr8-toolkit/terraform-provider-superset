// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SavedQuery represents a Superset saved query.
type SavedQuery struct {
	ID                 int64  `json:"id"`
	DatabaseID         int64  `json:"db_id"`
	DatabaseName       string `json:"database_name"` // populated from nested database.database_name
	Label              string `json:"label"`
	Description        string `json:"description"`
	Catalog            string `json:"catalog"`
	Schema             string `json:"schema"`
	SQL                string `json:"sql"`
	TemplateParameters string `json:"template_parameters"`
	ExtraJSON          string `json:"extra_json"`
}

// savedQueryResponse is the raw API response shape for a single saved query.
// The nested database object is flattened into SavedQuery.DatabaseName by UnmarshalSavedQuery.
type savedQueryResponse struct {
	ID         int64 `json:"id"`
	DatabaseID int64 `json:"db_id"`
	Database   *struct {
		ID           int64  `json:"id"`
		DatabaseName string `json:"database_name"`
	} `json:"database"`
	Label              string  `json:"label"`
	Description        *string `json:"description"`
	Catalog            *string `json:"catalog"`
	Schema             *string `json:"schema"`
	SQL                string  `json:"sql"`
	TemplateParameters *string `json:"template_parameters"`
	ExtraJSON          *string `json:"extra_json"`
}

func parseSavedQueryResponse(raw savedQueryResponse) *SavedQuery {
	sq := &SavedQuery{
		ID:         raw.ID,
		DatabaseID: raw.DatabaseID,
		Label:      raw.Label,
		SQL:        raw.SQL,
	}
	if raw.Database != nil {
		sq.DatabaseName = raw.Database.DatabaseName
		if sq.DatabaseID == 0 {
			sq.DatabaseID = raw.Database.ID
		}
	}
	// All these fields can be null from the API — treat null as empty string.
	if raw.Description != nil {
		sq.Description = *raw.Description
	}
	if raw.Catalog != nil {
		sq.Catalog = *raw.Catalog
	}
	if raw.Schema != nil {
		sq.Schema = *raw.Schema
	}
	if raw.TemplateParameters != nil {
		sq.TemplateParameters = *raw.TemplateParameters
	}
	if raw.ExtraJSON != nil {
		sq.ExtraJSON = *raw.ExtraJSON
	}
	return sq
}

// CreateSavedQuery creates a new saved query via POST /api/v1/saved_query/.
func (c *Client) CreateSavedQuery(payload map[string]interface{}) (*SavedQuery, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("POST", "/api/v1/saved_query/", payload, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create saved query, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID     int64              `json:"id"`
		Result savedQueryResponse `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	sq := parseSavedQueryResponse(result.Result)
	if sq.ID == 0 {
		sq.ID = result.ID
	}
	return sq, nil
}

// GetSavedQuery retrieves a saved query by ID via GET /api/v1/saved_query/{id}.
func (c *Client) GetSavedQuery(id int64) (*SavedQuery, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/saved_query/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("failed to get saved query, status code: 404")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get saved query, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result savedQueryResponse `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return parseSavedQueryResponse(result.Result), nil
}

// UpdateSavedQuery updates a saved query via PUT /api/v1/saved_query/{id}.
func (c *Client) UpdateSavedQuery(id int64, payload map[string]interface{}) (*SavedQuery, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("PUT", fmt.Sprintf("/api/v1/saved_query/%d", id), payload, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("failed to update saved query, status code: 404")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update saved query, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result savedQueryResponse `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return parseSavedQueryResponse(result.Result), nil
}

// DeleteSavedQuery deletes a saved query by ID via DELETE /api/v1/saved_query/{id}.
func (c *Client) DeleteSavedQuery(id int64) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("DELETE", fmt.Sprintf("/api/v1/saved_query/%d", id), nil, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("failed to delete saved query, status code: 404")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete saved query, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}
