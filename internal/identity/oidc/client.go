// Package oidcclient integrates Atlas with allow-listed OpenID Connect providers.
package oidcclient

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	coreoidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/MichaelSeveen/atlas/internal/identity"
	"github.com/MichaelSeveen/atlas/internal/platform/clock"
	"go.opentelemetry.io/otel/attribute"
	metricapi "go.opentelemetry.io/otel/metric"
)

type PopulationConfig struct {
	Population          identity.Population
	Issuer              string
	PublicOrigin        string
	ClientID            string
	RedirectURL         string
	SupportedAlgorithms []string
}

type discoveredProvider struct {
	provider *coreoidc.Provider
	oauth    oauth2.Config
	config   PopulationConfig
}

type Client struct {
	configs          map[identity.Population]PopulationConfig
	clock            clock.Clock
	http             *http.Client
	mu               sync.RWMutex
	cache            map[identity.Population]*discoveredProvider
	providerCounter  metricapi.Int64Counter
	providerDuration metricapi.Float64Histogram
}

func New(configs []PopulationConfig, sourceClock clock.Clock, httpClient *http.Client) (*Client, error) {
	return NewWithMeter(configs, sourceClock, httpClient, nil)
}

func NewWithMeter(
	configs []PopulationConfig,
	sourceClock clock.Clock,
	httpClient *http.Client,
	meter metricapi.Meter,
) (*Client, error) {
	if sourceClock == nil {
		sourceClock = clock.System{}
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 5 * time.Second}
	}
	if httpClient.Timeout <= 0 || httpClient.Timeout > 10*time.Second {
		return nil, errors.New("OIDC HTTP timeout is invalid")
	}
	client := &Client{
		configs: make(map[identity.Population]PopulationConfig, len(configs)),
		clock:   sourceClock, http: httpClient,
		cache: make(map[identity.Population]*discoveredProvider, len(configs)),
	}
	for _, config := range configs {
		if err := validateConfig(config); err != nil {
			return nil, err
		}
		if _, duplicate := client.configs[config.Population]; duplicate {
			return nil, errors.New("duplicate OIDC population")
		}
		copied := config
		copied.SupportedAlgorithms = append([]string(nil), config.SupportedAlgorithms...)
		client.configs[config.Population] = copied
	}
	if len(client.configs) != 3 {
		return nil, errors.New("OIDC population configuration is incomplete")
	}
	if meter != nil {
		var err error
		client.providerCounter, err = meter.Int64Counter(
			"atlas.identity.provider.request.count",
			metricapi.WithDescription("Completed allow-listed OIDC provider requests."),
			metricapi.WithUnit("{request}"),
		)
		if err != nil {
			return nil, errors.New("create OIDC provider request counter")
		}
		client.providerDuration, err = meter.Float64Histogram(
			"atlas.identity.provider.request.duration",
			metricapi.WithDescription("Allow-listed OIDC discovery and token request duration."),
			metricapi.WithUnit("s"),
		)
		if err != nil {
			return nil, errors.New("create OIDC provider request duration")
		}
	}
	return client, nil
}

func (client *Client) AuthorizationURL(
	ctx context.Context,
	population identity.Population,
	state string,
	nonce string,
	pkceVerifier string,
	kind identity.TransactionKind,
) (string, error) {
	if !safeProtocolToken(state) || !safeProtocolToken(nonce) || !safeProtocolToken(pkceVerifier) {
		return "", errors.New("OIDC authorization input is invalid")
	}
	discovered, err := client.provider(ctx, population)
	if err != nil {
		return "", err
	}
	options := []oauth2.AuthCodeOption{
		coreoidc.Nonce(nonce),
		oauth2.S256ChallengeOption(pkceVerifier),
	}
	if kind == identity.TransactionStepUp {
		options = append(options,
			oauth2.SetAuthURLParam("prompt", "login"),
			oauth2.SetAuthURLParam("max_age", "0"),
			oauth2.SetAuthURLParam("acr_values", "2 3"),
		)
	}
	internalURL := discovered.oauth.AuthCodeURL(state, options...)
	return rewriteOrigin(internalURL, discovered.config.PublicOrigin)
}

