# Description

This package encapsulates Sentry integration. So far:
- only for status-go (including when running as part of desktop and mobile)
- only for panics (no other error reporting)

Sentry is only enabled for users that **both**:
1. Opted-in for metrics
2. Use builds from our release CI

## 🛬 Where

We use self-hosted Sentry: https://sentry.infra.status.im/

## 🕐 When

### Which panics are reported:

- When running inside `status-desktop`/`status-mobile`:
  - during API calls in `/mobile/status.go`
  - inside all goroutines
- When running `status-backend`:
  - any panic

### Which panics are NOT reported:

- When running inside `status-desktop`/`status-mobile`:
  - during API calls in `/services/**/api.go` \
    NOTE: These endpoints are executed through `go-ethereum`'s JSON-RPC server, which internally recovers all panics and doesn't provide any events or option to set an interceptor. 
    The only way to catch these panics is to replace the JSON-RPC server implementation.
- When running `status-go` unit tests:
  - any panic \
    NOTE: Go internally runs tests in a goroutine. The only way to catch these panics in tests is to manually `defer sentry.Recover()` in each test. This also requires a linter (similar to `lint-panics`) that checks this call is present. \
    This is not a priority right now, because:
    1. We have direct access to failed tests logs, which contain the panic stacktrace.
    2. Panics are not expected to happen. Test must be passing to be able to merge the code. So it's only possible with a flaky test.
    3. This would only apply to nightly/develop jobs, as we don't gather panic reports from PR-level jobs.
    

## 📦 What

Full list can be found in `sentry.Event`. 

Notes regarding identity-disclosing properties:
- `ServerName` - completely removed from all events
- `Stacktrace`:
  - No private user paths are exposed, as we only gather reports from CI-built binaries.
- `TraceID` - so far will be unique for each event
  >Trace: A collection of spans representing the end-to-end journey of a request through your system that all share the same trace ID.

  More details in [sentry docs](https://docs.sentry.io/product/explore/traces/#key-concepts).

# Configuration

There are 2 main tags to identify the error. The configuration is a bit complicated, but provides full information.

## Parameters

### `Environment`

|                   |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                        |
|-------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Defining question | Where it is running?                                                                                                                                                                                                                                                                                                                                                                                                                                                                   |
| Set time          | - `production` can only be set at build time to prevent users from hacking the environment<br>- All others can be set at runtime, because on CI we sometimes use same build for multiple environments                                                                                                                                                                                                                                                                                  |
| Expected values   | <table><thead><tr><th>Value</th><th>Description</th></tr></thead><tr><td>`production`</td><td>End user machine</td></tr><tr><td>~~`development`~~</td><td>Developer machine</td></tr><tr><td>~~`ci-pr`~~</td><td>PR-level CI runs</td></tr><tr><td>`ci-main`</td><td>CI runs for stable branch</td></tr><tr><td>`ci-nightly`</td><td>CI nightly jobs on stable branch</td></tr></table>`development` and `ci-pr` are dropped, because we only want to consider panics from stable code |

### `Context`


|                   |                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       |
|-------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Defining question | What is the executable for the library?                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                               |
| Set time          | Always at build-time                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
| Expected values   | <table><thead><tr><th>Value</th><th>Running as...</th></tr></thead><tr><td>`status-desktop`</td><td>Library embedded into [status-desktop](https://github.com/status-im/status-desktop)</td></tr><tr><td>`status-mobile`</td><td>Library embedded into [status-mobile](https://github.com/status-im/status-mobile)</td></tr><tr><td>`status-backend`</td><td>Part of `cmd/status-backend`<br>Can be other `cmd/*` as well.</td></tr><tr><td>`matterbridge`</td><td>Part of [Status/Discord bridge app](https://github.com/status-im/matterbridge)</td></tr><tr><td>`status-go-tests`</td><td>Inside status-go tests</td></tr></table> |


## Environment variables

To cover these requirements, I added these environment variables:

| Environment variable                              | Provide time                                                                                            | Description                                                                                                                                             |
|---------------------------------------------------|---------------------------------------------------------------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------|
| `SENTRY_DSN`                                      | - At build time with direct call to `sentry.Init`<br>- At runtime with `InitializeApplication` endpoint | [Sentry DSN](https://docs.sentry.io/concepts/key-terms/dsn-explainer/) to be used                                                                       |
| `SENTRY_CONTEXT_NAME`<br>`SENTRY_CONTEXT_VERSION` | Build time                                                                                              | Execution context of status-go                                                                                                                          |
| `SENTRY_PRODUCTION`                               | Build time                                                                                              | When `true` or `1`:<br>-Defines if this is a production build<br>-Sets environment to `production`<br>-Has precedence over runtime `SENTRY_ENVIRONMENT` |
| `SENTRY_ENVIRONMENT`                              | Run time                                                                                                | Sets the environment. Has no effect when `SENTRY_PRODUCTION` is set                                                                                    |

# Client integration

1. Set `SENTRY_CONTEXT_NAME` and `SENTRY_CONTEXT_VERSION` at status-go build time
2. Provide `sentryDSN` to the `InitializeApplication` call.
   DSN must be kept private and will be provided by CI. Expect a `STATUS_GO_SENTRY_DSN` environment variable to be provided by CI. 

    <details>
        <summary>Why can't we consume `STATUS_GO_SENTRY_DSN` directly in status-go build?</summary>

        In theory, we could. But this would require us to mix approaches of getting the env variable to the code.
        Right now we prefer `go:generate + go:embed` approach (e.g. https://github.com/status-im/status-go/pull/6014), but we can't do it in this case, because we must not write the DSN to a local file, which would be a bit vulnerable. And I don't want to go back to `-ldflags="-X github.com/status-im/status-go/internal/sentry.sentryDSN=$(STATUS_GO_SENTRY_DSN:v%=%)` approach.
    </details>

# Implementation details

- We recover from panics in here:
  https://github.com/status-im/status-go/blob/fcedb013166785e7def8710118086f4b650c33b1/common/utils.go#L102 https://github.com/status-im/status-go/blob/fcedb013166785e7def8710118086f4b650c33b1/mobile/callog/status_request_log.go#L69 https://github.com/status-im/status-go/blob/fcedb013166785e7def8710118086f4b650c33b1/cmd/status-backend/main.go#L40
  This covers all goroutines, because we have a linter to check that all goroutines have `defer common.LogOnPanic`.
- Sentry is currently initialized in 2 places:
    - `InitializeApplication` - covers desktop/mobile clients
      https://github.com/status-im/status-go/blob/fcedb013166785e7def8710118086f4b650c33b1/mobile/status.go#L105-L108
    - in `status-backend` - covers functional tests:
      https://github.com/status-im/status-go/blob/fcedb013166785e7def8710118086f4b650c33b1/cmd/status-backend/main.go#L36-L39
