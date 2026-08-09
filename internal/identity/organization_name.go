package identity

import (
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/unicode/norm"
)

// normalizeOrganizationName implements the ADR 0014 NFKC + case-fold
// projection. Confusable skeleton generation remains an explicit caller-owned
// input until Atlas exposes an organization-creation boundary; persistence
// rejects collisions for both projections independently.
func normalizeOrganizationName(displayName string) string {
	normalized := norm.NFKC.String(strings.TrimSpace(displayName))
	normalized = cases.Fold().String(normalized)
	return strings.Join(strings.Fields(normalized), " ")
}
