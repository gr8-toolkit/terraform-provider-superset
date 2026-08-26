// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

const userGetResponse = `{
	"result": {
		"id": 100,
		"username": "jdoe",
		"first_name": "John",
		"last_name": "Doe",
		"email": "jdoe@example.com",
		"active": true,
		"roles": [
			{"id": 4, "name": "Gamma"}
		]
	}
}`

func TestFetchUsers(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/users/?q=(page_size:5000)",
		httpmock.NewStringResponder(200, `{"result": [
			{"id": 1, "username": "admin", "first_name": "Admin", "last_name": "User", "email": "admin@localhost", "active": true, "roles": []},
			{"id": 100, "username": "jdoe", "first_name": "John", "last_name": "Doe", "email": "jdoe@example.com", "active": true, "roles": [{"id": 4, "name": "Gamma"}]}
		]}`))

	users, err := c.FetchUsers()
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, int64(1), users[0].ID)
	assert.Equal(t, "admin", users[0].Username)
}

func TestGetUser(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/users/100",
		httpmock.NewStringResponder(200, userGetResponse))

	user, err := c.GetUser(100)
	assert.NoError(t, err)
	assert.Equal(t, int64(100), user.ID)
	assert.Equal(t, "jdoe", user.Username)
	assert.Equal(t, "John", user.FirstName)
	assert.Equal(t, "Doe", user.LastName)
	assert.Equal(t, "jdoe@example.com", user.Email)
	assert.True(t, user.Active)
	assert.Equal(t, []int64{4}, user.Roles)
}

func TestGetUser_NotFound(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("GET", "http://test-host/api/v1/security/users/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	_, err := c.GetUser(99)
	assert.Error(t, err)
}

func TestCreateUser(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/security/users/",
		httpmock.NewStringResponder(201, `{"id": 101}`))

	id, err := c.CreateUser("newuser", "New", "User", "new@example.com", "Pass123!", true, []int64{4})
	assert.NoError(t, err)
	assert.Equal(t, int64(101), id)
}

func TestCreateUser_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("POST", "http://test-host/api/v1/security/users/",
		httpmock.NewStringResponder(400, `{"message": "bad request"}`))

	_, err := c.CreateUser("bad", "", "", "", "", false, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "status code: 400")
}

func TestUpdateUser(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/security/users/100",
		httpmock.NewStringResponder(200, `{}`))

	err := c.UpdateUser(100, "jdoe", "John", "Updated", "jdoe@example.com", "", true, []int64{4})
	assert.NoError(t, err)
}

func TestUpdateUser_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("PUT", "http://test-host/api/v1/security/users/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	err := c.UpdateUser(99, "x", "", "", "", "", false, nil)
	assert.Error(t, err)
}

func TestDeleteUser(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/security/users/100",
		httpmock.NewStringResponder(204, ""))

	err := c.DeleteUser(100)
	assert.NoError(t, err)
}

func TestDeleteUser_Error(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	c := &Client{Host: "http://test-host", Token: "tok"}
	httpmock.RegisterResponder("DELETE", "http://test-host/api/v1/security/users/99",
		httpmock.NewStringResponder(404, `{"message": "Not found"}`))

	err := c.DeleteUser(99)
	assert.Error(t, err)
}
