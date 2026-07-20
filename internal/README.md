# Internal modules

Each first-level directory under `internal/` is an Atlas bounded context or a narrowly scoped foundation package. Contexts own their domain behavior and future authoritative tables. Cross-context use is limited to the target context's package root or `application` API; persistence and private subpackages are never cross-imported.

`platform/` is reserved for the small primitives explicitly allowed by ADR 0001. `architecture/` contains source-boundary tooling only. Empty context directories are S01 ownership markers, not implemented capabilities.

See [module boundaries](../docs/engineering/MODULE_BOUNDARIES.md) and run `go test ./internal/architecture -count=1` after changing imports or layout.
