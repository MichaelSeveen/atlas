# S02 safe-primitives verification report

## Evidence identity

| Field | Value |
|---|---|
| Evidence version | `S02-primitives-report` v1 |
| Verification date / revalidation date | `2026-07-20` / revalidate after every source, toolchain, policy, or fixture change |
| Requirements | `FND-005`, `FND-006` |
| Threats | `THR-001`, `THR-002`, `THR-009`, `THR-029`, `THR-030`, `THR-031` |
| Adversarial cases (S02 primitive facets only) | `ADV-LED-004`, `ADV-WEB-007`, `ADV-WEB-008`, `ADV-PRV-001`, `ADV-REL-010` |
| Base revision | `a59c45e209279dae66e7b20fec7193bc6c8a8645` |
| Source revision | `UNCOMMITTED_WORKTREE(base=a59c45e209279dae66e7b20fec7193bc6c8a8645)` |
| Go toolchain | language baseline `1.25.0`; toolchain `1.25.7` |
| Toolchain scope | Go only; no Node.js or package manager added |

The source-revision value is intentionally not a commit claim. S02 was implemented and verified after the last committed S01 baseline, and this turn did not authorize creating or pushing an S02 commit. This report, its transcripts, code, tests, and documentation belong to the same uncommitted worktree and must be revalidated against the resulting commit if an owner later authorizes one.

## Scope and results

S02 adds six narrowly owned packages under `internal/platform`: bounded integer money/currency, cryptographically random opaque identifiers, UTC clock abstraction, actor context, correlation/causation context, and stable data-minimizing domain errors. It extends the existing architecture scanner with domain-only floating-money and direct-wall-clock bans. It adds no endpoint, schema, broker, identity integration, wallet UI, ledger/posting behavior, provider behavior, or financial workflow.

| Proof | Expected | Observed |
|---|---|---|
| `go test ./...` | All repository tests pass and clean source has no architecture-policy violation. | PASS |
| `go build ./cmd/api ./cmd/worker ./cmd/simulator` | Each inert process entry point remains independently buildable. | PASS |
| `go test ./internal/platform/... -count=1` | All primitive table, boundary, property, JSON, deterministic-clock, sanitization, and seed-corpus tests pass. | PASS |
| Static canary command | Float-money and domain wall-clock fixtures are rejected; safe controls remain accepted. | PASS; see `S02-static-canaries.txt`. |
| Three `-fuzztime=100x` campaigns | Every committed baseline corpus completes, followed by exactly 100 bounded executions per target with no failure. | PASS; see `S02-fuzz-summary.txt`. |
| `scripts/test-s02-mutation.ps1` | Removing the `Amount.Add` currency guard causes the targeted invariant test to fail. | PASS; mutant KILLED; see `S02-mutation-result.txt`. |
| Repository layout / PRD integrity | S02 paths exist; all canonical manifest entries match; all eleven retained root copies remain identical. | PASS. |

Integrity digests captured after the traceability/manifest update:

```text
51fb1c9c13b8b416ce2aba8f9fe09d1b010c80200d7a99db8e7ecacbe5133b03  docs/atlas-prd/MANIFEST.sha256
60c1540f96b33435ab9daa4193e8fb4aab69e1b9f20136ad7ba9519b84fb1fc2  evidence/phase-00/primitives/S02-static-canaries.txt
c6478a97f82931d214cf0d4d296c21e63003c7862bfc53a2ace8b92c5eca0a3e  evidence/phase-00/primitives/S02-fuzz-summary.txt
11caecf43ca3ae20c51a435f94c80768ab8e7451caacf77fa75d8fb7aef04b4c  evidence/phase-00/primitives/S02-mutation-result.txt
```

## Boundary and abuse coverage

- Money parsing rejects maximum-plus-one, `math.MinInt64`, leading signs/zeroes, decimal points, whitespace, comma locale formatting, non-ASCII digit formatting, JSON numeric amounts, extra fields, absent currency, and unsupported currency. Amount arithmetic rejects cross-currency operations and overflow; `math/big` is an independent fuzz oracle.
- JSON fixtures include `9007199254740993`, immediately beyond JavaScript's exact safe-integer range, and retain it as a decimal string. The Go-only scope deliberately does not add a JavaScript/TypeScript runtime to prove this invariant.
- Opaque IDs enforce the canonical prefix/Crockford pattern, use 128 cryptographic random bits when generated, reject malformed/zero wire values, and never echo attacker input in errors.
- `Fixed` clock tests normalize UTC and exercise an exact expiry boundary so callers can define deterministic inclusive/exclusive semantics without wall-clock dependence.
- Actor and correlation primitives require explicit validated IDs. Correlation telemetry projection is bounded to request/correlation/causation IDs; there is no arbitrary attacker-controlled metadata map.
- Domain errors expose only validated stable codes, a closed kind, and a retry hint. They contain no free-form message, wrapped cause, or metadata that could leak personal or secret values.
- Syntax canaries cover direct float fields/types/literals/conversions/results, aliased `time.Now`, dot-imported `Now`, a safe non-money measurement, and the sole system-clock adapter.

## Limitations and follow-up

1. Evidence is based on an uncommitted worktree. Re-run the verifier and add a new evidence version after an authorized commit; do not rewrite this report.
2. The frontend package remains an empty ownership marker and the user required S02 to remain Go-only. The decimal-string fixture is ready for a future frontend consumer, but no TypeScript helper, Node.js runtime, dependency manifest, or package manager was added.
3. The canonical opaque-ID regex excludes `I`, `L`, `O`, and `U`, while several current OpenAPI/AsyncAPI examples use an `ATLAS` mnemonic containing `L`. S02 follows the normative regex and records the example defect for contract-first correction in S03.
4. The float-money rule is intentionally conservative and syntax/name based so non-financial measurements can remain floats. It is enforceable on the current domain tree but cannot prove a deliberately misleading name is non-monetary; review remains mandatory.
5. Protected CI enforcement is absent and remains `FND-020`/S07 work. Passing local static tests does not claim host branch protection.
6. This requirement-scoped result does not complete Phase 00 or any S03+ requirement.
7. `ADV-PRV-001` coverage here proves only that the domain-error primitive cannot carry arbitrary sensitive detail; full logger/trace/breadcrumb sink scanning remains `FND-041`/S06. `ADV-REL-010` coverage proves deterministic UTC clock injection and an exact time boundary only; durable ordering and multi-node tolerance require later workflows.

## Reproduction and sanitization

From repository root:

```powershell
pwsh -NoProfile -File ./scripts/verify-s02.ps1
```

The command uses ignored repository-local Go build/module caches and replays S01 before S02. Evidence contains only synthetic identifiers, currencies, integer values, static source snippets, public repository metadata, and tool versions. No secrets, credentials, tokens, customer data, production endpoints, runtime payloads, or personal records were used or retained.
