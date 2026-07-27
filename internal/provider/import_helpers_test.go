package provider

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestExportDir(t *testing.T) string {
	t.Helper()

	// Simulate a single dashboard export directory
	root := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(root, "datasets", "db_alpha"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "databases"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "charts"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "dashboards"), 0755))

	require.NoError(t, os.WriteFile(filepath.Join(root, "metadata.yaml"),
		[]byte("version: 1.0.0\ntype: Dashboard\ntimestamp: '2024-01-01T00:00:00+00:00'\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "datasets", "db_alpha", "dataset_a.yaml"),
		[]byte("table_name: dataset_a\nuuid: aaaa-1111\ndatabase_uuid: db-uuid-1\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "datasets", "db_alpha", "dataset_b.yaml"),
		[]byte("table_name: dataset_b\nuuid: bbbb-2222\ndatabase_uuid: db-uuid-1\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "databases", "db_alpha.yaml"),
		[]byte("database_name: db_alpha\nuuid: db-uuid-1\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "charts", "chart_a.yaml"),
		[]byte("slice_name: Chart A\nuuid: chart-a-uuid\ndataset_uuid: aaaa-1111\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "charts", "chart_b.yaml"),
		[]byte("slice_name: Chart B\nuuid: chart-b-uuid\ndataset_uuid: bbbb-2222\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "dashboards", "my_dashboard.yaml"),
		[]byte("dashboard_title: My Dashboard\nuuid: dash-uuid-1\n"), 0644))

	return root
}

func TestComputeFilteredFileHashes_Datasets(t *testing.T) {
	root := setupTestExportDir(t)

	hashes, err := computeFilteredFileHashes(root, []string{"datasets/", "databases/"}, nil, nil)
	require.NoError(t, err)

	// Should include datasets and databases, not charts or dashboards
	assert.Contains(t, hashes, "datasets/db_alpha/dataset_a.yaml")
	assert.Contains(t, hashes, "datasets/db_alpha/dataset_b.yaml")
	assert.Contains(t, hashes, "databases/db_alpha.yaml")
	assert.NotContains(t, hashes, "charts/chart_a.yaml")
	assert.NotContains(t, hashes, "dashboards/my_dashboard.yaml")
	assert.Equal(t, 3, len(hashes))
}

func TestComputeFilteredFileHashes_Charts(t *testing.T) {
	root := setupTestExportDir(t)

	hashes, err := computeFilteredFileHashes(root, []string{"charts/", "datasets/", "databases/"}, nil, nil)
	require.NoError(t, err)

	assert.Contains(t, hashes, "charts/chart_a.yaml")
	assert.Contains(t, hashes, "charts/chart_b.yaml")
	assert.Contains(t, hashes, "datasets/db_alpha/dataset_a.yaml")
	assert.Contains(t, hashes, "datasets/db_alpha/dataset_b.yaml")
	assert.Contains(t, hashes, "databases/db_alpha.yaml")
	assert.NotContains(t, hashes, "dashboards/my_dashboard.yaml")
	assert.Equal(t, 5, len(hashes))
}

func TestComputeFilteredFileHashes_DatabaseOverrides(t *testing.T) {
	root := setupTestExportDir(t)

	overrides := map[string]map[string]interface{}{
		"db-uuid-1": {"sqlalchemy_uri": "starrocks://new-host:9030"},
	}

	hashes, err := computeFilteredFileHashes(root, []string{"databases/"}, overrides, nil)
	require.NoError(t, err)

	// Hash should differ from the non-overridden version
	hashesNoOverride, _ := computeFilteredFileHashes(root, []string{"databases/"}, nil, nil)
	assert.NotEqual(t, hashes["databases/db_alpha.yaml"], hashesNoOverride["databases/db_alpha.yaml"])
}

func TestZipDirectoryFiltered_Datasets(t *testing.T) {
	root := setupTestExportDir(t)

	zipData, err := zipDirectoryFiltered(root, nil, []string{"datasets/", "databases/"}, "SqlaTable", nil)
	require.NoError(t, err)
	require.NotEmpty(t, zipData)

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)

	// Collect paths relative to the ZIP root dir
	var relPaths []string
	for _, f := range reader.File {
		// Strip the root dir prefix (e.g. "tmpdir123/datasets/..." -> "datasets/...")
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) == 2 && parts[1] != "" {
			relPaths = append(relPaths, parts[1])
		}
	}

	assert.Contains(t, relPaths, "datasets/db_alpha/dataset_a.yaml")
	assert.Contains(t, relPaths, "datasets/db_alpha/dataset_b.yaml")
	assert.Contains(t, relPaths, "databases/db_alpha.yaml")
	assert.Contains(t, relPaths, "metadata.yaml")
	assert.NotContains(t, relPaths, "charts/chart_a.yaml")
	assert.NotContains(t, relPaths, "dashboards/my_dashboard.yaml")
}

func TestZipDirectoryFiltered_Charts(t *testing.T) {
	root := setupTestExportDir(t)

	zipData, err := zipDirectoryFiltered(root, nil, []string{"charts/", "datasets/", "databases/"}, "Slice", nil)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)

	var relPaths []string
	for _, f := range reader.File {
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) == 2 && parts[1] != "" {
			relPaths = append(relPaths, parts[1])
		}
	}

	assert.Contains(t, relPaths, "charts/chart_a.yaml")
	assert.Contains(t, relPaths, "charts/chart_b.yaml")
	assert.Contains(t, relPaths, "datasets/db_alpha/dataset_a.yaml")
	assert.Contains(t, relPaths, "databases/db_alpha.yaml")
	assert.Contains(t, relPaths, "metadata.yaml")
	assert.NotContains(t, relPaths, "dashboards/my_dashboard.yaml")
}

func TestZipDirectoryFiltered_MetadataType(t *testing.T) {
	root := setupTestExportDir(t)

	zipData, err := zipDirectoryFiltered(root, nil, []string{"datasets/", "databases/"}, "SqlaTable", nil)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)

	for _, f := range reader.File {
		if strings.HasSuffix(f.Name, "/metadata.yaml") {
			rc, err := f.Open()
			require.NoError(t, err)
			buf := make([]byte, 1024)
			n, _ := rc.Read(buf)
			rc.Close()
			content := string(buf[:n])
			assert.Contains(t, content, "SqlaTable")
			assert.NotContains(t, content, "Dashboard")
			return
		}
	}
	t.Fatal("metadata.yaml not found in ZIP")
}

