// Package application composes Identity-owned runtime adapters behind its application boundary.
package application

import (
	"context"
	"crypto/rand"
	"errors"

	auditapplication "github.com/MichaelSeveen/atlas/internal/audit/application"
	"github.com/MichaelSeveen/atlas/internal/identity"
	oidcclient "github.com/MichaelSeveen/atlas/internal/identity/oidc"
	identitypersistence "github.com/MichaelSeveen/atlas/internal/identity/persistence"
	"github.com/MichaelSeveen/atlas/internal/platform/database"
	"github.com/MichaelSeveen/atlas/internal/platform/environment"
	metricapi "go.opentelemetry.io/otel/metric"
)

// NewRuntime composes the bounded PostgreSQL store, synthetic OIDC providers,
// transaction encryption, and CSRF protection for the API process.
func NewRuntime(
	ctx context.Context,
	config environment.Config,
	transactionKey []byte,
	csrfKey []byte,
	meter metricapi.Meter,
) (*identity.Service, func(), error) {
	databaseConfig, err := database.ConfigFromEnvironment()
	if err != nil {
		return nil, nil, errors.New("invalid identity database configuration")
	}
	pool, err := database.NewApplicationPool(ctx, databaseConfig, 4)
	if err != nil {
		return nil, nil, errors.New("invalid identity database configuration")
	}
	closePool := func() { pool.Close() }
	store, err := identitypersistence.NewSessionStore(pool, auditapplication.NewRecorder())
	if err != nil {
		closePool()
		return nil, nil, errors.New("invalid identity persistence configuration")
	}

	providers := make([]oidcclient.PopulationConfig, 0, len(config.OIDCProviders))
	policies := make(map[identity.Population]identity.SessionPolicy, len(config.OIDCProviders))
	for _, configured := range config.OIDCProviders {
		population, parseErr := identity.ParsePopulation(configured.Population)
		if parseErr != nil {
			closePool()
			return nil, nil, errors.New("invalid identity provider population")
		}
		providers = append(providers, oidcclient.PopulationConfig{
			Population: population, Issuer: configured.Issuer,
			PublicOrigin: configured.PublicOrigin, ClientID: configured.ClientID,
			RedirectURL:         configured.RedirectURL,
			SupportedAlgorithms: append([]string(nil), configured.SupportedAlgorithms...),
		})
		policy := identity.DefaultSessionPolicies[population]
		policy.AllowedReturnTo = configured.AllowedReturnTo
		policies[population] = policy
	}
	provider, err := oidcclient.NewWithMeter(providers, nil, nil, meter)
	if err != nil {
		closePool()
		return nil, nil, errors.New("invalid identity provider configuration")
	}
	cryptor, err := identity.NewAESGCMCryptor([]identity.VersionedKey{
		{Version: 1, Material: transactionKey},
	}, nil)
	if err != nil {
		closePool()
		return nil, nil, errors.New("invalid identity transaction key")
	}
	csrf, err := identity.NewHMACCSRFProtector(csrfKey)
	if err != nil {
		closePool()
		return nil, nil, errors.New("invalid identity CSRF key")
	}
	service, err := identity.NewService(identity.ServiceOptions{
		Store: store, Provider: provider, Cryptor: cryptor, CSRF: csrf,
		Entropy: rand.Reader, SessionPolicies: policies,
	})
	if err != nil {
		closePool()
		return nil, nil, errors.New("invalid identity service configuration")
	}
	return service, closePool, nil
}
