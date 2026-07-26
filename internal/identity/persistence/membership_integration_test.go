package persistence

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

func TestMembershipRepositoryRealPostgresTenantIsolation(t *testing.T) {
	databaseURL := os.Getenv("ATLAS_P01_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("ATLAS_P01_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close(context.Background())

	repository := NewMembershipRepository(connection)
	membershipID, err := identifier.Parse("mem_01JAT1AS00000000000001")
	if err != nil {
		t.Fatal(err)
	}
	owningTenant, err := identity.NewTenantContext("ten_01JAT1AS00000000000001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ByID(ctx, owningTenant, membershipID); err != nil {
		t.Fatalf("owning tenant lookup failed: %v", err)
	}
	otherTenant, err := identity.NewTenantContext("ten_01JAT1AS00000000000002")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.ByID(ctx, otherTenant, membershipID); !errors.Is(err, ErrMembershipNotFound) {
		t.Fatalf("valid cross-tenant ID was not concealed: %v", err)
	}
}
