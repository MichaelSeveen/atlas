package environment

import (
	"context"
	"errors"
)

// FlagSource is a replaceable runtime source. Configuration defaults remain
// authoritative when the source is absent or unavailable.
type FlagSource interface {
	Lookup(context.Context, string) (bool, error)
}

type FlagSourceFunc func(context.Context, string) (bool, error)

func (f FlagSourceFunc) Lookup(ctx context.Context, key string) (bool, error) {
	return f(ctx, key)
}

// FlagSet is immutable after construction and therefore safe for concurrent reads.
type FlagSet struct {
	flags map[string]FeatureFlag
}

func NewFlagSet(config Config) FlagSet {
	flags := make(map[string]FeatureFlag, len(config.FeatureFlags))
	for _, flag := range config.FeatureFlags {
		flags[flag.Key] = flag
	}
	return FlagSet{flags: flags}
}

func (s FlagSet) Enabled(ctx context.Context, key string, source FlagSource) (bool, error) {
	flag, found := s.flags[key]
	if !found {
		return false, errors.New("unknown feature flag")
	}
	if source == nil {
		return flag.Default, nil
	}
	value, err := source.Lookup(ctx, key)
	if err != nil {
		return flag.Default, nil
	}
	if flag.Risk == "high" && value != flag.Default {
		return flag.Default, errors.New("high-risk feature flag transition requires a reviewed deployment")
	}
	return value, nil
}

func (s FlagSet) RollbackBehavior(key string) (string, error) {
	flag, found := s.flags[key]
	if !found {
		return "", errors.New("unknown feature flag")
	}
	return flag.Rollback, nil
}
