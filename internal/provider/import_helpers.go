package provider

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// shouldSkipFile checks whether a file (by its base name or relative path) matches
// any of the user-provided skip patterns. Each pattern is treated as a regex.
// If the pattern contains no regex metacharacters, it is matched as an exact filename.
func shouldSkipFile(name string, relPath string, skipPatterns []*regexp.Regexp) bool {
	for _, re := range skipPatterns {
		if re.MatchString(name) || re.MatchString(relPath) {
			return true
		}
	}
	return false
}

// compileSkipPatterns compiles a list of string patterns into regexps.
// Invalid patterns are silently skipped.
func compileSkipPatterns(patterns []string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

// zipDirectoryFiltered creates a ZIP of sourceDir including only the specified subdirectory prefixes.
// It generates a metadata.yaml with the given type and current timestamp.
// Database overrides are applied to databases/*.yaml files.
// Files matching skipPatterns are excluded from the ZIP.
func zipDirectoryFiltered(sourceDir string, overrides map[string]map[string]interface{}, includePrefixes []string, metadataType string, skipPatterns []*regexp.Regexp) ([]byte, error) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	base := filepath.Base(sourceDir)

	// Create root dir entry
	if _, err := w.Create(base + "/"); err != nil {
		return nil, err
	}

	err := filepath.WalkDir(sourceDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(sourceDir, p)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)

		// Skip root and metadata.yaml (we generate our own)
		if relSlash == "." || relSlash == "metadata.yaml" {
			return nil
		}

		// Skip files matching user-provided patterns
		if !d.IsDir() && shouldSkipFile(d.Name(), relSlash, skipPatterns) {
			return nil
		}

		// Check if this path is under one of the included prefixes
		included := false
		for _, prefix := range includePrefixes {
			if strings.HasPrefix(relSlash+"/", prefix) || strings.HasPrefix(relSlash, prefix) {
				included = true
				break
			}
		}
		if !included {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		zipPath := filepath.ToSlash(filepath.Join(base, rel))
		if d.IsDir() {
			_, err := w.Create(zipPath + "/")
			return err
		}

		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.HasPrefix(relSlash, "databases/") && strings.HasSuffix(relSlash, ".yaml") {
			data, _ = applyDatabaseOverrides(data, overrides)
		}
		f, err := w.Create(zipPath)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		return err
	})
	if err != nil {
		return nil, err
	}

	// Generate metadata.yaml with overridden type and current timestamp
	metaContent, err := buildMetadataFromDir(sourceDir, metadataType)
	if err != nil {
		return nil, err
	}
	metaPath := filepath.ToSlash(filepath.Join(base, "metadata.yaml"))
	f, err := w.Create(metaPath)
	if err != nil {
		return nil, err
	}
	if _, err := f.Write(metaContent); err != nil {
		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// computeFilteredFileHashes computes SHA256 hashes for files in sourceDir matching the given prefixes.
// Database overrides are applied to databases/*.yaml before hashing.
// Files matching skipPatterns are excluded.
func computeFilteredFileHashes(sourceDir string, prefixes []string, overrides map[string]map[string]interface{}, skipPatterns []*regexp.Regexp) (map[string]string, error) {
	hashes := make(map[string]string)
	err := filepath.WalkDir(sourceDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(sourceDir, p)
		relSlash := filepath.ToSlash(rel)

		// Skip files matching user-provided patterns
		if shouldSkipFile(d.Name(), relSlash, skipPatterns) {
			return nil
		}

		// Only include files under the specified prefixes
		included := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(relSlash, prefix) {
				included = true
				break
			}
		}
		if !included {
			return nil
		}

		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if strings.HasPrefix(relSlash, "databases/") && strings.HasSuffix(relSlash, ".yaml") {
			data, _ = applyDatabaseOverrides(data, overrides)
		}
		h := sha256.Sum256(data)
		hashes[relSlash] = fmt.Sprintf("%x", h)
		return nil
	})
	return hashes, err
}

// buildMetadataFromDir reads metadata.yaml from sourceDir, overrides the type field,
// and sets the timestamp to the current UTC time.
func buildMetadataFromDir(sourceDir string, metadataType string) ([]byte, error) {
	metaPath := filepath.Join(sourceDir, "metadata.yaml")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		// Fallback if no metadata.yaml exists — not an error, just generate one
		fallback := fmt.Sprintf("version: 1.0.0\ntype: %s\ntimestamp: '%s'\n", metadataType, time.Now().UTC().Format(time.RFC3339))
		return []byte(fallback), nil //nolint:nilerr // intentional fallback
	}

	var doc map[string]interface{}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		// Fallback if metadata.yaml is malformed — generate a clean one
		fallback := fmt.Sprintf("version: 1.0.0\ntype: %s\ntimestamp: '%s'\n", metadataType, time.Now().UTC().Format(time.RFC3339))
		return []byte(fallback), nil //nolint:nilerr // intentional fallback
	}

	doc["type"] = metadataType
	doc["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// readUUIDsFromDir reads all YAML files under sourceDir matching the given prefix
// and extracts the "uuid" field from each. Used for deletion on destroy.
func readUUIDsFromDir(sourceDir string, prefix string) ([]string, error) {
	var uuids []string
	targetDir := filepath.Join(sourceDir, filepath.FromSlash(strings.TrimSuffix(prefix, "/")))

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return nil, nil
	}

	err := filepath.WalkDir(targetDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".yaml") {
			return nil
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		var doc struct {
			UUID string `yaml:"uuid"`
		}
		if err := yaml.Unmarshal(data, &doc); err == nil && doc.UUID != "" {
			uuids = append(uuids, doc.UUID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return uuids, nil
}
