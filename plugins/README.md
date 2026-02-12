# Writing a happyDomain Plugin

happyDomain supports external **test plugins** — shared libraries (`.so` files) that run domain health checks and diagnostics. Plugins are loaded at runtime and integrate seamlessly into happyDomain's domain and service testing UI.

## Overview

A plugin is a Go shared library (`-buildmode=plugin`) that exports a single entry point: `NewTestPlugin`. At startup, happyDomain scans its configured plugin directories, loads each `.so` file it finds, calls `NewTestPlugin`, and registers the returned plugin by its declared names.

A plugin implements the `TestPlugin` interface from `git.happydns.org/happyDomain/model`:

```go
type TestPlugin interface {
    PluginEnvName() []string
    Version() PluginVersionInfo
    AvailableOptions() PluginOptionsDocumentation
    RunTest(options PluginOptions, meta map[string]string) (*PluginResult, error)
}
```

---

## Project Structure

A minimal plugin lives in its own directory with `package main`:

```
myplugin/
├── go.mod
├── Makefile
└── plugin.go      (or split across multiple .go files)
```

### go.mod

Your plugin must declare the same module path as its source tree and depend on the happyDomain model:

```
module git.happydns.org/happyDomain/plugins/myplugin

go 1.25

require git.happydns.org/happyDomain v0.0.0
replace git.happydns.org/happyDomain => ../../
```

The `replace` directive points to your local happyDomain checkout, ensuring the plugin is compiled against the exact same types.

> **Important:** A Go plugin and the host program must be built with the same Go toolchain version and the same versions of all shared dependencies. Any mismatch will cause a runtime load error.

---

## The Entry Point

Every plugin must export a `NewTestPlugin` function with this exact signature:

```go
package main

import "git.happydns.org/happyDomain/model"

func NewTestPlugin() (happydns.TestPlugin, error) {
    return &MyPlugin{}, nil
}
```

You can use the constructor to perform one-time initialisation (read config files, create HTTP clients, etc.) and return an error if the plugin cannot function.

---

## Implementing the Interface

### `PluginEnvName() []string`

Returns one or more string identifiers for the plugin. These names are used internally to look up the plugin and to store its configuration. Use short, lowercase, collision-resistant names:

```go
func (p *MyPlugin) PluginEnvName() []string {
    return []string{"myplugin"}
}
```

If two plugins claim the same name, the second one is silently ignored and a conflict is logged.

---

### `Version() PluginVersionInfo`

Describes the plugin and declares where it applies:

```go
func (p *MyPlugin) Version() happydns.PluginVersionInfo {
    return happydns.PluginVersionInfo{
        Name:    "My Plugin",
        Version: "1.0",
        AvailableOn: happydns.PluginAvailability{
            ApplyToDomain:    true,
            ApplyToService:   false,
            LimitToProviders: []string{},       // empty = all providers
            LimitToServices:  []string{"abstract.MatrixIM"},
        },
    }
}
```

`PluginAvailability` fields:

| Field | Type | Description |
|---|---|---|
| `ApplyToDomain` | `bool` | Plugin can be run against a whole domain |
| `ApplyToService` | `bool` | Plugin can be run against a specific service |
| `LimitToProviders` | `[]string` | Restrict to certain DNS provider identifiers (empty = no restriction) |
| `LimitToServices` | `[]string` | Restrict to certain service type identifiers, e.g. `"abstract.MatrixIM"` (empty = no restriction) |

---

### `AvailableOptions() PluginOptionsDocumentation`

Declares all configurable options, grouped by **who sets them** and **at which scope**:

```go
func (p *MyPlugin) AvailableOptions() happydns.PluginOptionsDocumentation {
    return happydns.PluginOptionsDocumentation{
        RunOpts:     []happydns.PluginOptionDocumentation{ /* per-run options */ },
        ServiceOpts: []happydns.PluginOptionDocumentation{ /* per-service options */ },
        DomainOpts:  []happydns.PluginOptionDocumentation{ /* per-domain options */ },
        UserOpts:    []happydns.PluginOptionDocumentation{ /* per-user options */ },
        AdminOpts:   []happydns.PluginOptionDocumentation{ /* admin-only options */ },
    }
}
```

#### Option scopes

| Field | Who sets it | Typical use |
|---|---|---|
| `RunOpts` | The user at test time | Test-specific parameters (e.g. domain to test) |
| `ServiceOpts` | The user, per service | Configuration scoped to a DNS service |
| `DomainOpts` | The user, per domain | Configuration scoped to a whole domain |
| `UserOpts` | The user, globally | Personal preferences (e.g. language) |
| `AdminOpts` | The instance administrator | Backend URLs, API keys shared by all users |

Options from all scopes are **merged** before `RunTest` is called, with more-specific scopes overriding less-specific ones.

#### PluginOptionDocumentation fields

Each option is described by a `PluginOptionDocumentation` (an alias for `Field`):

