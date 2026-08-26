// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// MetaDatabase represents a Superset meta-database (superset:// dialect).
type MetaDatabase struct {
	ID                  int64   `json:"id"`
	DatabaseName        string  `json:"database_name"`
	Engine              string  `json:"engine"`
	ConfigurationMethod string  `json:"configuration_method"`
	SqlalchemyURI       string  `json:"sqlalchemy_uri"`
	ExposeInSqllab      bool    `json:"expose_in_sqllab"`
	AllowCtas           bool    `json:"allow_ctas"`
	AllowCvas           bool    `json:"allow_cvas"`
	AllowDml            bool    `json:"allow_dml"`
	AllowRunAsync       bool    `json:"allow_run_async"`
	Extra               string  `json:"extra"`
	ServerCert          *string `json:"server_cert"`
	IsManagedExternally bool    `json:"is_managed_externally"`
	ExternalURL         *string `json:"external_url"`
	// AllowedDBs is populated from extra.engine_params.allowed_dbs; not a direct API field.
	AllowedDBs []string `json:"-"`
}

func buildMetaDatabasePayload(metaDB *MetaDatabase) (map[string]interface{}, error) {
	extraData := map[string]interface{}{
		"metadata_params": map[string]interface{}{},
		"engine_params": map[string]interface{}{
			"allowed_dbs": metaDB.AllowedDBs,
		},
		"metadata_cache_timeout":         map[string]interface{}{},
		"schemas_allowed_for_csv_upload": []string{},
	}
	extraJSON, err := json.Marshal(extraData)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"database_name":         metaDB.DatabaseName,
		"engine":                metaDB.Engine,
		"configuration_method":  metaDB.ConfigurationMethod,
		"sqlalchemy_uri":        metaDB.SqlalchemyURI,
		"expose_in_sqllab":      metaDB.ExposeInSqllab,
		"allow_ctas":            metaDB.AllowCtas,
		"allow_cvas":            metaDB.AllowCvas,
		"allow_dml":             metaDB.AllowDml,
		"allow_run_async":       metaDB.AllowRunAsync,
		"extra":                 string(extraJSON),
		"server_cert":           metaDB.ServerCert,
		"is_managed_externally": metaDB.IsManagedExternally,
		"external_url":          metaDB.ExternalURL,
	}, nil
}

// CreateMetaDatabase creates a meta database and returns its numeric ID.
func (c *Client) CreateMetaDatabase(metaDB *MetaDatabase) (int64, error) {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return 0, err
	}
	payload, err := buildMetaDatabasePayload(metaDB)
	if err != nil {
		return 0, err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("POST", "/api/v1/database/", payload, headers, cookies)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to create meta database, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	id, ok := result["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("failed to retrieve meta database ID from response")
	}
	return int64(id), nil
}

// GetMetaDatabase retrieves a meta database by ID.
// Uses the /connection endpoint which returns the full extra field.
func (c *Client) GetMetaDatabase(id int64) (*MetaDatabase, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/database/%d/connection", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch meta database, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Result MetaDatabase `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	metaDB := &result.Result

	// Parse allowed_dbs from the extra JSON blob.
	if metaDB.Extra != "" {
		var extraData map[string]interface{}
		if err := json.Unmarshal([]byte(metaDB.Extra), &extraData); err == nil {
			if ep, ok := extraData["engine_params"].(map[string]interface{}); ok {
				if dbs, ok := ep["allowed_dbs"].([]interface{}); ok {
					metaDB.AllowedDBs = make([]string, 0, len(dbs))
					for _, db := range dbs {
						if s, ok := db.(string); ok {
							metaDB.AllowedDBs = append(metaDB.AllowedDBs, s)
						}
					}
				}
			}
		}
	}
	return metaDB, nil
}

// UpdateMetaDatabase updates a meta database.
func (c *Client) UpdateMetaDatabase(id int64, metaDB *MetaDatabase) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}
	payload, err := buildMetaDatabasePayload(metaDB)
	if err != nil {
		return err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("PUT", fmt.Sprintf("/api/v1/database/%d", id), payload, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update meta database, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

// DeleteMetaDatabase deletes a meta database.
func (c *Client) DeleteMetaDatabase(id int64) error {
	return c.DeleteDatabase(id)
}

// FindMetaDatabaseByName finds a meta database by name (sqlalchemy_uri == "superset://").
// Returns nil, nil if not found.
func (c *Client) FindMetaDatabaseByName(databaseName string) (*MetaDatabase, error) {
	dbs, err := c.GetAllDatabases()
	if err != nil {
		return nil, err
	}
	for _, db := range dbs {
		name, _ := db["database_name"].(string)
		uri, _ := db["sqlalchemy_uri"].(string)
		if name == databaseName && uri == "superset://" {
			if id, ok := db["id"].(float64); ok {
				return c.GetMetaDatabase(int64(id))
			}
		}
	}
	return nil, nil
}
