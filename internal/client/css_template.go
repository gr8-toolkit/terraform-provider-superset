package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// CSSTemplate represents a CSS template in Superset.
type CSSTemplate struct {
	ID           int    `json:"id"`
	TemplateName string `json:"template_name"`
	CSS          string `json:"css"`
}

// truncateBody truncates a string to maxLen characters, appending "... [truncated]" if truncated.
func truncateBody(body string, maxLen int) string {
	if len(body) > maxLen {
		return body[:maxLen] + "... [truncated]"
	}
	return body
}

// CreateCSSTemplate creates a new CSS template in Superset.
// POST /api/v1/css_template/ with {"template_name": "...", "css": "..."}.
func (c *Client) CreateCSSTemplate(templateName, css string) (*CSSTemplate, error) {
	endpoint := "/api/v1/css_template/"
	payload := map[string]string{
		"template_name": templateName,
		"css":           css,
	}

	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("POST", endpoint, payload, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if authErr := c.authenticate(); authErr != nil {
			return nil, authErr
		}
		csrfToken, cookies, err = c.GetCSRFToken()
		if err != nil {
			return nil, err
		}
		headers = map[string]string{
			"X-CSRFToken": csrfToken,
			"Referer":     c.Host,
		}
		resp, err = c.DoRequestWithHeadersAndCookies("POST", endpoint, payload, headers, cookies)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create CSS template, status code: %d, response: %s", resp.StatusCode, truncateBody(string(body), 1024))
	}

	var result struct {
		ID     int `json:"id"`
		Result struct {
			ID           int    `json:"id"`
			TemplateName string `json:"template_name"`
			CSS          string `json:"css"`
		} `json:"result"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	tmpl := &CSSTemplate{
		ID:           result.Result.ID,
		TemplateName: result.Result.TemplateName,
		CSS:          result.Result.CSS,
	}

	// If the result block doesn't have an ID, use the top-level id.
	if tmpl.ID == 0 {
		tmpl.ID = result.ID
	}

	return tmpl, nil
}

// GetCSSTemplate retrieves a CSS template by its ID.
// GET /api/v1/css_template/{id}.
func (c *Client) GetCSSTemplate(id int) (*CSSTemplate, error) {
	endpoint := fmt.Sprintf("/api/v1/css_template/%d", id)

	resp, err := c.DoRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if authErr := c.authenticate(); authErr != nil {
			return nil, authErr
		}
		resp, err = c.DoRequest("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("CSS template with ID %d not found", id)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch CSS template, status code: %d, response: %s", resp.StatusCode, truncateBody(string(body), 1024))
	}

	var result struct {
		Result struct {
			ID           int    `json:"id"`
			TemplateName string `json:"template_name"`
			CSS          string `json:"css"`
		} `json:"result"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &CSSTemplate{
		ID:           result.Result.ID,
		TemplateName: result.Result.TemplateName,
		CSS:          result.Result.CSS,
	}, nil
}

// UpdateCSSTemplate updates a CSS template by its ID.
// PUT /api/v1/css_template/{id} with full payload (both fields always sent).
func (c *Client) UpdateCSSTemplate(id int, templateName, css string) (*CSSTemplate, error) {
	endpoint := fmt.Sprintf("/api/v1/css_template/%d", id)
	payload := map[string]string{
		"template_name": templateName,
		"css":           css,
	}

	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return nil, err
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("PUT", endpoint, payload, headers, cookies)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if authErr := c.authenticate(); authErr != nil {
			return nil, authErr
		}
		csrfToken, cookies, err = c.GetCSRFToken()
		if err != nil {
			return nil, err
		}
		headers = map[string]string{
			"X-CSRFToken": csrfToken,
			"Referer":     c.Host,
		}
		resp, err = c.DoRequestWithHeadersAndCookies("PUT", endpoint, payload, headers, cookies)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("CSS template with ID %d not found", id)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update CSS template, status code: %d, response: %s", resp.StatusCode, truncateBody(string(body), 1000))
	}

	var result struct {
		Result struct {
			ID           int    `json:"id"`
			TemplateName string `json:"template_name"`
			CSS          string `json:"css"`
		} `json:"result"`
	}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, err
	}

	return &CSSTemplate{
		ID:           result.Result.ID,
		TemplateName: result.Result.TemplateName,
		CSS:          result.Result.CSS,
	}, nil
}

// DeleteCSSTemplate deletes a CSS template by its ID.
// DELETE /api/v1/css_template/{id}; 404 is treated as success.
func (c *Client) DeleteCSSTemplate(id int) error {
	endpoint := fmt.Sprintf("/api/v1/css_template/%d", id)

	csrfToken, cookies, err := c.GetCSRFToken()
	if err != nil {
		return err
	}

	headers := map[string]string{
		"X-CSRFToken": csrfToken,
		"Referer":     c.Host,
	}

	resp, err := c.DoRequestWithHeadersAndCookies("DELETE", endpoint, nil, headers, cookies)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		if authErr := c.authenticate(); authErr != nil {
			return authErr
		}
		csrfToken, cookies, err = c.GetCSRFToken()
		if err != nil {
			return err
		}
		headers = map[string]string{
			"X-CSRFToken": csrfToken,
			"Referer":     c.Host,
		}
		resp, err = c.DoRequestWithHeadersAndCookies("DELETE", endpoint, nil, headers, cookies)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
	}

	// 404 is treated as success (resource already gone).
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete CSS template, status code: %d, response: %s", resp.StatusCode, truncateBody(string(body), 1024))
	}

	return nil
}

// FindCSSTemplatesByName finds all CSS templates matching the exact name.
// Returns a slice of all matching templates (may be 0, 1, or multiple).
func (c *Client) FindCSSTemplatesByName(name string) ([]CSSTemplate, error) {
	page := 0
	pageSize := 100
	var matches []CSSTemplate

	for {
		q := fmt.Sprintf("(filters:!((col:template_name,opr:eq,value:'%s')),page:%d,page_size:%d)", name, page, pageSize)
		endpoint := fmt.Sprintf("/api/v1/css_template/?q=%s", url.QueryEscape(q))

		resp, err := c.DoRequest("GET", endpoint, nil)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			if authErr := c.authenticate(); authErr != nil {
				return nil, authErr
			}
			resp, err = c.DoRequest("GET", endpoint, nil)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("failed to search CSS templates, status code: %d, response: %s", resp.StatusCode, truncateBody(string(body), 1024))
		}

		var result struct {
			Result []struct {
				ID           int    `json:"id"`
				TemplateName string `json:"template_name"`
				CSS          string `json:"css"`
			} `json:"result"`
			Count int `json:"count"`
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		if err != nil {
			return nil, err
		}

		for _, tmpl := range result.Result {
			if tmpl.TemplateName == name {
				matches = append(matches, CSSTemplate{
					ID:           tmpl.ID,
					TemplateName: tmpl.TemplateName,
					CSS:          tmpl.CSS,
				})
			}
		}

		if len(result.Result) < pageSize {
			break
		}
		page++
	}

	return matches, nil
}

// FindCSSTemplateByName finds a single CSS template by exact name match.
// Returns an error if no match is found or if multiple matches exist (ambiguous).
func (c *Client) FindCSSTemplateByName(name string) (*CSSTemplate, error) {
	matches, err := c.FindCSSTemplatesByName(name)
	if err != nil {
		return nil, err
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("CSS template with name %q not found", name)
	}

	if len(matches) > 1 {
		return nil, fmt.Errorf("multiple CSS templates found with name %q (%d matches), result is ambiguous", name, len(matches))
	}

	return &matches[0], nil
}
