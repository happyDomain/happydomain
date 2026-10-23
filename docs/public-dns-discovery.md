# Public DNS discovery

The `internal/discovery` package recomposes the zone of a domain hosted where
happyDomain has no access, from what the authoritative servers of the domain
publicly answer. It never changes the zone.

This document is for administrators, who need to know what the discovery sends
where. Contributors find how to add an enumeration technique in the comments
of the code.

## How a zone is recomposed

A zone cannot be listed over plain DNS: names can only be asked about one by one.
`discovery.Resolve` (`internal/discovery/resolver.go`) therefore:

1. finds the authoritative servers of the domain;
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
30 seconds. These limits are the defaults of `discovery.Options`, which no
setting changes. A domain whose names are not all explored in that budget gets
a partial zone, and a log line telling the exploration was cut short.
