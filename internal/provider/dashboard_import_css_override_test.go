package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyCSSOverride(t *testing.T) {
	yamlContent := []byte(`dashboard_title: Test Dashboard
description: null
css: 'body { background: blue; }'
slug: test-dashboard
published: true
uuid: 11111111-1111-1111-1111-111111111111
version: 1.0.0
`)

	result, err := applyCSSOverride(yamlContent, "body { background: red !important; }")

	require.NoError(t, err)
	assert.Contains(t, string(result), "body { background: red !important; }")
	assert.NotContains(t, string(result), "background: blue")
	assert.Contains(t, string(result), "dashboard_title: Test Dashboard")
	assert.Contains(t, string(result), "uuid: 11111111-1111-1111-1111-111111111111")
}

func TestApplyCSSOverride_EmptyCSS(t *testing.T) {
	yamlContent := []byte(`dashboard_title: Test
css: ''
uuid: 22222222-2222-2222-2222-222222222222
version: 1.0.0
`)

	result, err := applyCSSOverride(yamlContent, ".new { color: green; }")

	require.NoError(t, err)
	assert.Contains(t, string(result), ".new { color: green; }")
}

func TestApplyCSSOverride_NoCSSField(t *testing.T) {
	yamlContent := []byte(`dashboard_title: No CSS Dashboard
uuid: 33333333-3333-3333-3333-333333333333
version: 1.0.0
`)

	result, err := applyCSSOverride(yamlContent, "body { color: red; }")

	require.NoError(t, err)
	assert.Contains(t, string(result), "body { color: red; }")
	assert.Contains(t, string(result), "dashboard_title: No CSS Dashboard")
}

func TestApplyCSSOverride_PreservesOtherFields(t *testing.T) {
	yamlContent := []byte(`dashboard_title: Full Dashboard
description: A test dashboard
css: 'old css here'
slug: full-dash
certified_by: admin
published: true
uuid: 44444444-4444-4444-4444-444444444444
version: 1.0.0
position:
  DASHBOARD_VERSION_KEY: v2
`)

	result, err := applyCSSOverride(yamlContent, "new css")

	require.NoError(t, err)
	resultStr := string(result)
	assert.Contains(t, resultStr, "new css")
	assert.Contains(t, resultStr, "dashboard_title: Full Dashboard")
	assert.Contains(t, resultStr, "description: A test dashboard")
	assert.Contains(t, resultStr, "slug: full-dash")
	assert.Contains(t, resultStr, "certified_by: admin")
	assert.Contains(t, resultStr, "44444444-4444-4444-4444-444444444444")
	assert.NotContains(t, resultStr, "old css here")
}

func TestApplyCSSOverride_InvalidYAML(t *testing.T) {
	invalidYAML := []byte(`{invalid yaml: [`)

	result, err := applyCSSOverride(invalidYAML, "body {}")

	// applyCSSOverride returns original data and an error on parse failure
	assert.Error(t, err)
	assert.Equal(t, invalidYAML, result)
}
