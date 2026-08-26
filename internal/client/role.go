// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// rawRoleModel is the wire shape returned by the roles list endpoint.
type rawRoleModel struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Permission represents a single permission-view-menu pair in Superset.
type Permission struct {
	ID             int64  `json:"id"`
	PermissionName string `json:"permission_name"`
	ViewMenuName   string `json:"view_menu_name"`
}

// Role represents a Superset role.
type Role struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// GetRoleIDByName returns the numeric ID of a role by its name.
func (c *Client) GetRoleIDByName(roleName string) (int64, error) {
	resp, err := c.DoRequest("GET", "/api/v1/security/roles?q=(page_size:5000)", nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to fetch roles from Superset, status code: %d", resp.StatusCode)
	}

	var result struct {
		Roles []struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	for _, role := range result.Roles {
		if role.Name == roleName {
			return role.ID, nil
		}
	}
	return 0, fmt.Errorf("role %s not found", roleName)
}

// FetchRoles returns all roles from Superset.
func (c *Client) FetchRoles() ([]rawRoleModel, error) {
	resp, err := c.DoRequest("GET", "/api/v1/security/roles?q=(page_size:5000)", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch roles from Superset, status code: %d", resp.StatusCode)
	}

	var result struct {
		Roles []rawRoleModel `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Roles, nil
}

// CreateRole creates a role; returns the existing ID if the role already exists.
func (c *Client) CreateRole(name string) (int64, error) {
	if id, err := c.GetRoleIDByName(name); err == nil {
		return id, nil
	}

	resp, err := c.DoRequest("POST", "/api/v1/security/roles/", map[string]string{"name": name})
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to create role, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	switch v := result["id"].(type) {
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	default:
		return 0, fmt.Errorf("failed to retrieve role ID from response")
	}
}

// GetRole retrieves a role by numeric ID.
func (c *Client) GetRole(id int64) (*Role, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/security/roles/%d", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch role, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var result struct {
		Result struct {
			ID   int64  `json:"id"`
			Name string `json:"name"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}
	return &Role{ID: result.Result.ID, Name: result.Result.Name}, nil
}

// UpdateRole renames a role by ID.
func (c *Client) UpdateRole(id int64, name string) error {
	existing, err := c.GetRole(id)
	if err != nil {
		return err
	}
	if existing.Name == name {
		return nil // nothing to do
	}

	resp, err := c.DoRequest("PUT", fmt.Sprintf("/api/v1/security/roles/%d", id), map[string]string{"name": name})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update role, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

// DeleteRole deletes a role by ID.
func (c *Client) DeleteRole(id int64) error {
	resp, err := c.DoRequest("DELETE", fmt.Sprintf("/api/v1/security/roles/%d", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete role, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetRolePermissions returns the permissions assigned to a role.
func (c *Client) GetRolePermissions(roleID int64) ([]Permission, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/security/roles/%d/permissions/", roleID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch permissions from Superset, status code: %d", resp.StatusCode)
	}

	var result struct {
		Permissions []Permission `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Permissions, nil
}

// GetPermissionViewMenuIDs resolves permission+view_menu name pairs to their numeric IDs.
func (c *Client) GetPermissionViewMenuIDs(permissions []map[string]string) ([]int64, error) {
	page := 0
	pageSize := 100
	var ids []int64
	found := make(map[string]bool)
	for _, p := range permissions {
		found[p["permission"]+"|"+p["view_menu"]] = false
	}

	for {
		url := fmt.Sprintf("%s/api/v1/security/permissions-resources/?q=(page:%d,page_size:%d)", c.Host, page, pageSize)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("failed to fetch permissions resources from Superset, status code: %d", resp.StatusCode)
		}

		var result struct {
			Resources []struct {
				ID         int64 `json:"id"`
				Permission struct {
					Name string `json:"name"`
				} `json:"permission"`
				ViewMenu struct {
					Name string `json:"name"`
				} `json:"view_menu"`
			} `json:"result"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, err
		}

		for _, p := range permissions {
			key := p["permission"] + "|" + p["view_menu"]
			if !found[key] {
				for _, res := range result.Resources {
					if res.Permission.Name == p["permission"] && res.ViewMenu.Name == p["view_menu"] {
						ids = append(ids, res.ID)
						found[key] = true
						break
					}
				}
			}
		}

		allFound := true
		for _, v := range found {
			if !v {
				allFound = false
				break
			}
		}
		if allFound || len(result.Resources) < pageSize {
			break
		}
		page++
	}
	return ids, nil
}

// GetPermissionIDByNameAndView resolves a single permission+view_menu name pair.
func (c *Client) GetPermissionIDByNameAndView(permissionName, viewMenuName string) (int64, error) {
	page := 0
	pageSize := 100
	for {
		resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/security/permissions-resources?q=(page:%d,page_size:%d)", page, pageSize), nil)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return 0, fmt.Errorf("failed to fetch permissions resources from Superset, status code: %d", resp.StatusCode)
		}

		var result struct {
			Resources []struct {
				ID         int64 `json:"id"`
				Permission struct {
					Name string `json:"name"`
				} `json:"permission"`
				ViewMenu struct {
					Name string `json:"name"`
				} `json:"view_menu"`
			} `json:"result"`
			Count int `json:"count"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return 0, err
		}
		for _, r := range result.Resources {
			if r.Permission.Name == permissionName && r.ViewMenu.Name == viewMenuName {
				return r.ID, nil
			}
		}
		if len(result.Resources) < pageSize {
			break
		}
		page++
	}
	return 0, fmt.Errorf("permission %s with view menu %s not found", permissionName, viewMenuName)
}

// UpdateRolePermissions replaces all permissions on a role with the given IDs.
func (c *Client) UpdateRolePermissions(roleID int64, permissionIDs []int64) error {
	url := fmt.Sprintf("%s/api/v1/security/roles/%d/permissions", c.Host, roleID)
	data, err := json.Marshal(map[string][]int64{"permission_view_menu_ids": permissionIDs})
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update role permissions, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ClearRolePermissions removes all permissions from a role.
func (c *Client) ClearRolePermissions(roleID int64) error {
	resp, err := c.DoRequest("POST", fmt.Sprintf("/api/v1/security/roles/%d/permissions", roleID),
		map[string]interface{}{"permission_view_menu_ids": []int64{}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to clear role permissions, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}
