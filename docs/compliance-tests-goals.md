# Records compliance: goals & scenarios

## Architecture

Frontend-only, registry-based. `$lib/services/compliance.ts` defines the
`ComplianceIssue` shape and exposes `registerValidators(svctype, …)`; each
service ships its checks in `$lib/services/<svctype>/compliance.ts`, which
`EditorCompliance.svelte` picks up by globbing the service folders, so wiring a
new service type is a single `register` call in the right folder. See
[`adding-a-service.md`](adding-a-service.md) for the layout of a service.
`EditorCompliance.svelte` is mounted under every editor and stays hidden when no
validator is registered or no issue is returned.

Validators come in two layers:

- **Sync** runs on every keystroke from the parsed value. It covers syntax, RFC
  field constraints, and zero-network cross-record checks.

- **Async** is debounced, allowed to call the resolver backend for checks that
  need authoritative DNS or HTTPS lookups.


## Scenarios covered per validator

### SPF (`svcs.SPF`)

Sync:
- missing `v=` or wrong version (error)
- multiple `all` directive, multiple `redirect=`, unknown mechanism, lookup-mechanism without value (error)
- invalid qualifier prefix, qualifier on a modifier (error)
- `ip4`/`ip6` without address, address that is not a literal or belongs to the other family, prefix length out of range (error)
- target that is not a domain name or whose top label is numeric, on `include`, `exists`, `a`, `mx`, `ptr`, `redirect=` and `exp=` (error)
- malformed macro, unknown macro letter, `c`/`r`/`t` outside an explanation (error)
- malformed `a`/`mx` dual-CIDR, prefix length on a mechanism taking none (error)
- `exp=` without value, multiple `exp=` (error)
- `all` not last, `redirect` with `all`, no `all`/`redirect`, `ptr` deprecated, unknown modifier, empty term (warning)
- single-label target, `%{p}` macro (warning)
- duplicate mechanism, record > 255 chars, `exp=` on a record that never fails (info)

Async (through new `spf-flatten` resolver route):
- recursive lookup budget > 10, include loop (error)
- ≥ 8 lookups, > 2 void lookups, include with no SPF record (warning)
- per-child resolver/timeout error (info)

### DKIM (`svcs.DKIMRecord`)

Sync:
- missing/invalid selector, parse error, wrong `v=`, missing `p=`, invalid base64, weak RSA key (< 1024) (error)
- Ed25519 key not 32 octets long, key that contradicts the announced `k=` (error)
- `p=` holding only spaces, placeholder key (too short or a single repeated character) (error)
- selector label over 63 octets, `<selector>._domainkey.<domain>` over 253 octets (error)
- selector label outside the LDH grammar of RFC 6376 sec. 3.1, several records sharing one selector (RFC 6376 sec. 3.6.2.2) (warning)
- revoked key (`p=` empty), Ed25519 key wrapped in a SubjectPublicKeyInfo, short RSA key (< 2048), deprecated hash `sha1`, unknown key type/hash/`t=` flag (warning)
- testing mode `t=y`, unknown service type, deprecated `g=` (info)

### DMARC (`svcs.DMARC`)

Sync:
- wrong owner name, parse error, missing/invalid `v=`/`p=`/`sp=`, invalid `adkim`/`aspf`/`pct`/`ri`, invalid URI scheme, malformed `mailto:`, no alignment source while enforcing (error)
- `rua`/`ruf` http(s) URI that does not parse or whose host is not a valid name (error)
- invalid `fo`, unknown `rf`, empty URI, no alignment source (`p=none`), `adkim=s` without DKIM in zone (warning)
- report URI in plain http, pointing at an address literal or at a single-label host (warning)
- zone relying on SPF only (no DKIM selector), every DKIM selector of the zone revoked or in testing mode (warning)
- `p=none`, `pct<100`, external reporting destination detected (info)

Async (through `dmarc-report-auth`):
- external reporting authorization missing/external domain has no DMARC (error)
- resolver error during the lookup (warning)

### MTA-STS (`svcs.MTA_STS`)

Sync:
- wrong owner name, parse error, missing/invalid `v=`/`id=` (error)

Async (through `mta-sts-policy`):
- policy DNS/TLS/not-found/too-large, invalid version/mode, missing `mx:` while filtering, missing/invalid `max_age`, zone MX not covered (in `enforce`) (error)
- HTTP redirect or non-200, fetch error, `mode=none`, `max_age` < 1 day, zone has no apex MX, zone MX not covered (in `testing`) (warning)
- `mode=testing`, policy `mx:` pattern unused by zone (info)

