# Building a happyDomain Service Plugin

This page documents how to ship a happyDomain **service** (a high-level
abstraction over a set of DNS records (e.g. "Mailbox", "Web server",
"Matrix homeserver") as an in-process Go plugin. Read the
[provider plugin guide](provider-plugin.md) first if you've never built a
happyDomain plugin.

> ⚠️ **Security note.** A `.so` plugin is loaded into the happyDomain process
> and runs with the same privileges. happyDomain refuses to load plugins from
> a directory that is group- or world-writable.

---

## What a service plugin must export

happyDomain's loader looks for a single exported symbol named
`NewServicePlugin` with this exact signature:

```go
func NewServicePlugin() (
    happydns.ServiceCreator,
    svcs.ServiceAnalyzer,
    happydns.ServiceInfos,
    uint32,   // weight (analyzer priority, lower runs first)
    []string, // optional aliases
    error,
)
```

- `ServiceCreator` is `func() happydns.ServiceBody`. Each call must return a
  fresh, zero-value instance of your service struct.
- `ServiceAnalyzer` is the optional analyzer that recognises this service in
  an existing zone (`nil` is allowed for "manual-only" services).
- `aliases` lets a single struct be reachable under several legacy names; the
  loader will refuse to register an alias that collides with an existing one.

### Sub-services and the `pathToSvcsModule` filter

happyDomain's built-in service registry walks each registered struct and
records every nested struct type as a *sub-service* so the storage layer can
(de)serialise polymorphic payloads later. To avoid registering random types
pulled in from third-party libraries, that walk is restricted to types whose
package path starts with `git.happydns.org/happyDomain/services`.

Plugin services live in a completely different module path. The plugin loader
calls a dedicated walker (`svcs.RegisterPluginSubServices`) on every plugin
service so that nested types declared by the plugin **are** registered. You
get the same nested-struct support as a built-in service: there is nothing
to do on your side. The only constraint is that nested types must be **named
struct types** (anonymous structs cannot be looked up by name later).

### Collisions

If your plugin tries to register a service or alias whose name already
exists, the registration is **refused with a warning** rather than
overwriting the previous entry. The first one wins.

---

## Minimal example (`service-dummy/plugin/plugin.go`)

```go
// Build with:
//   go build -buildmode=plugin -o service-dummy.so ./plugin
package main

import (
    svcs "git.happydns.org/happyDomain/internal/service"
    "git.happydns.org/happyDomain/model"
)

type DummyDetail struct {
    Note string `json:"note"`
}

type DummyService struct {
    Hostname string      `json:"hostname"`
    Detail   DummyDetail `json:"detail"`
}

func (d *DummyService) GetNbResources() int { return 1 }
func (d *DummyService) GenComment() string  { return d.Hostname }
func (d *DummyService) GetRecords(domain string, ttl uint32, origin string) ([]happydns.Record, error) {
    return nil, nil
}

func NewServicePlugin() (
    happydns.ServiceCreator,
    svcs.ServiceAnalyzer,
    happydns.ServiceInfos,
    uint32,
    []string,
    error,
) {
    creator := func() happydns.ServiceBody { return &DummyService{} }
    infos := happydns.ServiceInfos{
        Name:        "Dummy service",
        Description: "Example service plugin, replace with real logic.",
    }
    return creator, nil, infos, 100, nil, nil
}
```

Build and deploy:

```bash
go build -buildmode=plugin -o service-dummy.so ./plugin
sudo install -m 0644 -o happydomain service-dummy.so /var/lib/happydomain/plugins/
sudo systemctl restart happydomain
```

happyDomain will log:

```
Registering new service: main.DummyService
Registering new plugin subservice: main.DummyDetail
Plugin service "Dummy service" (.../service-dummy.so) loaded
```

---

## Build constraints

The same Go plugin caveats (toolchain version, dependency versions,
`CGO_ENABLED=1`, `GOOS`/`GOARCH`) apply to service plugins. See
[checker-plugin.md](checker-plugin.md#build-constraints-and-platform-support)
for the full list.

---

## Licensing

Service plugins import both `git.happydns.org/happyDomain/model` **and**
`git.happydns.org/happyDomain/internal/service`, both of which are AGPL-3.0.
A `.so` linked against them is therefore considered a derivative work of
happyDomain and must itself be AGPL-compatible.

For checker plugins see [checker-plugin.md](checker-plugin.md#licensing),
which uses a separate (Apache-2.0) SDK module and is not subject to these
AGPL constraints.
