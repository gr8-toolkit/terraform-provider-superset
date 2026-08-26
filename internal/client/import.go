// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
)

// ImportDashboard imports a dashboard from a ZIP archive.
// passwords is a JSON string mapping "databases/<file>.yaml" to its password.
func (c *Client) ImportDashboard(zipData []byte, overwrite bool, passwords string) error {
	return c.importViaEndpoint("/api/v1/dashboard/import/", zipData, overwrite, passwords)
}

// ImportDataset imports datasets from a ZIP archive.
func (c *Client) ImportDataset(zipData []byte, overwrite bool, passwords string) error {
	return c.importViaEndpoint("/api/v1/dataset/import/", zipData, overwrite, passwords)
}

// ImportChart imports charts from a ZIP archive.
func (c *Client) ImportChart(zipData []byte, overwrite bool, passwords string) error {
	return c.importViaEndpoint("/api/v1/chart/import/", zipData, overwrite, passwords)
}

// importViaEndpoint posts a ZIP file to any Superset import endpoint.
func (c *Client) importViaEndpoint(endpoint string, zipData []byte, overwrite bool, passwords string) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("formData", "export.zip")
	if err != nil {
		return err
	}
	if _, err := part.Write(zipData); err != nil {
		return err
	}
	if overwrite {
		_ = writer.WriteField("overwrite", "true")
	}
	if passwords != "" {
		_ = writer.WriteField("passwords", passwords)
	}
	writer.Close()

	req, err := http.NewRequest("POST", fmt.Sprintf("%s%s", c.Host, endpoint), &body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-CSRFToken", csrfToken)
	req.Header.Set("Referer", c.Host)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	for _, cookie := range c.Cookies {
		req.AddCookie(cookie)
	}

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("import failed at %s, status code: %d, response: %s", endpoint, resp.StatusCode, string(respBody))
	}
	return nil
}

// GetDashboardIDByUUID finds a dashboard numeric ID by its UUID.
func (c *Client) GetDashboardIDByUUID(uuid string) (int64, error) {
	resp, err := c.DoRequest("GET",
		fmt.Sprintf("/api/v1/dashboard/?q=(filters:!((col:uuid,opr:eq,value:'%s')))", uuid), nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to fetch dashboard by uuid %q, status code: %d, response: %s", uuid, resp.StatusCode, string(body))
	}

	var result struct {
		Result []struct {
			ID float64 `json:"id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}
	if len(result.Result) == 0 {
		return 0, fmt.Errorf("dashboard with uuid %q not found", uuid)
	}
	return int64(result.Result[0].ID), nil
}

// DashboardExistsByID checks whether a dashboard with the given ID exists.
func (c *Client) DashboardExistsByID(id int64) (bool, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/dashboard/%d", id), nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("failed to check dashboard existence, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return true, nil
}

// ClearDashboardLayout resets position_json and json_metadata on a dashboard.
func (c *Client) ClearDashboardLayout(dashboardID int64) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}
	payload := map[string]interface{}{"position_json": "{}", "json_metadata": "{}"}

	resp, err := c.DoRequestWithHeadersAndCookies("PUT", fmt.Sprintf("/api/v1/dashboard/%d", dashboardID), payload, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to clear dashboard layout, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetDashboardChartUUIDs returns a map of chart UUID → chart ID for all charts on a dashboard.
func (c *Client) GetDashboardChartUUIDs(dashboardID int64) (map[string]int64, error) {
	resp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/dashboard/%d/charts", dashboardID), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get dashboard charts, status code: %d, response: %s", resp.StatusCode, string(body))
	}

	var chartsResult struct {
		Result []struct {
			ID int64 `json:"id"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&chartsResult); err != nil {
		return nil, err
	}

	result := make(map[string]int64)
	for _, chart := range chartsResult.Result {
		chartResp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/chart/%d", chart.ID), nil)
		if err != nil {
			continue
		}
		var chartData struct {
			Result struct {
				UUID string `json:"uuid"`
			} `json:"result"`
		}
		if err := json.NewDecoder(chartResp.Body).Decode(&chartData); err == nil && chartData.Result.UUID != "" {
			result[chartData.Result.UUID] = chart.ID
		}
		chartResp.Body.Close()
	}
	return result, nil
}

// UnlinkChartsFromDashboard removes a dashboard from each chart's dashboard list.
func (c *Client) UnlinkChartsFromDashboard(chartIDs []int64, dashboardID int64) error {
	if len(chartIDs) == 0 {
		return nil
	}
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	for _, chartID := range chartIDs {
		chartResp, err := c.DoRequest("GET", fmt.Sprintf("/api/v1/chart/%d", chartID), nil)
		if err != nil {
			continue
		}
		var chartData struct {
			Result struct {
				Dashboards []struct {
					ID int64 `json:"id"`
				} `json:"dashboards"`
			} `json:"result"`
		}
		if err := json.NewDecoder(chartResp.Body).Decode(&chartData); err != nil {
			chartResp.Body.Close()
			continue
		}
		chartResp.Body.Close()

		remaining := make([]int64, 0)
		for _, d := range chartData.Result.Dashboards {
			if d.ID != dashboardID {
				remaining = append(remaining, d.ID)
			}
		}

		updateResp, err := c.DoRequestWithHeadersAndCookies("PUT",
			fmt.Sprintf("/api/v1/chart/%d", chartID),
			map[string]interface{}{"dashboards": remaining}, headers, cookies)
		if err != nil {
			return fmt.Errorf("failed to update chart %d: %w", chartID, err)
		}
		updateResp.Body.Close()
		if updateResp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to unlink chart %d from dashboard, status code: %d", chartID, updateResp.StatusCode)
		}
	}
	return nil
}

// SetDashboardRoles assigns roles to a dashboard.
func (c *Client) SetDashboardRoles(dashboardID int64, roleIDs []int64) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("PUT",
		fmt.Sprintf("/api/v1/dashboard/%d", dashboardID),
		map[string]interface{}{"roles": roleIDs}, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to set dashboard roles, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}

// DeleteDashboard deletes a dashboard by ID (used by the import resource).
func (c *Client) DeleteDashboard(id int64) error {
	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}
	headers := map[string]string{"X-CSRFToken": csrfToken, "Referer": c.Host}

	resp, err := c.DoRequestWithHeadersAndCookies("DELETE", fmt.Sprintf("/api/v1/dashboard/%d", id), nil, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete dashboard, status code: %d, response: %s", resp.StatusCode, string(body))
	}
	return nil
}
