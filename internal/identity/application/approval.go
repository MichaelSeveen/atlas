package application

import (
	"github.com/MichaelSeveen/atlas/internal/audit"
	"github.com/MichaelSeveen/atlas/internal/identity"
	identitypersistence "github.com/MichaelSeveen/atlas/internal/identity/persistence"
)

// NewApprovalBoundary exposes Identity's caller-transaction approval adapter
// through the context application API. Other contexts must not construct or
// import Identity persistence directly.
func NewApprovalBoundary(recorder audit.Recorder) (identity.ApprovalBoundary, error) {
	return identitypersistence.NewApprovalBoundary(recorder)
}
