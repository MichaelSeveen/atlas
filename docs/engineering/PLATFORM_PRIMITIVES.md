# Platform primitives and static safety policy

S02 implements only the cross-cutting Go primitives required by `FND-005` and the source checks required by `FND-006`. These packages contain no wallet, ledger, transfer, identity-provider, authorization, persistence, HTTP, broker, or frontend behavior.

## Package boundaries

| Package | Contract and invariant |
|---|---|
| `internal/platform/money` | A signed bounded `int64` minor-unit count plus a catalog-backed currency. `math.MinInt64` is excluded so negation is closed over the supported range. Arithmetic checks currency and overflow. Boundary constructors make signed, non-negative, and positive policies explicit. JSON uses an `amount_minor` canonical decimal string and a `currency` string, rejects unknown fields, and never accepts a JSON number. |
| `internal/platform/identifier` | An opaque identifier matching `^[a-z]{2,8}_[0-9A-HJKMNP-TV-Z]{20,32}$`. New identifiers use 128 bits from `crypto/rand` and a 26-character Crockford Base32 body. Parse and wire errors never echo rejected input. Prefixes are routing/type hints only and confer no permission or tenancy. |
| `internal/platform/clock` | Domain code accepts the `Clock` interface. `Fixed` is immutable and UTC-normalized for deterministic tests. `System` is the sole host-wall-clock adapter and the only foundation location permitted to call `time.Now()` directly. |
| `internal/platform/actor` | An immutable explicit actor type and opaque ID. The closed types are `customer`, `merchant_user`, `workforce`, `machine`, and `system`. A context does not authenticate, authorize, infer tenancy, or replace an authorization decision. |
| `internal/platform/correlation` | Immutable request, correlation, and optional causation IDs. `SafeFields` exposes only three bounded validated IDs and accepts no arbitrary labels or attacker-controlled metadata map. |
| `internal/platform/domainerror` | Stable uppercase machine code, closed transport-independent kind, and retry hint. The error type deliberately has no free-form message, wrapped cause, or metadata, so its safe rendering is only the stable code. |

The initial currency catalog contains explicit `NGN` and `USD` entries, both with exponent 2. Currency exponents are catalog data for boundary/display use; they are never inferred and do not permit floating-point domain arithmetic. Adding a currency requires a reviewed catalog and fixture change.

## Static bans

The repository architecture checker parses every Go file. Inside registered domain contexts it rejects:

- `float32` or `float64` declarations, conversions, literals, and function results whose names indicate money (`amount`, `money`, `balance`, `fee`, `price`, `credit`, `debit`, minor-unit, tax, gross/net, or rate terms);
- direct `time.Now` use, including aliased imports and captured function values; and
- dot imports of `time`, which would obscure a wall-clock call.

Non-financial measurement floats remain legal, and the platform clock adapter is intentionally outside the domain-rule scope. The checker is a conservative syntax policy, not a substitute for financial review: reviewers must reject obscure aliases or names used to conceal monetary floats. S07 will wire the already-reproducible checker into protected CI.

## Contract note

The canonical OpenAPI/AsyncAPI opaque-ID regex excludes the Crockford letters `I`, `L`, `O`, and `U`; several current example strings contain `L` in an `ATLAS` mnemonic. S02 follows the normative regex and deliberately rejects those examples. The example defect must be corrected contract-first in S03; this slice does not silently loosen the identifier invariant or edit product contracts.

## Reproduction

Run the complete S02 proof with:

```powershell
pwsh -NoProfile -File ./scripts/verify-s02.ps1
```

The verifier replays S01, runs all platform and architecture checks, exercises bounded fuzz campaigns, and proves the currency invariant test kills a seeded mutation. It uses repository-local ignored Go caches and adds no Node.js or package-manager toolchain.
