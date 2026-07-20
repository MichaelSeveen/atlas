package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBoundaryCheckerRejectsFloatingPointMoney(t *testing.T) {
	tests := []struct {
		name   string
		source string
	}{
		{name: "field", source: "package transfer\ntype Command struct { Amount float64 }\n"},
		{name: "type alias", source: "package transfer\ntype Money float32\n"},
		{name: "inferred variable", source: "package transfer\nvar Balance = 1.25\n"},
		{name: "short declaration", source: "package transfer\nfunc f() { fee := float64(1); _ = fee }\n"},
		{name: "financial result", source: "package transfer\nfunc calculateAmount() float64 { return 1 }\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			violations := checkPolicyFixture(t, "internal/transfer/service.go", test.source)
			if !hasRule(violations, ruleFloatMoney) {
				t.Fatalf("expected floating-money violation, got:\n%s", formatViolations(violations))
			}
		})
	}
}

func TestBoundaryCheckerRejectsDirectTimeNow(t *testing.T) {
	fixture := `package transfer
import wallclock "time"
func startedAt() { _ = wallclock.Now() }
`
	violations := checkPolicyFixture(t, "internal/transfer/service.go", fixture)
	if !hasRule(violations, ruleTimeNow) {
		t.Fatalf("expected time.Now violation, got:\n%s", formatViolations(violations))
	}
}

func TestBoundaryCheckerRejectsDotImportedTime(t *testing.T) {
	fixture := `package transfer
import . "time"
func startedAt() { _ = Now() }
`
	violations := checkPolicyFixture(t, "internal/transfer/service.go", fixture)
	if !hasRule(violations, ruleDotTime) || !hasRule(violations, ruleTimeNow) {
		t.Fatalf("expected dot-import and time.Now violations, got:\n%s", formatViolations(violations))
	}
}

func TestDomainPolicyAllowsNonMoneyFloatAndClockAdapter(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "internal/risk/metrics.go", "package risk\ntype Metrics struct { CPUUtilization float64 }\n")
	writeFixture(t, root, "internal/platform/clock/system.go", "package clock\nimport \"time\"\nfunc now() time.Time { return time.Now().UTC() }\n")

	violations, err := Check(root, "github.com/MichaelSeveen/atlas")
	if err != nil {
		t.Fatal(err)
	}
	if len(violations) != 0 {
		t.Fatalf("safe source rejected:\n%s", formatViolations(violations))
	}
}

func checkPolicyFixture(t *testing.T, relative, source string) []Violation {
	t.Helper()
	root := t.TempDir()
	writeFixture(t, root, relative, source)
	violations, err := Check(root, "github.com/MichaelSeveen/atlas")
	if err != nil {
		t.Fatal(err)
	}
	return violations
}

func writeFixture(t *testing.T, root, relative, source string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
}

func hasRule(violations []Violation, rule string) bool {
	for _, violation := range violations {
		if strings.EqualFold(violation.Rule, rule) {
			return true
		}
	}
	return false
}
