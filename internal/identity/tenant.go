// Package identity owns Atlas principals, tenancy, sessions, and authorization state.
package identity

import (
	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

var ErrTenantContextInvalid = domainerror.New(
	domainerror.MustCode("TENANT_CONTEXT_INVALID"),
	domainerror.KindInvalidArgument,
	false,
)

// TenantContext makes the authorization boundary explicit for every tenant repository call.
type TenantContext struct {
	tenantID identifier.ID
}

// NewTenantContext validates and binds an opaque tenant identifier.
func NewTenantContext(value string) (TenantContext, error) {
	tenantID, err := identifier.Parse(value)
	if err != nil || tenantID.Prefix() != "ten" {
		return TenantContext{}, ErrTenantContextInvalid
	}
	return TenantContext{tenantID: tenantID}, nil
}

// TenantID returns the validated tenant identifier.
func (context TenantContext) TenantID() identifier.ID {
	return context.tenantID
}

// IsZero reports whether the context was not constructed successfully.
func (context TenantContext) IsZero() bool {
	return context.tenantID.IsZero()
}
