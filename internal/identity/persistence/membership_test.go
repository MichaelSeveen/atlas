package persistence

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

type captureQueryer struct {
	query string
	args  []any
	row   pgx.Row
}

func (queryer *captureQueryer) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	queryer.query = query
	queryer.args = args
	return queryer.row
}

type valueRow struct {
	values []any
	err    error
}

func (row valueRow) Scan(destinations ...any) error {
	if row.err != nil {
		return row.err
	}
	for index, destination := range destinations {
		switch target := destination.(type) {
		case *string:
			*target = row.values[index].(string)
		case *int64:
			*target = row.values[index].(int64)
		default:
			return errors.New("unsupported test scan target")
		}
	}
	return nil
}

func TestMembershipRepositoryRequiresAndUsesTenantContext(t *testing.T) {
	tenant, err := identity.NewTenantContext("ten_01JAT1AS00000000000001")
	if err != nil {
		t.Fatal(err)
	}
	membershipID, err := identifier.Parse("mem_01JAT1AS00000000000001")
	if err != nil {
		t.Fatal(err)
	}
	queryer := &captureQueryer{row: valueRow{values: []any{
		"mem_01JAT1AS00000000000001",
		"ten_01JAT1AS00000000000001",
		"usr_01JAT1AS00000000000001",
		"customer_self",
		"active",
		int64(1),
		int64(1),
	}}}
	repository := NewMembershipRepository(queryer)
	membership, err := repository.ByID(context.Background(), tenant, membershipID)
	if err != nil {
		t.Fatal(err)
	}
	if membership.TenantID.String() != tenant.TenantID().String() ||
		len(queryer.args) != 2 ||
		queryer.args[0] != tenant.TenantID().String() ||
		queryer.args[1] != membershipID.String() {
		t.Fatal("membership repository did not bind tenant before object identifier")
	}
}

func TestMissingTenantPredicateMutationIsRejected(t *testing.T) {
	if err := validateMembershipQuery(membershipByIDSQL); err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(membershipByIDSQL, "tenant_id = $1 AND ", "", 1)
	if err := validateMembershipQuery(mutated); err == nil {
		t.Fatal("membership query without tenant predicate passed the policy canary")
	}
}

func TestCrossTenantShapeIsConcealedAsNotFound(t *testing.T) {
	tenant, err := identity.NewTenantContext("ten_01JAT1AS00000000000002")
	if err != nil {
		t.Fatal(err)
	}
	membershipID, err := identifier.Parse("mem_01JAT1AS00000000000001")
	if err != nil {
		t.Fatal(err)
	}
	repository := NewMembershipRepository(&captureQueryer{row: valueRow{err: pgx.ErrNoRows}})
	if _, err := repository.ByID(context.Background(), tenant, membershipID); !errors.Is(err, ErrMembershipNotFound) {
		t.Fatalf("cross-tenant miss = %v, want concealed not found", err)
	}
}
