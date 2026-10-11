# Decision Engine

A Go decision library built on Expr: typed value factors, expression factors,
dependency ordering, conditional decisions, and aggregated action descriptions.

## Status

Initial extraction / pre-release. This is not a hosted service or a complete risk
management product. Module path: `github.com/wilebeast/decision-engine`.
Repository: https://github.com/wilebeast/decision-engine.

## Run

Requires Go 1.24 or later.

```sh
go test ./...
go test -race ./...
go vet ./...
go run ./examples/basic
```

Expected example output:

```text
count=5 decision=pass actions=0
count=20 decision=continue actions=1
count=200 decision=block actions=0
```

## Model and execution

`Config → EventDefinition → StrategyDefinition → Rule / Decision / Action`

1. Define VALUE factors using `ParamKey` and EXPRESSION factors using Expr.
2. Compile configuration once with `engine.Compile`; dependency errors fail compilation.
3. Supply facts using `RiskInput.Param` or a server-owned `ParamReader` that maps keys
   to typed context getters. Keep data loading outside the engine.
4. Call `Snapshot.Execute(ctx, input)` and check its error before consuming decisions.
5. Interpret returned actions in the calling application. The library does not execute them.

Matching strategies are aggregated, not reduced to a priority winner. `block`
dominates `continue`, which dominates `pass`; blocked evaluations return no final
actions. Decision cases use first-match order, then the configured fallback.
No matched strategy yields `pass`. This is a computation result, not an approval.

The `model` package (Go package name `risk`, retained for compatibility) provides
fact/result types and actions. INT, FLOAT, BOOL, STRING, STRINGS, DATE and OBJECT
value declarations are supported. OBJECT validation belongs to the caller.
`model.ParamValue` distinguishes missing facts from read errors; missing results
are not equivalent to false. Guard potentially missing values in expressions.
An execution error may include partial diagnostic results; never treat it as pass.

## Scope and limits

- Preserves Factor / Provider / Registry / Runtime and separate RuntimeState abstractions.
- Includes conditional action fields/options from the source engine and their tests.
- Does not include customer models, product policies, identity providers, SDK tokens,
  persistence, HTTP/gRPC endpoints, or any production configuration.
- GoFrame error types remain a dependency in this initial extraction. Its effective
  source version, 2.7.2, is pinned to avoid mixing extraction with a dependency upgrade.
- Custom functions are trusted Go code, not an untrusted-code sandbox. Register
  providers at startup; do not mutate registries or shared inputs during execution.
- No new guarantees about hostile expressions, execution quotas, concurrent custom
  functions, or arbitrary OBJECT deep copying are made by this extraction.
- Treat input and diagnostic result values as potentially sensitive. Do not log
  complete facts or publish real customer data in fixtures.

## License and publication

Apache-2.0; see LICENSE. Third-party dependencies retain their own licenses.
Before publishing: confirm source-owner permission and attribution obligations,
choose the public module path, review dependency licenses, scan files for secrets,
and run all tests. See RELEASE_CHECKLIST.md. This local preparation is not legal
clearance or a full security audit.
