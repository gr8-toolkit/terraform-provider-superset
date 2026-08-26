package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestCreateRowLevelSecurity(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "test-csrf-token"}`))

	httpmock.RegisterResponder("POST", "http://test-host/api/v1/rowlevelsecurity/",
		httpmock.NewStringResponder(201, `{"id": 123}`))

	id, err := client.CreateRowLevelSecurity("test_rls", []int64{1}, "user_id = 1", []int64{2}, "group1", "Regular", "Test RLS")

	assert.NoError(t, err)
	assert.Equal(t, int64(123), id)
}

func TestGetRowLevelSecurity(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", "http://test-host/api/v1/rowlevelsecurity/123",
		httpmock.NewStringResponder(200, `{
			"result": {
				"id": 123,
				"name": "test_rls",
				"tables": [{"id": 1}],
				"clause": "user_id = 1",
				"group_key": "group1",
				"filter_type": "Regular",
				"description": "Test RLS",
				"roles": [{"id": 2}]
			}
		}`))

	rls, err := client.GetRowLevelSecurity(123)

	assert.NoError(t, err)
	assert.Equal(t, int64(123), rls.ID)
	assert.Equal(t, "test_rls", rls.Name)
	assert.Equal(t, []int64{1}, rls.Tables)
	assert.Equal(t, "user_id = 1", rls.Clause)
	assert.Equal(t, []int64{2}, rls.RoleIDs)
}

func TestUpdateRowLevelSecurity(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "test-csrf-token"}`))

	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/rowlevelsecurity/123",
		httpmock.NewStringResponder(200, `{}`))

	err := client.UpdateRowLevelSecurity(123, "updated_rls", []int64{1, 2}, "user_id = 2", []int64{3}, "group2", "Base", "Updated RLS")

	assert.NoError(t, err)
}

func TestDeleteRowLevelSecurity(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	client := &Client{
		Host:  "http://test-host",
		Token: "test-token",
	}

	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/csrf_token/",
		httpmock.NewStringResponder(200, `{"result": "test-csrf-token"}`))

	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/rowlevelsecurity/123",
		httpmock.NewStringResponder(200, ``))

	err := client.DeleteRowLevelSecurity(123)

	assert.NoError(t, err)
}