func (client *Client) Exchange(
	ctx context.Context,
	population identity.Population,
	code string,
	pkceVerifier string,
	authorizationIssuer string,
) (_ identity.ProviderClaims, returnErr error) {
	if len(code) < 16 || len(code) > 2048 || !safeProtocolToken(pkceVerifier) {
		return identity.ProviderClaims{}, errors.New("OIDC exchange input is invalid")
	}
	config, found := client.configs[population]
	if !found {
		return identity.ProviderClaims{}, errors.New("OIDC population is not configured")
	}
	publicIssuer, err := rewriteOrigin(config.Issuer, config.PublicOrigin)
	if err != nil {
		return identity.ProviderClaims{}, identity.ErrProviderInvalid
	}
	if authorizationIssuer != "" &&
		authorizationIssuer != config.Issuer &&
		authorizationIssuer != publicIssuer {
		return identity.ProviderClaims{}, identity.ErrProviderInvalid
	}
	discovered, err := client.provider(ctx, population)
	if err != nil {
		return identity.ProviderClaims{}, err
	}
	started := client.clock.Now()
	defer func() {
		client.observeProvider(ctx, population, "token", started, returnErr)
	}()
	exchangeContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	exchangeContext = coreoidc.ClientContext(exchangeContext, client.http)
	token, err := discovered.oauth.Exchange(
		exchangeContext,
		code,
		oauth2.VerifierOption(pkceVerifier),
	)
	if err != nil {
		return identity.ProviderClaims{}, identity.ErrProviderUnavailable
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return identity.ProviderClaims{}, identity.ErrProviderInvalid
	}
	verifier := discovered.provider.VerifierContext(exchangeContext, &coreoidc.Config{
		ClientID:             discovered.config.ClientID,
		SupportedSigningAlgs: append([]string(nil), discovered.config.SupportedAlgorithms...),
		SkipIssuerCheck:      true,
		Now: func() time.Time {
			return client.clock.Now().UTC().Add(-time.Minute)
		},
	})
	idToken, err := verifier.Verify(exchangeContext, rawIDToken)
	if err != nil {
		return identity.ProviderClaims{}, identity.ErrProviderInvalid
	}
	if idToken.Issuer != discovered.config.Issuer && idToken.Issuer != publicIssuer {
		return identity.ProviderClaims{}, identity.ErrProviderInvalid
	}
	if idToken.AccessTokenHash != "" {
		if err := idToken.VerifyAccessToken(token.AccessToken); err != nil {
			return identity.ProviderClaims{}, identity.ErrProviderInvalid
		}
	}
	var claims struct {
		Nonce     string `json:"nonce"`
		ACR       string `json:"acr"`
		AuthTime  int64  `json:"auth_time"`
		NotBefore int64  `json:"nbf"`
	}
	if err := idToken.Claims(&claims); err != nil || claims.Nonce == "" || claims.AuthTime <= 0 {
		return identity.ProviderClaims{}, identity.ErrProviderInvalid
	}
	now := client.clock.Now().UTC()
	if idToken.IssuedAt.After(now.Add(time.Minute)) ||
		(claims.NotBefore > 0 && time.Unix(claims.NotBefore, 0).After(now.Add(time.Minute))) {
		return identity.ProviderClaims{}, identity.ErrProviderInvalid
	}
	assurance, err := assuranceFromACR(claims.ACR)
	if err != nil {
		return identity.ProviderClaims{}, err
	}
	return identity.ProviderClaims{
		Issuer: discovered.config.Issuer, Subject: idToken.Subject, Nonce: claims.Nonce,
		Assurance: assurance, AuthenticatedAt: time.Unix(claims.AuthTime, 0).UTC(),
	}, nil
}

func (client *Client) provider(
	ctx context.Context,
	population identity.Population,
) (_ *discoveredProvider, returnErr error) {
	client.mu.RLock()
	cached := client.cache[population]
	client.mu.RUnlock()
	if cached != nil {
		return cached, nil
	}
	config, found := client.configs[population]
	if !found {
		return nil, errors.New("OIDC population is not configured")
	}
	started := client.clock.Now()
	defer func() {
		client.observeProvider(ctx, population, "discovery", started, returnErr)
	}()
	discoveryContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	discoveryContext = coreoidc.ClientContext(discoveryContext, client.http)
	provider, err := coreoidc.NewProvider(discoveryContext, config.Issuer)
	if err != nil {
		return nil, identity.ErrProviderUnavailable
	}
	endpoint := provider.Endpoint()
	if err := validateDiscoveredEndpoint(config.Issuer, endpoint.AuthURL); err != nil {
		return nil, err
	}
	if err := validateDiscoveredEndpoint(config.Issuer, endpoint.TokenURL); err != nil {
		return nil, err
	}
	discovered := &discoveredProvider{
		provider: provider,
		oauth: oauth2.Config{
			ClientID: config.ClientID, RedirectURL: config.RedirectURL,
			Endpoint: endpoint, Scopes: []string{coreoidc.ScopeOpenID},
		},
		config: config,
	}
	client.mu.Lock()
	if existing := client.cache[population]; existing != nil {
		discovered = existing
	} else {
		client.cache[population] = discovered
	}
	client.mu.Unlock()
	return discovered, nil
}

