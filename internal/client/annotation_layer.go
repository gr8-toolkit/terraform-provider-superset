// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// AnnotationLayer represents a Superset annotation layer.
type AnnotationLayer struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"descr"` // API field is "descr", not "description".
}

// CreateAnnotationLayer creates a new annotation layer via POST /api/v1/annotation_layer/.
func (c *Client) CreateAnnotationLayer(name, description string) (*AnnotationLayer, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"name":  name,
		"descr": description,
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("POST", "/api/v1/annotation_layer/", payload, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create annotation layer, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		ID     int64           `json:"id"`
		Result AnnotationLayer `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	al := result.Result
	if al.ID == 0 {
		al.ID = result.ID
	}
	return &al, nil
}

// GetAnnotationLayer retrieves an annotation layer by ID via GET /api/v1/annotation_layer/{id}.
func (c *Client) GetAnnotationLayer(id int64) (*AnnotationLayer, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/annotation_layer/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("failed to get annotation layer, status code: 404")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get annotation layer, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result AnnotationLayer `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// UpdateAnnotationLayer updates an annotation layer via PUT /api/v1/annotation_layer/{id}.
func (c *Client) UpdateAnnotationLayer(id int64, name, description string) (*AnnotationLayer, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"name":  name,
		"descr": description,
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("PUT", fmt.Sprintf("/api/v1/annotation_layer/%d", id), payload, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("failed to update annotation layer, status code: 404")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update annotation layer, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result AnnotationLayer `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result.Result, nil
}

// DeleteAnnotationLayer deletes an annotation layer via DELETE /api/v1/annotation_layer/{id}.
func (c *Client) DeleteAnnotationLayer(id int64) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("DELETE", fmt.Sprintf("/api/v1/annotation_layer/%d", id), nil, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("failed to delete annotation layer, status code: 404")
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete annotation layer, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}
