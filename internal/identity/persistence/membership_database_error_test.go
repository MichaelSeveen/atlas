package persistence

import (
	"errors"
	"testing"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestMembershipMutationDatabaseErrorsClassifyTransientContention(t *testing.T) {
	t.Parallel()
	for _, code := range []string{"40001", "40P01", "55P03"} {
		t.Run(code, func(t *testing.T) {
			t.Parallel()
			databaseError := &pgconn.PgError{Code: code}
			if err := memberRoleChangeDatabaseError(databaseError); !errors.Is(err, errRetryMemberRoleChange) {
				t.Fatalf("role-change SQLSTATE %s classified as %v", code, err)
			}
			if err := memberRevocationDatabaseError(databaseError); !errors.Is(err, errRetryMemberRevocation) {
				t.Fatalf("revocation SQLSTATE %s classified as %v", code, err)
			}
		})
	}
}

func TestMembershipMutationDatabaseErrorsFailClosedForNonTransientFailures(t *testing.T) {
	t.Parallel()
	databaseError := &pgconn.PgError{Code: "22000"}
	for name, err := range map[string]error{
		"role change": memberRoleChangeDatabaseError(databaseError),
		"revocation":  memberRevocationDatabaseError(databaseError),
	} {
		if !errors.Is(err, identity.ErrIdentityUnavailable) {
			t.Fatalf("%s SQLSTATE classified as %v", name, err)
		}
	}
}
