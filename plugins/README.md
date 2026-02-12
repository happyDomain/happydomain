# Writing a happyDomain Plugin

happyDomain supports external **check plugins** — shared libraries (`.so` files) that run domain health checks and diagnostics. Plugins are loaded at runtime and integrate seamlessly into happyDomain's domain and service testing UI.

## Overview

A plugin is a Go shared library (`-buildmode=plugin`) that exports a single entry point: `NewCheckPlugin`. At startup, happyDomain scans its configured plugin directories, loads each `.so` file it finds, calls `NewCheckPlugin`, and registers the returned checker under the declared name.

A plugin implements the `Checker` interface from `git.happydns.org/happyDomain/model`:

```go
type Checker interface {
    ID() string
    Name() string
    Availability() CheckerAvailability
    Options() CheckerOptionsDocumentation
    RunCheck(options CheckerOptions, meta map[string]string) (*CheckResult, error)
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

Every plugin must export a `NewCheckPlugin` function with this exact signature:

```go
package main

import "git.happydns.org/happyDomain/model"

func NewCheckPlugin() (string, happydns.Checker, error) {
    return "myplugin", &MyPlugin{}, nil
}
```

The first return value is the unique registration name for the checker. You can use the constructor to perform one-time initialisation (read config files, create HTTP clients, etc.) and return an error if the plugin cannot function.

---

## Implementing the Interface

### `ID() string`

Returns the unique string identifier for the checker. This name is used internally to look up the checker and to store its configuration. Use a short, lowercase, collision-resistant name:

```go
func (p *MyPlugin) ID() string {
    return "myplugin"
}
```

The value returned here should match the name returned by `NewCheckPlugin`. If two checkers claim the same ID, the second one is silently ignored and a conflict is logged.

---

### `Name() string`

Returns a human-readable display name for the checker:

```go
func (p *MyPlugin) Name() string {
    return "My Plugin"
}
```

---

### `Availability() CheckerAvailability`

Declares where the checker applies:

```go
func (p *MyPlugin) Availability() happydns.CheckerAvailability {
    return happydns.CheckerAvailability{
        ApplyToDomain:    true,
        ApplyToService:   false,
        LimitToProviders: []string{},       // empty = all providers
        LimitToServices:  []string{"abstract.MatrixIM"},
    }
}
```

`CheckerAvailability` fields:

| Field | Type | Description |
|---|---|---|
| `ApplyToDomain` | `bool` | Checker can be run against a whole domain |
| `ApplyToService` | `bool` | Checker can be run against a specific service |
| `LimitToProviders` | `[]string` | Restrict to certain DNS provider identifiers (empty = no restriction) |
| `LimitToServices` | `[]string` | Restrict to certain service type identifiers, e.g. `"abstract.MatrixIM"` (empty = no restriction) |

---

### `Options() CheckerOptionsDocumentation`

Declares all configurable options, grouped by **who sets them** and **at which scope**:

```go
func (p *MyPlugin) Options() happydns.CheckerOptionsDocumentation {
    return happydns.CheckerOptionsDocumentation{
        RunOpts:     []happydns.CheckerOptionDocumentation{ /* per-run options */ },
        ServiceOpts: []happydns.CheckerOptionDocumentation{ /* per-service options */ },
        DomainOpts:  []happydns.CheckerOptionDocumentation{ /* per-domain options */ },
        UserOpts:    []happydns.CheckerOptionDocumentation{ /* per-user options */ },
        AdminOpts:   []happydns.CheckerOptionDocumentation{ /* admin-only options */ },
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

Options from all scopes are **merged** before `RunCheck` is called, with more-specific scopes overriding less-specific ones.

#### CheckerOptionDocumentation fields

Each option is described by a `CheckerOptionDocumentation` (an alias for `Field`):

| Field | Type | Description |
|---|---|---|
| `Id` | `string` | **Required.** Key used in `CheckerOptions` map |
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

### `RunCheck(options CheckerOptions, meta map[string]string) (*CheckResult, error)`

This is where the actual check happens. `options` is the merged map of all scoped options (keyed by option `Id`). `meta` carries additional context provided by the scheduler (currently reserved for future use).

```go
func (p *MyPlugin) RunCheck(options happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) {
    domain, ok := options["domainName"].(string)
    if !ok || domain == "" {
        return nil, fmt.Errorf("domainName is required")
    }

    // ... perform the check ...

    return &happydns.CheckResult{
        Status:     happydns.CheckResultStatusOK,
        StatusLine: "All good",
        Report:     myDetailedReport,
    }, nil
}
```

Return a non-nil `error` for hard failures (network errors, invalid options). Return a `CheckResult` with a `KO` status for expected failures (e.g. the DNS check failed).

#### CheckResult fields set by the plugin

| Field | Type | Description |
|---|---|---|
| `Status` | `CheckResultStatus` | Overall result level |
| `StatusLine` | `string` | Short human-readable summary |
| `Report` | `any` | Arbitrary data (serialised to JSON and stored) |

The remaining fields (`Id`, `CheckerName`, `ExecutedAt`, etc.) are filled in by happyDomain automatically.

#### CheckResultStatus values (ordered worst → best)

| Constant | Meaning |
|---|---|
| `CheckResultStatusKO` | Check failed |
| `CheckResultStatusWarn` | Check passed with warnings |
| `CheckResultStatusInfo` | Informational result |
| `CheckResultStatusOK` | Check fully passed |

---

## Full Example

The matrix federation checker plugin (`matrix/`) illustrates a real-world plugin:

**`main.go`** — exports the entry point:

```go
package main

import "git.happydns.org/happyDomain/model"

func NewCheckPlugin() (string, happydns.Checker, error) {
    return "matrixim", &MatrixTester{
        TesterURI: "https://federationtester.matrix.org/api/report?server_name=%s",
    }, nil
}
```

**`test.go`** — implements the interface on a struct:

```go
func (p *MatrixTester) ID() string { return "matrixim" }

func (p *MatrixTester) Name() string { return "Matrix Federation Tester" }

func (p *MatrixTester) Availability() happydns.CheckerAvailability {
    return happydns.CheckerAvailability{
        ApplyToService:  true,
        LimitToServices: []string{"abstract.MatrixIM"},
    }
}

func (p *MatrixTester) Options() happydns.CheckerOptionsDocumentation { /* ... */ }

func (p *MatrixTester) RunCheck(options happydns.CheckerOptions, meta map[string]string) (*happydns.CheckResult, error) { /* ... */ }
```

The built-in Zonemaster checker (`checks/zonemaster.go`) shows a more complex flow: it starts an asynchronous test, polls for completion, and aggregates results across multiple severity levels. Although it is compiled in rather than loaded as a `.so`, it implements the same `Checker` interface and is a useful reference.

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

> **Naming convention:** happyDomain looks for any `.so` file in the plugin directory, but using the prefix `happydomain-plugin-test-` makes the purpose clear.

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

Multiple directories can be specified as a comma-separated list. happyDomain scans each directory at startup and attempts to load every `.so` file it finds. Loading errors are logged but do not prevent the server from starting.

### 3. Verify

Check the server logs at startup for a line like:

```
Plugin myplugin loaded
```

If a name conflict or load error occurs, a warning is logged with the filename and reason.
