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

// PermissionKey builds the lookup key used by GetPermissionIDMap.
func PermissionKey(permissionName, viewMenuName string) string {
	return permissionName + "|" + viewMenuName
}

// GetPermissionIDMap returns the IDs of all permission+view_menu pairs, keyed by PermissionKey.
// Pages are ordered by id: without an explicit order the database may return overlapping
// pages and silently skip entries.
func (c *Client) GetPermissionIDMap() (map[string]int64, error) {
	pageSize := 100
	ids := make(map[string]int64)
	for page := 0; ; page++ {
		endpoint := fmt.Sprintf("/api/v1/security/permissions-resources/?q=(order_column:id,order_direction:asc,page:%d,page_size:%d)", page, pageSize)
		resp, err := c.DoRequest("GET", endpoint, nil)
		if err != nil {
			return nil, err
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
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return nil, fmt.Errorf("failed to fetch permissions resources from Superset, status code: %d", resp.StatusCode)
		}
		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		for _, r := range result.Resources {
			ids[PermissionKey(r.Permission.Name, r.ViewMenu.Name)] = r.ID
		}
		if len(result.Resources) < pageSize {
			return ids, nil
		}
	}
}

// GetPermissionViewMenuIDs resolves permission+view_menu name pairs to their numeric IDs.
// Pairs that don't exist in Superset are skipped.
func (c *Client) GetPermissionViewMenuIDs(permissions []map[string]string) ([]int64, error) {
	idMap, err := c.GetPermissionIDMap()
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	for _, p := range permissions {
		if id, ok := idMap[PermissionKey(p["permission"], p["view_menu"])]; ok {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// GetPermissionIDByNameAndView resolves a single permission+view_menu name pair.
func (c *Client) GetPermissionIDByNameAndView(permissionName, viewMenuName string) (int64, error) {
	idMap, err := c.GetPermissionIDMap()
	if err != nil {
		return 0, err
	}
	if id, ok := idMap[PermissionKey(permissionName, viewMenuName)]; ok {
		return id, nil
	}
	return 0, fmt.Errorf("permission %s with view menu %s not found", permissionName, viewMenuName)
}

// UpdateRolePermissions replaces all permissions on a role with the given IDs.
func (c *Client) UpdateRolePermissions(roleID int64, permissionIDs []int64) error {
	// a nil slice is marshalled as null, which the API rejects
	if permissionIDs == nil {
		permissionIDs = []int64{}
	}
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
