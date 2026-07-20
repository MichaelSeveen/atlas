# ADR 0006 — Money uses integer minor units and explicit currency metadata

- **Status:** Accepted
- **Date:** 2026-07-20
- **Related requirements:** FND-005, FND-006, LED-011 through LED-013, LED-050 through LED-053

## Context

Binary floating point cannot represent many decimal amounts exactly. JavaScript also cannot exactly represent all large integers as `number`. Financial operations require explicit currency exponent, rounding, overflow, and conversion rules.

## Decision

- Stored and transported money is an integer count of minor units plus currency code.
- JSON encodes `amount_minor` as a decimal string.
- Go domain type wraps a bounded integer representation and currency; raw integer arithmetic is not scattered through handlers.
- TypeScript treats amount as string/BigInt-safe domain value; it never converts to unsafe `number` for business logic.
- Currency catalog defines exponent and supported operations; code does not assume all currencies have two decimals.
- Rates use decimal/rational representation with explicit precision and rounding mode.
- Rounding remainder is posted explicitly, not discarded.
- Every amount boundary validates positivity/sign policy, maximum, currency, and conversion context.

## Consequences

Contracts are slightly more verbose and UI formatting requires explicit helpers. Exactness, overflow handling, and cross-language consistency are testable.

## Rejected alternatives

- JSON number and `float64`.
- Arbitrary decimal strings with no minor-unit/currency invariant.
- Assuming two decimal places globally.

## Verification

Static rule rejecting float money; max/overflow tests; Go/TypeScript contract fixtures; locale/large-number browser tests; FX/rounding properties.