### TLS-RPT (`svcs.TLS_RPT`)

Sync:
- wrong owner name, parse error, missing/invalid `v=`, missing `rua=`, invalid `rua` scheme, malformed `mailto:` (error)
- empty `rua` entry (warning)

### MX (`svcs.MXs`)

Sync:
- null MX mixed with other records, invalid hostname, invalid preference, in-zone target is a CNAME (error)
- null MX with non-zero preference, duplicate target, in-zone target has no A/AAAA (warning)

### BIMI (`svcs.BIMI`)

Sync:
- wrong owner name, missing/invalid selector, invalid `v=`, missing `l=` (outside declination), `l=` or `a=` not HTTPS (error)
- DMARC policy is `none` everywhere, no DMARC in zone, `l=` not `.svg`, `e=` not HTTPS (warning)
- declination detected, missing VMC `a=`, `a=` not `.pem` (info)

### For sale (`svcs.ForSale`)

Sync:
- wrong owner name, missing `v=FORSALE1;`, content that is not a tag-value pair, several pairs in one record, duplicated pair, empty or over-long value, invalid price, invalid URI (error)
- unknown tag, unusual URI scheme, TTL above an hour, `_for-sale` records disagreeing on their TTL (warning)
- the domain is announced for sale without any detail (info)

### Aliases (`svcs.Alias`, `svcs.SpecialCNAME`)

Sync:
- CNAME at the apex, CNAME sharing its name with another record, DNAME conflicting with a CNAME at the same name (error)
- empty target, syntactically invalid target, record aliasing its own name (error)
- in-zone target that is itself an alias, in-zone target publishing nothing (warning)

The target checks apply to every kind of alias, but only a CNAME and a DNAME
are walked: the provider-resolved kinds (ALIAS, ANAME, R53_ALIAS, ...) are
flattened into addresses, so neither the chain nor an empty target concerns
them. `svcs.SpecialCNAME` shares those checks, with RFC 8552 underscore labels
allowed in the target, and keys its coexistence rule on the `_service._proto`
name it carries rather than on the subdomain it is attached to.

### Delegation (`abstract.Delegation`)

Sync, name servers:
- no NS at all, empty or invalid target, target that is a CNAME (RFC 2181 sec. 10.3) (error)
- target inside the delegated subtree with no glue published in the parent zone (error)
- single name server, duplicate target, in-zone target outside the subtree with no A/AAAA (warning)

Sync, DS records:
- DS without any NS, unknown digest type, key tag out of the uint16 range, digest whose length or alphabet does not match its digest type (error)
- SHA-1 digest, algorithm deprecated by RFC 8624, duplicate DS (warning)

### SRV (`svcs.UnknownSRV` and the SRV-based abstract services)

The checks live in `$lib/services/srv-compliance` and are registered by every
service built on SRV records: `svcs.UnknownSRV`, `abstract.RFC6186`,
`abstract.LibravatarServer`, `abstract.SIP`, `abstract.XMPP`, `abstract.LDAP`,
`abstract.MatrixIM`, `abstract.CalDAV`, `abstract.CardDAV` and
`abstract.Kerberos`, each naming the body fields to read. Records are grouped
by owner name first, so two sets of a same service stay independent.

Sync:
- owner not spelled `_service._protocol`, target of `.` next to a real one, invalid target, priority/weight/port outside the uint16 range, port 0 on a real target, target that is a CNAME (error)
- duplicate host and port, in-zone target with no A/AAAA (warning)
- weight of 0 next to non-zero ones at the same priority (info)

### PTR (`svcs.PTR`)

Sync:
- empty or invalid target, target that is a CNAME (error)
- reverse name carrying other records, owner labels that cannot be read as an address in reverse (warning)
- target left relative, PTR published outside of any reverse tree (info)

### SSHFP (`svcs.SSHFPs`)

Sync:
- unknown key algorithm or fingerprint type, non-hexadecimal fingerprint, fingerprint whose length does not match its type, owner name that is a CNAME (error)
- key published with SHA-1 only, DSA key, duplicate, owner with no A/AAAA in the zone (warning)
- service publishing nothing yet (info)

RFC 4255 sec. 5 only trusts these records under DNSSEC, but neither the zone
nor the domain carries a signing state to key a message off, so nothing is
raised about it.

The editor also fills the fingerprints in from the server itself, through
`POST /api/resolver/ssh-hostkeys`. Unlike the other resolver routes, that one
is authenticated and rate limited: it opens a TCP connection to a host and a
port the caller picks, which netguard restricts to globally routable addresses
but which would still make a port prober out of an anonymous endpoint.