func (client *Client) observeProvider(
	ctx context.Context,
	population identity.Population,
	operation string,
	started time.Time,
	err error,
) {
	outcome := "ok"
	if err != nil {
		outcome = "error"
	}
	duration := client.clock.Now().Sub(started).Seconds()
	if duration < 0 {
		duration = 0
	}
	attributes := []attribute.KeyValue{
		attribute.String("atlas.identity.provider.operation", operation),
		attribute.String("atlas.identity.population", string(population)),
		attribute.String("atlas.outcome", outcome),
	}
	defer func() { _ = recover() }()
	if client.providerCounter != nil {
		client.providerCounter.Add(ctx, 1, metricapi.WithAttributes(attributes...))
	}
	if client.providerDuration != nil {
		client.providerDuration.Record(ctx, duration, metricapi.WithAttributes(attributes...))
	}
}

func validateConfig(config PopulationConfig) error {
	switch config.Population {
	case identity.PopulationCustomer, identity.PopulationMerchant, identity.PopulationWorkforce:
	default:
		return errors.New("OIDC population is invalid")
	}
	issuer, err := canonicalAbsoluteURL(config.Issuer)
	if err != nil || issuer.Path == "" || strings.HasSuffix(issuer.Path, "/") {
		return errors.New("OIDC issuer is invalid")
	}
	publicOrigin, err := canonicalAbsoluteURL(config.PublicOrigin)
	if err != nil || publicOrigin.Path != "" {
		return errors.New("OIDC public origin is invalid")
	}
	redirect, err := canonicalAbsoluteURL(config.RedirectURL)
	if err != nil || redirect.Path != "/v1/auth/callback" {
		return errors.New("OIDC redirect URL is invalid")
	}
	if config.ClientID == "" || len(config.ClientID) > 128 || strings.ContainsAny(config.ClientID, " \t\r\n") {
		return errors.New("OIDC client ID is invalid")
	}
	if len(config.SupportedAlgorithms) != 1 || config.SupportedAlgorithms[0] != coreoidc.RS256 {
		return errors.New("OIDC signing algorithm allow-list must contain only RS256")
	}
	return nil
}

func canonicalAbsoluteURL(value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil ||
		parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("URL is not a canonical absolute HTTP URL")
	}
	if parsed.String() != value {
		return nil, errors.New("URL is not canonical")
	}
	return parsed, nil
}

func validateDiscoveredEndpoint(issuer, endpoint string) error {
	issuerURL, err := canonicalAbsoluteURL(issuer)
	if err != nil {
		return err
	}
	endpointURL, err := canonicalAbsoluteURL(endpoint)
	if err != nil {
		return errors.New("OIDC discovery returned an invalid endpoint")
	}
	if endpointURL.Scheme != issuerURL.Scheme || endpointURL.Host != issuerURL.Host ||
		!strings.HasPrefix(endpointURL.Path, issuerURL.Path+"/") {
		return errors.New("OIDC discovery endpoint escaped the allow-listed issuer")
	}
	return nil
}

func rewriteOrigin(value, publicOrigin string) (string, error) {
	target, err := url.Parse(value)
	if err != nil || target.Scheme == "" || target.Host == "" || target.User != nil ||
		target.Fragment != "" || (target.Scheme != "http" && target.Scheme != "https") {
		return "", errors.New("OIDC authorization endpoint is invalid")
	}
	public, err := canonicalAbsoluteURL(publicOrigin)
	if err != nil {
		return "", err
	}
	target.Scheme = public.Scheme
	target.Host = public.Host
	return target.String(), nil
}

func assuranceFromACR(value string) (identity.Assurance, error) {
	switch value {
	case "", "0", "1", "urn:atlas:assurance:baseline":
		return identity.AssuranceBaseline, nil
	case "2", "urn:atlas:assurance:stepped-up":
		return identity.AssuranceSteppedUp, nil
	case "3", "urn:atlas:assurance:phishing-resistant":
		return identity.AssurancePhishingResistant, nil
	default:
		return "", errors.New("OIDC assurance claim is not allow-listed")
	}
}

func safeProtocolToken(value string) bool {
	if len(value) < 43 || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if (character >= 'A' && character <= 'Z') ||
			(character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' {
			continue
		}
		return false
	}
	return true
}

var _ identity.Provider = (*Client)(nil)
