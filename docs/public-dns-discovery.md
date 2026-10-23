# Public DNS discovery

The "Public DNS discovery" provider lets users monitor a domain hosted where
happyDomain has no access. It never changes the zone: it recomposes it from what
the authoritative servers of the domain publicly answer, then hands the records
to the regular import pipeline, which analyses them into services that the
checkers look after like those of any other zone.

The user documentation is on the help center, under *Monitor a domain hosted
elsewhere*. This document is for administrators, who need to know what the
discovery sends where, and for contributors adding an enumeration technique or a
certificate transparency service.

## How a zone is recomposed

A zone cannot be listed over plain DNS: names can only be asked about one by one.
`discovery.Resolve` (`internal/discovery/resolver.go`) therefore:

1. finds the authoritative servers of the domain, through the recursive resolver
   of the provider settings, or the one of `/etc/resolv.conf`;
2. asks the techniques handing the whole zone, such as a zone transfer: when one
   succeeds, its records are the zone and the exploration stops there;
3. explores the apex, then asks a random name to learn what a wildcard, if any,
   answers, so that names merely reflecting it are left out;
4. explores the subdomains found by previous retrievals (the provider actuator
   implements `happydns.DiscoveryZoneGetter`, which receives them), so a name
   found once keeps being looked after and the diff view does not report it as
   removed;
5. explores the names yielded by each technique, in priority order, stopping
   after a technique that found every name of the zone, such as a complete NSEC
   walk;
6. follows the in-zone targets of the answers (`CNAME`, `MX`, `SRV`...), up to
   a bounded number of hops (`Options.MaxDepth`).

Each name is asked for a set of types chosen after its shape (apex, host, or
`_`-prefixed service name), unless its technique tells which types exist there.

The zone is only retrieved when the user imports it, or fetches the current
deployed zone: it is not refreshed in the background. Viewing the differences
runs a full exploration too, to compare the stored zone with what the servers
answer now.

## Techniques

| ID | Priority | What it yields | Setting disabling it |
|----|----------|----------------|----------------------|
| `axfr` | 10 | The whole zone, when a server allows a zone transfer. | `disable_axfr` |
| `nsec` | 20 | Every name of a zone signed with NSEC, and the types at each. Not NSEC3, nor zones signed on the fly. | `disable_nsec_walk` |
| `wellknown` | 100 | Where usual services are published: `_dmarc`, DKIM selectors, SRV and TLSA names... | none (`extra_names` adds names) |
| `wordlist` | 900 | The 5,000 host names most often found in zones (SecLists). | `disable_wordlist` |

## For administrators

### Load on the authoritative servers

An exploration asks at most 7,000 questions, 100 a second to each authoritative
server, 16 names at once, and lasts at most 90 seconds. The questions are shared
among the servers. These limits are the defaults of `discovery.Options` and are
not configurable yet. A domain whose names are not all explored in that budget
gets a partial zone, and a log line telling the exploration was cut short.

### Outbound destinations

The addresses of the authoritative servers come from DNS answers, controlled by
whoever runs the domain. They therefore go through the outbound guard, like the
endpoints of other providers: see [outbound-targets.md](outbound-targets.md). A
domain whose name servers are on your own network can only be monitored once
`-outbound-allowed-target` names them.

The recursive resolver of the provider settings is an endpoint as well, vetted
the same way. When left empty, the resolver of `/etc/resolv.conf` is used: it is
chosen by the operator and therefore not vetted.

## For contributors

### Adding an enumeration technique

Add a file to `internal/discovery`, declaring the settings of the technique
and registering it from its `init` function:

```go
func init() {
	RegisterSource("mytechnique", 300, func(s *Settings) NameSource {
		if s.DisableMyTechnique {
			return nil
		}
		return MyTechniqueSource{}
	})
}

type MyTechniqueSettings struct {
	DisableMyTechnique bool `json:"disable_mytechnique,omitempty" happydomain:"label=Do not use my technique"`
}
```

then embed `MyTechniqueSettings` in `Settings`, in `settings.go`. No other file
changes: the provider embeds `Settings`, so their fields show in its form, and
builds its sources from the registry. The rules:

- The source implements `NameSource`, yielding names within `Target.Domain`.
  Implement `RecordSource` instead to hand the whole zone, or `ExhaustiveSource`
  to tell when every name was found.
- The priority orders the exploration, see `SourceFactory.Priority`: below 100
  for sources handing the whole zone, 100 to 499 for sources yielding few names,
  most of them real, 500 and above for long lists of guesses. The query budget
  leaves the names of the last techniques unasked when it runs out.
- The settings struct is tagged like provider settings. Its zero value must be
  the recommended setting, so a provider created with no settings explores
  fully, and its JSON names must be unique among all techniques, as they are
  embedded in a single struct.
- The `happydomain` tag uses commas as separators: labels and descriptions
  cannot contain any.
- DNS questions go through `Target.Query`, which asks the authoritative servers,
  already vetted by the outbound guard, within the budget and the pace of the
  exploration.

### Adding a certificate transparency service

The `internal/ct` package depends on nothing but the standard library, so it can
move out of happyDomain, such as next to a checker watching issuances. Each
service is a `ct.Backend` in a file of its own:

```go
var myService = Backend{
	ID:          "myservice",
	Name:        "My Service",
	TakesAPIKey: true,
	New: func(cfg Config) Searcher {
		return MyService{cfg}
	},
}
```

then a line in `internal/ct/backends.go`. Use `Config.get` for the requests: it
sets the User-Agent, caps the size of the answer and turns statuses into
`ErrRateLimited` and `ErrUnavailable`. Test against `httptest`, with
`Config.BaseURL` pointing to it.
