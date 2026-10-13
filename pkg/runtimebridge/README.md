# Runtime Bridge

`runtimebridge` provides shared helpers for trusted Go runtimes that need to
enter Matrix rulechains directly.

Use it from orchestrator or host-owned boundary code when the entry concern
cannot be represented as a normal DSL HTTP endpoint, for example OAuth
callbacks, verified webhook handshakes, or lifecycle-owned recovery paths.

Business orchestration should still live in DSL rulechains. The Go caller should
only establish the trusted context, build the initial `RuleMsg`, and call
the appropriate execution entry.

`ExecuteRootRuleChain` explicitly owns a new root execution. Supply a fresh,
non-empty `WithExecutionID` for every invocation. The root chain is the target
chain: conflicting root metadata, later options replacing an already declared
identity, or `fill` changing/removing either identity are errors. Repeating the
same identity is allowed. Lookup, option validation and `fill` failures do not
create a snapshot. After preparation, the helper registers one deferred
`FinalizeSnapshot` through the Engine's optional Trace API, covering every
runtime return and preserving the original `WithEndFunc` business callback.

`ExecuteRuleChain` retains its original behavior, including metadata overrides.
Use it for nested execution or calls inheriting an outer execution ID; it never
automatically finalizes a trace, even when root and child chain names match.
HTTP entries already owning request completion keep their existing finalizers.

A deadline/cancellation ends the root **wait**, not necessarily the physical
task. Late node logs may append to the retained snapshot and refresh
`LastUpdated`, without clearing `EndTs`. Native in-memory state creation is
atomic. The default 24-hour retention and cleanup interval are unchanged;
there is no new tombstone policy to prevent recreation after actual TTL deletion.
The required `types.MatrixEngine` and `types.Store` interfaces are unchanged.

```go
msg, err := runtimebridge.ExecuteRootRuleChain(
    ctx,
    engine,
    "<module-id>/rc-trusted-root",
    func(msg types.RuleMsg) error {
        return setProviderIdentity(msg, identity)
    },
    runtimebridge.WithExecutionID(executionID),
)
```
