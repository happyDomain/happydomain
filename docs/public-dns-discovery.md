# Public DNS discovery

The "Public DNS discovery" provider lets users monitor a domain hosted where
happyDomain has no access. It never changes the zone: it recomposes it from what
the authoritative servers of the domain publicly answer, then hands the records
to the regular import pipeline, which analyses them into services that the
checkers look after like those of any other zone.

The user documentation is on the help center, under *Monitor a domain hosted
elsewhere*. This document is for administrators, who need to know what the
discovery sends where. Contributors find how to add an enumeration technique in
the comments of the code.

## How a zone is recomposed

A zone cannot be listed over plain DNS: names can only be asked about one by one.
`discovery.Resolve` (`internal/discovery/resolver.go`) therefore:

1. finds the authoritative servers of the domain;
2. explores the apex, then asks a random name to learn what a wildcard, if any,
   answers, so that names merely reflecting it are left out;
3. explores the subdomains found by previous retrievals (the provider actuator
   implements `happydns.DiscoveryZoneGetter`, which receives them), so a name
   found once keeps being looked after and the diff view does not report it as
   removed;
4. explores the names yielded by each technique, in priority order;
5. follows the in-zone targets of the answers (`CNAME`, `MX`, `SRV`...), up to
   a bounded number of hops (`Options.MaxDepth`).

Each name is asked for a set of types chosen after its shape (apex, host, or
`_`-prefixed service name), unless its technique tells which types exist there.

The zone is only retrieved when the user imports it, or fetches the current
deployed zone: it is not refreshed in the background. Viewing the differences
runs a full exploration too, to compare the stored zone with what the servers
answer now.

## For administrators

### Load on the authoritative servers

An exploration asks at most 2,000 questions, 16 names at once, and lasts at most
30 seconds. These limits are the defaults of `discovery.Options`, which no
setting changes. A domain whose names are not all explored in that budget gets
a partial zone, and a log line telling the exploration was cut short.

### Outbound destinations

The addresses of the authoritative servers come from DNS answers, controlled by
whoever runs the domain. They therefore go through the outbound guard, like the
endpoints of other providers: see [outbound-targets.md](outbound-targets.md). A
domain whose name servers are on your own network can only be monitored once
`-outbound-allowed-target` names them.

The recursive resolvers asked to find the authoritative servers are those of
`-default-resolver`: chosen by the operator, they are not vetted by the outbound
guard.
