# CI required check failed

1. Do not merge, rerun with relaxed flags, or remove the required check. Record the job, source/base revisions, seed, first failing command, and report digest.
2. Reproduce through the repository-owned S07/S05/S06 command named by the job. Treat a difference between local and hosted results as an environment finding, not permission to bypass.
3. For contract failures, edit only the canonical contract, assess consumers and mixed-version behavior, and obtain the declared code owner. For migration failures, preserve the database/container evidence and follow the database migration runbook.
4. For secret findings, stop output sharing, revoke any real credential through its owner, sanitize retained evidence, and scan complete history. Never rewrite shared history without an approved incident plan.
5. For vulnerability/license/SAST findings, open a finding with severity, exploitability, affected digest, owner, remediation or documented mitigation, and regression command. Critical dependency findings block release.
6. Fix forward in a reviewed change and rerun every failed and dependent lane. Preserve the prior failing evidence; do not overwrite it.
7. If GitHub required checks or CODEOWNERS were disabled or bypassed, treat it as policy drift, restore the ruleset, identify merges during the window, and revalidate their revisions.
