// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GetDatabaseSchemasByID lists schemas inside a database.
func (c *Client) GetDatabaseSchemasByID(databaseID int64) ([]string, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/database/%d/schemas/", databaseID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch schemas from Superset, status code: %d", resp.StatusCode)
	}

	var result struct {
		Result []string `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Result, nil
}

// GetDatabaseConnectionByID returns the full connection record for a database.
func (c *Client) GetDatabaseConnectionByID(databaseID int64) (map[string]interface{}, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/database/%d/connection", databaseID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch database connection from Superset, status code: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetAllDatabases returns all databases; results are cached for 5 minutes.
func (c *Client) GetAllDatabases() ([]map[string]interface{}, error) {
	globalDatabasesCacheMutex.RLock()
	if len(globalDatabasesCache) > 0 && time.Since(globalDatabasesCacheTime) < globalDatabasesCacheTTL {
		fmt.Printf("DEBUG GetAllDatabases: Using global cached result with %d databases (age: %v)\n",
			len(globalDatabasesCache), time.Since(globalDatabasesCacheTime))
		result := globalDatabasesCache
		globalDatabasesCacheMutex.RUnlock()
		return result, nil
	}
	globalDatabasesCacheMutex.RUnlock()

	globalDatabasesCacheMutex.Lock()
	defer globalDatabasesCacheMutex.Unlock()

	if len(globalDatabasesCache) > 0 && time.Since(globalDatabasesCacheTime) < globalDatabasesCacheTTL {
		fmt.Printf("DEBUG GetAllDatabases: Using global cached result (double-check) with %d databases\n", len(globalDatabasesCache))
		return globalDatabasesCache, nil
	}

	endpoint := "/api/v1/database/?q=(page_size:5000)"
	fmt.Printf("DEBUG GetAllDatabases: Making API call to %s\n", endpoint)
	resp, err := c.DoRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch databases from Superset, status code: %d", resp.StatusCode)
	}

	var result struct {
		Result []map[string]interface{} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	globalDatabasesCache = result.Result
	globalDatabasesCacheTime = time.Now()
	fmt.Printf("DEBUG GetAllDatabases: Retrieved and cached globally %d databases total\n", len(result.Result))
	return result.Result, nil
}

// GetDatabasesInfos returns a summary of up to 100 databases with their schemas and URIs.
func (c *Client) GetDatabasesInfos() (map[string]interface{}, error) {
	dbs, err := c.GetAllDatabases()
	if err != nil {
		return nil, err
	}

	limit := 100
	if len(dbs) < limit {
		limit = len(dbs)
	}

	list := make([]map[string]interface{}, 0, limit)
	for _, db := range dbs[:limit] {
		dbID, ok := db["id"].(float64)
		if !ok {
			continue
		}
		details, err := c.GetDatabaseConnectionByID(int64(dbID))
		if err != nil {
			return nil, err
		}

		var uri, name string
		if res, ok := details["result"].(map[string]interface{}); ok {
			uri, _ = res["sqlalchemy_uri"].(string)
			name, _ = res["database_name"].(string)
		}
		if uri == "" {
			uri = "URI not provided"
		}
		if name == "" {
			name = "Name not provided"
		}

		schemas, err := c.GetDatabaseSchemasByID(int64(dbID))
		if err != nil {
			return nil, err
		}

		list = append(list, map[string]interface{}{
			"id":             int64(dbID),
			"database_name":  name,
			"schemas":        schemas,
			"sqlalchemy_uri": uri,
		})
	}
	return map[string]interface{}{"databases": list}, nil
}

// CreateDatabase creates a new database connection.
func (c *Client) CreateDatabase(payload map[string]interface{}) (map[string]interface{}, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("POST", "/api/v1/database/", payload, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create database, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateDatabase updates a database connection.
func (c *Client) UpdateDatabase(databaseID int64, payload map[string]interface{}) (map[string]interface{}, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("PUT", fmt.Sprintf("/api/v1/database/%d", databaseID), payload, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update database, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteDatabase deletes a database connection.
func (c *Client) DeleteDatabase(databaseID int64) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("DELETE", fmt.Sprintf("/api/v1/database/%d", databaseID), nil, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete database, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetDatabaseIDByName resolves a database name to its numeric ID using the cache.
func (c *Client) GetDatabaseIDByName(databaseName string) (int64, error) {
	dbs, err := c.GetAllDatabases()
	if err != nil {
		return 0, fmt.Errorf("failed to fetch databases: %w", err)
	}
	for _, db := range dbs {
		if name, ok := db["database_name"].(string); ok && name == databaseName {
			if id, ok := db["id"].(float64); ok {
				return int64(id), nil
			}
		}
	}
	return 0, fmt.Errorf("database with name '%s' not found", databaseName)
}

// GetDatabaseNameByID resolves a numeric database ID to its name using the cache.
func (c *Client) GetDatabaseNameByID(databaseID int64) (string, error) {
	dbs, err := c.GetAllDatabases()
	if err != nil {
		return "", fmt.Errorf("failed to fetch databases: %w", err)
	}
	for _, db := range dbs {
		if id, ok := db["id"].(float64); ok && int64(id) == databaseID {
			if name, ok := db["database_name"].(string); ok {
				return name, nil
			}
		}
	}
	return "", fmt.Errorf("database with ID %d not found", databaseID)
}
