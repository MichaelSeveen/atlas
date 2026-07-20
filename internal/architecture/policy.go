package architecture

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

const (
	ruleFloatMoney = "floating-point money is forbidden; use internal/platform/money"
	ruleTimeNow    = "direct time.Now use is forbidden in domain code; inject internal/platform/clock"
	ruleDotTime    = "dot-import of time is forbidden in domain code"
)

func checkDomainPolicies(relative string, file *ast.File, fileSet *token.FileSet) []Violation {
	var violations []Violation
	timeAliases := map[string]struct{}{}
	dotImportedTime := false

	for _, imported := range file.Imports {
		path, err := strconv.Unquote(imported.Path.Value)
		if err != nil || path != "time" {
			continue
		}
		alias := "time"
		if imported.Name != nil {
			alias = imported.Name.Name
		}
		switch alias {
		case "_":
			continue
		case ".":
			dotImportedTime = true
			violations = append(violations, policyViolation(relative, fileSet, imported.Pos(), ruleDotTime))
		default:
			timeAliases[alias] = struct{}{}
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch typed := node.(type) {
		case *ast.SelectorExpr:
			alias, ok := typed.X.(*ast.Ident)
			if ok && typed.Sel.Name == "Now" {
				if _, isTime := timeAliases[alias.Name]; isTime {
					violations = append(violations, policyViolation(relative, fileSet, typed.Pos(), ruleTimeNow))
				}
			}
		case *ast.CallExpr:
			if dotImportedTime {
				if function, ok := typed.Fun.(*ast.Ident); ok && function.Name == "Now" {
					violations = append(violations, policyViolation(relative, fileSet, typed.Pos(), ruleTimeNow))
				}
			}
		case *ast.TypeSpec:
			if financialName(typed.Name.Name) && isFloatType(typed.Type) {
				violations = append(violations, policyViolation(relative, fileSet, typed.Pos(), ruleFloatMoney))
			}
		case *ast.Field:
			if isFloatType(typed.Type) {
				for _, name := range typed.Names {
					if financialName(name.Name) {
						violations = append(violations, policyViolation(relative, fileSet, name.Pos(), ruleFloatMoney))
					}
				}
			}
		case *ast.ValueSpec:
			floatValue := isFloatType(typed.Type) || expressionsUseFloat(typed.Values)
			if floatValue {
				for _, name := range typed.Names {
					if financialName(name.Name) {
						violations = append(violations, policyViolation(relative, fileSet, name.Pos(), ruleFloatMoney))
					}
				}
			}
		case *ast.AssignStmt:
			if expressionsUseFloat(typed.Rhs) {
				for _, left := range typed.Lhs {
					name, ok := left.(*ast.Ident)
					if ok && financialName(name.Name) {
						violations = append(violations, policyViolation(relative, fileSet, name.Pos(), ruleFloatMoney))
					}
				}
			}
		case *ast.FuncDecl:
			if typed.Type.Results != nil && financialName(typed.Name.Name) {
				for _, result := range typed.Type.Results.List {
					if isFloatType(result.Type) {
						violations = append(violations, policyViolation(relative, fileSet, typed.Name.Pos(), ruleFloatMoney))
						break
					}
				}
			}
		}
		return true
	})

	return violations
}

func policyViolation(relative string, fileSet *token.FileSet, position token.Pos, rule string) Violation {
	return Violation{
		File: relative,
		Line: fileSet.Position(position).Line,
		Rule: rule,
	}
}

func isFloatType(expression ast.Expr) bool {
	identifier, ok := expression.(*ast.Ident)
	return ok && (identifier.Name == "float32" || identifier.Name == "float64")
}

func expressionsUseFloat(expressions []ast.Expr) bool {
	for _, expression := range expressions {
		usesFloat := false
		ast.Inspect(expression, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.BasicLit:
				usesFloat = usesFloat || typed.Kind == token.FLOAT
			case *ast.CallExpr:
				usesFloat = usesFloat || isFloatType(typed.Fun)
			}
			return !usesFloat
		})
		if usesFloat {
			return true
		}
	}
	return false
}

func financialName(name string) bool {
	normalized := strings.ToLower(strings.ReplaceAll(name, "_", ""))
	for _, marker := range []string{
		"amount",
		"money",
		"balance",
		"fee",
		"price",
		"credit",
		"debit",
		"minorunit",
		"grossamount",
		"netamount",
		"taxamount",
		"fxrate",
		"exchangerate",
		"conversionrate",
	} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
