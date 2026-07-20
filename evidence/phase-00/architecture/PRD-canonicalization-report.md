# Canonical PRD cleanup verification report

## Evidence identity

| Field | Value |
|---|---|
| Verification date | `2026-07-20` |
| S02 implementation revision | `dc638d2949335fc5808aea39906618406cd5c042` |
| Verified cleanup source revision | `240adbf32b73951062b9e2233e1aa6257b4d386d` |
| Branch / remote | `main` / `https://github.com/MichaelSeveen/atlas.git` |
| Toolchain | Go language `1.25.0`; Go toolchain `1.25.7`; Go-only verification |
| Scope | Remove the eleven non-authoritative root PRD/contract copies; preserve `docs/atlas-prd/` as the sole canonical pack |

This evidence is stored in a subsequent evidence-only revision so it can name the already committed and tested cleanup source without a self-referential commit hash. No S03 implementation or later Phase 00 completion is claimed.

## Pre-deletion safety proof

Immediately before deletion, a repository-root-contained ordered map resolved each target and canonical path, required every target to be a direct root child, required every canonical path to exist under `docs/atlas-prd/`, and compared SHA-256 values. Observed result: `verified_duplicate_pairs=11`; every pair reported `match=True`.

The verified/deleted root paths were:

```text
00_PRODUCT_CHARTER.md
00_SYSTEM_ARCHITECTURE.md
01_SECURITY_AND_TRUST_MODEL.md
02_DATA_ARCHITECTURE_AND_LEDGER_MODEL.md
ADVERSARIAL_TEST_CATALOG.md
CONTENT_CALENDAR.md
PHASE-03_LEDGER_CORE.md
REQUIREMENTS_TRACEABILITY.csv
THREAT_REGISTER.csv
asyncapi.yaml
openapi.yaml
```

No path under `docs/atlas-prd/` was deleted. Git reports exactly eleven deletions in cleanup revision `240adbf`; its parent `dc638d2` remains the recoverable historical source for the removed copies.

## Post-commit verification

Command:

```powershell
pwsh -NoProfile -File ./scripts/verify-s02.ps1
```

Observed at clean source revision `240adbf32b73951062b9e2233e1aa6257b4d386d`:

- `TestNoRootPRDDuplicates`: PASS; every canonical target exists and all eleven root paths are absent.
- `TestCanonicalPRDManifest`: PASS; 58/58 manifest entries match across the 59-file canonical pack (manifest excludes itself).
- S01 repository tests, process builds, layout, import boundaries, and forbidden-import canary: PASS.
- All six S02 platform packages and static float-money/wall-clock canaries: PASS.
- `FuzzParseMinorUnits`, `FuzzCheckedAddition`, and identifier `FuzzParse`: PASS with exactly 100 executions each after baseline corpus replay.
- Currency-guard mutation: `KILLED`.
- Final verifier marker: `s02_verification=PASS`.

## Result, recovery, and sanitization

Result: `docs/atlas-prd/` is the single mutable PRD/contract source, and a repository test now rejects reintroduction of any historical root duplicate. The cleanup is recoverable from Git history, but restoring root copies would intentionally fail architecture verification.

Evidence contains public repository paths, source revisions, tool versions, and synthetic test results only. It contains no secrets, credentials, tokens, personal data, customer records, production endpoints, or runtime payloads. Revalidate after changes to canonical ownership, the manifest, repository layout tests, or verification scripts.
