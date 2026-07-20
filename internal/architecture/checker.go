// Package architecture enforces Atlas's source dependency boundaries.
// It is engineering tooling and must never contain domain behavior.
package architecture

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Violation describes one import that bypasses an Atlas module boundary.
type Violation struct {
	File       string
	ImportPath string
	Rule       string
}

func (v Violation) String() string {
	if v.ImportPath == "" {
		return fmt.Sprintf("%s: %s", v.File, v.Rule)
	}
	return fmt.Sprintf("%s: %s imports %s", v.File, v.Rule, v.ImportPath)
}

var domainModules = map[string]struct{}{
	"audit":          {},
	"customer":       {},
	"identity":       {},
	"ledger":         {},
	"operations":     {},
	"payment":        {},
	"provider":       {},
	"reconciliation": {},
	"reporting":      {},
	"risk":           {},
	"settlement":     {},
	"transfer":       {},
	"wallet":         {},
}

var foundationModules = map[string]struct{}{
	"architecture": {},
	"platform":     {},
}

var processCommands = map[string]struct{}{
	"api":       {},
	"simulator": {},
	"worker":    {},
}

var forbiddenSharedModules = map[string]struct{}{
	"common": {},
	"models": {},
	"shared": {},
}

var skippedDirectories = map[string]struct{}{
	".git":     {},
	".tmp":     {},
	"evidence": {},
	"vendor":   {},
}

// Check scans Go imports under root and returns deterministic boundary violations.
func Check(root, modulePath string) ([]Violation, error) {
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		return nil, fmt.Errorf("module path is required")
	}

	var violations []Violation
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root {
				if _, skip := skippedDirectories[entry.Name()]; skip {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}

		relative, err := filepath.Rel(root, path)
		if err != nil {
			return fmt.Errorf("make %q relative to %q: %w", path, root, err)
		}
		relative = filepath.ToSlash(relative)
		sourceModule, sourceKind := sourceOwnership(relative)
		if sourceKind == ownershipUnregistered {
			violations = append(violations, Violation{
				File: relative,
				Rule: "unregistered top-level module or command " + sourceModule,
			})
		}
		if sourceKind == ownershipForbidden {
			violations = append(violations, Violation{
				File: relative,
				Rule: "forbidden shared domain module " + sourceModule,
			})
		}

		parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return fmt.Errorf("parse imports in %q: %w", relative, err)
		}
		for _, imported := range parsed.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				return fmt.Errorf("unquote import in %q: %w", relative, err)
			}
			if violation, found := checkImport(relative, sourceModule, sourceKind, importPath, modulePath); found {
				violations = append(violations, violation)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].File != violations[j].File {
			return violations[i].File < violations[j].File
		}
		if violations[i].ImportPath != violations[j].ImportPath {
			return violations[i].ImportPath < violations[j].ImportPath
		}
		return violations[i].Rule < violations[j].Rule
	})
	return violations, nil
}

type ownership int

const (
	ownershipOther ownership = iota
	ownershipCommand
	ownershipDomain
	ownershipFoundation
	ownershipForbidden
	ownershipUnregistered
)

func sourceOwnership(relative string) (string, ownership) {
	parts := strings.Split(relative, "/")
	if len(parts) >= 2 && parts[0] == "cmd" {
		module := parts[1]
		if _, found := processCommands[module]; found {
			return module, ownershipCommand
		}
		return module, ownershipUnregistered
	}
	if len(parts) < 2 || parts[0] != "internal" {
		return "", ownershipOther
	}

	module := parts[1]
	if _, found := domainModules[module]; found {
		return module, ownershipDomain
	}
	if _, found := foundationModules[module]; found {
		return module, ownershipFoundation
	}
	if _, found := forbiddenSharedModules[module]; found {
		return module, ownershipForbidden
	}
	return module, ownershipUnregistered
}

func checkImport(file, sourceModule string, sourceKind ownership, importPath, modulePath string) (Violation, bool) {
	prefix := strings.TrimSuffix(modulePath, "/") + "/internal/"
	if !strings.HasPrefix(importPath, prefix) {
		return Violation{}, false
	}

	remainder := strings.TrimPrefix(importPath, prefix)
	parts := strings.Split(remainder, "/")
	targetModule := parts[0]

	if _, forbidden := forbiddenSharedModules[targetModule]; forbidden {
		return newViolation(file, importPath, "shared domain dumping-ground import is forbidden"), true
	}
	_, targetIsDomain := domainModules[targetModule]
	if !targetIsDomain {
		if targetModule == "architecture" && sourceKind == ownershipDomain {
			return newViolation(file, importPath, "domain code cannot import architecture tooling"), true
		}
		return Violation{}, false
	}

	if sourceKind == ownershipFoundation {
		return newViolation(file, importPath, "foundation code cannot depend on a domain context"), true
	}
	if sourceKind == ownershipDomain && sourceModule == targetModule {
		return Violation{}, false
	}

	if sourceKind == ownershipDomain || sourceKind == ownershipCommand {
		if len(parts) == 1 || parts[1] == "application" {
			return Violation{}, false
		}
		return newViolation(file, importPath, "cross-context import must target the context root or application API"), true
	}

	return Violation{}, false
}

func newViolation(file, importPath, rule string) Violation {
	return Violation{File: file, ImportPath: importPath, Rule: rule}
}
