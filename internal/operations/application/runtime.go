// Package application composes Operations-owned runtime adapters behind its
// application boundary.
package application

import (
	"context"
	"errors"

	auditapplication "github.com/MichaelSeveen/atlas/internal/audit/application"
	"github.com/MichaelSeveen/atlas/internal/identity"
	identityapplication "github.com/MichaelSeveen/atlas/internal/identity/application"
	"github.com/MichaelSeveen/atlas/internal/operations"
	operationspersistence "github.com/MichaelSeveen/atlas/internal/operations/persistence"
	"github.com/MichaelSeveen/atlas/internal/platform/database"
)

// NewRuntime creates the Operations pool and composes the typed Identity
// target boundary. Both contexts share one caller-owned transaction only while
// an approval execution is in progress.
func NewRuntime(
	ctx context.Context,
	identityService *identity.Service,
) (*operations.Service, func(), error) {
	if identityService == nil {
		return nil, nil, errors.New("identity service is required for approvals")
	}
	databaseConfig, err := database.ConfigFromEnvironment()
	if err != nil {
		return nil, nil, errors.New("invalid operations database configuration")
	}
	pool, err := database.NewApplicationPool(ctx, databaseConfig, 4)
	if err != nil {
		return nil, nil, errors.New("invalid operations database configuration")
	}
	closePool := func() { pool.Close() }
	recorder := auditapplication.NewRecorder()
	identityBoundary, err := identityapplication.NewApprovalBoundary(recorder)
	if err != nil {
		closePool()
		return nil, nil, errors.New("invalid approval identity boundary")
	}
	store, err := operationspersistence.NewStore(pool, recorder, identityBoundary)
	if err != nil {
		closePool()
		return nil, nil, errors.New("invalid approval persistence configuration")
	}
	service, err := operations.NewService(operations.ServiceOptions{
		Store: store, Identity: identityService,
	})
	if err != nil {
		closePool()
		return nil, nil, errors.New("invalid approval service configuration")
	}
	return service, closePool, nil
}
