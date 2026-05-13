# Runtime Bridge

`runtimebridge` provides shared helpers for trusted Go runtimes that need to
enter Matrix rulechains directly.

Use it from orchestrator or host-owned boundary code when the entry concern
cannot be represented as a normal DSL HTTP endpoint, for example OAuth
callbacks, verified webhook handshakes, or lifecycle-owned recovery paths.

Business orchestration should still live in DSL rulechains. The Go caller should
only establish the trusted context, build the initial `RuleMsg`, and call
`ExecuteRuleChain`.

```go
msg, err := runtimebridge.ExecuteRuleChain(
    ctx,
    engine,
    "identityx/rc-auth-sync-user-after-login",
    func(msg types.RuleMsg) error {
        return setProviderIdentity(msg, identity)
    },
    runtimebridge.WithExecutionID(executionID),
    runtimebridge.WithStartRuleChainID("identityx/rc-auth-sync-user-after-login"),
)
```
