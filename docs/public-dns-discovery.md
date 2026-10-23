# Public DNS discovery

The `internal/discovery` package recomposes the zone of a domain hosted where
happyDomain has no access, from what the authoritative servers of the domain
publicly answer. It never changes the zone.

This document is for administrators, who need to know what the discovery sends
where, and for contributors adding an enumeration technique.

## How a zone is recomposed

A zone cannot be listed over plain DNS: names can only be asked about one by one.
`discovery.Resolve` (`internal/discovery/resolver.go`) therefore:

1. finds the authoritative servers of the domain, through the recursive resolver
   of `Options.Resolver`, or the one of `/etc/resolv.conf`;
2. explores the apex, then asks a random name to learn what a wildcard, if any,
   answers, so that names merely reflecting it are left out;
3. explores the subdomains found by previous retrievals, handed by the caller,
   so a name found once keeps being looked after;
4. explores the names yielded by each technique, in priority order;
5. follows the in-zone targets of the answers (`CNAME`, `MX`, `SRV`...), up to
   a bounded number of hops (`Options.MaxDepth`).

Each name is asked for a set of types chosen after its shape (apex, host, or
`_`-prefixed service name), unless its technique tells which types exist there.

## For administrators

### Load on the authoritative servers

An exploration asks at most 2,000 questions, 16 names at once, and lasts at most
30 seconds. These limits are the defaults of `discovery.Options` and are not
configurable yet. A domain whose names are not all explored in that budget gets
a partial zone, and a log line telling the exploration was cut short.

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
changes: whoever exposes the discovery, such as a provider, embeds `Settings`
and builds its sources from the registry. The rules:

- The source implements `NameSource`, yielding names within `Target.Domain`.
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
- The authoritative servers of `Target.Authoritative` are already vetted by the
  outbound guard.
