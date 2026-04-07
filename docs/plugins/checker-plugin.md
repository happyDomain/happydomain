# Building a happyDomain Checker Plugin

This page documents how to ship a **checker** as an in-process Go plugin
that happyDomain loads at startup. Checker plugins extend happyDomain with
automated diagnostics on zones, domains, services or users.

If you've never built a happyDomain plugin before, read
[`checker-dummy`](https://git.happydns.org/checker-dummy) first; it is the
reference implementation that this page mirrors.

> ⚠️ **Security note.** A `.so` plugin is loaded into the happyDomain process
> and runs with the same privileges. happyDomain refuses to load plugins from
> a directory that is group- or world-writable; keep your plugin directory
> owned and writable only by the happyDomain user.

---

## What a checker plugin must export

happyDomain's loader looks for a single exported symbol named
`NewCheckerPlugin` with this exact signature:

```go
func NewCheckerPlugin() (
    *checker.CheckerDefinition,
    checker.ObservationProvider,
    error,
)
```

where `checker` is `git.happydns.org/checker-sdk-go/checker` (see
[Licensing](#licensing) below for why the SDK lives in a separate module).

- `*CheckerDefinition` describes the checker: ID, name, version, options
  documentation, rules, optional aggregator, scheduling interval, and
  whether the checker exposes HTML reports or metrics. The `ID` field is
  the persistent key: pick something stable and namespaced
  (`com.example.dnssec-freshness`, not `dnssec`).
- `ObservationProvider` is the data-collection half of the checker. It
  exposes a `Key()` (the observation key the rules will look up) and a
  `Collect(ctx, opts)` method that returns the raw observation payload.
  happyDomain serialises the result to JSON and caches it per
  `ObservationContext`.
- Return a non-nil `error` if your plugin cannot initialise (missing
  environment variable, broken cgo dependency, …); the host will log it and
  skip the file rather than aborting startup.

### Registration and collisions

The loader calls `RegisterExternalizableChecker` and
`RegisterObservationProvider` from the SDK registry. Pick globally unique
identifiers: if your checker ID or observation key collides with a built-in
or another plugin, the duplicate is ignored.

The same `.so` may export both `NewCheckerPlugin` and (e.g.)
`NewProviderPlugin`. The loader runs every known plugin loader against
every file, so a single binary can ship a checker, a provider and a service
at once.

---

## Minimal example

```go
// Command plugin is the happyDomain plugin entrypoint for the dummy checker.
//
// Build with:
//   go build -buildmode=plugin -o checker-dummy.so ./plugin
package main

import (
    "context"

    sdk "git.happydns.org/checker-sdk-go/checker"
)

type dummyProvider struct{}

func (dummyProvider) Key() sdk.ObservationKey { return "dummy.observation" }

func (dummyProvider) Collect(ctx context.Context, opts sdk.CheckerOptions) (any, error) {
    return map[string]string{"hello": "world"}, nil
}

// NewCheckerPlugin is the symbol resolved by happyDomain at startup.
func NewCheckerPlugin() (*sdk.CheckerDefinition, sdk.ObservationProvider, error) {
    def := &sdk.CheckerDefinition{
        ID:              "com.example.dummy",
        Name:            "Dummy checker",
        Version:         "0.1.0",
        ObservationKeys: []sdk.ObservationKey{"dummy.observation"},
        // Add Rules / Aggregator / Options here in a real plugin.
    }
    return def, dummyProvider{}, nil
}
```

Build and deploy:

```bash
go build -buildmode=plugin -o checker-dummy.so ./plugin
sudo install -m 0644 -o happydomain checker-dummy.so /var/lib/happydomain/plugins/
sudo systemctl restart happydomain
```

happyDomain will log:

```
Plugin com.example.dummy (.../checker-dummy.so) loaded
```

---

## Build constraints and platform support

Go's `plugin` package is unforgiving:

- The plugin **must be built with the same Go version** as happyDomain
  itself, including the same toolchain patch level.
- It **must use the same versions of every shared dependency**. Vendor the
  exact module versions happyDomain ships, or pin them in your `go.mod`
  with `replace` directives.
- `CGO_ENABLED=1` is required.
- `GOOS`/`GOARCH` must match the host binary.

If any of these don't match, `plugin.Open` will fail with a (sometimes
cryptic) error like *"plugin was built with a different version of package
…"*. The host will log it and skip the file.

Go's `plugin` package only works on **linux**, **darwin** and **freebsd**.
On other platforms (Windows, plan9, …) happyDomain is built without plugin
support and `--plugins-directory` is silently ignored apart from a warning
log line at startup.

---

## Licensing

Checker plugins import only `git.happydns.org/checker-sdk-go/checker`,
which is licensed under **Apache-2.0**. This is intentional: the
checker SDK is a small, stable public API for third-party checkers,
deliberately split out of the AGPL-3.0 happyDomain core so that
permissively-licensed checker plugins are possible.

You may therefore distribute your checker `.so` under any license compatible
with Apache-2.0. Note that this only covers checker plugins; provider and
service plugins still link against AGPL code and remain subject to the
AGPL-3.0 reciprocity rules described in their respective documentation
([provider](provider-plugin.md), [service](service-plugin.md)).