| Field | Type | Description |
|---|---|---|
| `Id` | `string` | **Required.** Key used in `PluginOptions` map |
| `Type` | `string` | Input type: `"string"`, `"select"`, … |
| `Label` | `string` | Human-readable label shown in the UI |
| `Placeholder` | `string` | Input placeholder text |
| `Default` | `any` | Default value pre-filled in the form |
| `Choices` | `[]string` | Available choices for `"select"` type inputs |
| `Required` | `bool` | Whether the field must be filled before running |
| `Secret` | `bool` | Marks the field as sensitive (e.g. API key) |
| `Hide` | `bool` | Hides the field from the user |
| `Textarea` | `bool` | Displays a multiline text area |
| `Description` | `string` | Help text shown below the field |
| `AutoFill` | `string` | Automatically populate the field from context (see below) |

#### Auto-fill variables

When a field's `AutoFill` is set, happyDomain populates it from the test context — the user does not need to fill it in:

| Constant | Value | Filled with |
|---|---|---|
| `happydns.AutoFillDomainName` | `"domain_name"` | The FQDN of the domain under test (e.g. `"example.com."`) |
| `happydns.AutoFillSubdomain` | `"subdomain"` | Subdomain relative to the zone (service-scoped tests only) |
| `happydns.AutoFillServiceType` | `"service_type"` | Service type identifier (service-scoped tests only) |

```go
{
    Id:       "domainName",
    Type:     "string",
    Label:    "Domain name",
    AutoFill: happydns.AutoFillDomainName,
    Required: true,
},
```

---

### `RunTest(options PluginOptions, meta map[string]string) (*PluginResult, error)`

This is where the actual check happens. `options` is the merged map of all scoped options (keyed by option `Id`). `meta` carries additional context provided by the scheduler (currently reserved for future use).

```go
func (p *MyPlugin) RunTest(options happydns.PluginOptions, meta map[string]string) (*happydns.PluginResult, error) {
    domain, ok := options["domainName"].(string)
    if !ok || domain == "" {
        return nil, fmt.Errorf("domainName is required")
    }

    // ... perform the check ...

    return &happydns.PluginResult{
        Status:     happydns.PluginResultStatusOK,
        StatusLine: "All good",
        Report:     myDetailedReport,
    }, nil
}
```

Return a non-nil `error` for hard failures (network errors, invalid options). Return a `PluginResult` with a `KO` status for expected failures (e.g. the DNS check failed).

#### PluginResult

| Field | Type | Description |
|---|---|---|
| `Status` | `PluginResultStatus` | Overall result level |
| `StatusLine` | `string` | Short human-readable summary |
| `Report` | `any` | Arbitrary data (serialised to JSON and stored) |

#### PluginResultStatus values (ordered worst → best)

| Constant | Meaning |
|---|---|
| `PluginResultStatusKO` | Check failed |
| `PluginResultStatusWarn` | Check passed with warnings |
| `PluginResultStatusInfo` | Informational result |
| `PluginResultStatusOK` | Check fully passed |

---

## Full Example

The matrix federation checker plugin (`matrix/`) illustrates a real-world plugin:

**`main.go`** — exports the entry point:

```go
package main

import "git.happydns.org/happyDomain/model"

func NewTestPlugin() (happydns.TestPlugin, error) {
    return &MatrixTester{
        TesterURI: "https://federationtester.matrix.org/api/report?server_name=%s",
    }, nil
}
```

**`test.go`** — implements the interface on a struct.

The zonemaster plugin (`zonemaster/`) shows a more complex flow: it starts an asynchronous test, polls for completion, and aggregates results across multiple severity levels.

---

## Building

Use `-buildmode=plugin`:

```bash
go build -buildmode=plugin -o happydomain-plugin-test-myplugin.so \
    git.happydns.org/happyDomain/plugins/myplugin
```

A minimal `Makefile`:

```makefile
PLUGIN_NAME=myplugin
TARGET=../happydomain-plugin-test-$(PLUGIN_NAME).so

all: $(TARGET)

$(TARGET): *.go
	go build -buildmode=plugin -o $@ git.happydns.org/happyDomain/plugins/$(PLUGIN_NAME)
```

> **Naming convention:** happyDomain looks for any file in the plugin directory, but using the prefix `happydomain-plugin-test-` makes the purpose clear.

---

## Deployment

### 1. Copy the `.so` file to a plugin directory

```bash
cp happydomain-plugin-test-myplugin.so /usr/lib/happydomain/plugins/
```

### 2. Configure happyDomain to load that directory

In your `happydomain.conf`:

```
plugins-directories=/usr/lib/happydomain/plugins
```

Or via an environment variable:

```bash
HAPPYDOMAIN_PLUGINS_DIRECTORIES=/usr/lib/happydomain/plugins
```

Multiple directories can be specified as a comma-separated list. happyDomain scans each directory at startup and attempts to load every file it finds. Loading errors are logged but do not prevent the server from starting.

### 3. Verify

Check the server logs at startup for a line like:

```
Plugin My Plugin loaded (version 1.0)
```

If a name conflict or load error occurs, a warning is logged with the filename and reason.
