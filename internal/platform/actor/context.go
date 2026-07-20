// Package actor defines an explicitly passed, identity-provider-independent
// actor context for audit and authorization inputs.
package actor

import (
	"github.com/MichaelSeveen/atlas/internal/platform/domainerror"
	"github.com/MichaelSeveen/atlas/internal/platform/identifier"
)

// Type is the closed actor vocabulary shared with the canonical audit contract.
type Type string

const (
	TypeCustomer     Type = "customer"
	TypeMerchantUser Type = "merchant_user"
	TypeWorkforce    Type = "workforce"
	TypeMachine      Type = "machine"
	TypeSystem       Type = "system"
)

var ErrInvalid = domainerror.New(
	domainerror.MustCode("ACTOR_CONTEXT_INVALID"),
	domainerror.KindInvalidArgument,
	false,
)

// Valid reports whether the actor type belongs to the closed vocabulary.
func (t Type) Valid() bool {
	switch t {
	case TypeCustomer, TypeMerchantUser, TypeWorkforce, TypeMachine, TypeSystem:
		return true
	default:
		return false
	}
}

// Context is an immutable actor identity input. It does not confer permission,
// infer a tenant, or replace an authorization decision.
type Context struct {
	typ Type
	id  identifier.ID
}

// New validates an explicit actor type and opaque identifier.
func New(typ Type, id identifier.ID) (Context, error) {
	if !typ.Valid() || id.IsZero() {
		return Context{}, ErrInvalid
	}
	return Context{typ: typ, id: id}, nil
}

// Type returns the actor classification.
func (c Context) Type() Type {
	return c.typ
}

// ID returns the opaque actor identifier.
func (c Context) ID() identifier.ID {
	return c.id
}

// IsZero reports whether the context has not been initialized.
func (c Context) IsZero() bool {
	return !c.typ.Valid() || c.id.IsZero()
}
