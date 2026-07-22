# S08 Phase 00 acceptance report

- **Evidence ID:** EVD-P00-S08-001
- **Created:** 2026-07-22T15:37:21Z
- **Source revision:** `UNCOMMITTED_WORKTREE(base=f6ad53553e739ea44718cc1336920a37c3fd05bc)`
- **Base tree:** `d6a0d97d4905d35b263faf85c0cb23a3420f365c`
- **Requirements:** all 37 Phase 00 requirements reviewed; focused proof for FND-010, FND-020..024, FND-040..043, and FND-064
- **Threats:** THR-009, THR-013, THR-014, THR-019, THR-020, THR-025, THR-027, THR-030, THR-042, THR-045, THR-054, and THR-060
- **Named skipped tests:** #1..#3 and #5..#9 preserved; #10 added; #4 explicitly not applicable until an outbox/worker claim protocol exists
- **Synthetic seeds:** evidence-content tamper, stale catalogue source, one-connection concurrent readiness pressure, and all preserved S01–S07 canaries
- **Result:** PASS for implemented S08 static, history/security, supply-chain, live, and restore controls; Phase 00 completion is not claimed
- **Revalidate by:** 2026-08-22 or on any source, evidence, workflow, contract, dependency, migration, configuration, image, scanner, recovery, telemetry, ownership, or release change

## Boundaries

S08 adds only acceptance, evidence-integrity, and test orchestration. It does not change an API/event contract or introduce a product endpoint, schema, broker stream, identity integration, financial state, wallet behavior, outbox/inbox, job, or provider. Authorization, tenant, idempotency, money, transaction, and before/after-commit semantics remain unchanged.

The constrained-pool test uses only the existing feature-free migration catalogue and the `atlas_api` readiness role. The restore procedure uses the internal-only recovery service and synthetic marker; it cannot write into the active database namespace.

## Reproduction

```powershell
pwsh -NoProfile -File ./scripts/verify-s08.ps1
pwsh -NoProfile -File ./scripts/verify-s08.ps1 -Live -History -SupplyChain -ContainerRuntime podman
# After commit, from a clean tree:
pwsh -NoProfile -File ./scripts/verify-s08.ps1 -Live -History -SupplyChain -CleanClone -ContainerRuntime podman
```

Expected: S01–S07 static/contract/security controls pass; the evidence catalogue matches exact artifact hashes and rejects disposable content/source mutations; the full stack starts; a fixed W3C trace covers API/readiness/database; collector outage leaves readiness authoritative; real PostgreSQL/NATS roles and migrations pass; long-lock abort and isolated recovery pass; 24 concurrent readiness requests complete through one database connection; teardown runs even on failure.

Observed static result on the Windows host:

```text
s07_verification=PASS
s08_evidence_tamper_canary=PASS
s08_evidence_stale-source_canary=PASS
s08_evidence_integrity=PASS
s08_phase_00_completion=NOT_CLAIMED
s08_verification=PASS
```

Observed history/security and supply-chain result:

```text
complete_history_commits=16 no_leaks_found=true
s07_deleted_history_secret_canary=PASS
reachable_vulnerabilities=0
s07_sbom_surfaces=backend-source,frontend-source,backend-image,web-image
s07_vulnerability_threshold=critical
s07_image_runtime=non-root,read-only,cap-drop,no-new-privileges
s07_supply_chain=PASS
```

Observed live result:

```text
s04_live_smoke=PASS
golden_trace_spans=api,readiness,database
telemetry_outage_readiness=200
database_long_lock_abort=PASS elapsed_seconds=1
database_integration_broker=REAL_NATS_JETSTREAM
database_base_backup=PASS duration_seconds=27
database_wal_archive=PASS
database_isolated_pitr_restore=PASS
database_restore_rto_seconds=62
TestMostAgentsSkip10ConstrainedDatabasePool=PASS duration_seconds=0.10 pool_connections=1
s08_constrained_pool_race=NOT_AVAILABLE(cgo-disabled-host;required-in-hosted-Linux-lane)
s08_live_stack_trace_restore=PASS
s04_environment_down=PASS
s06_environment_down=PASS
s08_verification=PASS
```

## Hosted S07 prerequisite

PR #1 run `29928153984` passed all five configured Linux jobs against `747cc80f058d570851f64592c0eb3a9ca0e33adc`, then merged as `f6ad53553e739ea44718cc1336920a37c3fd05bc`. This closes the previously absent hosted-execution evidence. It does not close FND-020 enforcement: the rulesets API returned an empty list. No release workflow run exists, so FND-023 registry promotion, FND-024 signature/provenance, and FND-026 enforced independent code-owner review remain partial.

## Tests and adversarial disposition

The exact ten-test matrix and `ADV-REL`/`ADV-DR` applicability review are in `docs/engineering/PHASE-00-ACCEPTANCE.md`. Test #4 is not implemented because the repository contains no outbox table or worker claim protocol and Phase 00 forbids inventing those semantics. Test #10 uses a real migrated PostgreSQL instance, a pool maximum of one, 24 simultaneous calls, bounded completion, connection release, and `go test -race` whenever CGO is available. The hosted PR lane requires race support rather than accepting a skip.

## Evidence integrity and sanitization

The versioned catalogue binds the current source identity to exact SHA-256 digests for the S01–S08 reports and acceptance/limitations documents. Its verifier checks closed identities, repository-confined paths, artifact presence, hashes, and source identity; disposable mutation canaries prove changed content and a stale source are rejected.

No credential, token, connection string, customer/identity data, product payload, raw scanner database, or runtime environment file is retained. GitHub evidence contains public repository/run/job/revision identifiers only. Generated runtime state and disposable SBOM/scanner outputs remain ignored under `.tmp/`. Syft emitted its already documented non-fatal Windows temporary-directory cleanup warning after producing the image artifact.

## Limitations and completion decision

This evidence is pre-commit and cannot identify the final S08 tree. Clean-clone execution is intentionally unavailable until the implementation is committed. The Windows host has CGO disabled, so local race proof must come from the required hosted Linux lane. During teardown Podman waited ten seconds for the stateless Bun web container and then used SIGKILL; the isolated environment still completed its down path, but graceful web termination is not claimed. Independent clean-machine, branch-ruleset, code-owner approval, GHCR digest promotion, keyless signature/provenance, managed secret custody, encrypted backup storage, alert routing, and nonexistent product replay state remain open in `docs/engineering/PHASE-00-KNOWN-LIMITATIONS.md`.

S08 implementation passing does not by itself pass the Phase 00 gate. A post-commit evidence revision and the applicable external gates are required before changing the phase state to complete.
