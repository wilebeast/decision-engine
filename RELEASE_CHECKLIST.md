# Release checklist

- [ ] Confirm authorization from all relevant source-code rights holders.
- [ ] Confirm required copyright / attribution / NOTICE text; do not invent ownership.
- [x] Repository selected: wilebeast/decision-engine; module path updated.
- [ ] Review direct and transitive dependency licenses using the final go.mod/go.sum.
- [ ] Review source and fixtures for secrets, private endpoints, personal data and policies.
- [ ] Review exported API stability and document known limitations.
- [x] Tests (including race detection), go vet ./... and the basic example pass.
- [ ] Create a fresh repository without importing private source history.
- [ ] Publish only after explicit approval; choose a pre-1.0 initial version.

The local extraction deliberately preserves existing decision semantics. It does
not claim that all production hardening or public API compatibility work is complete.
