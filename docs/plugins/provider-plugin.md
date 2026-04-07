# Building a happyDomain Provider Plugin

This page documents how to ship a DNS **provider** as an in-process Go plugin
that happyDomain loads at startup. It mirrors the layout of
[`checker-dummy`](https://git.happydns.org/checker-dummy); read that first if
you've never built a happyDomain plugin before.

For checker and service plugins see [checker-plugin.md](checker-plugin.md)
and [service-plugin.md](service-plugin.md).

> ⚠️ **Security note.** A `.so` plugin is loaded into the happyDomain process
> and runs with the same privileges. happyDomain refuses to load plugins from a
> directory that is group- or world-writable; keep your plugin directory owned
> and writable only by the happyDomain user.

---

## What a provider plugin must export

happyDomain's loader looks for a single exported symbol named
`NewProviderPlugin` with this exact signature:

```go
func NewProviderPlugin() (
    happydns.ProviderCreatorFunc,
    happydns.ProviderInfos,
    error,
)
```

- `ProviderCreatorFunc` is `func() happydns.ProviderBody`. Each call must
  return a fresh, zero-value instance of your provider struct so happyDomain
  can decode user-supplied configuration into it.
- `ProviderInfos` carries the human-readable name, description, capabilities
  and help link displayed in the UI.
- Return a non-nil `error` if your plugin cannot initialise (missing
  environment variable, broken cgo dependency, …); the host will log it and
  skip the file rather than aborting startup.

### Registration name and collisions

Plugin-registered providers are stored under their **fully qualified Go type
name** (`packagename.TypeName`), not the short type name used by built-in
providers. This is deliberate: two plugins shipping a `Provider` struct in
different packages would otherwise silently overwrite each other in the
global registry.

If your plugin tries to register a name that already exists (because it is
loaded twice, or because it shadows a built-in), the second registration is
**refused with a warning** rather than overwriting the first. The first one
wins; restart with the duplicate removed.

---

## Minimal example (`provider-dummy/plugin/plugin.go`)

```go
// Command plugin is the happyDomain plugin entrypoint for the dummy provider.
//
// Build with:
//   go build -buildmode=plugin -o provider-dummy.so ./plugin
package main

import (
    "git.happydns.org/happyDomain/model"
)

// Version is overridden at link time:
//   go build -buildmode=plugin \
//       -ldflags "-X main.Version=$(git describe --tags)" \
//       -o provider-dummy.so ./plugin
var Version = "custom-build"

// DummyProvider is the provider body that happyDomain stores and edits.
// Exported fields become the user-facing configuration form.
type DummyProvider struct {
    Endpoint string `json:"endpoint"`
    Token    string `json:"token"`
}

func (d *DummyProvider) InstantiateProvider() (happydns.ProviderActuator, error) {
    // Return your real ProviderActuator implementation here.
    return nil, nil
}

// NewProviderPlugin is the symbol resolved by happyDomain at startup.
func NewProviderPlugin() (happydns.ProviderCreatorFunc, happydns.ProviderInfos, error) {
    creator := func() happydns.ProviderBody { return &DummyProvider{} }
    infos := happydns.ProviderInfos{
        Name:        "Dummy provider (" + Version + ")",
        Description: "Example provider plugin, replace with real DNS code.",
        HelpLink:    "https://example.com/docs/dummy-provider",
    }
    return creator, infos, nil
}
```

Build and deploy:

```bash
go build -buildmode=plugin -o provider-dummy.so ./plugin
sudo install -m 0644 -o happydomain provider-dummy.so /var/lib/happydomain/plugins/
sudo systemctl restart happydomain
```

happyDomain will log:

```
Registering new provider: main.DummyProvider
Plugin provider "Dummy provider (...)" registered as "main.DummyProvider" (.../provider-dummy.so)
```

---

## Build constraints

The same Go plugin caveats (toolchain version, dependency versions,
`CGO_ENABLED=1`, `GOOS`/`GOARCH`) apply to provider plugins. See
[checker-plugin.md](checker-plugin.md#build-constraints-and-platform-support)
for the full list.

---

## Licensing

Provider plugins import `git.happydns.org/happyDomain/model`, which is part
of happyDomain and licensed under **AGPL-3.0**. A `.so` linked against the
model package is therefore considered a derivative work of happyDomain and
must itself be AGPL-compatible. If you need a permissively-licensed
provider, run it as a separate process behind happyDomain's HTTP API
instead.
