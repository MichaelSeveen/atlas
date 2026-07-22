// Package contractcompat validates canonical contracts and rejects removals
// without creating a second mutable contract source.
package contractcompat

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type document map[string]any

// Lint validates the document kind and every internal reference.
func Lint(path string) error {
	doc, err := load(path)
	if err != nil {
		return err
	}
	if _, err := kind(doc); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	var refs []string
	collectRefs(doc, &refs)
	for _, ref := range refs {
		if !strings.HasPrefix(ref, "#/") {
			return fmt.Errorf("%s: external reference %q is not allowed", path, ref)
		}
		if !resolves(doc, ref) {
			return fmt.Errorf("%s: reference %q does not resolve", path, ref)
		}
	}
	return nil
}

// Compare rejects removal of a compatibility-significant API or event surface.
// Additions still require CODEOWNERS review.
func Compare(baselinePath, candidatePath string) error {
	baseline, err := load(baselinePath)
	if err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	candidate, err := load(candidatePath)
	if err != nil {
		return fmt.Errorf("candidate: %w", err)
	}
	baseKind, err := kind(baseline)
	if err != nil {
		return fmt.Errorf("baseline: %w", err)
	}
	candidateKind, err := kind(candidate)
	if err != nil {
		return fmt.Errorf("candidate: %w", err)
	}
	if baseKind != candidateKind {
		return fmt.Errorf("contract kind changed from %s to %s", baseKind, candidateKind)
	}
	var surfaces [][]string
	if baseKind == "openapi" {
		surfaces = openAPISurfaces(baseline)
	} else {
		surfaces = asyncAPISurfaces(baseline)
	}
	var removed []string
	for _, path := range surfaces {
		if !hasPath(candidate, path) {
			removed = append(removed, strings.Join(path, "/"))
		}
	}
	if len(removed) != 0 {
		sort.Strings(removed)
		return fmt.Errorf("breaking removals: %s", strings.Join(removed, ", "))
	}
	return Lint(candidatePath)
}

func load(path string) (document, error) {
	// #nosec G304 -- contractctl accepts only local repository contract paths in this trust boundary.
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if strings.ContainsRune(string(content), '\t') {
		return nil, fmt.Errorf("tab indentation is forbidden")
	}
	var doc document
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("parse YAML: %w", err)
	}
	return doc, nil
}

func kind(doc document) (string, error) {
	if version, ok := doc["openapi"].(string); ok {
		if version != "3.1.1" {
			return "", fmt.Errorf("OpenAPI version must be 3.1.1, got %q", version)
		}
		if !nonEmptyMap(doc["paths"]) {
			return "", fmt.Errorf("OpenAPI paths must be non-empty")
		}
		return "openapi", nil
	}
	if version, ok := doc["asyncapi"].(string); ok {
		if version != "3.0.0" {
			return "", fmt.Errorf("AsyncAPI version must be 3.0.0, got %q", version)
		}
		if !nonEmptyMap(doc["channels"]) {
			return "", fmt.Errorf("AsyncAPI channels must be non-empty")
		}
		return "asyncapi", nil
	}
	return "", fmt.Errorf("expected OpenAPI or AsyncAPI root version")
}

func nonEmptyMap(value any) bool {
	m, ok := asMap(value)
	return ok && len(m) != 0
}

func asMap(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case document:
		return map[string]any(typed), true
	default:
		return nil, false
	}
}

func collectRefs(value any, refs *[]string) {
	if m, ok := asMap(value); ok {
		for key, child := range m {
			if key == "$ref" {
				if ref, ok := child.(string); ok {
					*refs = append(*refs, ref)
				}
			}
			collectRefs(child, refs)
		}
		return
	}
	if list, ok := value.([]any); ok {
		for _, child := range list {
			collectRefs(child, refs)
		}
	}
}

func resolves(doc document, ref string) bool {
	parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
	var current any = doc
	for _, escaped := range parts {
		part := strings.ReplaceAll(strings.ReplaceAll(escaped, "~1", "/"), "~0", "~")
		m, ok := asMap(current)
		if !ok {
			return false
		}
		current, ok = m[part]
		if !ok {
			return false
		}
	}
	return true
}

func openAPISurfaces(doc document) [][]string {
	var result [][]string
	paths, _ := asMap(doc["paths"])
	for path, pathValue := range paths {
		result = append(result, []string{"paths", path})
		methods, _ := asMap(pathValue)
		for method, operationValue := range methods {
			if !httpMethod(method) {
				continue
			}
			result = append(result, []string{"paths", path, method})
			operation, _ := asMap(operationValue)
			responses, _ := asMap(operation["responses"])
			for status := range responses {
				result = append(result, []string{"paths", path, method, "responses", status})
			}
		}
	}
	appendSchemaSurfaces(doc, []string{"components", "schemas"}, &result)
	return result
}

func asyncAPISurfaces(doc document) [][]string {
	var result [][]string
	for _, root := range []string{"channels", "operations"} {
		values, _ := asMap(doc[root])
		for name := range values {
			result = append(result, []string{root, name})
		}
	}
	components, _ := asMap(doc["components"])
	messages, _ := asMap(components["messages"])
	for name := range messages {
		result = append(result, []string{"components", "messages", name})
	}
	appendSchemaSurfaces(doc, []string{"components", "schemas"}, &result)
	return result
}

func appendSchemaSurfaces(doc document, root []string, result *[][]string) {
	value, ok := lookup(doc, root)
	if !ok {
		return
	}
	schemas, _ := asMap(value)
	for name, schemaValue := range schemas {
		schemaPath := appendPath(root, name)
		*result = append(*result, schemaPath)
		schema, _ := asMap(schemaValue)
		properties, _ := asMap(schema["properties"])
		for property := range properties {
			*result = append(*result, appendPath(schemaPath, "properties", property))
		}
		for _, required := range stringList(schema["required"]) {
			*result = append(*result, appendPath(schemaPath, "required", required))
		}
	}
}

func stringList(value any) []string {
	list, _ := value.([]any)
	result := make([]string, 0, len(list))
	for _, item := range list {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}
	return result
}

func hasPath(doc document, path []string) bool {
	if len(path) >= 2 && path[len(path)-2] == "required" {
		value, ok := lookup(doc, path[:len(path)-1])
		if !ok {
			return false
		}
		for _, item := range stringList(value) {
			if item == path[len(path)-1] {
				return true
			}
		}
		return false
	}
	_, ok := lookup(doc, path)
	return ok
}

func lookup(doc document, path []string) (any, bool) {
	var current any = doc
	for _, part := range path {
		m, ok := asMap(current)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func appendPath(path []string, parts ...string) []string {
	result := append([]string(nil), path...)
	return append(result, parts...)
}

func httpMethod(value string) bool {
	switch value {
	case "get", "put", "post", "delete", "options", "head", "patch", "trace":
		return true
	default:
		return false
	}
}
