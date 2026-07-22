# S08 solo-maintainer governance and ruleset verification

- **Evidence ID:** EVD-P00-S08-004
- **Created:** 2026-07-22T19:05:57Z
- **Implementation revision:** `08762a3e1333043d021264a875b8e5e222e9c34c`
- **Hosted source revision:** `8c1032333356fe2d10b91ab46328f0a187290024`
- **Pull request:** [MichaelSeveen/atlas#19](https://github.com/MichaelSeveen/atlas/pull/19)
- **Workflow run:** [atlas-pr run 29949126130](https://github.com/MichaelSeveen/atlas/actions/runs/29949126130)
- **Ruleset:** [main-solo-maintainer-protection #19577130](https://github.com/MichaelSeveen/atlas/rules/19577130)
- **Requirements:** verifies the hosted required-check facet of FND-020 and the active compensating-control facet of FND-026; FND-026 remains an accepted solo-maintainer deviation, not independent-review proof
- **Risk/threats:** RSK-031; THR-007; THR-013; THR-014; THR-018; THR-020; THR-040; THR-060
- **Result:** PASS for the closed policy, seeded failures, sensitive PR declaration, all five hosted jobs, and active `main` rules; independent human review remains unavailable and is not claimed
- **Revalidate by:** 2026-08-22 or on any policy, workflow, required context, repository rule, owner, scope, provider/data boundary, or release change

## Hosted sensitive-change and CI proof

| Job | Job ID | Duration | Result |
|---|---:|---:|---|
| `static-contracts-secret-history` | 89022080193 | 1m56s | PASS |
| `postgres-nats-migration-lanes` | 89022080251 | 1m38s | PASS |
| `codeql-go-typescript-go` | 89022080227 | 1m44s | PASS |
| `codeql-go-typescript-javascript-typescript` | 89022080521 | 1m05s | PASS |
| `sbom-vulnerability-license-container` | 89022080432 | 3m03s | PASS |

The static job inspected the full PR diff and reported sensitive paths in `.github/` plus `scripts/verify-s07.ps1` and `scripts/verify-s08.ps1`. Its exact bounded markers were:

```text
solo_governance_attestations=PASS
solo_governance_independent_review=UNAVAILABLE_NOT_CLAIMED
solo_governance_seeded_canaries=PASS
solo_governance_verification=PASS
```

The repository-owned negative canary rejects an incomplete declaration. The checked PR declaration states that Codex performed an automated/fresh-context review under owner authorization; it does not attribute an independent human approval to MichaelSeveen, Fennie8, or any other identity.

## Active `main` ruleset proof

The repository API created ruleset `19577130` at 2026-07-22T20:04:39+01:00 and subsequently returned:

- target `branch`, condition `~DEFAULT_BRANCH`, enforcement `active`;
- rule types `deletion`, `non_fast_forward`, `pull_request`, and `required_status_checks`;
- pull-request merge method `merge`, conversation resolution required, zero approvals, code-owner approval disabled, and latest-push approval disabled while ADR 0012 solo mode is active;
- strict required contexts matching the five successful jobs above;
- an empty bypass actor list and `current_user_can_bypass=never`;
- the branch-rules endpoint for `main` returned the same four applicable rule types.

Zero approvals is intentional and bounded: the sole owner cannot create independent judgment by approving through another controlled account. ADR 0012 prohibits real money, real personal/identity data, production credentials/providers, and production-ready or independently-reviewed claims while this deviation is active. Genuine qualified review becomes release-blocking at any policy revalidation trigger.

## Reproduce

```powershell
pwsh -NoProfile -File ./scripts/test-solo-maintainer-governance.ps1
go test ./internal/architecture -count=1
pwsh -NoProfile -File ./scripts/verify-s08.ps1
gh run view 29949126130
gh api repos/MichaelSeveen/atlas/rulesets/19577130
gh api repos/MichaelSeveen/atlas/rules/branches/main
```

## Sanitization and limitations

This derivative retains public repository/ruleset/run/job/revision identities, policy markers, and bounded results only. It contains no token, credential, connection string, customer data, identity data, product payload, scanner database, or complete job log.

The ruleset prevents an ordinary direct update, force push, or deletion through the configured branch policy, but the personal repository owner retains administrative power to edit or delete the ruleset. GitHub history and future rule queries detect that change; they cannot create organizational separation of duties. Independent clean-host, hosted release/GHCR/signature/provenance, encrypted backup custody, alert routing, and product-deferred gaps remain outside this evidence.
