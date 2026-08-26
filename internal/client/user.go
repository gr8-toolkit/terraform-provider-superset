// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// User represents a Superset user.
type User struct {
	ID        int64   `json:"id"`
	Username  string  `json:"username"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Email     string  `json:"email"`
	Active    bool    `json:"active"`
	Roles     []int64 `json:"roles,omitempty"`
}

// rawUserModel is the wire shape returned by the users list and get endpoints.
type rawUserModel struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Active    bool   `json:"active"`
	Roles     []struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	} `json:"roles"`
}

// FetchUsers returns all users.
func (c *Client) FetchUsers() ([]rawUserModel, error) {
	resp, err := c.DoRequest("GET", "/api/v1/security/users/?q=(page_size:5000)", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch users from Superset, status code: %d", resp.StatusCode)
	}

	var result struct {
		Users []rawUserModel `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Users, nil
}

// GetUser retrieves a user by numeric ID.
func (c *Client) GetUser(id int64) (*User, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/security/users/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result struct {
		Result rawUserModel `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	user := &User{
		ID:        result.Result.ID,
		Username:  result.Result.Username,
		FirstName: result.Result.FirstName,
		LastName:  result.Result.LastName,
		Email:     result.Result.Email,
		Active:    result.Result.Active,
		Roles:     make([]int64, len(result.Result.Roles)),
	}
	for i, r := range result.Result.Roles {
		user.Roles[i] = r.ID
	}
	return user, nil
}

// CreateUser creates a new user.
func (c *Client) CreateUser(username, firstName, lastName, email, password string, active bool, roles []int64) (int64, error) {
	payload := map[string]interface{}{
		"username":   username,
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
		"password":   password,
		"active":     active,
		"roles":      roles,
	}
	resp, err := c.DoRequest("POST", "/api/v1/security/users/", payload)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to create user, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	id, ok := result["id"].(float64)
	if !ok {
		return 0, fmt.Errorf("failed to retrieve user ID from response")
	}
	return int64(id), nil
}

// UpdateUser updates an existing user.
func (c *Client) UpdateUser(id int64, username, firstName, lastName, email, password string, active bool, roles []int64) error {
	payload := map[string]interface{}{
		"username":   username,
		"first_name": firstName,
		"last_name":  lastName,
		"email":      email,
		"active":     active,
		"roles":      roles,
	}
	if password != "" {
		payload["password"] = password
	}

	resp, err := c.DoRequest("PUT", fmt.Sprintf("/api/v1/security/users/%d", id), payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update user, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

// DeleteUser deletes a user by ID.
func (c *Client) DeleteUser(id int64) error {
	resp, err := c.DoRequest("DELETE", fmt.Sprintf("/api/v1/security/users/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete user, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}