func TestZipDirectoryFiltered_DatabaseOverrides(t *testing.T) {
	root := setupTestExportDir(t)

	overrides := map[string]map[string]interface{}{
		"db-uuid-1": {"sqlalchemy_uri": "starrocks://overridden:9030"},
	}

	zipData, err := zipDirectoryFiltered(root, overrides, []string{"databases/"}, "SqlaTable", nil)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)

	for _, f := range reader.File {
		if strings.HasSuffix(f.Name, "/databases/db_alpha.yaml") {
			rc, err := f.Open()
			require.NoError(t, err)
			buf := make([]byte, 1024)
			n, _ := rc.Read(buf)
			rc.Close()
			assert.Contains(t, string(buf[:n]), "starrocks://overridden:9030")
			return
		}
	}
	t.Fatal("database file not found in ZIP")
}

func setupTestExportDirWithExtraFiles(t *testing.T) string {
	t.Helper()
	root := setupTestExportDir(t)

	// Add terragrunt manifest files that should be skippable
	require.NoError(t, os.WriteFile(filepath.Join(root, ".terragrunt-source-manifest"),
		[]byte("hash: abc123\n"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".terragrunt-module-manifest"),
		[]byte("module: something\n"), 0644))
	// Add a .terraform.lock.hcl in a subdirectory
	require.NoError(t, os.WriteFile(filepath.Join(root, "databases", ".terraform.lock.hcl"),
		[]byte("provider stuff\n"), 0644))

	return root
}

