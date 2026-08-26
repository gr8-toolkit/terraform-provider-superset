// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

const metaDBConnectionResponse = `{
	"result": {
		"id": 50,
		"database_name": "Superset Meta",
		"engine": "superset",
		"configuration_method": "sqlalchemy_form",
		"sqlalchemy_uri": "superset://",
		"expose_in_sqllab": true,
		"allow_ctas": false,
		"allow_cvas": false,
		"allow_dml": false,
		"allow_run_async": true,
		"is_managed_externally": false,
		"extra": "{\"metadata_params\":{},\"engine_params\":{\"allowed_dbs\":[\"db1\",\"db2\"]},\"metadata_cache_timeout\":{},\"schemas_allowed_for_csv_upload\":[]}"
	}
}`

func TestCreateMetaDatabase(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/database/",
		httpmock.NewStringResponder(201, `{"id": 50}`))

	metaDB := &MetaDatabase{
		DatabaseName:        "Superset Meta",
		Engine:              "superset",
		ConfigurationMethod: "sqlalchemy_form",
		SqlalchemyURI:       "superset://",
		ExposeInSqllab:      true,
		AllowRunAsync:       true,
		AllowedDBs:          []string{"db1", "db2"},
	}

	id, err := c.CreateMetaDatabase(metaDB)
	assert.NoError(t, err)
	assert.Equal(t, int64(50), id)
}

func TestCreateMetaDatabase_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/database/",
		httpmock.NewStringResponder(400, `{"message": "bad request"}`))

	_, err := c.CreateMetaDatabase(&MetaDatabase{DatabaseName: "bad"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 400")
}

func TestGetMetaDatabase(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/database/50/connection",
		httpmock.NewStringResponder(200, metaDBConnectionResponse))

	metaDB, err := c.GetMetaDatabase(50)
	assert.NoError(t, err)
	assert.Equal(t, int64(50), metaDB.ID)
	assert.Equal(t, "Superset Meta", metaDB.DatabaseName)
	assert.Equal(t, "superset://", metaDB.SqlalchemyURI)
	assert.Equal(t, []string{"db1", "db2"}, metaDB.AllowedDBs)
}

func TestGetMetaDatabase_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/database/99/connection",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	_, err := c.GetMetaDatabase(99)
	assert.Error(t, err)
}

func TestUpdateMetaDatabase(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "csrf"}`))
	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/database/50",
		httpmock.NewStringResponder(200, `{}`))

	metaDB := &MetaDatabase{
		DatabaseName:  "Superset Meta",
		SqlalchemyURI: "superset://",
		AllowedDBs:    []string{"db1", "db2", "db3"},
	}
	err := c.UpdateMetaDatabase(50, metaDB)
	assert.NoError(t, err)
}

func TestFindMetaDatabaseByName_Found(t *testing.T) {
	ClearGlobalDatabaseCache()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/database/?q=(page_size:5000)",
		httpmock.NewStringResponder(200, `{"result": [
			{"id": 50, "database_name": "Superset Meta", "sqlalchemy_uri": "superset://"}
		]}`))
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/database/50/connection",
		httpmock.NewStringResponder(200, metaDBConnectionResponse))

	metaDB, err := c.FindMetaDatabaseByName("Superset Meta")
	assert.NoError(t, err)
	assert.NotNil(t, metaDB)
	assert.Equal(t, int64(50), metaDB.ID)
}

func TestFindMetaDatabaseByName_NotFound(t *testing.T) {
	ClearGlobalDatabaseCache()
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/database/?q=(page_size:5000)",
		httpmock.NewStringResponder(200, `{"result": []}`))

	metaDB, err := c.FindMetaDatabaseByName("Nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, metaDB)
}
