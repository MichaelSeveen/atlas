package persistence

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrganizationNameProjectionRealPostgresRejectsUnicodeConfusableCollisions(t *testing.T) {
	migrationURL := os.Getenv("ATLAS_P01_MIGRATION_DATABASE_URL")
	if migrationURL == "" {
		t.Skip("real Phase 01 migration database URL is not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, migrationURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	now := time.Date(2026, 8, 9, 10, 0, 0, 0, time.UTC)

	for _, test := range []struct {
		name                            string
		firstDisplay, firstNormalized   string
		firstSkeleton                   string
		secondDisplay, secondNormalized string
		secondSkeleton                  string
	}{
		{
			name:         "uts39-skeleton",
			firstDisplay: "Atlas", firstNormalized: "atlas", firstSkeleton: "atlas",
			secondDisplay: "Αtlas", secondNormalized: "αtlas", secondSkeleton: "atlas",
		},
		{
			name:         "nfkc-casefold",
			firstDisplay: "Atlas Treasury", firstNormalized: "atlas treasury", firstSkeleton: "atlas treasury",
			secondDisplay: "ＡＴＬＡＳ ＴＲＥＡＳＵＲＹ", secondNormalized: "atlas treasury", secondSkeleton: "atlas treasury",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			transaction, err := pool.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = transaction.Rollback(ctx) }()
			firstTenant := newIntegrationID(t, "ten")
			secondTenant := newIntegrationID(t, "ten")
			if _, err := transaction.Exec(ctx, `
INSERT INTO atlas_identity.organizations (
    tenant_id, organization_type, display_name, normalized_name, confusable_skeleton,
    status, authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', $2, $3, $4, 'active', 1, 1, $5, $5)`,
				firstTenant.String(), test.firstDisplay, test.firstNormalized, test.firstSkeleton, now,
			); err != nil {
				t.Fatal(err)
			}
			_, err = transaction.Exec(ctx, `
INSERT INTO atlas_identity.organizations (
    tenant_id, organization_type, display_name, normalized_name, confusable_skeleton,
    status, authorization_version, version, created_at, updated_at
) VALUES ($1, 'merchant', $2, $3, $4, 'active', 1, 1, $5, $5)`,
				secondTenant.String(), test.secondDisplay, test.secondNormalized, test.secondSkeleton, now,
			)
			var databaseError *pgconn.PgError
			if !errors.As(err, &databaseError) || databaseError.Code != "23505" {
				t.Fatalf("confusable organization collision error=%v", err)
			}
		})
	}
}
