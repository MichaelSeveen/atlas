// Package persistence contains Identity-owned PostgreSQL adapters.
package persistence

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

const membershipByIDSQL = `
SELECT membership_id, tenant_id, principal_id, role_id, status, authorization_version, version
FROM atlas_identity.memberships
WHERE tenant_id = $1 AND membership_id = $2`

var (
	ErrMembershipNotFound = domainerror.New(
		domainerror.MustCode("MEMBERSHIP_NOT_FOUND"),
		domainerror.KindNotFound,
		false,
	)
	ErrMembershipReadFailed = domainerror.New(
		domainerror.MustCode("MEMBERSHIP_READ_FAILED"),
		domainerror.KindUnavailable,
		true,
	)
)

// Queryer is the narrow database capability required by this repository.
type Queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

// Membership is the authority-bearing subset returned by a tenant-scoped lookup.
type Membership struct {
	MembershipID         identifier.ID
	TenantID             identifier.ID
	PrincipalID          identifier.ID
	RoleID               string
	Status               string
	AuthorizationVersion int64
	Version              int64
}

// MembershipRepository owns tenant-scoped membership reads.
type MembershipRepository struct {
	queryer Queryer
}

// NewMembershipRepository constructs a repository over an explicit query capability.
func NewMembershipRepository(queryer Queryer) MembershipRepository {
	return MembershipRepository{queryer: queryer}
}

// ByID reads a membership only inside the caller-supplied tenant boundary.
func (repository MembershipRepository) ByID(
	ctx context.Context,
	tenant identity.TenantContext,
	membershipID identifier.ID,
) (Membership, error) {
	if repository.queryer == nil || tenant.IsZero() || membershipID.IsZero() || membershipID.Prefix() != "mem" {
		return Membership{}, identity.ErrTenantContextInvalid
	}

	var membershipIDText, tenantIDText, principalIDText string
	var result Membership
	err := repository.queryer.QueryRow(
		ctx,
		membershipByIDSQL,
		tenant.TenantID().String(),
		membershipID.String(),
	).Scan(
		&membershipIDText,
		&tenantIDText,
		&principalIDText,
		&result.RoleID,
		&result.Status,
		&result.AuthorizationVersion,
		&result.Version,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Membership{}, ErrMembershipNotFound
	}
	if err != nil {
		return Membership{}, ErrMembershipReadFailed
	}
	result.MembershipID, err = identifier.Parse(membershipIDText)
	if err != nil {
		return Membership{}, ErrMembershipReadFailed
	}
	result.TenantID, err = identifier.Parse(tenantIDText)
	if err != nil {
		return Membership{}, ErrMembershipReadFailed
	}
	result.PrincipalID, err = identifier.Parse(principalIDText)
	if err != nil {
		return Membership{}, ErrMembershipReadFailed
	}
	return result, nil
}

func validateMembershipQuery(source string) error {
	normalized := strings.Join(strings.Fields(strings.ToLower(source)), " ")
	if !strings.Contains(normalized, "where tenant_id = $1 and membership_id = $2") {
		return errors.New("tenant membership query lacks its leading tenant predicate")
	}
	return nil
}