func TestComputeFilteredFileHashes_SkipFiles(t *testing.T) {
	root := setupTestExportDirWithExtraFiles(t)

	// Without skip patterns — extra files in databases/ prefix are included
	hashesAll, err := computeFilteredFileHashes(root, []string{"datasets/", "databases/"}, nil, nil)
	require.NoError(t, err)
	assert.Contains(t, hashesAll, "databases/.terraform.lock.hcl")

	// With skip patterns — terraform lock file excluded
	skip := compileSkipPatterns([]string{`\.terraform\.lock\.hcl`})
	hashesSkipped, err := computeFilteredFileHashes(root, []string{"datasets/", "databases/"}, nil, skip)
	require.NoError(t, err)
	assert.NotContains(t, hashesSkipped, "databases/.terraform.lock.hcl")
	assert.Contains(t, hashesSkipped, "databases/db_alpha.yaml")
}

func TestComputeFilteredFileHashes_SkipFilesByRegex(t *testing.T) {
	root := setupTestExportDirWithExtraFiles(t)

	// Skip anything with "terragrunt" in the name
	skip := compileSkipPatterns([]string{`.*terragrunt.*`})
	hashes, err := computeFilteredFileHashes(root, []string{"datasets/", "databases/"}, nil, skip)
	require.NoError(t, err)

	// Terragrunt files are at root level, outside prefixes, so they wouldn't be included anyway.
	// But the pattern matching logic itself works — let's verify with a broader prefix.
	hashesAll, err := computeFilteredFileHashes(root, []string{"datasets/", "databases/", ""}, nil, nil)
	require.NoError(t, err)
	assert.Contains(t, hashesAll, ".terragrunt-source-manifest")

	hashesSkipped, err := computeFilteredFileHashes(root, []string{"datasets/", "databases/", ""}, nil, skip)
	require.NoError(t, err)
	assert.NotContains(t, hashesSkipped, ".terragrunt-source-manifest")
	assert.NotContains(t, hashesSkipped, ".terragrunt-module-manifest")

	// Non-matching files still present
	assert.Contains(t, hashesSkipped, "databases/db_alpha.yaml")
	_ = hashes
}

func TestZipDirectoryFiltered_SkipFiles(t *testing.T) {
	root := setupTestExportDirWithExtraFiles(t)

	skip := compileSkipPatterns([]string{`\.terraform\.lock\.hcl`})
	zipData, err := zipDirectoryFiltered(root, nil, []string{"datasets/", "databases/"}, "SqlaTable", skip)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	require.NoError(t, err)

	var relPaths []string
	for _, f := range reader.File {
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) == 2 && parts[1] != "" {
			relPaths = append(relPaths, parts[1])
		}
	}

	assert.NotContains(t, relPaths, "databases/.terraform.lock.hcl")
	assert.Contains(t, relPaths, "databases/db_alpha.yaml")
	assert.Contains(t, relPaths, "datasets/db_alpha/dataset_a.yaml")
}

func TestShouldSkipFile(t *testing.T) {
	patterns := compileSkipPatterns([]string{`.*terragrunt.*`, `^init$`})

	assert.True(t, shouldSkipFile(".terragrunt-source-manifest", ".terragrunt-source-manifest", patterns))
	assert.True(t, shouldSkipFile(".terragrunt-module-manifest", "sub/.terragrunt-module-manifest", patterns))
	assert.True(t, shouldSkipFile("init", "charts/init", patterns))
	assert.False(t, shouldSkipFile("chart_a.yaml", "charts/chart_a.yaml", patterns))
	assert.False(t, shouldSkipFile("initialize.yaml", "charts/initialize.yaml", patterns))
}

func TestCompileSkipPatterns_InvalidRegex(t *testing.T) {
	// Invalid regex should be silently skipped
	patterns := compileSkipPatterns([]string{`[invalid`, `.*valid.*`})
	assert.Equal(t, 1, len(patterns))
	assert.True(t, shouldSkipFile("valid_file", "valid_file", patterns))
}
