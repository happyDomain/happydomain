# Outbound destinations

happyDomain connects to addresses its users choose. A DNS provider is configured
with an API endpoint, the TLSA editor probes a TLS port, the resolver tool
queries a name server, the MTA-STS checker fetches a policy file, and a
notification channel posts to a webhook.

Left unchecked, those features reach whatever the server itself can reach: a
database on the same host, an appliance on the LAN, the admin socket on
loopback, or the cloud metadata endpoint at `169.254.169.254`. The provider form
is the sharpest of them, because the API key configured on the provider is sent
to whatever address the endpoint field names.

happyDomain therefore refuses any destination that is not publicly routable.
Two options re-open the ranges a deployment genuinely needs.

> **Upgrading?** This is a behaviour change. If you run PowerDNS, BIND, AdGuard
> Home, OpenWrt, Mikrotik, UniFi or FortiGate on your own network, read
> [Providers on your own network](#providers-on-your-own-network) before you
> upgrade.

## What is refused by default

Everything outside the public unicast Internet:

- loopback (`127.0.0.0/8`, `::1`) and the unspecified address;
- private ranges (`10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `fc00::/7`);
- link-local, which includes the cloud metadata endpoint `169.254.169.254`, and
  `fe80::/10`;
- carrier-grade NAT (`100.64.0.0/10`);
- multicast and broadcast;
- the reserved, documentation and benchmarking ranges (`0.0.0.0/8`,
  `192.0.0.0/24`, `192.0.2.0/24`, `198.18.0.0/15`, `198.51.100.0/24`,
  `203.0.113.0/24`, `240.0.0.0/4`, `2001:db8::/32`, `100::/64`);
- Teredo (`2001::/32`), 6to4 (`2002::/16`) and NAT64 (`64:ff9b::/96`,
  `64:ff9b:1::/48`), which wrap an IPv4 address and would otherwise be a way
  around the list above.

An IPv4-mapped IPv6 literal such as `::ffff:127.0.0.1` is treated as the IPv4
address it contains, so it cannot be used to disguise one.

## `-outbound-allowed-target`

Applies to every HTTP destination the application fetches on a caller's behalf:
DNS provider endpoints, certificate probes, MTA-STS policy fetches, and
notification webhooks.

```
-outbound-allowed-target 127.0.0.1
```

Most of those are reached only by a signed-in user's actions, but the MTA-STS
policy fetch is not: `/api/resolver/mta-sts-policy` needs no account, and the
host it fetches (`https://mta-sts.<domain>/.well-known/mta-sts.txt`) is derived
from a domain the caller supplies. An anonymous caller who controls a domain's
DNS can point `mta-sts.example.com` at any address on this list and get back the
response body. **Everything you list here is therefore reachable, over HTTPS on
port 443, by anyone who can call the API**, including the loopback entry a local
PowerDNS or BIND provider needs.

## `-resolver-allowed-target`

Applies to the DNS server picked in the resolver tool.

```
-resolver-allowed-target 10.0.0.53
```

It is a separate list because it opens DNS servers rather than HTTP endpoints,
and because the resolver tool needs no account at all, so opening it up is a
broader decision than opening up an action a registered user takes.

It does **not** cover `-default-ns`, nor the "Local resolver" choice that reads
`/etc/resolv.conf`. Those are operator-chosen and routinely point at loopback,
so they keep working with no configuration.

## Syntax

Both options take an IP address or a CIDR block, and can be repeated:

```
happyDomain -outbound-allowed-target 192.168.1.1 -outbound-allowed-target 10.0.0.0/24
```

They also accept a comma separated list, which is the only usable form for the
environment variable and the config file, as those can only be given once:

```
HAPPYDOMAIN_OUTBOUND_ALLOWED_TARGET=127.0.0.1,192.168.1.0/24
HAPPYDOMAIN_RESOLVER_ALLOWED_TARGET=10.0.0.53
```

```
# happydomain.conf
outbound-allowed-target=127.0.0.1,192.168.1.0/24
resolver-allowed-target=10.0.0.53
```

The keyword `none` empties the list. The config file, the environment and the
command line all feed the same accumulating option, so without it a value
inherited from a lower-precedence source could only ever be widened:

```
# whatever the compose file or happydomain.conf set, allow only this
happyDomain -outbound-allowed-target none,127.0.0.1
```

An invalid value stops happyDomain at startup rather than silently changing what
is allowed. Two forms are refused rather than reinterpreted:

- a block with host bits set (`192.168.1.5/24`), which reads like one host but
  covers the whole subnet: write `192.168.1.0/24` or `192.168.1.5`;
- an IPv4-mapped IPv6 literal (`::ffff:192.168.1.1`): write `192.168.1.1`.

**Hostnames are not accepted.** The lists are matched against the addresses a
name resolves to, at the moment of the request, and every address a name
resolves to must be allowed. A hostname entry would be matched before
resolution, and the name could then be pointed anywhere afterwards.

## Providers on your own network

Several providers exist precisely to manage a server or a router you host
yourself, and their placeholders say so: `http://127.0.0.1:3000` for AdGuard
Home, `http://192.168.88.1:8080` for Mikrotik, `https://192.168.1.1` for UniFi.
Those providers stop working until you name their address.

PowerDNS or BIND on the same host:

```
-outbound-allowed-target 127.0.0.1
```

An appliance on the LAN:

```
-outbound-allowed-target 192.168.1.1
```

Bear in mind that these entries are not limited to the provider you opened them
for: anything listening on port 443 at that address becomes fetchable through
the MTA-STS endpoint, by anyone, as described above.

The DDNS/AXFR provider is worth calling out: leaving its **Server** field empty
means `127.0.0.1` to the backend, so it is refused unless loopback is allowed,
even though the field looks unset.

Providers already stored keep their configuration, but the next time one is
edited or applied it goes through the same check, so a missing entry surfaces as
a failure to reach the provider rather than at startup.

## Internal notification webhooks

This is the one place the change relaxes something: webhooks and UnifiedPush
endpoints used to be blocked outright with no way to allow them. An internal
collector now works once its address is listed:

```
-outbound-allowed-target 10.42.0.0/16
```

## Choosing a value

Name the exact address whenever you can. Allowing a range allows everything else
listening in it: `10.0.0.0/8` on a container network or a corporate LAN is a far
larger set of machines than the one you meant, and any account on the instance
can then reach all of them, read what they return, and in the provider case send
an API key to them. On port 443, an anonymous caller can read them too, through
the MTA-STS endpoint.

Neither list is safe to widen on an instance with open registration or multiple
tenants, and neither one is account-gated end to end: keep both as narrow as
possible, and prefer leaving `-resolver-allowed-target` empty, since the
resolver tool it backs is entirely callable without an account.

happyDomain logs the resulting posture at startup whenever either list is not
empty. Nothing is logged when they are, because that is the safe default.
